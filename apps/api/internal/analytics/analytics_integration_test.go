//go:build integration

//	MONGO_TEST_URI="mongodb://127.0.0.1:27017/?replicaSet=rs0&directConnection=true" \
//	  go test -tags integration ./internal/analytics/
package analytics

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

func TestReportIntegration(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx := context.Background()
	client, db, err := mongox.Connect(ctx, uri, "atc_it_"+bson.NewObjectID().Hex(), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	loc, _ := time.LoadLocation("America/Vancouver")
	s := New(db, nil, "pepper", Options{
		Location: loc, StorefrontURL: "https://shop.test",
		ImageURL: func(k string) string { return "https://cdn.test/" + k },
	})
	now := time.Date(2026, 10, 20, 18, 0, 0, 0, time.UTC) // 11:00 in Vancouver
	s.now = func() time.Time { return now }
	if err := s.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	productID := bson.NewObjectID()
	if _, err := db.Collection("products").InsertOne(ctx, bson.M{
		"_id": productID, "name": bson.M{"en": "Vase"}, "images": bson.A{bson.M{"id": "1", "key": "products/vase.webp"}},
	}); err != nil {
		t.Fatal(err)
	}
	order := func(at time.Time, status string, cents int64, qty int) bson.M {
		return bson.M{
			"_id": bson.NewObjectID(), "createdAt": at, "status": status, "totalCents": cents,
			"items": bson.A{bson.M{"productId": productID, "quantity": qty, "lineTotalCents": cents, "name": bson.M{"en": "Vase"}}},
		}
	}
	_, err = db.Collection("orders").InsertMany(ctx, []any{
		order(now.Add(-1*time.Hour), "pending", 5000, 1),
		order(now.Add(-26*time.Hour), "completed", 7000, 2),
		order(now.Add(-2*time.Hour), "cancelled", 9900, 3),  // not revenue, but in the breakdown
		order(now.AddDate(0, 0, -10), "completed", 1000, 1), // previous 7-day period
	})
	if err != nil {
		t.Fatal(err)
	}

	browser := "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile/15E148"
	for _, v := range []PageView{
		{Path: "/en", Referrer: "https://www.google.com/", UserAgent: browser, Country: "CA"},
		{Path: "/en/products/vase?x=1", Referrer: "https://shop.test/en", UserAgent: browser, Country: "CA"},
		{Path: "/en", UserAgent: "Googlebot/2.1"},       // bot: ignored
		{Path: "/en", UserAgent: browser, OptOut: true}, // DNT: ignored
		{Path: "not-a-path", UserAgent: browser},        // invalid: ignored
	} {
		if err := s.RecordPageView(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	s.RecordProductView(ctx, productID)
	s.RecordProductView(ctx, productID)

	rep, err := s.Report(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Totals.Orders != 2 || rep.Totals.RevenueCents != 12000 || rep.Totals.AvgOrderCents != 6000 {
		t.Errorf("totals = %+v, want 2 orders / 12000 revenue", rep.Totals)
	}
	if rep.Previous.Orders != 1 || rep.Previous.RevenueCents != 1000 {
		t.Errorf("previous = %+v", rep.Previous)
	}
	if rep.Totals.PageViews != 2 || rep.Totals.ProductViews != 2 {
		t.Errorf("traffic totals = %+v, want 2 page views and 2 product views", rep.Totals)
	}
	if len(rep.Daily) != 7 || rep.Daily[6].Date != "2026-10-20" || rep.Daily[6].Orders != 1 || rep.Daily[5].Orders != 1 {
		t.Errorf("daily = %+v", rep.Daily)
	}
	statuses := map[string]int64{}
	for _, sc := range rep.OrdersByStatus {
		statuses[sc.Status] = sc.Count
	}
	if statuses["cancelled"] != 1 || statuses["pending"] != 1 || statuses["completed"] != 1 {
		t.Errorf("ordersByStatus = %+v", rep.OrdersByStatus)
	}
	if len(rep.TopSelling) != 1 || rep.TopSelling[0].UnitsSold != 3 || rep.TopSelling[0].Views != 2 {
		t.Errorf("topSelling = %+v", rep.TopSelling)
	}
	if len(rep.TopViewed) != 1 || rep.TopViewed[0].ImageURL != "https://cdn.test/products/vase.webp" {
		t.Errorf("topViewed = %+v", rep.TopViewed)
	}
	refs := map[string]int64{}
	for _, c := range rep.Referrers {
		refs[c.Key] = c.Count
	}
	if refs["google.com"] != 1 || refs[DirectReferrer] != 1 {
		t.Errorf("referrers = %+v", rep.Referrers)
	}
	if len(rep.Devices) != 1 || rep.Devices[0] != (Count{"mobile", 2}) {
		t.Errorf("devices = %+v", rep.Devices)
	}
	if len(rep.Countries) != 1 || rep.Countries[0] != (Count{"CA", 2}) {
		t.Errorf("countries = %+v", rep.Countries)
	}
}

func TestBufferedCountsAndKeyCap(t *testing.T) {
	uri, redisURL := os.Getenv("MONGO_TEST_URI"), os.Getenv("REDIS_TEST_URL")
	if uri == "" || redisURL == "" {
		t.Skip("MONGO_TEST_URI and REDIS_TEST_URL are required")
	}
	ctx := context.Background()
	client, db, err := mongox.Connect(ctx, uri, "atc_it_"+bson.NewObjectID().Hex(), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	opts.DB = 14 // not the database other packages' tests flush concurrently
	rdb := redis.NewClient(opts)
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
		_ = rdb.FlushDB(context.Background()).Err()
		_ = rdb.Close()
	})
	_ = rdb.FlushDB(ctx).Err()
	s := New(db, rdb, "pepper", Options{StorefrontURL: "https://shop.test"})
	if err := s.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	old := maxKeysPerDay[dimPath]
	maxKeysPerDay[dimPath] = 3
	t.Cleanup(func() { maxKeysPerDay[dimPath] = old })

	browser := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	for i := range 5 { // 5 distinct paths, cap 3: two land in "other"
		if err := s.RecordPageView(ctx, PageView{Path: fmt.Sprintf("/en/p%d", i), UserAgent: browser, IP: fmt.Sprintf("10.0.0.%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordPageView(ctx, PageView{Path: "/en/p0", UserAgent: browser, IP: "10.0.0.0"}); err != nil {
		t.Fatal(err)
	}
	s.RecordProductView(ctx, bson.NewObjectID())

	// Nothing reaches MongoDB until the flush.
	if n, _ := db.Collection("analytics_daily").CountDocuments(ctx, bson.M{}); n != 0 {
		t.Fatalf("daily docs before flush = %d, want 0", n)
	}
	if err := s.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	rep, err := s.Report(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Totals.PageViews != 6 || rep.Totals.ProductViews != 1 || rep.Totals.Visitors != 5 {
		t.Fatalf("totals = %+v, want 6 page views, 1 product view, 5 visitors", rep.Totals)
	}
	pages := map[string]int64{}
	for _, c := range rep.Pages {
		pages[c.Key] = c.Count
	}
	if len(pages) != 4 || pages[OtherKey] != 2 || pages["/en/p0"] != 2 {
		t.Fatalf("pages = %v, want 3 capped paths plus other=2", pages)
	}
	// A second flush with nothing new changes nothing.
	if err := s.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if again, _ := s.Report(ctx, 1); again.Totals.PageViews != 6 {
		t.Fatalf("page views after empty flush = %d", again.Totals.PageViews)
	}
}
