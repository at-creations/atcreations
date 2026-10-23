package orders

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/textx"
)

// Unambiguous alphabet (no 0/O, 1/I/L).
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

type Service struct {
	client   *mongo.Client
	orders   *mongo.Collection
	products *mongo.Collection
	storage  media.Storage
	mailer   *Mailer
	audit    *audit.Log
	pepper   []byte
	cache    *redisx.Cache
	emails   sync.WaitGroup // background notifications, drained on shutdown
}

func NewService(client *mongo.Client, db *mongo.Database, storage media.Storage, mailer *Mailer, a *audit.Log, cache *redisx.Cache, pepper string) *Service {
	return &Service{
		cache:  cache,
		client: client, orders: db.Collection("orders"), products: db.Collection("products"),
		storage: storage, mailer: mailer, audit: a, pepper: []byte(pepper),
	}
}

func (s *Service) EnsureIndexes(ctx context.Context) error {
	return mongox.EnsureIndexes(ctx, s.orders,
		mongox.Index{Keys: bson.D{{Key: "number", Value: 1}}, Unique: true},
		mongox.Index{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "customer.email", Value: 1}, {Key: "createdAt", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "searchWords", Value: 1}, {Key: "createdAt", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "items.imageKey", Value: 1}}}, // media sweep keeps sold images
		// total_desc / total_asc (exact reverses), with and without a status filter.
		mongox.Index{Keys: bson.D{{Key: "totalCents", Value: -1}, {Key: "createdAt", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "status", Value: 1}, {Key: "totalCents", Value: -1}, {Key: "createdAt", Value: -1}}},
		// Covers the analytics report's sales and status queries (no document reads).
		mongox.Index{Keys: bson.D{{Key: "createdAt", Value: 1}, {Key: "status", Value: 1}, {Key: "totalCents", Value: 1}}},
	)
}

// maxHistory bounds an order's history (the oldest events are dropped beyond it), so repeated
// note or shipping edits cannot grow a document without limit.
const maxHistory = 200

func pushHistory(ev Event) bson.M {
	return bson.M{"history": bson.M{"$each": bson.A{ev}, "$slice": -maxHistory}}
}

func orderSearchWords(o *Order) []string {
	return textx.SearchWords(o.Number, o.Customer.Name, o.Customer.Email)
}

// BackfillSearchWords sets searchWords on orders placed before it existed (runs once).
func (s *Service) BackfillSearchWords(ctx context.Context) error {
	return mongox.Backfill(ctx, s.orders, bson.M{"searchWords": bson.M{"$exists": false}},
		bson.M{"number": 1, "customer": 1},
		func(o *Order) bson.ObjectID { return o.ID },
		func(o *Order) bson.M { return bson.M{"searchWords": orderSearchWords(o)} })
}

func randomCode(n int) string {
	var b strings.Builder
	max := big.NewInt(int64(len(codeAlphabet)))
	for range n {
		i, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err) // crypto/rand failure is unrecoverable
		}
		b.WriteByte(codeAlphabet[i.Int64()])
	}
	return b.String()
}

