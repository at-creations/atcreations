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
	"golang.org/x/sync/singleflight"
)

// Cache namespaces. A write bumps its namespace version, which instantly
// orphans every cached entry of that namespace (they then expire by TTL), so
// invalidation never needs KEYS/SCAN.
const (
	NSCatalog   = "catalog"
	NSStore     = "store"
	NSAnalytics = "analytics" // never invalidated: entries just expire
)

const maxCachedBody = 1 << 20 // 1 MiB

// Cache caches successful public GET responses. A nil *Cache is a no-op.
type Cache struct {
	rdb    *redis.Client
	prefix string
	flight singleflight.Group
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

// recorder buffers a handler's response so it can be cached and shared.
type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header { return r.header }
func (r *recorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
}
func (r *recorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(p)
}

func (r *recorder) replay(w http.ResponseWriter) {
	for k, v := range r.header {
		w.Header()[k] = v
	}
	w.WriteHeader(r.status)
	_, _ = w.Write(r.body.Bytes())
}

// Middleware serves cached 200 JSON responses for GET requests in namespace ns and stores
// fresh ones for ttl. Only the query parameters in params make up the key; a request with any
// other parameter bypasses the cache, so junk parameters can neither miss on purpose nor fill
// Redis. Concurrent misses for one key run the handler once per instance (singleflight), so an
// invalidation does not send every waiting request to the database at once. Redis errors bypass
// the cache.
func (c *Cache) Middleware(ns string, ttl time.Duration, params ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(params))
	for _, p := range params {
		allowed[p] = true
	}
	return func(next http.Handler) http.Handler {
		if c == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			for k := range r.URL.Query() {
				if !allowed[k] {
					w.Header().Set("X-Cache", "BYPASS")
					next.ServeHTTP(w, r)
					return
				}
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
			v, _, shared := c.flight.Do(key, func() (any, error) {
				// Detached from the first caller, whose cancellation must not fail the others.
				rec := &recorder{header: http.Header{}}
				next.ServeHTTP(rec, r.WithContext(context.WithoutCancel(ctx)))
				if rec.status == http.StatusOK && rec.body.Len() > 0 && rec.body.Len() < maxCachedBody {
					if err := c.rdb.Set(context.WithoutCancel(ctx), key, rec.body.Bytes(), ttl).Err(); err != nil {
						slog.DebugContext(ctx, "cache store failed", "err", err)
					}
				}
				return rec, nil
			})
			rec := v.(*recorder)
			w.Header().Set("X-Cache", map[bool]string{true: "SHARED", false: "MISS"}[shared])
			rec.replay(w)
		})
	}
}
