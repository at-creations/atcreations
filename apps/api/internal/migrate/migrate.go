// Package migrate imports catalogue data from the legacy atstore-server-v2
// MongoDB into the new schema. The legacy database is only ever read.
//
// Mapping (see docs/MIGRATION.md):
//   - categories and products keep their ObjectIDs, so re-runs replace
//     documents instead of duplicating them;
//   - prices in dollars become integer cents; a price <= 0 ("contact for
//     price" in the old UI) becomes null;
//   - thumbnail + images become one ordered image list (cover first), each
//     image downloaded from the legacy CDN, re-processed by media.Process and
//     stored under a new random key; downloaded images are remembered in the
//     migration_media collection so re-runs do not upload them again;
//   - store info: email/phone/address and business hours.
//
// Users, refresh tokens, API keys and raw page views (which contain IP
// addresses) are intentionally not migrated.
package migrate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
	"github.com/atcreations/atcreations/apps/api/internal/platform/textx"
	"github.com/atcreations/atcreations/apps/api/internal/storeinfo"
)

// ---- legacy documents (read-only) ----

type oldCategory struct {
	ID            bson.ObjectID `bson:"_id"`
	Name          string        `bson:"name"`
	NameVI        string        `bson:"nameVI"`
	Description   string        `bson:"description"`
	DescriptionVI string        `bson:"descriptionVI"`
	Slug          string        `bson:"slug"`
	Thumbnail     string        `bson:"thumbnail"`
	CreatedAt     time.Time     `bson:"createdAt"`
	UpdatedAt     time.Time     `bson:"updatedAt"`
}

type oldProduct struct {
	ID            bson.ObjectID   `bson:"_id"`
	Name          string          `bson:"name"`
	NameVI        string          `bson:"nameVI"`
	Description   string          `bson:"description"`
	DescriptionVI string          `bson:"descriptionVI"`
	Price         float64         `bson:"price"`
	Discount      float64         `bson:"discount"`
	Stock         float64         `bson:"stock"`
	Featured      bool            `bson:"featured"`
	CategoryIDs   []bson.ObjectID `bson:"categoryIds"`
	Thumbnail     string          `bson:"thumbnail"`
	Images        []string        `bson:"images"`
	Views         int64           `bson:"views"`
	CreatedAt     time.Time       `bson:"createdAt"`
	UpdatedAt     time.Time       `bson:"updatedAt"`
}

type oldStoreInfo struct {
	Email         string `bson:"email"`
	Phone         string `bson:"phone"`
	Address       string `bson:"address"`
	BusinessHours []struct {
		Day       string `bson:"day"`
		OpenTime  string `bson:"openTime"`
		CloseTime string `bson:"closeTime"`
	} `bson:"businessHours"`
}

// ---- runner ----

type Options struct {
	Apply       bool   // false = dry run: read and plan, write nothing
	Overwrite   bool   // replace categories/products that already exist (only before go-live)
	OldCDN      string // base URL serving legacy object keys
	Concurrency int
}

type Runner struct {
	old     *mongo.Database
	catalog *catalog.Service
	store   *storeinfo.Service
	storage media.Storage
	mapped  *mongo.Collection // migration_media: legacy key -> new image
	http    *http.Client
	opt     Options
}

func NewRunner(old *mongo.Database, target *mongo.Database, cat *catalog.Service, store *storeinfo.Service, storage media.Storage, opt Options) *Runner {
	if opt.Concurrency <= 0 {
		opt.Concurrency = 4
	}
	return &Runner{
		old: old, catalog: cat, store: store, storage: storage,
		mapped: target.Collection("migration_media"),
		http:   &http.Client{Timeout: 60 * time.Second},
		opt:    opt,
	}
}

type Report struct {
	Categories, Products, ProductsWithoutPrice, Images, ImagesReused, StoreInfo int
	// Already in the target and left untouched (no -overwrite).
	CategoriesKept, ProductsKept int
	Warnings                     []string
	mu                           sync.Mutex
}

func (m *Runner) mode() catalog.ImportMode {
	if m.opt.Overwrite {
		return catalog.ImportOverwrite
	}
	return catalog.ImportKeep
}