func (s *Service) hashCode(code string) string {
	m := hmac.New(sha256.New, s.pepper)
	m.Write([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return hex.EncodeToString(m.Sum(nil))
}

func newOrderNumber(now time.Time) string {
	return "AT-" + now.Format("060102") + "-" + randomCode(6)
}

// Create places an order. Stock is reserved atomically inside a transaction,
// and all prices come from the catalogue, never from the client.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*Order, string, error) {
	in.Customer.Email = strings.ToLower(strings.TrimSpace(in.Customer.Email))
	qty := map[bson.ObjectID]int{}
	var orderOfIDs []bson.ObjectID
	for _, it := range in.Items {
		if _, ok := qty[it.ProductID]; !ok {
			orderOfIDs = append(orderOfIDs, it.ProductID)
		}
		qty[it.ProductID] += it.Quantity
	}
	for id, q := range qty {
		if q > 99 {
			return nil, "", httpx.Validation(httpx.FieldError{Field: "items", Message: "quantity per product must be at most 99 (product " + id.Hex() + ")"})
		}
	}

	accessCode := randomCode(10)
	now := time.Now().UTC()
	order := &Order{
		ID: bson.NewObjectID(), AccessCodeHash: s.hashCode(accessCode), Status: Pending,
		Customer: in.Customer, Shipping: in.Shipping, Currency: catalog.Currency,
		CustomerNote: strings.TrimSpace(in.CustomerNote),
		History:      []Event{{Type: "created", At: now, Actor: "customer", ToStatus: Pending}},
		CreatedAt:    now, UpdatedAt: now,
	}

	sess, err := s.client.StartSession()
	if err != nil {
		return nil, "", err
	}
	defer sess.EndSession(ctx)

	// A clashing order number (unique index) aborts the transaction, so a retry with a new
	// number has to run the whole transaction again; the callback is re-runnable.
	for range 5 {
		_, err = sess.WithTransaction(ctx, func(tx context.Context) (any, error) {
			order.Items = order.Items[:0]
			order.SubtotalCents, order.DiscountCents = 0, 0
			for _, id := range orderOfIDs {
				q := qty[id]
				var p catalog.Product
				err := s.products.FindOneAndUpdate(tx,
					bson.M{"_id": id, "status": catalog.StatusPublished, "hasPrice": true, "stock": bson.M{"$gte": q}},
					bson.M{"$inc": bson.M{"stock": -q, "version": 1}},
					options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&p)
				if mongox.IsNotFound(err) {
					return nil, httpx.Conflict("product " + id.Hex() + " is unavailable or does not have enough stock")
				}
				if err != nil {
					return nil, err
				}
				unit := *p.PriceCents
				final := *catalog.FinalPrice(p.PriceCents, p.DiscountPercent)
				item := Item{
					ProductID: p.ID, Slug: p.Slug, Name: p.Name, UnitPriceCents: unit,
					DiscountPercent: p.DiscountPercent, Quantity: q, LineTotalCents: final * int64(q),
				}
				if len(p.Images) > 0 {
					item.ImageURL = s.storage.URL(p.Images[0].Key)
					item.ImageKey = p.Images[0].Key
				}
				order.Items = append(order.Items, item)
				order.SubtotalCents += unit * int64(q)
				order.DiscountCents += (unit - final) * int64(q)
			}
			order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents + order.TaxCents
			order.Number = newOrderNumber(now)
			order.SearchWords = orderSearchWords(order)
			_, err := s.orders.InsertOne(tx, order)
			return nil, err
		})
		if !mongox.IsDuplicateKey(err) {
			break
		}
	}
	if mongox.IsDuplicateKey(err) {
		err = fmt.Errorf("could not allocate a unique order number: %w", err)
	}
	if err != nil {
		return nil, "", err
	}
	s.cache.Invalidate(ctx, redisx.NSCatalog) // stock changed

	s.notifyAsync(ctx, order.ID, func(ctx context.Context) error {
		return s.mailer.Confirmation(ctx, order, accessCode)
	})
	return order, accessCode, nil
}

// notifyAsync sends an email in the background (the request does not wait for it). A panic
// in the send is contained, and Wait lets shutdown finish sends already started.
func (s *Service) notifyAsync(ctx context.Context, id bson.ObjectID, send func(context.Context) error) {
	ctx = context.WithoutCancel(ctx)
	s.emails.Go(func() {
		defer func() {
			if v := recover(); v != nil {
				slog.ErrorContext(ctx, "order email panicked", "panic", v, "order", id.Hex())
			}
		}()
		s.notify(ctx, id, send)
	})
}

// Wait blocks until background emails finish or ctx ends.
func (s *Service) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.emails.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("order emails still sending: %w", ctx.Err())
	}
}

