// Package analytics records first-party, cookieless storefront traffic and builds the admin's
// sales and traffic report.
//
// Privacy: no cookies, no IP addresses and no user agents are stored. Unique visitors are
// counted with a keyed hash of (day, IP, user agent) added to a Redis HyperLogLog that expires
// after two days; only the resulting per-day count reaches MongoDB. Visitors sending
// `DNT: 1` or `Sec-GPC: 1`, and known bots, are not counted.
package analytics

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

const (
	// Traffic rollups are kept for a little over a year.
	retention = 400 * 24 * time.Hour
	// HyperLogLog keys only need to outlive the day they count.
	visitorKeyTTL = 48 * time.Hour

	maxPathLen = 200

	dimPath    = "path"
	dimRef     = "ref"
	dimDevice  = "device"
	dimCountry = "country"
	dimProduct = "product"

	// DirectReferrer is reported for visits without an external referrer.
	DirectReferrer = "direct"
)

// Service owns the analytics collections:
//
//	analytics_daily  {_id: "2006-01-02", date, pageViews, visitors, productViews}
//	analytics_dims   {day, dim, key, count, date}  (one counter per day and dimension value)
type Service struct {
	daily    *mongo.Collection
	dims     *mongo.Collection
	orders   *mongo.Collection
	products *mongo.Collection
	rdb      *redis.Client
	pepper   []byte
	loc      *time.Location
	// Host of the storefront; referrals from it count as internal navigation.
	ownHost  string
	imageURL func(key string) string
	now      func() time.Time
}

type Options struct {
	Location      *time.Location
	StorefrontURL string
	// ImageURL turns a stored image key into its public URL.
	ImageURL func(key string) string
}

func New(db *mongo.Database, rdb *redis.Client, pepper string, opts Options) *Service {
	loc := opts.Location
	if loc == nil {
		loc = time.UTC
	}
	own := ""
	if u, err := url.Parse(opts.StorefrontURL); err == nil {
		own = strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	}
	imageURL := opts.ImageURL
	if imageURL == nil {
		imageURL = func(string) string { return "" }
	}
	return &Service{
		daily:    db.Collection("analytics_daily"),
		dims:     db.Collection("analytics_dims"),
		orders:   db.Collection("orders"),
		products: db.Collection("products"),
		rdb:      rdb,
		pepper:   []byte(pepper),
		loc:      loc,
		ownHost:  own,
		imageURL: imageURL,
		now:      time.Now,
	}
}

func (s *Service) EnsureIndexes(ctx context.Context) error {
	ttl := mongox.Int32(int32(retention / time.Second))
	if err := mongox.EnsureIndexes(ctx, s.daily,
		mongox.Index{Keys: bson.D{{Key: "date", Value: 1}}, TTL: ttl, Name: "date_ttl"},
	); err != nil {
		return err
	}
	return mongox.EnsureIndexes(ctx, s.dims,
		mongox.Index{Keys: bson.D{{Key: "day", Value: 1}, {Key: "dim", Value: 1}, {Key: "key", Value: 1}}, Unique: true},
		mongox.Index{Keys: bson.D{{Key: "dim", Value: 1}, {Key: "date", Value: 1}}},
		mongox.Index{Keys: bson.D{{Key: "date", Value: 1}}, TTL: ttl, Name: "date_ttl"},
	)
}

// PageView is one storefront navigation as reported by the browser and the request.
type PageView struct {
	Path      string
	Referrer  string
	IP        string
	UserAgent string
	// Country is an ISO 3166-1 alpha-2 code from a trusted edge proxy, or "".
	Country string
	// OptOut is set when the browser sent DNT or Global Privacy Control.
	OptOut bool
}

// RecordPageView counts a page view. Opted-out visitors and bots are silently ignored.
func (s *Service) RecordPageView(ctx context.Context, v PageView) error {
	if v.OptOut || IsBot(v.UserAgent) {
		slog.DebugContext(ctx, "analytics: page view ignored", "optOut", v.OptOut, "bot", !v.OptOut)
		return nil
	}
	path, ok := NormalizePath(v.Path)
	if !ok {
		slog.DebugContext(ctx, "analytics: page view ignored", "reason", "invalid path")
		return nil
	}
	now := s.now()
	day, dayStart := s.dayOf(now)

	counters := []counter{
		{dimPath, path},
		{dimRef, s.ReferrerHost(v.Referrer)},
		{dimDevice, Device(v.UserAgent)},
	}
	if c := NormalizeCountry(v.Country); c != "" {
		counters = append(counters, counter{dimCountry, c})
	}
	if s.rdb != nil {
		err := s.buffer(ctx, day, "pageViews", s.visitorID(day, v.IP, v.UserAgent), counters)
		if err == nil {
			return nil
		}
		slog.WarnContext(ctx, "analytics: buffering failed, writing directly", "err", err)
	}

	inc := bson.M{"pageViews": 1}
	update := bson.M{"$inc": inc, "$setOnInsert": bson.M{"date": dayStart}}
	if n, ok := s.countVisitor(ctx, day, v.IP, v.UserAgent); ok {
		update["$max"] = bson.M{"visitors": n}
	}
	if _, err := s.daily.UpdateOne(ctx, bson.M{"_id": day}, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return fmt.Errorf("record page view: %w", err)
	}

	models := make([]mongo.WriteModel, 0, len(counters))
	for _, c := range counters {
		models = append(models, s.dimIncrement(day, dayStart, c.dim, c.key))
	}
	if _, err := s.dims.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
		return fmt.Errorf("record page view dimensions: %w", err)
	}
	return nil
}