func (r *Report) warn(format string, args ...any) {
	r.mu.Lock()
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
	r.mu.Unlock()
}

func (r *Report) add(f func()) {
	r.mu.Lock()
	f()
	r.mu.Unlock()
}

// Run migrates categories, then products, then store info.
func (m *Runner) Run(ctx context.Context) (*Report, error) {
	rep := &Report{}
	cats, err := m.migrateCategories(ctx, rep)
	if err != nil {
		return rep, err
	}
	if err := m.migrateProducts(ctx, rep, cats); err != nil {
		return rep, err
	}
	if err := m.migrateStoreInfo(ctx, rep); err != nil {
		return rep, err
	}
	return rep, nil
}

func readAll[T any](ctx context.Context, coll *mongo.Collection) ([]T, error) {
	cur, err := coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", coll.Name(), err)
	}
	var out []T
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode %s: %w", coll.Name(), err)
	}
	return out, nil
}

func (m *Runner) migrateCategories(ctx context.Context, rep *Report) (map[bson.ObjectID]bool, error) {
	olds, err := readAll[oldCategory](ctx, m.old.Collection("categories"))
	if err != nil {
		return nil, err
	}
	known := map[bson.ObjectID]bool{}
	for _, o := range olds {
		slug := o.Slug
		if !textx.ValidSlug(slug) {
			slug = textx.Slugify(o.Name)
			rep.warn("category %s: slug %q is not URL-safe, using %q", o.ID.Hex(), o.Slug, slug)
		}
		known[o.ID] = true
		if !m.opt.Overwrite {
			// Existing records hold live edits: skip them before any image work.
			if exists, err := m.catalog.CategoryExists(ctx, o.ID); err != nil {
				return nil, err
			} else if exists {
				rep.CategoriesKept++
				continue
			}
		}
		c := &catalog.Category{
			ID: o.ID, Slug: slug,
			Name:        catalog.LocalizedText{EN: strings.TrimSpace(o.Name), VI: strings.TrimSpace(o.NameVI)},
			Description: catalog.LocalizedText{EN: strings.TrimSpace(o.Description), VI: strings.TrimSpace(o.DescriptionVI)},
			CreatedAt:   o.CreatedAt, UpdatedAt: o.UpdatedAt,
		}
		if o.Thumbnail != "" {
			img, reused, err := m.image(ctx, o.Thumbnail, catalog.CategoryImagePrefix+o.ID.Hex())
			if err != nil {
				rep.warn("category %s: image %s skipped: %v", slug, o.Thumbnail, err)
			} else {
				c.Image = img // nil in dry-run mode
				rep.add(func() {
					rep.Images++
					if reused {
						rep.ImagesReused++
					}
				})
			}
		}
		if m.opt.Apply {
			if err := m.catalog.ImportCategory(ctx, c, m.mode()); errors.Is(err, catalog.ErrImportExists) {
				rep.CategoriesKept++
				continue
			} else if err != nil {
				return nil, err
			}
		}
		rep.Categories++
		slog.InfoContext(ctx, "category", "slug", slug, "image", c.Image != nil)
	}
	return known, nil
}

func (m *Runner) migrateProducts(ctx context.Context, rep *Report, categories map[bson.ObjectID]bool) error {
	olds, err := readAll[oldProduct](ctx, m.old.Collection("products"))
	if err != nil {
		return err
	}
	slugs, err := m.assignSlugs(ctx, olds)
	if err != nil {
		return err
	}

	work := make(chan oldProduct)
	errs := make(chan error, m.opt.Concurrency)
	var wg sync.WaitGroup
	for range m.opt.Concurrency {
		wg.Go(func() {
			for o := range work {
				if err := m.migrateProduct(ctx, rep, o, slugs[o.ID], categories); err != nil {
					errs <- err
					return
				}
			}
		})
	}
	var firstErr error
feed:
	for _, o := range olds {
		select {
		case work <- o:
		case firstErr = <-errs:
			break feed
		case <-ctx.Done():
			firstErr = ctx.Err()
			break feed
		}
	}
	close(work)
	wg.Wait()
	close(errs)
	if firstErr == nil {
		firstErr = <-errs
	}
	return firstErr
}

