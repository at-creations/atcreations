package analytics

import (
	"context"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/sync/errgroup"

	"github.com/atcreations/atcreations/apps/api/internal/catalog"
)

const (
	topProducts = 5
	topCounts   = 8
)

// Orders in these statuses never brought in money.
var excludedStatuses = bson.A{"cancelled", "refunded"}

type Totals struct {
	RevenueCents  int64 `json:"revenueCents"`
	Orders        int64 `json:"orders"`
	AvgOrderCents int64 `json:"avgOrderCents"`
	Visitors      int64 `json:"visitors"`
	PageViews     int64 `json:"pageViews"`
	ProductViews  int64 `json:"productViews"`
}

type Day struct {
	Date         string `json:"date"`
	RevenueCents int64  `json:"revenueCents"`
	Orders       int64  `json:"orders"`
	Visitors     int64  `json:"visitors"`
	PageViews    int64  `json:"pageViews"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type ProductStat struct {
	ProductID    string                `json:"productId"`
	Name         catalog.LocalizedText `json:"name"`
	ImageURL     string                `json:"imageUrl,omitempty"`
	UnitsSold    int64                 `json:"unitsSold"`
	RevenueCents int64                 `json:"revenueCents"`
	Views        int64                 `json:"views"`
}

type Count struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type Report struct {
	From           string        `json:"from"`
	To             string        `json:"to"`
	Timezone       string        `json:"timezone"`
	Totals         Totals        `json:"totals"`
	Previous       Totals        `json:"previous"`
	Daily          []Day         `json:"daily"`
	OrdersByStatus []StatusCount `json:"ordersByStatus"`
	TopSelling     []ProductStat `json:"topSelling"`
	TopViewed      []ProductStat `json:"topViewed"`
	Pages          []Count       `json:"pages"`
	Referrers      []Count       `json:"referrers"`
	Devices        []Count       `json:"devices"`
	Countries      []Count       `json:"countries"`
}

// period is a half-open range of whole store-local days [start, end).
type period struct{ start, end time.Time }

// periods returns the last `days` days including today, and the same-length period before it.
func (s *Service) periods(days int) (cur, prev period) {
	local := s.now().In(s.loc)
	todayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc)
	cur = period{start: todayStart.AddDate(0, 0, -(days - 1)), end: todayStart.AddDate(0, 0, 1)}
	prev = period{start: cur.start.AddDate(0, 0, -days), end: cur.start}
	return cur, prev
}

// Report builds the sales and traffic report for the last `days` days (today included).
func (s *Service) Report(ctx context.Context, days int) (*Report, error) {
	cur, prev := s.periods(days)
	rep := &Report{
		From:     cur.start.Format(time.DateOnly),
		To:       cur.end.AddDate(0, 0, -1).Format(time.DateOnly),
		Timezone: s.loc.String(),
	}

	var (
		salesByDay              map[string]Day
		trafficByDay            map[string]Day
		curTraffic, prevTraffic Totals
		productViews            map[string]int64
		dimCounts               map[string][]Count
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { salesByDay, rep.Totals, err = s.sales(gctx, cur, true); return })
	g.Go(func() (err error) { _, rep.Previous, err = s.sales(gctx, prev, false); return })
	g.Go(func() (err error) { trafficByDay, curTraffic, err = s.traffic(gctx, cur); return })
	g.Go(func() (err error) { _, prevTraffic, err = s.traffic(gctx, prev); return })
	g.Go(func() (err error) { rep.OrdersByStatus, err = s.statusBreakdown(gctx, cur); return })
	g.Go(func() (err error) { rep.TopSelling, err = s.topSelling(gctx, cur); return })
	g.Go(func() (err error) { productViews, dimCounts, err = s.dimensions(gctx, cur); return })
	if err := g.Wait(); err != nil {
		return nil, err
	}
	addTraffic(&rep.Totals, curTraffic)
	addTraffic(&rep.Previous, prevTraffic)

	// One entry per day, oldest first, zero-filled.
	for d := cur.start; d.Before(cur.end); d = d.AddDate(0, 0, 1) {
		key := d.Format(time.DateOnly)
		day := Day{Date: key}
		if v, ok := salesByDay[key]; ok {
			day.RevenueCents, day.Orders = v.RevenueCents, v.Orders
		}
		if v, ok := trafficByDay[key]; ok {
			day.Visitors, day.PageViews = v.Visitors, v.PageViews
		}
		rep.Daily = append(rep.Daily, day)
	}

	for i := range rep.TopSelling {
		rep.TopSelling[i].Views = productViews[rep.TopSelling[i].ProductID]
	}
	var err error
	if rep.TopViewed, err = s.topViewed(ctx, productViews); err != nil {
		return nil, err
	}
	rep.Pages = nonNil(dimCounts[dimPath])
	rep.Referrers = nonNil(dimCounts[dimRef])
	rep.Devices = nonNil(dimCounts[dimDevice])
	rep.Countries = nonNil(dimCounts[dimCountry])
	rep.OrdersByStatus = nonNil(rep.OrdersByStatus)
	rep.TopSelling = nonNil(rep.TopSelling)
	return rep, nil
}

// sales returns revenue and order counts per day (when byDay) and in total.
func (s *Service) sales(ctx context.Context, p period, byDay bool) (map[string]Day, Totals, error) {
	groupKey := any(nil)
	if byDay {
		groupKey = bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt", "timezone": s.loc.String()}}
	}
	cur, err := s.orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": p.start, "$lt": p.end},
			"status":    bson.M{"$nin": excludedStatuses},
		}}},
		{{Key: "$group", Value: bson.M{"_id": groupKey, "revenue": bson.M{"$sum": "$totalCents"}, "n": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, Totals{}, err
	}
	var rows []struct {
		ID      *string `bson:"_id"`
		Revenue int64   `bson:"revenue"`
		N       int64   `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, Totals{}, err
	}
	out := make(map[string]Day, len(rows))
	var t Totals
	for _, r := range rows {
		t.RevenueCents += r.Revenue
		t.Orders += r.N
		if r.ID != nil {
			out[*r.ID] = Day{RevenueCents: r.Revenue, Orders: r.N}
		}
	}
	if t.Orders > 0 {
		t.AvgOrderCents = t.RevenueCents / t.Orders
	}
	return out, t, nil
}

