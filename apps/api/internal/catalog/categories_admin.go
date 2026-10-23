package catalog

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
)

// AdminCategoryDTO adds what the dashboard shows next to each category.
type AdminCategoryDTO struct {
	CategoryDTO
	// Products (any status) in the category.
	ProductCount int64 `json:"productCount"`
}

// ListAdminCategories returns every category with its product count, in display order.
func (s *Service) ListAdminCategories(ctx context.Context) ([]AdminCategoryDTO, error) {
	cats, err := s.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	cur, err := s.products.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$unwind", Value: "$categoryIds"}},
		{{Key: "$group", Value: bson.M{"_id": "$categoryIds", "n": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
		N  int64         `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	counts := make(map[bson.ObjectID]int64, len(rows))
	for _, r := range rows {
		counts[r.ID] = r.N
	}
	out := make([]AdminCategoryDTO, len(cats))
	for i, c := range cats {
		out[i] = AdminCategoryDTO{CategoryDTO: c, ProductCount: counts[c.ID]}
	}
	return out, nil
}

// ReorderCategories sets the display order from a complete list of category ids. Requiring
// every id keeps a stale browser tab from silently scrambling the order.
func (s *Service) ReorderCategories(ctx context.Context, ids []bson.ObjectID) error {
	seen := make(map[bson.ObjectID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return httpx.Validation(httpx.FieldError{Field: "categoryIds", Message: "contains duplicates"})
		}
		seen[id] = true
	}
	total, err := s.categories.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	found, err := s.categories.CountDocuments(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return err
	}
	if int64(len(ids)) != total || found != total {
		return httpx.Conflict("the categories changed; reload and try again")
	}
	now := time.Now().UTC()
	models := make([]mongo.WriteModel, len(ids))
	for i, id := range ids {
		models[i] = mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": id}).
			SetUpdate(bson.M{"$set": bson.M{"sortOrder": (i + 1) * 10, "updatedAt": now}})
	}
	if _, err := s.categories.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
		return err
	}
	s.audit.Record(ctx, "category.reorder", "category", "")
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

func (s *Service) HandleAdminCategories(w http.ResponseWriter, r *http.Request) error {
	cats, err := s.ListAdminCategories(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, cats)
	return nil
}

func (s *Service) HandleReorderCategories(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		CategoryIDs []string `json:"categoryIds" validate:"required,max=500,dive,len=24,hexadecimal"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	ids := make([]bson.ObjectID, len(in.CategoryIDs))
	for i, raw := range in.CategoryIDs {
		id, err := bson.ObjectIDFromHex(raw)
		if err != nil {
			return httpx.Validation(httpx.FieldError{Field: "categoryIds", Message: "must be category ids"})
		}
		ids[i] = id
	}
	if err := s.ReorderCategories(r.Context(), ids); err != nil {
		return err
	}
	cats, err := s.ListAdminCategories(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, cats)
	return nil
}
