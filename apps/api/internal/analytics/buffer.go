package analytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// With Redis, views are counted in a per-day Redis hash (one script call per view) and Flush
// moves the totals to MongoDB in one bulk write every few seconds. Every view would otherwise
// update the same few documents (today's totals, "direct", "mobile"), which serialises writes
// under load. Without Redis, or when a Redis call fails, views are written directly.
//
// New dimension values are capped per day (maxKeysPerDay); further values are counted as
// "other", so made-up paths or referrers cannot create unbounded rows.

const (
	// Hash per day: totals and "d|<dim>|<key>" counters. No TTL, so Redis' volatile-lru eviction
	// never drops counts; Flush deletes it.
	bufferPrefix   = "analytics:buf:"
	flushingPrefix = "analytics:flushing:" // a buffer being flushed
	keysPrefix     = "analytics:keys:"     // set of known values per day and dimension
	pendingKey     = "analytics:pending"   // days with buffered counts
	flushingKey    = "analytics:flushing"  // days with a flush in progress (or failed)
	bufferTTL      = 48 * time.Hour

	// OtherKey collects the values of a dimension beyond its daily cap.
	OtherKey = "other"
)

// maxKeysPerDay caps the distinct values stored per day; 0 = no cap (the value set is closed).
var maxKeysPerDay = map[string]int{
	dimPath:    2000,
	dimRef:     500,
	dimCountry: 300,
}

// recordScript counts one view: totals, visitor, and each dimension value (capped).
//
//	KEYS: buffer hash, visitors HyperLogLog, pending days set
//	ARGV: day, visitor id ("" = none), ttl seconds, totals field, then (dim, key, cap) triples
var recordScript = redis.NewScript(`
redis.call('HINCRBY', KEYS[1], ARGV[4], 1)
if ARGV[2] ~= '' then
  redis.call('PFADD', KEYS[2], ARGV[2])
  redis.call('EXPIRE', KEYS[2], ARGV[3])
end
for i = 5, #ARGV, 3 do
  local dim, key, cap = ARGV[i], ARGV[i + 1], tonumber(ARGV[i + 2])
  if cap > 0 then
    local set = '` + keysPrefix + `' .. ARGV[1] .. ':' .. dim
    if redis.call('SADD', set, key) == 1 then
      redis.call('EXPIRE', set, ARGV[3])
      if redis.call('SCARD', set) > cap then
        redis.call('SREM', set, key)
        key = '` + OtherKey + `'
      end
    end
  end
  redis.call('HINCRBY', KEYS[1], 'd|' .. dim .. '|' .. key, 1)
end
redis.call('SADD', KEYS[3], ARGV[1])
return 1
`)

type counter struct{ dim, key string }

// buffer counts a view in Redis. totalsField is "pageViews" or "productViews".
func (s *Service) buffer(ctx context.Context, day, totalsField, visitorID string, counters []counter) error {
	args := []any{day, visitorID, int(bufferTTL / time.Second), totalsField}
	for _, c := range counters {
		args = append(args, c.dim, c.key, maxKeysPerDay[c.dim])
	}
	return recordScript.Run(ctx, s.rdb, []string{bufferPrefix + day, visitorsKey(day), pendingKey}, args...).Err()
}

func visitorsKey(day string) string { return "analytics:uv:" + day }

// Flush moves buffered counts to MongoDB. It runs as a leased job, so one replica flushes at
// a time. A failed flush keeps its data and is retried; a crash after the MongoDB write but
// before the cleanup can count that batch twice (rare and bounded to one batch).
func (s *Service) Flush(ctx context.Context) error {
	if s.rdb == nil {
		return nil
	}
	days, err := s.rdb.SUnion(ctx, pendingKey, flushingKey).Result()
	if err != nil {
		return err
	}
	var errs []error
	for _, day := range days {
		if err := s.flushDay(ctx, day); err != nil {
			errs = append(errs, fmt.Errorf("flush %s: %w", day, err))
		}
	}
	return errors.Join(errs...)
}

func (s *Service) flushDay(ctx context.Context, day string) error {
	flushing := flushingPrefix + day
	// A previous flush of this day that failed or crashed goes first.
	exists, err := s.rdb.Exists(ctx, flushing).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		// Unregister the day before taking its buffer: a view arriving meanwhile registers it
		// again, so nothing is left behind.
		if err := s.rdb.SRem(ctx, pendingKey, day).Err(); err != nil {
			return err
		}
		if err := s.rdb.SAdd(ctx, flushingKey, day).Err(); err != nil {
			return err
		}
		if err := s.rdb.Rename(ctx, bufferPrefix+day, flushing).Err(); err != nil {
			if strings.Contains(err.Error(), "no such key") {
				return s.rdb.SRem(ctx, flushingKey, day).Err()
			}
			return err
		}
	}
	fields, err := s.rdb.HGetAll(ctx, flushing).Result()
	if err != nil {
		return err
	}
	visitors, err := s.rdb.PFCount(ctx, visitorsKey(day)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	if err := s.apply(ctx, day, fields, visitors); err != nil {
		return err
	}
	if err := s.rdb.Del(ctx, flushing).Err(); err != nil {
		return err
	}
	return s.rdb.SRem(ctx, flushingKey, day).Err()
}

// apply adds one buffered batch to the daily totals and dimension counters.
func (s *Service) apply(ctx context.Context, day string, fields map[string]string, visitors int64) error {
	start, err := time.ParseInLocation(time.DateOnly, day, s.loc)
	if err != nil {
		return fmt.Errorf("bad day %q: %w", day, err)
	}
	dayStart := start.UTC()
	totals := bson.M{}
	var dims []mongo.WriteModel
	for field, raw := range fields {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n == 0 {
			continue
		}
		if rest, ok := strings.CutPrefix(field, "d|"); ok {
			dim, key, ok := strings.Cut(rest, "|")
			if !ok {
				continue
			}
			dims = append(dims, mongo.NewUpdateOneModel().
				SetFilter(bson.M{"day": day, "dim": dim, "key": key}).
				SetUpdate(bson.M{"$inc": bson.M{"count": n}, "$setOnInsert": bson.M{"date": dayStart}}).
				SetUpsert(true))
			continue
		}
		totals[field] = n
	}
	update := bson.M{"$setOnInsert": bson.M{"date": dayStart}}
	if len(totals) > 0 {
		update["$inc"] = totals
	}
	if visitors > 0 {
		update["$max"] = bson.M{"visitors": visitors}
	}
	if _, err := s.daily.UpdateOne(ctx, bson.M{"_id": day}, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return err
	}
	const batch = 1000
	for i := 0; i < len(dims); i += batch {
		if _, err := s.dims.BulkWrite(ctx, dims[i:min(i+batch, len(dims))], options.BulkWrite().SetOrdered(false)); err != nil {
			return err
		}
	}
	if len(dims) > 0 {
		slog.DebugContext(ctx, "analytics flushed", "day", day, "counters", len(dims))
	}
	return nil
}