func addTraffic(dst *Totals, t Totals) {
	dst.Visitors, dst.PageViews, dst.ProductViews = t.Visitors, t.PageViews, t.ProductViews
}

// traffic reads the daily rollups: per day, and the period's traffic totals.
func (s *Service) traffic(ctx context.Context, p period) (map[string]Day, Totals, error) {
	var t Totals
	cur, err := s.daily.Find(ctx, bson.M{"date": bson.M{"$gte": p.start.UTC(), "$lt": p.end.UTC()}})
	if err != nil {
		return nil, t, err
	}
	var rows []struct {
		ID           string `bson:"_id"`
		PageViews    int64  `bson:"pageViews"`
		Visitors     int64  `bson:"visitors"`
		ProductViews int64  `bson:"productViews"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, t, err
	}
	out := make(map[string]Day, len(rows))
	for _, r := range rows {
		t.PageViews += r.PageViews
		t.Visitors += r.Visitors
		t.ProductViews += r.ProductViews
		out[r.ID] = Day{PageViews: r.PageViews, Visitors: r.Visitors}
	}
	return out, t, nil
}

func (s *Service) statusBreakdown(ctx context.Context, p period) ([]StatusCount, error) {
	cur, err := s.orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"createdAt": bson.M{"$gte": p.start, "$lt": p.end}}}},
		{{Key: "$group", Value: bson.M{"_id": "$status", "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Status string `bson:"_id"`
		Count  int64  `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]StatusCount, len(rows))
	for i, r := range rows {
		out[i] = StatusCount(r)
	}
	return out, nil
}