// notify sends an email and records it on the order.
func (s *Service) notify(ctx context.Context, id bson.ObjectID, send func(context.Context) error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := send(ctx); err != nil {
		slog.ErrorContext(ctx, "order email failed", "err", err, "order", id.Hex())
		return
	}
	_, err := s.orders.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$push": pushHistory(Event{Type: "customer_notified", At: time.Now().UTC(), Actor: "system"})})
	if err != nil {
		slog.WarnContext(ctx, "record notification failed", "err", err)
	}
}

// Track returns a reduced view when number, email and access code all match.
// Every mismatch yields the same 404 so the endpoint cannot be used to probe.
func (s *Service) Track(ctx context.Context, number, email, code string) (*Tracking, error) {
	var o Order
	err := s.orders.FindOne(ctx, bson.M{"number": strings.ToUpper(strings.TrimSpace(number))}).Decode(&o)
	if err != nil && !mongox.IsNotFound(err) {
		return nil, err
	}
	want := o.AccessCodeHash
	if want == "" {
		want = strings.Repeat("0", 64) // keep timing similar for unknown numbers
	}
	codeOK := hmac.Equal([]byte(s.hashCode(code)), []byte(want))
	emailOK := strings.EqualFold(strings.TrimSpace(email), o.Customer.Email)
	if err != nil || !codeOK || !emailOK {
		return nil, httpx.NewError(http.StatusNotFound, "no order matches these details")
	}
	t := &Tracking{Number: o.Number, Status: o.Status, CreatedAt: o.CreatedAt, TrackingNumber: o.TrackingNumber}
	for _, it := range o.Items {
		t.Items = append(t.Items, struct {
			Name     catalog.LocalizedText `json:"name"`
			Quantity int                   `json:"quantity"`
		}{it.Name, it.Quantity})
	}
	return t, nil
}

