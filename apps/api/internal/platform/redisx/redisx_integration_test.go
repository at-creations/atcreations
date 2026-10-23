//go:build integration

// docker compose -f docker-compose.dev.yml up -d redis
// REDIS_TEST_URL="redis://:dev-redis-password@127.0.0.1:6379/15" go test -tags integration ./internal/platform/redisx/
package redisx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func connect(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("REDIS_TEST_URL")
	if url == "" {
		t.Skip("REDIS_TEST_URL not set")
	}
	rdb, err := Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = rdb.FlushDB(context.Background()).Err()
		_ = rdb.Close()
	})
	return rdb
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	return rec
}

func TestCacheHitMissAndInvalidate(t *testing.T) {
	c := NewCache(connect(t))
	var calls atomic.Int32
	h := c.Middleware(NSCatalog, time.Minute, "a", "b", "fail")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("fail") != "" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprintf(w, `{"data":"v%d"}`, calls.Load())
	}))

	first := get(t, h, "/v1/products?b=2&a=1")
	if first.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("first request: %s", first.Header().Get("X-Cache"))
	}
	// Same query in a different order must hit the same entry.
	second := get(t, h, "/v1/products?a=1&b=2")
	if second.Header().Get("X-Cache") != "HIT" || second.Body.String() != first.Body.String() || calls.Load() != 1 {
		t.Fatalf("expected cache hit with identical body, got %s %q (calls=%d)", second.Header().Get("X-Cache"), second.Body.String(), calls.Load())
	}

	c.Invalidate(context.Background(), NSCatalog)
	third := get(t, h, "/v1/products?a=1&b=2")
	if third.Header().Get("X-Cache") != "MISS" || calls.Load() != 2 {
		t.Fatalf("invalidate did not take effect: %s calls=%d", third.Header().Get("X-Cache"), calls.Load())
	}

	// Errors are never cached.
	get(t, h, "/v1/products?fail=1")
	if rec := get(t, h, "/v1/products?fail=1"); rec.Code != http.StatusInternalServerError || calls.Load() != 4 {
		t.Fatalf("error response was cached (calls=%d)", calls.Load())
	}
}

func TestCacheBypassesUnknownParamsAndSharesMisses(t *testing.T) {
	c := NewCache(connect(t))
	var calls atomic.Int32
	release := make(chan struct{})
	h := c.Middleware(NSCatalog, time.Minute, "page")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("page") == "slow" {
			<-release
		}
		_, _ = w.Write([]byte(`{"data":1}`))
	}))
	// Unknown parameters never create cache entries.
	for i := range 3 {
		if rec := get(t, h, fmt.Sprintf("/v1/products?junk=%d", i)); rec.Header().Get("X-Cache") != "BYPASS" {
			t.Fatalf("junk param: X-Cache = %q", rec.Header().Get("X-Cache"))
		}
	}
	calls.Store(0)
	// Concurrent misses for one key run the handler once.
	done := make(chan *httptest.ResponseRecorder, 5)
	for range 5 {
		go func() { done <- get(t, h, "/v1/products?page=slow") }()
	}
	time.Sleep(200 * time.Millisecond)
	close(release)
	for range 5 {
		if rec := <-done; rec.Code != http.StatusOK || rec.Body.String() != `{"data":1}` {
			t.Fatalf("waiter got %d %q", rec.Code, rec.Body.String())
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("handler ran %d times for concurrent misses, want 1", n)
	}
}

func TestNilCacheIsPassthrough(t *testing.T) {
	var c *Cache
	h := c.Middleware(NSCatalog, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	if rec := get(t, h, "/x"); rec.Body.String() != "ok" || rec.Header().Get("X-Cache") != "" {
		t.Fatal("nil cache must be a transparent no-op")
	}
	c.Invalidate(context.Background(), NSCatalog) // must not panic
}

func TestLockIsExclusiveAndTokenSafe(t *testing.T) {
	rdb := connect(t)
	ctx := context.Background()

	release, err := Lock(ctx, rdb, "t", 300*time.Millisecond, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(ctx, rdb, "t", time.Second, 100*time.Millisecond); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("second holder: err = %v, want ErrLockHeld", err)
	}

	// After expiry another holder takes over; the stale release must not free it.
	time.Sleep(400 * time.Millisecond)
	release2, err := Lock(ctx, rdb, "t", time.Second, 0)
	if err != nil {
		t.Fatal(err)
	}
	release() // stale token: no effect
	if _, err := Lock(ctx, rdb, "t", time.Second, 100*time.Millisecond); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("stale release freed someone else's lock: %v", err)
	}
	release2()
	if r, err := Lock(ctx, rdb, "t", time.Second, 0); err != nil {
		t.Fatalf("lock should be free after release: %v", err)
	} else {
		r()
	}
}
