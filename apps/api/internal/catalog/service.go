package catalog

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/textx"
)

const (
	ProductImagePrefix  = "products/"
	CategoryImagePrefix = "categories/"
	viewDedupWindow     = 15 * time.Minute
)

type Service struct {
	products   *mongo.Collection
	categories *mongo.Collection
	views      *mongo.Collection
	storage    media.Storage
	audit      *audit.Log
	pepper     []byte
	cache      *redisx.Cache // nil-safe; invalidated after every catalogue write
}

func NewService(db *mongo.Database, storage media.Storage, audit *audit.Log, cache *redisx.Cache, pepper string) *Service {
	return &Service{
		cache:      cache,
		products:   db.Collection("products"),
		categories: db.Collection("categories"),
		views:      db.Collection("product_views"),
		storage:    storage,
		audit:      audit,
		pepper:     []byte(pepper),
	}
}

func (s *Service) EnsureIndexes(ctx context.Context) error {
	err := mongox.EnsureIndexes(ctx, s.products,
		mongox.Index{Keys: bson.D{{Key: "slug", Value: 1}}, Unique: true},
		mongox.Index{Keys: bson.D{{Key: "status", Value: 1}, {Key: "featured", Value: -1}, {Key: "recentViews", Value: -1}, {Key: "views", Value: -1}, {Key: "createdAt", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "status", Value: 1}, {Key: "hasPrice", Value: -1}, {Key: "finalPriceCents", Value: 1}}},
		mongox.Index{Keys: bson.D{{Key: "categoryIds", Value: 1}, {Key: "status", Value: 1}}},
		mongox.Index{Keys: bson.D{{Key: "stock", Value: 1}}},
	)
	if err != nil {
		return err
	}
	err = mongox.EnsureIndexes(ctx, s.categories,
		mongox.Index{Keys: bson.D{{Key: "slug", Value: 1}}, Unique: true},
		mongox.Index{Keys: bson.D{{Key: "sortOrder", Value: 1}, {Key: "name.en", Value: 1}}},
	)
	if err != nil {
		return err
	}
	return mongox.EnsureIndexes(ctx, s.views,
		mongox.Index{Keys: bson.D{{Key: "productId", Value: 1}, {Key: "at", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "at", Value: 1}}, TTL: mongox.Int32(7 * 24 * 3600), Name: "at_ttl"},
	)
}

// ---------- DTO mapping ----------

func (s *Service) imageDTO(img *Image) *ImageDTO {
	if img == nil {
		return nil
	}
	return &ImageDTO{ID: img.ID, Key: img.Key, URL: s.storage.URL(img.Key), Width: img.Width, Height: img.Height, Alt: img.Alt}
}