func (s *Service) Get(ctx context.Context, id bson.ObjectID) (*Order, error) {
	var o Order
	if err := s.orders.FindOne(ctx, bson.M{"_id": id}).Decode(&o); err != nil {
		if mongox.IsNotFound(err) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

type ListQuery struct {
	Status   string
	Text     string
	From, To *time.Time
	Sort     string
	Page     httpx.Page
}

func (s *Service) List(ctx context.Context, q ListQuery) ([]Order, httpx.PageMeta, error) {
	f := bson.M{}
	if q.Status != "" {
		f["status"] = q.Status
	}
	// Every term must start a word of the number, name or email ("AT-2601", "nguyen", "jane@exa").
	if terms := textx.PrefixTerms(q.Text); len(terms) > 0 {
		f["searchWords"] = textx.WordPrefixes(terms)
	}
	if q.From != nil || q.To != nil {
		r := bson.M{}
		if q.From != nil {
			r["$gte"] = *q.From
		}
		if q.To != nil {
			r["$lt"] = q.To.AddDate(0, 0, 1) // the whole local day, even across a DST change
		}
		f["createdAt"] = r
	}
	sort := bson.D{{Key: "createdAt", Value: -1}}
	switch q.Sort {
	case "oldest":
		sort = bson.D{{Key: "createdAt", Value: 1}}
	case "total_desc":
		sort = bson.D{{Key: "totalCents", Value: -1}, {Key: "createdAt", Value: -1}}
	case "total_asc": // the exact reverse of total_desc, so the same index serves both
		sort = bson.D{{Key: "totalCents", Value: 1}, {Key: "createdAt", Value: 1}}
	}
	total, capped, err := mongox.CountForPage(ctx, s.orders, f)
	if err != nil {
		return nil, httpx.PageMeta{}, err
	}
	cur, err := s.orders.Find(ctx, f, options.Find().SetSort(sort).SetSkip(q.Page.Skip()).SetLimit(int64(q.Page.Size)))
	if err != nil {
		return nil, httpx.PageMeta{}, err
	}
	out := []Order{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, httpx.PageMeta{}, err
	}
	return out, q.Page.MetaCapped(total, capped), nil
}

type TransitionInput struct {
	Status         Status `json:"status" validate:"required,oneof=pending confirmed processing shipped delivered completed cancelled refunded"`
	Note           string `json:"note" validate:"max=1000"`
	TrackingNumber string `json:"trackingNumber" validate:"max=100"`
	Notify         *bool  `json:"notify"`
}

// Transition moves an order along the state machine. Cancelling restocks
// items in the same transaction. The status filter makes the update a
// compare-and-swap, so concurrent transitions cannot both succeed.
func (s *Service) Transition(ctx context.Context, actor string, id bson.ObjectID, in *TransitionInput) (*Order, error) {
	sess, err := s.client.StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	var updated Order
	_, err = sess.WithTransaction(ctx, func(tx context.Context) (any, error) {
		var cur Order
		if err := s.orders.FindOne(tx, bson.M{"_id": id}).Decode(&cur); err != nil {
			if mongox.IsNotFound(err) {
				return nil, httpx.ErrNotFound
			}
			return nil, err
		}
		if !CanTransition(cur.Status, in.Status) {
			return nil, httpx.Conflict(fmt.Sprintf("cannot change status from %s to %s", cur.Status, in.Status))
		}
		now := time.Now().UTC()
		set := bson.M{"status": in.Status, "updatedAt": now}
		if tn := strings.TrimSpace(in.TrackingNumber); tn != "" {
			set["trackingNumber"] = tn
		}
		ev := Event{Type: "status_changed", At: now, Actor: actor, FromStatus: cur.Status, ToStatus: in.Status, Note: strings.TrimSpace(in.Note)}
		err := s.orders.FindOneAndUpdate(tx, bson.M{"_id": id, "status": cur.Status},
			bson.M{"$set": set, "$push": pushHistory(ev)},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
		if mongox.IsNotFound(err) {
			return nil, httpx.Conflict("order changed concurrently; reload and try again")
		}
		if err != nil {
			return nil, err
		}
		if in.Status == Cancelled {
			for _, it := range cur.Items {
				if _, err := s.products.UpdateOne(tx, bson.M{"_id": it.ProductID}, bson.M{"$inc": bson.M{"stock": it.Quantity, "version": 1}}); err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, "order.status."+string(in.Status), "order", id.Hex())
	if in.Status == Cancelled {
		s.cache.Invalidate(ctx, redisx.NSCatalog) // restocked
	}
	if in.Notify == nil || *in.Notify {
		o := updated
		s.notifyAsync(ctx, id, func(ctx context.Context) error { return s.mailer.StatusChanged(ctx, &o) })
	}
	return &updated, nil
}

func (s *Service) SetStaffNote(ctx context.Context, actor string, id bson.ObjectID, note string) (*Order, error) {
	now := time.Now().UTC()
	var o Order
	err := s.orders.FindOneAndUpdate(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"staffNote": strings.TrimSpace(note), "updatedAt": now},
			"$push": pushHistory(Event{Type: "note_updated", At: now, Actor: actor})},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&o)
	if mongox.IsNotFound(err) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, "order.note", "order", id.Hex())
	return &o, nil
}

func (s *Service) SetShipping(ctx context.Context, actor string, id bson.ObjectID, addr Address) (*Order, error) {
	now := time.Now().UTC()
	editable := bson.A{Pending, Confirmed, Processing}
	var o Order
	err := s.orders.FindOneAndUpdate(ctx, bson.M{"_id": id, "status": bson.M{"$in": editable}},
		bson.M{"$set": bson.M{"shipping": addr, "updatedAt": now},
			"$push": pushHistory(Event{Type: "shipping_updated", At: now, Actor: actor})},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&o)
	if mongox.IsNotFound(err) {
		if _, gerr := s.Get(ctx, id); gerr != nil {
			return nil, gerr
		}
		return nil, httpx.Conflict("shipping can no longer be changed for this order")
	}
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, "order.shipping", "order", id.Hex())
	return &o, nil
}
