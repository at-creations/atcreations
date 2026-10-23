//go:build integration

// Integration tests against a real MongoDB replica set:
//
//	docker compose -f docker-compose.dev.yml up -d
//	MONGO_TEST_URI="mongodb://127.0.0.1:27017/?replicaSet=rs0&directConnection=true" go test -tags integration ./internal/
package internal_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	"github.com/atcreations/atcreations/apps/api/internal/mail"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/orders"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

type memStorage struct {
	mu   sync.Mutex
	objs map[string]time.Time
}

func (m *memStorage) Put(_ context.Context, key string, _ []byte, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objs[key] = time.Now().Add(-48 * time.Hour)
	return nil
}
func (m *memStorage) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.objs, k)
	}
	return nil
}
func (m *memStorage) Walk(_ context.Context, prefix string, fn func(string, time.Time) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, t := range m.objs {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			if err := fn(k, t); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *memStorage) URL(key string) string { return "https://cdn.test/" + key }
func (m *memStorage) PutObject(ctx context.Context, in media.PutInput) error {
	return m.Put(ctx, in.Key, in.Data, in.ContentType)
}
func (m *memStorage) Get(_ context.Context, key string, _ int64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objs[key]; !ok {
		return nil, media.ErrObjectNotFound
	}
	return []byte{}, nil
}
func (m *memStorage) Head(_ context.Context, key string) (*media.ObjectInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objs[key]; !ok {
		return nil, media.ErrObjectNotFound
	}
	return &media.ObjectInfo{}, nil
}

type nopSender struct{}

func (nopSender) Send(context.Context, mail.Message) error { return nil }

const pepper = "integration-test-pepper-0123456789abcdef"

func setup(t *testing.T) (*catalog.Service, *orders.Service, *memStorage) {
	t.Helper()
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
	store := &memStorage{objs: map[string]time.Time{}}
	a := audit.New(db)
	cat := catalog.NewService(db, store, a, nil, pepper)
	ord := orders.NewService(client, db, store, orders.NewMailer(nopSender{}, "Test", "https://x"), a, nil, pepper)
	for _, f := range []func(context.Context) error{a.EnsureIndexes, cat.EnsureIndexes, ord.EnsureIndexes} {
		if err := f(ctx); err != nil {
			t.Fatal(err)
		}
	}
	return cat, ord, store
}

func i64(v int64) *int64 { return &v }

func status(err error) int {
	var e *httpx.Error
	if errors.As(err, &e) {
		return e.Status
	}
	return 0
}

func TestCatalogSlugsAndSearch(t *testing.T) {
	cat, _, _ := setup(t)
	ctx := context.Background()
	in := func() *catalog.ProductInput {
		return &catalog.ProductInput{Status: catalog.StatusPublished, Name: catalog.NameInput{EN: "Ceramic Vase", VI: "Bình gốm"}, PriceCents: i64(2500), Stock: 3}
	}
	p1, err := cat.CreateProduct(ctx, "u", in())
	if err != nil {
		t.Fatal(err)
	}
	p2, err := cat.CreateProduct(ctx, "u", in())
	if err != nil {
		t.Fatal(err)
	}
	if p1.Slug != "ceramic-vase" || p2.Slug != "ceramic-vase-2" {
		t.Fatalf("slugs = %s, %s", p1.Slug, p2.Slug)
	}
	explicit := in()
	explicit.Slug = "ceramic-vase"
	if _, err := cat.CreateProduct(ctx, "u", explicit); status(err) != http.StatusConflict {
		t.Fatalf("explicit duplicate slug: %v", err)
	}
	draft := in()
	draft.Status = catalog.StatusDraft
	draft.Name.EN = "Hidden draft"
	if _, err := cat.CreateProduct(ctx, "u", draft); err != nil {
		t.Fatal(err)
	}
	items, meta, err := cat.ListProducts(ctx, catalog.ProductQuery{Published: true, Text: "binh GOM", Page: httpx.Page{Page: 1, Size: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Total != 2 || len(items) != 2 {
		t.Fatalf("accent-insensitive search found %d, want 2 (drafts must be hidden)", meta.Total)
	}
}

func TestOrderLifecycle(t *testing.T) {
	cat, ord, _ := setup(t)
	ctx := context.Background()
	p, err := cat.CreateProduct(ctx, "u", &catalog.ProductInput{
		Status: catalog.StatusPublished, Name: catalog.NameInput{EN: "Lamp"}, PriceCents: i64(1000), DiscountPercent: 10, Stock: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	newInput := func(qty int) *orders.CreateInput {
		in := &orders.CreateInput{
			Customer: orders.Customer{Name: "Ann", Email: "Ann@Example.com"},
			Shipping: orders.Address{Name: "Ann", Phone: "1", Line1: "1 St", City: "Van", PostalCode: "V1", Country: "CA"},
		}
		in.Items = []orders.ItemInput{{ProductID: p.ID, Quantity: qty}}
		return in
	}

	// Server-side pricing: 2 × (1000 − 10%) = 1800.
	o, code, err := ord.Create(ctx, newInput(2))
	if err != nil {
		t.Fatal(err)
	}
	if o.TotalCents != 1800 || o.SubtotalCents != 2000 || o.DiscountCents != 200 || o.Customer.Email != "ann@example.com" {
		t.Fatalf("totals: %+v", o)
	}

	// Stock is exhausted; a further order must fail and leave stock untouched.
	if _, _, err := ord.Create(ctx, newInput(1)); status(err) != http.StatusConflict {
		t.Fatalf("oversell: %v", err)
	}

	// Tracking requires all three factors.
	if _, err := ord.Track(ctx, o.Number, "ann@example.com", "WRONGCODE1"); status(err) != http.StatusNotFound {
		t.Fatalf("wrong code: %v", err)
	}
	if tr, err := ord.Track(ctx, o.Number, "ANN@example.com", code); err != nil || tr.Status != orders.Pending {
		t.Fatalf("track: %v %+v", err, tr)
	}

	// Illegal transition is refused; cancelling restocks.
	if _, err := ord.Transition(ctx, "staff", o.ID, &orders.TransitionInput{Status: orders.Shipped}); status(err) != http.StatusConflict {
		t.Fatalf("pending→shipped: %v", err)
	}
	notify := false
	if _, err := ord.Transition(ctx, "staff", o.ID, &orders.TransitionInput{Status: orders.Cancelled, Notify: &notify}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ord.Create(ctx, newInput(2)); err != nil {
		t.Fatalf("stock should be restored after cancel: %v", err)
	}
}

func TestConcurrentOrdersNeverOversell(t *testing.T) {
	cat, ord, _ := setup(t)
	ctx := context.Background()
	p, err := cat.CreateProduct(ctx, "u", &catalog.ProductInput{
		Status: catalog.StatusPublished, Name: catalog.NameInput{EN: "Bowl"}, PriceCents: i64(500), Stock: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := &orders.CreateInput{
				Customer: orders.Customer{Name: "B", Email: "b@example.com"},
				Shipping: orders.Address{Name: "B", Phone: "1", Line1: "1", City: "c", PostalCode: "p", Country: "CA"},
			}
			in.Items = []orders.ItemInput{{ProductID: p.ID, Quantity: 1}}
			if _, _, err := ord.Create(ctx, in); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 5 {
		t.Fatalf("%d orders succeeded with stock 5", ok)
	}
}

func TestViewsAreDeduplicated(t *testing.T) {
	cat, _, _ := setup(t)
	ctx := context.Background()
	p, err := cat.CreateProduct(ctx, "u", &catalog.ProductInput{Status: catalog.StatusPublished, Name: catalog.NameInput{EN: "Cup"}})
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := cat.RecordView(ctx, p.ID, "203.0.113.1", "UA"); err != nil {
			t.Fatal(err)
		}
	}
	if err := cat.RecordView(ctx, p.ID, "203.0.113.2", "UA"); err != nil {
		t.Fatal(err)
	}
	items, _, err := cat.ListProducts(ctx, catalog.ProductQuery{Published: true, Page: httpx.Page{Page: 1, Size: 1}})
	if err != nil || items[0].Views != 2 {
		t.Fatalf("views = %d, want 2 (%v)", items[0].Views, err)
	}
	if err := cat.RefreshRecentViews(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOrphanSweepKeepsReferencedObjects(t *testing.T) {
	cat, _, store := setup(t)
	ctx := context.Background()
	store.objs["products/x/orphan.jpg"] = time.Now().Add(-48 * time.Hour)
	store.objs["products/x/fresh.jpg"] = time.Now()
	n, err := cat.SweepOrphanedMedia(ctx, 24*time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("swept %d (%v), want 1", n, err)
	}
	if _, ok := store.objs["products/x/fresh.jpg"]; !ok {
		t.Fatal("fresh upload was removed")
	}
}

func TestStaleProductEditCannotOverwriteStock(t *testing.T) {
	cat, ord, _ := setup(t)
	ctx := context.Background()
	in := &catalog.ProductInput{Status: catalog.StatusPublished, Name: catalog.NameInput{EN: "Mug"}, PriceCents: i64(900), Stock: 5}
	p, err := cat.CreateProduct(ctx, "u", in)
	if err != nil {
		t.Fatal(err)
	}
	loaded := p.Version // admin opens the edit form

	order := &orders.CreateInput{
		Customer: orders.Customer{Name: "C", Email: "c@example.com"},
		Shipping: orders.Address{Name: "C", Phone: "1", Line1: "1", City: "c", PostalCode: "p", Country: "CA"},
		Items:    []orders.ItemInput{{ProductID: p.ID, Quantity: 2}},
	}
	if _, _, err := ord.Create(ctx, order); err != nil {
		t.Fatal(err)
	}

	in.Version = &loaded // admin saves the stale form (stock still 5)
	if _, err := cat.UpdateProduct(ctx, "u", p.ID, in); status(err) != http.StatusConflict {
		t.Fatalf("stale save: err = %v, want 409", err)
	}
	if _, err := cat.UpdateProduct(ctx, "u", p.ID, &catalog.ProductInput{Status: catalog.StatusPublished, Name: catalog.NameInput{EN: "Mug"}, Stock: 5}); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("missing version: err = %v, want 422", err)
	}
	fresh, _, err := cat.ListProducts(ctx, catalog.ProductQuery{Published: true, Page: httpx.Page{Page: 1, Size: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if fresh[0].Stock != 3 {
		t.Fatalf("stock = %d, want 3", fresh[0].Stock)
	}
	in.Version = &fresh[0].Version
	in.Stock = 3
	if _, err := cat.UpdateProduct(ctx, "u", p.ID, in); err != nil {
		t.Fatalf("fresh save: %v", err)
	}
}

func TestCategoryReorderAndCounts(t *testing.T) {
	cat, _, _ := setup(t)
	ctx := context.Background()
	var ids []bson.ObjectID
	for _, name := range []string{"Pottery", "Woodcraft", "Paper"} {
		c, err := cat.CreateCategory(ctx, &catalog.CategoryInput{Name: catalog.NameInput{EN: name}})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, c.ID)
	}
	if _, err := cat.CreateProduct(ctx, "u", &catalog.ProductInput{
		Status: catalog.StatusDraft, Name: catalog.NameInput{EN: "Bowl"}, CategoryIDs: []bson.ObjectID{ids[0], ids[2]},
	}); err != nil {
		t.Fatal(err)
	}

	// Reverse the order.
	if err := cat.ReorderCategories(ctx, []bson.ObjectID{ids[2], ids[1], ids[0]}); err != nil {
		t.Fatal(err)
	}
	list, err := cat.ListAdminCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].ID != ids[2] || list[2].ID != ids[0] {
		t.Fatalf("order = %v", []bson.ObjectID{list[0].ID, list[1].ID, list[2].ID})
	}
	counts := map[bson.ObjectID]int64{}
	for _, c := range list {
		counts[c.ID] = c.ProductCount
	}
	if counts[ids[0]] != 1 || counts[ids[1]] != 0 || counts[ids[2]] != 1 {
		t.Fatalf("counts = %v", counts)
	}

	// Partial or duplicated lists are refused (a stale tab must not scramble the order).
	if err := cat.ReorderCategories(ctx, []bson.ObjectID{ids[0], ids[1]}); status(err) != http.StatusConflict {
		t.Fatalf("partial list: %v", err)
	}
	if err := cat.ReorderCategories(ctx, []bson.ObjectID{ids[0], ids[0], ids[1]}); status(err) != http.StatusUnprocessableEntity {
		t.Fatalf("duplicates: %v", err)
	}
}