func (s *Service) topSelling(ctx context.Context, p period) ([]ProductStat, error) {
	cur, err := s.orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": p.start, "$lt": p.end},
			"status":    bson.M{"$nin": excludedStatuses},
		}}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.M{
			"_id":     "$items.productId",
			"units":   bson.M{"$sum": "$items.quantity"},
			"revenue": bson.M{"$sum": "$items.lineTotalCents"},
			// The order snapshot keeps the name and image even if the product was deleted.
			"name":  bson.M{"$last": "$items.name"},
			"image": bson.M{"$last": "$items.imageUrl"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "revenue", Value: -1}, {Key: "units", Value: -1}}}},
		{{Key: "$limit", Value: topProducts}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID      bson.ObjectID         `bson:"_id"`
		Units   int64                 `bson:"units"`
		Revenue int64                 `bson:"revenue"`
		Name    catalog.LocalizedText `bson:"name"`
		Image   string                `bson:"image"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]ProductStat, len(rows))
	for i, r := range rows {
		out[i] = ProductStat{ProductID: r.ID.Hex(), Name: r.Name, ImageURL: r.Image, UnitsSold: r.Units, RevenueCents: r.Revenue}
	}
	return out, nil
}

// dimensions sums the per-day counters of the period: product views by product id, and the
// top values of every other dimension.
func (s *Service) dimensions(ctx context.Context, p period) (map[string]int64, map[string][]Count, error) {
	cur, err := s.dims.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"date": bson.M{"$gte": p.start.UTC(), "$lt": p.end.UTC()}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"dim": "$dim", "key": "$key"},
			"count": bson.M{"$sum": "$count"},
		}}},
	})
	if err != nil {
		return nil, nil, err
	}
	var rows []struct {
		ID struct {
			Dim string `bson:"dim"`
			Key string `bson:"key"`
		} `bson:"_id"`
		Count int64 `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, nil, err
	}
	views := map[string]int64{}
	counts := map[string][]Count{}
	for _, r := range rows {
		if r.ID.Dim == dimProduct {
			views[r.ID.Key] = r.Count
			continue
		}
		counts[r.ID.Dim] = append(counts[r.ID.Dim], Count{Key: r.ID.Key, Count: r.Count})
	}
	for dim, list := range counts {
		counts[dim] = topN(list, topCounts)
	}
	return views, counts, nil
}

// topViewed resolves the most viewed product ids to names and cover images.
func (s *Service) topViewed(ctx context.Context, views map[string]int64) ([]ProductStat, error) {
	list := make([]Count, 0, len(views))
	for id, n := range views {
		list = append(list, Count{Key: id, Count: n})
	}
	list = topN(list, topProducts)
	if len(list) == 0 {
		return []ProductStat{}, nil
	}
	ids := make([]bson.ObjectID, 0, len(list))
	for _, c := range list {
		if id, err := bson.ObjectIDFromHex(c.Key); err == nil {
			ids = append(ids, id)
		}
	}
	cur, err := s.products.Find(ctx, bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.M{"name": 1, "images": bson.M{"$slice": 1}}))
	if err != nil {
		return nil, err
	}
	var docs []struct {
		ID     bson.ObjectID         `bson:"_id"`
		Name   catalog.LocalizedText `bson:"name"`
		Images []catalog.Image       `bson:"images"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	byID := make(map[string]int, len(docs))
	for i, d := range docs {
		byID[d.ID.Hex()] = i
	}
	out := make([]ProductStat, 0, len(list))
	for _, c := range list {
		i, ok := byID[c.Key]
		if !ok {
			continue // deleted since
		}
		d := docs[i]
		stat := ProductStat{ProductID: c.Key, Name: d.Name, Views: c.Count}
		if len(d.Images) > 0 {
			stat.ImageURL = s.imageURL(d.Images[0].Key)
		}
		out = append(out, stat)
	}
	return out, nil
}

func topN(list []Count, n int) []Count {
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Key < list[j].Key
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
