package catalog

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Category product counts are stored on each category (productCount) and adjusted with $inc
// whenever a product joins or leaves it, so listing categories never scans the products.
// RecountCategoryProducts corrects any drift (e.g. an adjustment lost to a crash) and backfills
// counts; it runs as a scheduled job.

// categoryDiff returns the categories only in after (joined) and only in before (left).
func categoryDiff(before, after []bson.ObjectID) (joined, left []bson.ObjectID) {
	in := func(id bson.ObjectID, list []bson.ObjectID) bool {
		for _, x := range list {
			if x == id {
				return true
			}
		}
		return false
	}
	for _, id := range after {
		if !in(id, before) {
			joined = append(joined, id)
		}
	}
	for _, id := range before {
		if !in(id, after) {
			left = append(left, id)
		}
	}
	return joined, left
}

// adjustCategoryCounts applies +1 to joined and -1 to left categories. Best effort: a failure
// is logged and repaired by the next recount.
func (s *Service) adjustCategoryCounts(ctx context.Context, joined, left []bson.ObjectID) {
	if len(joined)+len(left) == 0 {
		return
	}
	models := make([]mongo.WriteModel, 0, len(joined)+len(left))
	for _, id := range joined {
		models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": id}).SetUpdate(bson.M{"$inc": bson.M{"productCount": 1}}))
	}
	for _, id := range left {
		models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": id}).SetUpdate(bson.M{"$inc": bson.M{"productCount": -1}}))
	}
	if _, err := s.categories.BulkWrite(context.WithoutCancel(ctx), models, options.BulkWrite().SetOrdered(false)); err != nil {
		slog.WarnContext(ctx, "category count update failed; the next recount repairs it", "err", err)
	}
}

// RecountCategoryProducts sets every category's productCount from the products themselves.
// Each count is an index-only scan of {categoryIds, status}.
func (s *Service) RecountCategoryProducts(ctx context.Context) error {
	cur, err := s.categories.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1, "productCount": 1}))
	if err != nil {
		return err
	}
	var cats []struct {
		ID    bson.ObjectID `bson:"_id"`
		Count int64         `bson:"productCount"`
	}
	if err := cur.All(ctx, &cats); err != nil {
		return err
	}
	var models []mongo.WriteModel
	for _, c := range cats {
		n, err := s.products.CountDocuments(ctx, bson.M{"categoryIds": c.ID})
		if err != nil {
			return err
		}
		if n != c.Count {
			models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": c.ID}).SetUpdate(bson.M{"$set": bson.M{"productCount": n}}))
		}
	}
	if len(models) == 0 {
		return nil
	}
	if _, err := s.categories.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
		return err
	}
	slog.InfoContext(ctx, "category product counts corrected", "categories", len(models))
	return nil
}
