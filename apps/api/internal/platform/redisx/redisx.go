// Package redisx connects to Redis and provides small primitives built on it:
// a distributed lock and a namespaced, version-invalidated response cache.
// Every helper is nil-safe so the API also runs without Redis (development).
package redisx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Connect parses a redis:// or rediss:// URL and verifies the connection.
// Timeouts are short so callers fail fast and fall back instead of hanging.
func Connect(ctx context.Context, rawURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	opts.ClientName = "atcreations-api"
	opts.DialTimeout = 2 * time.Second
	opts.ReadTimeout = 500 * time.Millisecond
	opts.WriteTimeout = 500 * time.Millisecond
	opts.MaxRetries = 1
	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

// ErrLockHeld is returned when another holder owns the lock.
var ErrLockHeld = errors.New("lock is held by another request")

// releaseScript deletes the key only if it still holds our token, so a lock
// that expired and was re-acquired by someone else is never released by us.
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0`)

// Lock acquires a named lock for at most ttl, retrying until wait elapses.
// The returned release function is safe to call once.
func Lock(ctx context.Context, rdb *redis.Client, name string, ttl, wait time.Duration) (func(), error) {
	var b [16]byte
	_, _ = rand.Read(b[:])
	token := hex.EncodeToString(b[:])
	key := "lock:" + name
	deadline := time.Now().Add(wait)
	for {
		ok, err := rdb.SetNX(ctx, key, token, ttl).Result()
		if err != nil {
			return nil, fmt.Errorf("acquire lock %s: %w", name, err)
		}
		if ok {
			return func() {
				_ = releaseScript.Run(context.WithoutCancel(ctx), rdb, []string{key}, token).Err()
			}, nil
		}
		if time.Now().After(deadline) {
			return nil, ErrLockHeld
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}
