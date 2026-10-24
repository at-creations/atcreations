package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
)

// Image uploads are processed asynchronously (internal/uploads + the image worker). The catalog
// side only reserves room for them and attaches finished images.

var (
	// ErrOwnerGone: the product or category was deleted while its image was processing.
	ErrOwnerGone = errors.New("owner no longer exists")
	// ErrImagesFull: the product already has MaxProductImages images.
	ErrImagesFull = errors.New("image limit reached")
)

// OwnerDeleted is called after a product or category is deleted, so pending uploads can be
// cancelled and their originals removed.
func (s *Service) OnOwnerDeleted(fn func(ctx context.Context, kind string, id bson.ObjectID)) {
	s.onOwnerDeleted = fn
}

func (s *Service) ownerDeleted(ctx context.Context, kind string, id bson.ObjectID) {
	if s.onOwnerDeleted != nil {
		s.onOwnerDeleted(ctx, kind, id)
	}
}

// ReserveProductImageSlot counts one more image in flight for the product, refusing when its
// images plus in-flight uploads would exceed MaxProductImages. The check and the increment are
// one atomic update, so concurrent uploads cannot overshoot the limit.
func (s *Service) ReserveProductImageSlot(ctx context.Context, id bson.ObjectID) error {
	res, err := s.products.UpdateOne(ctx, bson.M{
		"_id": id,
		"$expr": bson.M{"$lt": bson.A{
			bson.M{"$add": bson.A{bson.M{"$size": "$images"}, bson.M{"$ifNull": bson.A{"$pendingImages", 0}}}},
			MaxProductImages,
		}},
	}, bson.M{"$inc": bson.M{"pendingImages": 1}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 1 {
		return nil
	}
	if n, err := s.products.CountDocuments(ctx, bson.M{"_id": id}, options.Count().SetLimit(1)); err != nil {
		return err
	} else if n == 0 {
		return httpx.ErrNotFound
	}
	return httpx.Validation(httpx.FieldError{Field: "file", Message: fmt.Sprintf("a product can have at most %d images (including uploads in progress)", MaxProductImages)})
}

// ReleaseProductImageSlot undoes ReserveProductImageSlot once the upload has finished either way.
func (s *Service) ReleaseProductImageSlot(ctx context.Context, id bson.ObjectID) error {
	_, err := s.products.UpdateOne(ctx, bson.M{"_id": id, "pendingImages": bson.M{"$gt": 0}},
		bson.M{"$inc": bson.M{"pendingImages": -1}})
	return err
}

// AttachProductImage appends a processed image and releases the upload's reserved slot in the
// same atomic update, so the image count and the in-flight count never overlap. The image id is
// the upload job's id, which makes the call idempotent: a job retried after a lost
// acknowledgement cannot add the image (or release the slot) twice.
func (s *Service) AttachProductImage(ctx context.Context, actor string, id bson.ObjectID, img Image) error {
	res, err := s.products.UpdateOne(ctx, bson.M{
		"_id":       id,
		"images.id": bson.M{"$ne": img.ID},
		fmt.Sprintf("images.%d", MaxProductImages-1): bson.M{"$exists": false},
	}, mongo.Pipeline{{{Key: "$set", Value: bson.M{
		"images":        bson.M{"$concatArrays": bson.A{"$images", bson.A{img}}},
		"pendingImages": bson.M{"$max": bson.A{0, bson.M{"$subtract": bson.A{bson.M{"$ifNull": bson.A{"$pendingImages", 0}}, 1}}}},
		"updatedAt":     time.Now().UTC(),
		"updatedBy":     actor,
	}}}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		var p struct {
			Images []Image `bson:"images"`
		}
		err := s.products.FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"images.id": 1})).Decode(&p)
		switch {
		case mongox.IsNotFound(err):
			return ErrOwnerGone
		case err != nil:
			return err
		}
		for _, existing := range p.Images {
			if existing.ID == img.ID {
				return nil // already attached by an earlier attempt
			}
		}
		return ErrImagesFull
	}
	s.audit.Record(ctx, "product.image.add", "product", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

// SetCategoryImageKey replaces a category's image with an already stored object and deletes the
// previous one.
func (s *Service) SetCategoryImageKey(ctx context.Context, id bson.ObjectID, img Image) error {
	var before Category
	err := s.categories.FindOneAndUpdate(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"image": img, "updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&before)
	if mongox.IsNotFound(err) {
		return ErrOwnerGone
	}
	if err != nil {
		return err
	}
	if before.Image != nil && before.Image.Key != img.Key {
		s.deleteObjects(ctx, before.Image.Key)
	}
	s.audit.Record(ctx, "category.image.set", "category", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

// CategoryExists reports whether a category exists (for accepting an upload).
func (s *Service) CategoryExists(ctx context.Context, id bson.ObjectID) (bool, error) {
	n, err := s.categories.CountDocuments(ctx, bson.M{"_id": id}, options.Count().SetLimit(1))
	return n > 0, err
}

// ImageNeedingVariants is a stored image without smaller copies yet (saved before they existed).
type ImageNeedingVariants struct {
	OwnerKind string // "product" or "category"
	OwnerID   bson.ObjectID
	ImageID   string
	Key       string
}

// ImagesWithoutVariants returns up to limit images that have no variants recorded, skipping
// the image keys in skip (already attempted by the caller).
func (s *Service) ImagesWithoutVariants(ctx context.Context, limit int, skip map[string]bool) ([]ImageNeedingVariants, error) {
	var out []ImageNeedingVariants
	cur, err := s.products.Find(ctx, bson.M{"images": bson.M{"$elemMatch": bson.M{"variants": nil}}},
		options.Find().SetProjection(bson.M{"images": 1}))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	for cur.Next(ctx) && len(out) < limit {
		var p struct {
			ID     bson.ObjectID `bson:"_id"`
			Images []Image       `bson:"images"`
		}
		if err := cur.Decode(&p); err != nil {
			return nil, err
		}
		for _, img := range p.Images {
			if img.Variants == nil && !skip[img.Key] && len(out) < limit {
				out = append(out, ImageNeedingVariants{OwnerKind: "product", OwnerID: p.ID, ImageID: img.ID, Key: img.Key})
			}
		}
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	if len(out) >= limit {
		return out, nil
	}
	cats, err := s.categories.Find(ctx, bson.M{"image": bson.M{"$type": "object"}, "image.variants": nil},
		options.Find().SetProjection(bson.M{"image": 1}))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cats.Close(ctx) }()
	for cats.Next(ctx) && len(out) < limit {
		var c struct {
			ID    bson.ObjectID `bson:"_id"`
			Image Image         `bson:"image"`
		}
		if err := cats.Decode(&c); err != nil {
			return nil, err
		}
		if !skip[c.Image.Key] {
			out = append(out, ImageNeedingVariants{OwnerKind: "category", OwnerID: c.ID, ImageID: c.Image.ID, Key: c.Image.Key})
		}
	}
	return out, cats.Err()
}

// SetImageVariants records the variants of one stored image (matched by key, so an image
// replaced meanwhile is left alone). An empty list marks an image too small for any variant.
func (s *Service) SetImageVariants(ctx context.Context, img ImageNeedingVariants, variants []ImageVariant) error {
	if variants == nil {
		variants = []ImageVariant{}
	}
	var err error
	switch img.OwnerKind {
	case "product":
		_, err = s.products.UpdateOne(ctx, bson.M{"_id": img.OwnerID, "images": bson.M{"$elemMatch": bson.M{"id": img.ImageID, "key": img.Key}}},
			bson.M{"$set": bson.M{"images.$.variants": variants}})
	case "category":
		_, err = s.categories.UpdateOne(ctx, bson.M{"_id": img.OwnerID, "image.key": img.Key},
			bson.M{"$set": bson.M{"image.variants": variants}})
	default:
		return fmt.Errorf("unknown owner kind %q", img.OwnerKind)
	}
	if err == nil {
		s.cache.Invalidate(ctx, redisx.NSCatalog)
	}
	return err
}