// RecordProductView adds a (already de-duplicated) product view to the daily rollup.
// Failures are logged, never returned: analytics must not break the storefront.
func (s *Service) RecordProductView(ctx context.Context, productID bson.ObjectID) {
	day, dayStart := s.dayOf(s.now())
	if s.rdb != nil {
		err := s.buffer(ctx, day, "productViews", "", []counter{{dimProduct, productID.Hex()}})
		if err == nil {
			return
		}
		slog.WarnContext(ctx, "analytics: buffering failed, writing directly", "err", err)
	}
	_, err := s.daily.UpdateOne(ctx, bson.M{"_id": day},
		bson.M{"$inc": bson.M{"productViews": 1}, "$setOnInsert": bson.M{"date": dayStart}},
		options.UpdateOne().SetUpsert(true))
	if err == nil {
		_, err = s.dims.BulkWrite(ctx, []mongo.WriteModel{s.dimIncrement(day, dayStart, dimProduct, productID.Hex())})
	}
	if err != nil {
		slog.WarnContext(ctx, "analytics: product view not recorded", "err", err)
	}
}

func (s *Service) dimIncrement(day string, dayStart time.Time, dim, key string) mongo.WriteModel {
	return mongo.NewUpdateOneModel().
		SetFilter(bson.M{"day": day, "dim": dim, "key": key}).
		SetUpdate(bson.M{"$inc": bson.M{"count": 1}, "$setOnInsert": bson.M{"date": dayStart}}).
		SetUpsert(true)
}

// countVisitor adds the visitor to the day's HyperLogLog and returns the day's estimate.
// ok is false when Redis is unavailable (visitors are then simply not counted).
func (s *Service) countVisitor(ctx context.Context, day, ip, ua string) (int64, bool) {
	if s.rdb == nil {
		return 0, false
	}
	id := s.visitorID(day, ip, ua)
	key := visitorsKey(day)
	pipe := s.rdb.TxPipeline()
	pipe.PFAdd(ctx, key, id)
	pipe.Expire(ctx, key, visitorKeyTTL)
	count := pipe.PFCount(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		slog.WarnContext(ctx, "analytics: visitor count unavailable", "err", err)
		return 0, false
	}
	return count.Val(), true
}

// visitorID is a keyed hash of (day, IP, user agent): it cannot be linked across days.
func (s *Service) visitorID(day, ip, ua string) string {
	mac := hmac.New(sha256.New, s.pepper)
	_, _ = fmt.Fprintf(mac, "visitor|%s|%s|%s", day, ip, ua)
	return hex.EncodeToString(mac.Sum(nil)[:16])
}

// dayOf returns the store-local day key and the UTC instant at which that day starts.
func (s *Service) dayOf(t time.Time) (string, time.Time) {
	local := t.In(s.loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc)
	return start.Format(time.DateOnly), start.UTC()
}

// ---------- Input normalisation ----------

// NormalizePath keeps only the path of a storefront URL: no query, no fragment, no trailing
// slash, at most maxPathLen characters of printable text. ok is false for anything else.
func NormalizePath(p string) (string, bool) {
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") {
		return "", false
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	if p == "" {
		p = "/"
	}
	if len(p) > maxPathLen {
		return "", false
	}
	for _, r := range p {
		if !unicode.IsPrint(r) || unicode.IsSpace(r) {
			return "", false
		}
	}
	return strings.ToLower(p), true
}

// ReferrerHost reduces a referrer URL to its host ("google.com"), or DirectReferrer when it is
// empty, malformed or the storefront itself.
func (s *Service) ReferrerHost(ref string) string {
	if ref == "" || len(ref) > 1000 {
		return DirectReferrer
	}
	u, err := url.Parse(ref)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return DirectReferrer
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if host == "" || host == s.ownHost || len(host) > 100 {
		return DirectReferrer
	}
	return host
}

// NormalizeCountry accepts two ASCII letters ("ca", "VN"); Cloudflare's "XX"/"T1" are dropped.
func NormalizeCountry(c string) string {
	c = strings.ToUpper(strings.TrimSpace(c))
	if len(c) != 2 || c == "XX" || c == "T1" {
		return ""
	}
	for _, r := range c {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return c
}

// Device classifies a user agent as "mobile", "tablet" or "desktop".
func Device(ua string) string {
	l := strings.ToLower(ua)
	switch {
	case strings.Contains(l, "ipad") || strings.Contains(l, "tablet") ||
		(strings.Contains(l, "android") && !strings.Contains(l, "mobile")):
		return "tablet"
	case strings.Contains(l, "mobi") || strings.Contains(l, "iphone") || strings.Contains(l, "ipod"):
		return "mobile"
	default:
		return "desktop"
	}
}

var botMarkers = []string{
	"bot", "crawl", "spider", "slurp", "facebookexternalhit", "headless", "lighthouse",
	"preview", "python-requests", "curl/", "wget/", "go-http-client", "monitor", "pingdom",
}

// IsBot reports whether a user agent is empty or belongs to a crawler, preview fetcher or script.
func IsBot(ua string) bool {
	if strings.TrimSpace(ua) == "" {
		return true
	}
	l := strings.ToLower(ua)
	for _, m := range botMarkers {
		if strings.Contains(l, m) {
			return true
		}
	}
	return false
}
