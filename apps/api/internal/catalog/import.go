package catalog

import (
	"context"
	"errors"
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
// import never duplicates them. By default existing records are left alone:
// after go-live they hold edits, images and stock that the legacy data must not
// overwrite. Derived fields are computed exactly as for edits made through the API.

// ImportMode says what an import does with a record that already exists.
type ImportMode int

const (
	// ImportKeep inserts new records only and leaves existing ones untouched.
	ImportKeep ImportMode = iota
	// ImportOverwrite replaces existing records (only for re-runs before go-live).
	ImportOverwrite
)

// ErrImportExists is returned in ImportKeep mode when the record already exists.
var ErrImportExists = errors.New("already imported; kept as is")

// ProductExists reports whether a product with this id exists.
func (s *Service) ProductExists(ctx context.Context, id bson.ObjectID) (bool, error) {
	n, err := s.products.CountDocuments(ctx, bson.M{"_id": id}, options.Count().SetLimit(1))
	return n > 0, err
}

// insertNew inserts doc, mapping a clash on an existing id to ErrImportExists.
func insertNew(ctx context.Context, exists func(context.Context, bson.ObjectID) (bool, error), insert func() error, id bson.ObjectID) error {
	err := insert()
	if err == nil || !mongox.IsDuplicateKey(err) {
		return err
	}
	if found, ferr := exists(ctx, id); ferr == nil && found {
		return ErrImportExists
	}
	return err // another unique index (e.g. the slug) clashed
}

// ImportCategory inserts a category by id (ImportKeep) or replaces it (ImportOverwrite).
func (s *Service) ImportCategory(ctx context.Context, c *Category, mode ImportMode) error {
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
	var err error
	if mode == ImportOverwrite {
		_, err = s.categories.ReplaceOne(ctx, bson.M{"_id": c.ID}, c, options.Replace().SetUpsert(true))
	} else {
		err = insertNew(ctx, s.CategoryExists, func() error {
			_, err := s.categories.InsertOne(ctx, c)
			return err
		}, c.ID)
	}
	if errors.Is(err, ErrImportExists) {
		return err
	}
	if err != nil {
		return fmt.Errorf("import category %s: %w", c.Slug, err)
	}
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

// ImportProduct inserts a product by id (ImportKeep) or replaces it (ImportOverwrite) after
// computing its derived fields (final price, search text, sort keys). An overwrite bumps the
// version, so admin edit forms opened before it get a conflict, and only applies if the product
// did not change since it was read.
func (s *Service) ImportProduct(ctx context.Context, p *Product, mode ImportMode) error {
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
	p.SearchWords = productSearchWords(p.Name, p.Description)
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

	insert := func() error {
		p.Version = 1
		return insertNew(ctx, s.ProductExists, func() error {
			_, err := s.products.InsertOne(ctx, p)
			return err
		}, p.ID)
	}
	var err error
	if mode == ImportOverwrite {
		var existing struct {
			Version int64 `bson:"version"`
		}
		ferr := s.products.FindOne(ctx, bson.M{"_id": p.ID}, options.FindOne().SetProjection(bson.M{"version": 1})).Decode(&existing)
		switch {
		case ferr == nil:
			p.Version = existing.Version + 1
			// Conditional on the version read: an edit made meanwhile is not lost.
			res, rerr := s.products.ReplaceOne(ctx, bson.M{"_id": p.ID, "version": existing.Version}, p)
			if rerr == nil && res.MatchedCount == 0 {
				rerr = errors.New("changed while importing; run the import again")
			}
			err = rerr
		case mongox.IsNotFound(ferr):
			err = insert()
		default:
			err = ferr
		}
	} else {
		err = insert()
	}
	if errors.Is(err, ErrImportExists) {
		return err
	}
	if err != nil {
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
