package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	clientIPKey
)

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID propagates a sane incoming X-Request-ID or generates a new one.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID.MatchString(id) {
			var b [12]byte
			_, _ = rand.Read(b[:])
			id = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// ClientIP resolves the real client address. X-Forwarded-For is only honoured
// when the direct peer is a trusted proxy, and then the right-most address
// that is not itself a trusted proxy is used (left-most entries are
// client-controlled and spoofable).
func ClientIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	isTrusted := func(a netip.Addr) bool {
		for _, p := range trusted {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip, err := netip.ParseAddr(host)
			if err == nil && isTrusted(ip.Unmap()) {
				hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
				for i := len(hops) - 1; i >= 0; i-- {
					hop, perr := netip.ParseAddr(strings.TrimSpace(hops[i]))
					if perr != nil {
						break
					}
					ip = hop.Unmap()
					if !isTrusted(ip) {
						break
					}
				}
			}
			addr := "unknown"
			if ip.IsValid() {
				addr = ip.Unmap().String()
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, addr)))
		})
	}
}

func ClientIPFrom(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey).(string)
	return ip
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// AccessLog writes one structured line per request. Query strings are not
// logged because they can carry personal data (search terms, emails).
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		level := slog.LevelInfo
		if rec.status >= 500 {
			level = slog.LevelError
		}
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			level = slog.LevelDebug
		}
		slog.Log(r.Context(), level, "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", ClientIPFrom(r.Context()),
			"request_id", RequestIDFrom(r.Context()),
		)
	})
}

// Recover converts panics into 500 problems and logs the stack.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				slog.ErrorContext(r.Context(), "panic", "value", v, "stack", string(debug.Stack()))
				WriteError(w, r, NewError(http.StatusInternalServerError, "an unexpected error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders sets defensive headers suitable for a JSON API.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cross-Origin-Resource-Policy", "same-site")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// NoStore marks responses as uncacheable (used for authenticated routes).
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Authorization")
		next.ServeHTTP(w, r)
	})
}

// PublicCache marks catalogue responses as briefly cacheable by CDNs.
func PublicCache(maxAge time.Duration) func(http.Handler) http.Handler {
	v := "public, max-age=" + strconv.Itoa(int(maxAge.Seconds())) + ", stale-while-revalidate=300"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.Header().Set("Cache-Control", v)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout gives each request a deadline, so a slow query fails the request (503) instead of
// tying up the server until the write timeout. Requests for which skip returns true (long
// uploads) are left to the server's own timeouts.
func Timeout(d time.Duration, skip func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip != nil && skip(r) {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