// assignSlugs gives every product a unique slug derived from its English name.
func (m *Runner) assignSlugs(ctx context.Context, olds []oldProduct) (map[bson.ObjectID]string, error) {
	taken := map[string]bson.ObjectID{}
	out := map[bson.ObjectID]string{}
	for _, o := range olds {
		base := textx.Slugify(o.Name)
		if base == "" {
			base = "product"
		}
		for i := 1; ; i++ {
			slug := base
			if i > 1 {
				slug = fmt.Sprintf("%s-%d", base, i)
			}
			if owner, ok := taken[slug]; ok && owner != o.ID {
				continue
			}
			// A product created in the new system may already use the slug.
			owner, exists, err := m.catalog.ProductSlugOwner(ctx, slug)
			if err != nil {
				return nil, err
			}
			if exists && owner != o.ID {
				continue
			}
			taken[slug], out[o.ID] = o.ID, slug
			break
		}
	}
	return out, nil
}

func dollarsToCents(v float64) *int64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	c := int64(math.Round(v * 100))
	return &c
}

func (m *Runner) migrateProduct(ctx context.Context, rep *Report, o oldProduct, slug string, categories map[bson.ObjectID]bool) error {
	if !m.opt.Overwrite {
		// Existing records hold live edits (stock, prices, images): skip them before any image work.
		if exists, err := m.catalog.ProductExists(ctx, o.ID); err != nil {
			return err
		} else if exists {
			rep.add(func() { rep.ProductsKept++ })
			return nil
		}
	}
	p := &catalog.Product{
		ID: o.ID, Slug: slug, Status: catalog.StatusPublished,
		Name:            catalog.LocalizedText{EN: strings.TrimSpace(o.Name), VI: strings.TrimSpace(o.NameVI)},
		Description:     catalog.LocalizedText{EN: strings.TrimSpace(o.Description), VI: strings.TrimSpace(o.DescriptionVI)},
		PriceCents:      dollarsToCents(o.Price),
		DiscountPercent: int(math.Max(0, math.Min(90, math.Round(o.Discount)))),
		Stock:           int(math.Max(0, math.Round(o.Stock))),
		Featured:        o.Featured,
		Views:           o.Views,
		CreatedAt:       o.CreatedAt, UpdatedAt: o.UpdatedAt,
		CreatedBy: "migration", UpdatedBy: "migration",
	}
	for _, id := range o.CategoryIDs {
		if categories[id] {
			p.CategoryIDs = append(p.CategoryIDs, id)
		} else {
			rep.warn("product %s: unknown category %s dropped", slug, id.Hex())
		}
	}

	// Cover first (old thumbnail), then gallery images; skip duplicates.
	seen := map[string]bool{}
	var sources []string
	for _, k := range append([]string{o.Thumbnail}, o.Images...) {
		if k = strings.TrimSpace(k); k != "" && !seen[k] {
			seen[k] = true
			sources = append(sources, k)
		}
	}
	if len(sources) > catalog.MaxProductImages {
		rep.warn("product %s: %d images, keeping the first %d", slug, len(sources), catalog.MaxProductImages)
		sources = sources[:catalog.MaxProductImages]
	}
	for _, key := range sources {
		img, reused, err := m.image(ctx, key, catalog.ProductImagePrefix+o.ID.Hex())
		if err != nil {
			rep.warn("product %s: image %s skipped: %v", slug, key, err)
			continue
		}
		if img != nil {
			p.Images = append(p.Images, *img)
		}
		rep.add(func() {
			rep.Images++
			if reused {
				rep.ImagesReused++
			}
		})
	}

	if m.opt.Apply {
		if err := m.catalog.ImportProduct(ctx, p, m.mode()); errors.Is(err, catalog.ErrImportExists) {
			rep.add(func() { rep.ProductsKept++ })
			return nil
		} else if err != nil {
			return err
		}
	}
	rep.add(func() {
		rep.Products++
		if p.PriceCents == nil {
			rep.ProductsWithoutPrice++
		}
	})
	slog.InfoContext(ctx, "product", "slug", slug, "images", len(sources), "price_cents", p.PriceCents)
	return nil
}

