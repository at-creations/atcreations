package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/textx"
)

// Data import (used by cmd/migrate). Records keep their ids, so re-running an
// import replaces documents instead of duplicating them. Derived fields are
// computed exactly as for edits made through the API.

// ImportCategory upserts a category by id.
func (s *Service) ImportCategory(ctx context.Context, c *Category) error {
	if c.ID.IsZero() || !textx.ValidSlug(c.Slug) || strings.TrimSpace(c.Name.EN) == "" {
		return fmt.Errorf("category %s: id, valid slug and English name are required", c.ID.Hex())
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	if _, err := s.categories.ReplaceOne(ctx, bson.M{"_id": c.ID}, c, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("import category %s: %w", c.Slug, err)
	}
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

// ImportProduct upserts a product by id after computing its derived fields
// (final price, search text, sort keys). It never lowers an existing version,
// so admin edit forms opened before a re-import get a conflict, not a silent
// overwrite.
func (s *Service) ImportProduct(ctx context.Context, p *Product) error {
	if p.ID.IsZero() || !textx.ValidSlug(p.Slug) || strings.TrimSpace(p.Name.EN) == "" {
		return fmt.Errorf("product %s: id, valid slug and English name are required", p.ID.Hex())
	}
	if p.DiscountPercent < 0 || p.DiscountPercent > 90 || p.Stock < 0 || len(p.Images) > MaxProductImages {
		return fmt.Errorf("product %s: discount, stock or image count out of range", p.ID.Hex())
	}
	if p.PriceCents != nil && *p.PriceCents < 0 {
		return fmt.Errorf("product %s: negative price", p.ID.Hex())
	}
	p.FinalPriceCents = FinalPrice(p.PriceCents, p.DiscountPercent)
	p.HasPrice = p.PriceCents != nil
	p.Search = textx.SearchText(p.Name.EN, p.Name.VI, p.Description.EN, p.Description.VI)
	p.SortNameEN = textx.Fold(p.Name.EN)
	p.SortNameVI = textx.Fold(firstNonEmpty(p.Name.VI, p.Name.EN))
	p.CategoryIDs = nonNil(dedupe(p.CategoryIDs))
	p.Images = nonNil(p.Images)
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}

	var existing struct {
		Version int64 `bson:"version"`
	}
	err := s.products.FindOne(ctx, bson.M{"_id": p.ID}, options.FindOne().SetProjection(bson.M{"version": 1})).Decode(&existing)
	switch {
	case err == nil:
		p.Version = existing.Version + 1
	case mongox.IsNotFound(err):
		p.Version = 1
	default:
		return err
	}
	if _, err := s.products.ReplaceOne(ctx, bson.M{"_id": p.ID}, p, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("import product %s: %w", p.Slug, err)
	}
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

// ProductSlugOwner returns the id of the product using slug, if any.
func (s *Service) ProductSlugOwner(ctx context.Context, slug string) (bson.ObjectID, bool, error) {
	var doc struct {
		ID bson.ObjectID `bson:"_id"`
	}
	err := s.products.FindOne(ctx, bson.M{"slug": slug}, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&doc)
	if mongox.IsNotFound(err) {
		return bson.ObjectID{}, false, nil
	}
	return doc.ID, err == nil, err
}

// ProductImages returns a product's current images (nil if it does not exist).
func (s *Service) ProductImages(ctx context.Context, id bson.ObjectID) ([]Image, error) {
	var doc struct {
		Images []Image `bson:"images"`
	}
	err := s.products.FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"images": 1})).Decode(&doc)
	if mongox.IsNotFound(err) {
		return nil, nil
	}
	return doc.Images, err
}