func (s *Service) categoryDTO(c *Category) CategoryDTO {
	return CategoryDTO{
		ID: c.ID, Slug: c.Slug, Name: c.Name, Description: c.Description,
		Image: s.imageDTO(c.Image), SortOrder: c.SortOrder, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

// productDTOs maps products and resolves their category references in one query.
func (s *Service) productDTOs(ctx context.Context, ps []Product) ([]ProductDTO, error) {
	idSet := map[bson.ObjectID]struct{}{}
	for _, p := range ps {
		for _, id := range p.CategoryIDs {
			idSet[id] = struct{}{}
		}
	}
	refs := map[bson.ObjectID]CategoryRef{}
	if len(idSet) > 0 {
		ids := make([]bson.ObjectID, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		cur, err := s.categories.Find(ctx, bson.M{"_id": bson.M{"$in": ids}},
			options.Find().SetProjection(bson.M{"slug": 1, "name": 1}))
		if err != nil {
			return nil, err
		}
		var cats []Category
		if err := cur.All(ctx, &cats); err != nil {
			return nil, err
		}
		for _, c := range cats {
			refs[c.ID] = CategoryRef{ID: c.ID, Slug: c.Slug, Name: c.Name}
		}
	}
	out := make([]ProductDTO, len(ps))
	for i, p := range ps {
		d := ProductDTO{
			ID: p.ID, Slug: p.Slug, Status: p.Status, Name: p.Name, Description: p.Description,
			PriceCents: p.PriceCents, FinalPriceCents: p.FinalPriceCents, Currency: Currency,
			DiscountPercent: p.DiscountPercent, Stock: p.Stock, Featured: p.Featured,
			CategoryIDs: nonNil(p.CategoryIDs), Categories: []CategoryRef{}, Images: make([]ImageDTO, len(p.Images)),
			Views: p.Views, Version: p.Version, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		}
		for _, id := range p.CategoryIDs {
			if ref, ok := refs[id]; ok {
				d.Categories = append(d.Categories, ref)
			}
		}
		for j := range p.Images {
			d.Images[j] = *s.imageDTO(&p.Images[j])
		}
		out[i] = d
	}
	return out, nil
}

func (s *Service) productDTO(ctx context.Context, p *Product) (ProductDTO, error) {
	ds, err := s.productDTOs(ctx, []Product{*p})
	if err != nil {
		return ProductDTO{}, err
	}
	return ds[0], nil
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// ---------- Queries ----------

type ProductQuery struct {
	Published     bool // restrict to published (public API)
	Text          string
	Status        string
	CategoryID    *bson.ObjectID
	Uncategorized bool
	Featured      *bool
	InStock       *bool
	MinPrice      *int64
	MaxPrice      *int64
	Sort          string
	Locale        string
	Page          httpx.Page
}

func (q ProductQuery) filter() bson.M {
	f := bson.M{}
	if q.Published {
		f["status"] = StatusPublished
	} else if q.Status != "" {
		f["status"] = q.Status
	}
	if q.CategoryID != nil {
		f["categoryIds"] = *q.CategoryID
	} else if q.Uncategorized {
		f["categoryIds"] = bson.M{"$size": 0}
	}
	if q.Featured != nil {
		f["featured"] = *q.Featured
	}
	if q.InStock != nil {
		if *q.InStock {
			f["stock"] = bson.M{"$gt": 0}
		} else {
			f["stock"] = 0
		}
	}
	if q.MinPrice != nil || q.MaxPrice != nil {
		r := bson.M{}
		if q.MinPrice != nil {
			r["$gte"] = *q.MinPrice
		}
		if q.MaxPrice != nil {
			r["$lte"] = *q.MaxPrice
		}
		f["finalPriceCents"] = r
	}
	// Every search term must appear in the folded haystack (AND semantics).
	if terms := strings.Fields(textx.Fold(q.Text)); len(terms) > 0 {
		and := bson.A{}
		for _, t := range terms[:min(len(terms), 8)] {
			and = append(and, bson.M{"search": mongox.ContainsPattern(t)})
		}
		f["$and"] = and
	}
	return f
}

func (q ProductQuery) sort() bson.D {
	nameField := "sortNameEn"
	if q.Locale == "vi" {
		nameField = "sortNameVi"
	}
	switch q.Sort {
	case "newest":
		return bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}
	case "oldest":
		return bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}
	case "price_asc":
		return bson.D{{Key: "hasPrice", Value: -1}, {Key: "finalPriceCents", Value: 1}, {Key: "_id", Value: 1}}
	case "price_desc":
		return bson.D{{Key: "hasPrice", Value: -1}, {Key: "finalPriceCents", Value: -1}, {Key: "_id", Value: 1}}
	case "name_asc":
		return bson.D{{Key: nameField, Value: 1}, {Key: "_id", Value: 1}}
	case "name_desc":
		return bson.D{{Key: nameField, Value: -1}, {Key: "_id", Value: 1}}
	case "stock_asc":
		return bson.D{{Key: "stock", Value: 1}, {Key: "_id", Value: 1}}
	case "popular":
		return bson.D{{Key: "views", Value: -1}, {Key: "_id", Value: 1}}
	default: // relevance
		return bson.D{{Key: "featured", Value: -1}, {Key: "recentViews", Value: -1}, {Key: "views", Value: -1}, {Key: "createdAt", Value: -1}, {Key: "_id", Value: 1}}
	}
}

func (s *Service) ListProducts(ctx context.Context, q ProductQuery) ([]ProductDTO, httpx.PageMeta, error) {
	f := q.filter()
	total, err := s.products.CountDocuments(ctx, f)
	if err != nil {
		return nil, httpx.PageMeta{}, err
	}
	cur, err := s.products.Find(ctx, f, options.Find().SetSort(q.sort()).SetSkip(q.Page.Skip()).SetLimit(int64(q.Page.Size)))
	if err != nil {
		return nil, httpx.PageMeta{}, err
	}
	var ps []Product
	if err := cur.All(ctx, &ps); err != nil {
		return nil, httpx.PageMeta{}, err
	}
	dtos, err := s.productDTOs(ctx, ps)
	return dtos, q.Page.Meta(total), err
}

func (s *Service) findProduct(ctx context.Context, filter bson.M) (*Product, error) {
	var p Product
	if err := s.products.FindOne(ctx, filter).Decode(&p); err != nil {
		if mongox.IsNotFound(err) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *Service) CategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	var c Category
	if err := s.categories.FindOne(ctx, bson.M{"slug": slug}).Decode(&c); err != nil {
		if mongox.IsNotFound(err) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]CategoryDTO, error) {
	cur, err := s.categories.Find(ctx, bson.M{}, options.Find().
		SetSort(bson.D{{Key: "sortOrder", Value: 1}, {Key: "name.en", Value: 1}}).SetLimit(500))
	if err != nil {
		return nil, err
	}
	var cs []Category
	if err := cur.All(ctx, &cs); err != nil {
		return nil, err
	}
	out := make([]CategoryDTO, len(cs))
	for i := range cs {
		out[i] = s.categoryDTO(&cs[i])
	}
	return out, nil
}

// ---------- Product writes ----------

func (s *Service) validateProductInput(ctx context.Context, in *ProductInput) error {
	var fields []httpx.FieldError
	in.Name.EN, in.Name.VI = strings.TrimSpace(in.Name.EN), strings.TrimSpace(in.Name.VI)
	in.Slug = strings.TrimSpace(in.Slug)
	if in.Name.EN == "" {
		fields = append(fields, httpx.FieldError{Field: "name.en", Message: "is required"})
	}
	if in.Slug != "" && !textx.ValidSlug(in.Slug) {
		fields = append(fields, httpx.FieldError{Field: "slug", Message: "must contain only lowercase letters, digits and single dashes"})
	}
	in.CategoryIDs = dedupe(in.CategoryIDs)
	if len(in.CategoryIDs) > 0 {
		n, err := s.categories.CountDocuments(ctx, bson.M{"_id": bson.M{"$in": in.CategoryIDs}})
		if err != nil {
			return err
		}
		if int(n) != len(in.CategoryIDs) {
			fields = append(fields, httpx.FieldError{Field: "categoryIds", Message: "contains unknown categories"})
		}
	}
	if len(fields) > 0 {
		return httpx.Validation(fields...)
	}
	return nil
}

func dedupe(ids []bson.ObjectID) []bson.ObjectID {
	seen := map[bson.ObjectID]bool{}
	out := []bson.ObjectID{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func applyProductInput(p *Product, in *ProductInput) {
	p.Status = in.Status
	p.Name = LocalizedText(in.Name)
	p.Description = LocalizedText{EN: strings.TrimSpace(in.Description.EN), VI: strings.TrimSpace(in.Description.VI)}
	p.PriceCents = in.PriceCents
	p.DiscountPercent = in.DiscountPercent
	p.FinalPriceCents = FinalPrice(in.PriceCents, in.DiscountPercent)
	p.HasPrice = in.PriceCents != nil
	p.Stock = in.Stock
	p.Featured = in.Featured
	p.CategoryIDs = in.CategoryIDs
	p.Search = textx.SearchText(p.Name.EN, p.Name.VI, p.Description.EN, p.Description.VI)
	p.SortNameEN = textx.Fold(p.Name.EN)
	p.SortNameVI = textx.Fold(firstNonEmpty(p.Name.VI, p.Name.EN))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// insertWithSlug inserts doc, retrying with numbered suffixes when an
// auto-generated slug collides. Explicit slugs report a conflict instead.
func insertWithSlug(ctx context.Context, coll *mongo.Collection, base string, explicit bool, insert func(slug string) any) error {
	for i := 1; i <= 25; i++ {
		slug := base
		if i > 1 {
			slug = base + "-" + strconv.Itoa(i)
		}
		_, err := coll.InsertOne(ctx, insert(slug))
		if err == nil {
			return nil
		}
		if !mongox.IsDuplicateKey(err) {
			return err
		}
		if explicit {
			return httpx.Conflict("slug is already in use")
		}
	}
	return httpx.Conflict("could not generate a unique slug; please provide one")
}

func (s *Service) CreateProduct(ctx context.Context, actor string, in *ProductInput) (ProductDTO, error) {
	if err := s.validateProductInput(ctx, in); err != nil {
		return ProductDTO{}, err
	}
	now := time.Now().UTC()
	p := &Product{ID: bson.NewObjectID(), Images: []Image{}, Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: actor, UpdatedBy: actor}
	applyProductInput(p, in)
	base := in.Slug
	if base == "" {
		base = textx.Slugify(in.Name.EN)
		if base == "" {
			base = "product"
		}
	}
	err := insertWithSlug(ctx, s.products, base, in.Slug != "", func(slug string) any {
		p.Slug = slug
		return p
	})
	if err != nil {
		return ProductDTO{}, err
	}
	s.audit.Record(ctx, "product.create", "product", p.ID.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.productDTO(ctx, p)
}

func (s *Service) UpdateProduct(ctx context.Context, actor string, id bson.ObjectID, in *ProductInput) (ProductDTO, error) {
	if in.Version == nil {
		return ProductDTO{}, httpx.Validation(httpx.FieldError{Field: "version", Message: "is required"})
	}
	if err := s.validateProductInput(ctx, in); err != nil {
		return ProductDTO{}, err
	}
	p, err := s.findProduct(ctx, bson.M{"_id": id})
	if err != nil {
		return ProductDTO{}, err
	}
	applyProductInput(p, in)
	if in.Slug != "" {
		p.Slug = in.Slug
	}
	p.UpdatedAt, p.UpdatedBy = time.Now().UTC(), actor
	set := bson.M{
		"slug": p.Slug, "status": p.Status, "name": p.Name, "description": p.Description,
		"priceCents": p.PriceCents, "discountPercent": p.DiscountPercent, "finalPriceCents": p.FinalPriceCents,
		"hasPrice": p.HasPrice, "stock": p.Stock, "featured": p.Featured, "categoryIds": p.CategoryIDs,
		"search": p.Search, "sortNameEn": p.SortNameEN, "sortNameVi": p.SortNameVI,
		"updatedAt": p.UpdatedAt, "updatedBy": p.UpdatedBy,
	}
	// Compare-and-swap on version: a concurrent edit or a stock change from an
	// order makes this a 409 instead of silently overwriting live stock.
	res, err := s.products.UpdateOne(ctx, bson.M{"_id": id, "version": *in.Version},
		bson.M{"$set": set, "$inc": bson.M{"version": 1}})
	if mongox.IsDuplicateKey(err) {
		return ProductDTO{}, httpx.Conflict("slug is already in use")
	}
	if err != nil {
		return ProductDTO{}, err
	}
	if res.MatchedCount == 0 {
		return ProductDTO{}, httpx.Conflict("this product was changed by someone else (or its stock changed); reload and try again")
	}
	p.Version = *in.Version + 1
	s.audit.Record(ctx, "product.update", "product", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.productDTO(ctx, p)
}

func (s *Service) DeleteProduct(ctx context.Context, id bson.ObjectID) error {
	var p Product
	err := s.products.FindOneAndDelete(ctx, bson.M{"_id": id}).Decode(&p)
	if mongox.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	// Image objects are left for the nightly sweep, which keeps any that
	// past orders still reference.
	s.audit.Record(ctx, "product.delete", "product", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

// deleteObjects removes objects best-effort; the nightly orphan sweep catches failures.
func (s *Service) deleteObjects(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	if err := s.storage.Delete(context.WithoutCancel(ctx), keys...); err != nil {
		slog.WarnContext(ctx, "object delete failed; orphan sweep will retry", "err", err, "count", len(keys))
	}
}

// ---------- Product images ----------

func (s *Service) AddProductImage(ctx context.Context, actor string, id bson.ObjectID, img *media.Processed, alt *LocalizedText) (ProductDTO, error) {
	if _, err := s.findProduct(ctx, bson.M{"_id": id}); err != nil {
		return ProductDTO{}, err
	}
	key := media.NewKey(ProductImagePrefix+id.Hex(), img.Ext)
	if err := s.storage.Put(ctx, key, img.Data, img.ContentType); err != nil {
		return ProductDTO{}, err
	}
	rec := Image{ID: media.NewID(), Key: key, Width: img.Width, Height: img.Height, Alt: alt}
	// The array-index condition makes the 10-image limit atomic.
	filter := bson.M{"_id": id, fmt.Sprintf("images.%d", MaxProductImages-1): bson.M{"$exists": false}}
	var p Product
	err := s.products.FindOneAndUpdate(ctx, filter,
		bson.M{"$push": bson.M{"images": rec}, "$set": bson.M{"updatedAt": time.Now().UTC(), "updatedBy": actor}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&p)
	if err != nil {
		s.deleteObjects(ctx, key)
		if mongox.IsNotFound(err) {
			return ProductDTO{}, httpx.Validation(httpx.FieldError{Field: "file", Message: fmt.Sprintf("a product can have at most %d images", MaxProductImages)})
		}
		return ProductDTO{}, err
	}
	s.audit.Record(ctx, "product.image.add", "product", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.productDTO(ctx, &p)
}

func (s *Service) RemoveProductImage(ctx context.Context, actor string, id bson.ObjectID, imageID string) (ProductDTO, error) {
	var before Product
	err := s.products.FindOneAndUpdate(ctx, bson.M{"_id": id, "images.id": imageID},
		bson.M{"$pull": bson.M{"images": bson.M{"id": imageID}}, "$set": bson.M{"updatedAt": time.Now().UTC(), "updatedBy": actor}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&before)
	if mongox.IsNotFound(err) {
		return ProductDTO{}, httpx.ErrNotFound
	}
	if err != nil {
		return ProductDTO{}, err
	}
	after := before
	after.Images = nil
	for _, img := range before.Images {
		// The removed object is left for the nightly sweep (orders may reference it).
		if img.ID != imageID {
			after.Images = append(after.Images, img)
		}
	}
	s.audit.Record(ctx, "product.image.remove", "product", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.productDTO(ctx, &after)
}

func (s *Service) ReorderProductImages(ctx context.Context, actor string, id bson.ObjectID, order []string) (ProductDTO, error) {
	p, err := s.findProduct(ctx, bson.M{"_id": id})
	if err != nil {
		return ProductDTO{}, err
	}
	byID := map[string]Image{}
	for _, img := range p.Images {
		byID[img.ID] = img
	}
	if len(order) != len(p.Images) {
		return ProductDTO{}, httpx.Validation(httpx.FieldError{Field: "imageIds", Message: "must list every image exactly once"})
	}
	reordered := make([]Image, 0, len(order))
	for _, imgID := range order {
		img, ok := byID[imgID]
		if !ok {
			return ProductDTO{}, httpx.Validation(httpx.FieldError{Field: "imageIds", Message: "must list every image exactly once"})
		}
		delete(byID, imgID)
		reordered = append(reordered, img)
	}
	// Guard against a concurrent upload/delete changing the set meanwhile.
	res, err := s.products.UpdateOne(ctx, bson.M{"_id": id, "images": bson.M{"$size": len(p.Images)}},
		bson.M{"$set": bson.M{"images": reordered, "updatedAt": time.Now().UTC(), "updatedBy": actor}})
	if err != nil {
		return ProductDTO{}, err
	}
	if res.MatchedCount == 0 {
		return ProductDTO{}, httpx.Conflict("images changed while reordering; reload and try again")
	}
	p.Images = reordered
	s.audit.Record(ctx, "product.image.reorder", "product", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.productDTO(ctx, p)
}

// ---------- Views ----------

// RecordView counts a view once per visitor per product per 15 minutes. The
// visitor is identified by a keyed hash of IP and user agent; raw IPs are
// never stored.
func (s *Service) RecordView(ctx context.Context, id bson.ObjectID, ip, userAgent string) error {
	n, err := s.products.CountDocuments(ctx, bson.M{"_id": id, "status": StatusPublished}, options.Count().SetLimit(1))
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.ErrNotFound
	}
	now := time.Now().UTC()
	bucket := now.Truncate(viewDedupWindow).Unix()
	mac := hmac.New(sha256.New, s.pepper)
	_, _ = fmt.Fprintf(mac, "%s|%s|%s|%d", id.Hex(), ip, userAgent, bucket)
	_, err = s.views.InsertOne(ctx, bson.M{"_id": hex.EncodeToString(mac.Sum(nil)), "productId": id, "at": now})
	if mongox.IsDuplicateKey(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.products.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"views": 1}})
	return err
}

// RefreshRecentViews recomputes the 7-day view counter used for "trending" ordering.
func (s *Service) RefreshRecentViews(ctx context.Context) error {
	since := time.Now().UTC().Add(-7 * 24 * time.Hour)
	cur, err := s.views.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"at": bson.M{"$gte": since}}}},
		{{Key: "$group", Value: bson.M{"_id": "$productId", "n": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return err
	}
	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
		N  int64         `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return err
	}
	seen := make([]bson.ObjectID, 0, len(rows))
	models := make([]mongo.WriteModel, 0, len(rows)+1)
	for _, r := range rows {
		seen = append(seen, r.ID)
		models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": r.ID}).SetUpdate(bson.M{"$set": bson.M{"recentViews": r.N}}))
	}
	models = append(models, mongo.NewUpdateManyModel().
		SetFilter(bson.M{"_id": bson.M{"$nin": seen}, "recentViews": bson.M{"$ne": 0}}).
		SetUpdate(bson.M{"$set": bson.M{"recentViews": 0}}))
	if _, err = s.products.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
		return err
	}
	s.cache.Invalidate(ctx, redisx.NSCatalog) // relevance ordering changed
	return nil
}

// ---------- Category writes ----------

func validateCategoryInput(in *CategoryInput) error {
	in.Name.EN, in.Name.VI, in.Slug = strings.TrimSpace(in.Name.EN), strings.TrimSpace(in.Name.VI), strings.TrimSpace(in.Slug)
	var fields []httpx.FieldError
	if in.Name.EN == "" {
		fields = append(fields, httpx.FieldError{Field: "name.en", Message: "is required"})
	}
	if in.Slug != "" && !textx.ValidSlug(in.Slug) {
		fields = append(fields, httpx.FieldError{Field: "slug", Message: "must contain only lowercase letters, digits and single dashes"})
	}
	if len(fields) > 0 {
		return httpx.Validation(fields...)
	}
	return nil
}

func (s *Service) CreateCategory(ctx context.Context, in *CategoryInput) (CategoryDTO, error) {
	if err := validateCategoryInput(in); err != nil {
		return CategoryDTO{}, err
	}
	now := time.Now().UTC()
	c := &Category{
		ID: bson.NewObjectID(), Name: LocalizedText(in.Name),
		Description: LocalizedText{EN: strings.TrimSpace(in.Description.EN), VI: strings.TrimSpace(in.Description.VI)},
		SortOrder:   in.SortOrder, CreatedAt: now, UpdatedAt: now,
	}
	base := in.Slug
	if base == "" {
		if base = textx.Slugify(in.Name.EN); base == "" {
			base = "category"
		}
	}
	if err := insertWithSlug(ctx, s.categories, base, in.Slug != "", func(slug string) any { c.Slug = slug; return c }); err != nil {
		return CategoryDTO{}, err
	}
	s.audit.Record(ctx, "category.create", "category", c.ID.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.categoryDTO(c), nil
}

func (s *Service) UpdateCategory(ctx context.Context, id bson.ObjectID, in *CategoryInput) (CategoryDTO, error) {
	if err := validateCategoryInput(in); err != nil {
		return CategoryDTO{}, err
	}
	set := bson.M{
		"name":        LocalizedText(in.Name),
		"description": LocalizedText{EN: strings.TrimSpace(in.Description.EN), VI: strings.TrimSpace(in.Description.VI)},
		"sortOrder":   in.SortOrder, "updatedAt": time.Now().UTC(),
	}
	if in.Slug != "" {
		set["slug"] = in.Slug
	}
	var c Category
	err := s.categories.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&c)
	switch {
	case mongox.IsDuplicateKey(err):
		return CategoryDTO{}, httpx.Conflict("slug is already in use")
	case mongox.IsNotFound(err):
		return CategoryDTO{}, httpx.ErrNotFound
	case err != nil:
		return CategoryDTO{}, err
	}
	s.audit.Record(ctx, "category.update", "category", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.categoryDTO(&c), nil
}

func (s *Service) DeleteCategory(ctx context.Context, id bson.ObjectID) error {
	var c Category
	err := s.categories.FindOneAndDelete(ctx, bson.M{"_id": id}).Decode(&c)
	if mongox.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := s.products.UpdateMany(ctx, bson.M{"categoryIds": id}, bson.M{"$pull": bson.M{"categoryIds": id}}); err != nil {
		return fmt.Errorf("detach category from products: %w", err)
	}
	if c.Image != nil {
		s.deleteObjects(ctx, c.Image.Key)
	}
	s.audit.Record(ctx, "category.delete", "category", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return nil
}

func (s *Service) SetCategoryImage(ctx context.Context, id bson.ObjectID, img *media.Processed) (CategoryDTO, error) {
	key := media.NewKey(CategoryImagePrefix+id.Hex(), img.Ext)
	if err := s.storage.Put(ctx, key, img.Data, img.ContentType); err != nil {
		return CategoryDTO{}, err
	}
	rec := &Image{ID: media.NewID(), Key: key, Width: img.Width, Height: img.Height}
	var before Category
	err := s.categories.FindOneAndUpdate(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"image": rec, "updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&before)
	if err != nil {
		s.deleteObjects(ctx, key)
		if mongox.IsNotFound(err) {
			return CategoryDTO{}, httpx.ErrNotFound
		}
		return CategoryDTO{}, err
	}
	if before.Image != nil {
		s.deleteObjects(ctx, before.Image.Key)
	}
	before.Image = rec
	s.audit.Record(ctx, "category.image.set", "category", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.categoryDTO(&before), nil
}

func (s *Service) RemoveCategoryImage(ctx context.Context, id bson.ObjectID) (CategoryDTO, error) {
	var before Category
	err := s.categories.FindOneAndUpdate(ctx, bson.M{"_id": id},
		bson.M{"$unset": bson.M{"image": ""}, "$set": bson.M{"updatedAt": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&before)
	if mongox.IsNotFound(err) {
		return CategoryDTO{}, httpx.ErrNotFound
	}
	if err != nil {
		return CategoryDTO{}, err
	}
	if before.Image != nil {
		s.deleteObjects(ctx, before.Image.Key)
	}
	before.Image = nil
	s.audit.Record(ctx, "category.image.remove", "category", id.Hex())
	s.cache.Invalidate(ctx, redisx.NSCatalog)
	return s.categoryDTO(&before), nil
}

// ---------- Media sweep ----------

// ReferencedKeys returns every object key referenced by products and categories.
func (s *Service) ReferencedKeys(ctx context.Context) (map[string]struct{}, error) {
	keys := map[string]struct{}{}
	cur, err := s.products.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"images.key": 1}))
	if err != nil {
		return nil, err
	}
	for cur.Next(ctx) {
		var p struct {
			Images []struct {
				Key string `bson:"key"`
			} `bson:"images"`
		}
		if err := cur.Decode(&p); err != nil {
			return nil, err
		}
		for _, i := range p.Images {
			keys[i.Key] = struct{}{}
		}
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	cats, err := s.categories.Find(ctx, bson.M{"image": bson.M{"$exists": true}}, options.Find().SetProjection(bson.M{"image.key": 1}))
	if err != nil {
		return nil, err
	}
	for cats.Next(ctx) {
		var c struct {
			Image struct {
				Key string `bson:"key"`
			} `bson:"image"`
		}
		if err := cats.Decode(&c); err != nil {
			return nil, err
		}
		keys[c.Image.Key] = struct{}{}
	}
	if err := cats.Err(); err != nil {
		return nil, err
	}
	// Order line items keep showing the product image they were sold with.
	var ks []string
	err = s.products.Database().Collection("orders").
		Distinct(ctx, "items.imageKey", bson.M{"items.imageKey": bson.M{"$exists": true}}).Decode(&ks)
	if err != nil {
		return nil, err
	}
	for _, k := range ks {
		keys[k] = struct{}{}
	}
	return keys, nil
}

// SweepOrphanedMedia deletes stored objects no document references. Objects
// younger than grace are kept so in-flight uploads are never removed.
func (s *Service) SweepOrphanedMedia(ctx context.Context, grace time.Duration) (int, error) {
	refs, err := s.ReferencedKeys(ctx)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-grace)
	var orphans []string
	for _, prefix := range []string{ProductImagePrefix, CategoryImagePrefix} {
		err := s.storage.Walk(ctx, prefix, func(key string, modified time.Time) error {
			if _, ok := refs[key]; !ok && modified.Before(cutoff) {
				orphans = append(orphans, key)
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	if len(orphans) == 0 {
		return 0, nil
	}
	if err := s.storage.Delete(ctx, orphans...); err != nil {
		return 0, err
	}
	return len(orphans), nil
}