type mappedImage struct {
	OldKey string `bson:"_id"`
	Image  catalog.Image
}

// image returns the migrated image for a legacy object key, downloading and
// re-processing it on first use. In dry-run mode it only checks that the
// legacy object is reachable and returns nil.
func (m *Runner) image(ctx context.Context, oldKey, prefix string) (*catalog.Image, bool, error) {
	var prev mappedImage
	err := m.mapped.FindOne(ctx, bson.M{"_id": oldKey}).Decode(&prev)
	if err == nil {
		img := prev.Image
		return &img, true, nil
	}
	if !mongox.IsNotFound(err) {
		return nil, false, err
	}

	url := strings.TrimRight(m.opt.OldCDN, "/") + "/" + strings.TrimLeft(oldKey, "/")
	if !m.opt.Apply {
		req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		resp, err := m.http.Do(req)
		if err != nil {
			return nil, false, err
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, false, fmt.Errorf("legacy CDN returned %d", resp.StatusCode)
		}
		return nil, false, nil
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := m.http.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("legacy CDN returned %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, media.MaxUploadBytes+1))
	if err != nil {
		return nil, false, err
	}
	processed, err := media.Process(ctx, raw, media.Options{})
	if err != nil {
		return nil, false, fmt.Errorf("process: %w", err)
	}
	key := media.NewKey(prefix, processed.Ext)
	if err := m.storage.Put(ctx, key, processed.Data, processed.ContentType); err != nil {
		return nil, false, err
	}
	img := catalog.Image{ID: media.NewID(), Key: key, Width: processed.Width, Height: processed.Height}
	if _, err := m.mapped.InsertOne(ctx, mappedImage{OldKey: oldKey, Image: img}); err != nil {
		if mongox.IsDuplicateKey(err) { // a concurrent worker got there first
			_ = m.storage.Delete(context.WithoutCancel(ctx), key)
			return m.image(ctx, oldKey, prefix)
		}
		return nil, false, err
	}
	return &img, false, nil
}

var dayCodes = map[string]string{
	"monday": "mon", "tuesday": "tue", "wednesday": "wed", "thursday": "thu",
	"friday": "fri", "saturday": "sat", "sunday": "sun",
}

func (m *Runner) migrateStoreInfo(ctx context.Context, rep *Report) error {
	var o oldStoreInfo
	err := m.old.Collection("storeinfos").FindOne(ctx, bson.M{}).Decode(&o)
	if errors.Is(err, mongo.ErrNoDocuments) {
		rep.warn("no store info in the legacy database")
		return nil
	}
	if err != nil {
		return err
	}
	// Never overwrite store info that was already set in the new system
	// (e.g. edited in the admin after a first migration).
	if cur, err := m.store.Get(ctx); err != nil {
		return err
	} else if cur.Email != "" || cur.Phone != "" || cur.Address != "" || len(cur.BusinessHours) > 0 {
		rep.warn("store info already set in the new system; left unchanged")
		return nil
	}
	in := storeinfo.Input{Email: o.Email, Phone: o.Phone, Address: o.Address, BusinessHours: []storeinfo.BusinessHours{}}
	for _, h := range o.BusinessHours {
		day, ok := dayCodes[strings.ToLower(strings.TrimSpace(h.Day))]
		if !ok {
			rep.warn("store info: unknown day %q skipped", h.Day)
			continue
		}
		closed := strings.EqualFold(h.OpenTime, "closed") || strings.EqualFold(h.CloseTime, "closed")
		bh := storeinfo.BusinessHours{Day: day, Closed: closed}
		if !closed {
			bh.Open, bh.Close = h.OpenTime, h.CloseTime
		}
		in.BusinessHours = append(in.BusinessHours, bh)
	}
	if m.opt.Apply {
		if _, err := m.store.Import(ctx, in); err != nil {
			return fmt.Errorf("store info: %w", err)
		}
	}
	rep.StoreInfo = 1
	slog.InfoContext(ctx, "store info", "hours", len(in.BusinessHours))
	return nil
}
