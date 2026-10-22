package redisx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache namespaces. A write bumps its namespace version, which instantly
// orphans every cached entry of that namespace (they then expire by TTL), so
// invalidation never needs KEYS/SCAN.
const (
	NSCatalog = "catalog"
	NSStore   = "store"
)

const maxCachedBody = 1 << 20 // 1 MiB

// Cache caches successful public GET responses. A nil *Cache is a no-op.
type Cache struct {
	rdb    *redis.Client
	prefix string
}

func NewCache(rdb *redis.Client) *Cache {
	if rdb == nil {
		return nil
	}
	return &Cache{rdb: rdb, prefix: "cache:"}
}

func (c *Cache) versionKey(ns string) string { return c.prefix + ns + ":ver" }

// Invalidate bumps the version of each namespace. Failures are logged: the
// entries still expire by TTL, so staleness is bounded.
func (c *Cache) Invalidate(ctx context.Context, namespaces ...string) {
	if c == nil {
		return
	}
	pipe := c.rdb.Pipeline()
	for _, ns := range namespaces {
		pipe.Incr(ctx, c.versionKey(ns))
	}
	if _, err := pipe.Exec(context.WithoutCancel(ctx)); err != nil {
		slog.WarnContext(ctx, "cache invalidation failed; entries expire by TTL", "err", err, "namespaces", namespaces)
	}
}

// requestKey canonicalizes path and query (sorted) so equivalent requests share an entry.
func requestKey(r *http.Request) string {
	q := r.URL.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(r.URL.Path)
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		for _, v := range vals {
			b.WriteString("&" + k + "=" + v)
		}
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16])
}

type captureWriter struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (w *captureWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.buf.Len()+len(p) <= maxCachedBody {
		w.buf.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

// Middleware serves cached 200 JSON responses for GET requests in namespace
// ns and stores fresh ones for ttl. Redis errors bypass the cache.
func (c *Cache) Middleware(ns string, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if c == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()
			ver, err := c.rdb.Get(ctx, c.versionKey(ns)).Result()
			if err != nil && !errors.Is(err, redis.Nil) {
				slog.DebugContext(ctx, "cache unavailable, bypassing", "err", err)
				next.ServeHTTP(w, r)
				return
			}
			if ver == "" {
				ver = "0"
			}
			key := c.prefix + ns + ":" + ver + ":" + requestKey(r)
			if body, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				_, _ = w.Write(body)
				return
			}
			w.Header().Set("X-Cache", "MISS")
			cw := &captureWriter{ResponseWriter: w}
			next.ServeHTTP(cw, r)
			if cw.status == http.StatusOK && cw.buf.Len() > 0 && cw.buf.Len() < maxCachedBody {
				if err := c.rdb.Set(context.WithoutCancel(ctx), key, cw.buf.Bytes(), ttl).Err(); err != nil {
					slog.DebugContext(ctx, "cache store failed", "err", err)
				}
			}
		})
	}
}
