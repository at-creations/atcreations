//go:build integration

//	MONGO_TEST_URI="mongodb://127.0.0.1:27017/?replicaSet=rs0&directConnection=true" \
//	  go test -tags integration ./internal/uploads/
package uploads

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
	"github.com/atcreations/atcreations/apps/api/internal/imageworker"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/media/mediatest"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

const token = "0123456789abcdef0123456789abcdef"

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.RGBA{R: uint8(x), G: 90, B: 160, A: 255}) //nolint:gosec // test pattern
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func setup(t *testing.T, startWorker bool) (*Service, *catalog.Service, *mediatest.Store, bson.ObjectID, func() catalog.Product) {
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
	store := mediatest.New()
	cat := catalog.NewService(db, store, audit.New(db), nil, "integration-test-pepper-0123456789abcdef")

	// The real worker, in-process over gRPC.
	lis := bufconn.Listen(1 << 20)
	unary, stream := imageworker.AuthInterceptors(token)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(unary), grpc.ChainStreamInterceptor(stream))
	imagingv1.RegisterImageWorkerServiceServer(srv, imageworker.New(store, 2))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, m string, req, reply any, cc *grpc.ClientConn, inv grpc.UnaryInvoker, o ...grpc.CallOption) error {
			return inv(metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), m, req, reply, cc, o...)
		}),
		grpc.WithStreamInterceptor(func(ctx context.Context, d *grpc.StreamDesc, cc *grpc.ClientConn, m string, s grpc.Streamer, o ...grpc.CallOption) (grpc.ClientStream, error) {
			return s(metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), d, cc, m, o...)
		}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	svc := New(db, store, cat, imagingv1.NewImageWorkerServiceClient(conn), Options{Concurrency: 2, MaxEdge: 200, MaxAttempts: 3})
	if err := svc.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	cat.OnOwnerDeleted(svc.CancelOwner)
	if startWorker {
		runCtx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)
		svc.Run(runCtx)
	}

	pid := bson.NewObjectID()
	_, err = db.Collection("products").InsertOne(ctx, bson.M{
		"_id": pid, "slug": "vase-" + pid.Hex(), "status": "draft", "name": bson.M{"en": "Vase"},
		"images": bson.A{}, "categoryIds": bson.A{}, "version": 1, "createdAt": time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	get := func() catalog.Product {
		var p catalog.Product
		if err := db.Collection("products").FindOne(ctx, bson.M{"_id": pid}).Decode(&p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	return svc, cat, store, pid, get
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func jobStatus(t *testing.T, s *Service, id bson.ObjectID) Job {
	t.Helper()
	var j Job
	if err := s.jobs.FindOne(context.Background(), bson.M{"_id": id}).Decode(&j); err != nil {
		t.Fatal(err)
	}
	return j
}

func TestUploadsProcessInParallel(t *testing.T) {
	svc, _, store, pid, product := setup(t, true)
	ctx := context.Background()
	var ids []bson.ObjectID
	for i := range 3 {
		job, err := svc.Enqueue(ctx, "user_1", KindProduct, pid, "photo.png", pngBytes(t, 400+i, 300))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, job.ID)
	}
	waitFor(t, "three images attached", func() bool { return len(product().Images) == 3 })
	// Jobs are marked done and originals deleted right after the attach.
	waitFor(t, "jobs finished", func() bool {
		for _, id := range ids {
			if jobStatus(t, svc, id).Status != StatusDone {
				return false
			}
		}
		return len(store.Keys(media.UploadsPrefix)) == 0
	})
	p := product()
	if p.PendingImages != 0 {
		t.Fatalf("pendingImages = %d, want 0", p.PendingImages)
	}
	for _, img := range p.Images {
		if max(img.Width, img.Height) != 200 || !store.Has(img.Key) {
			t.Fatalf("bad image %+v", img)
		}
	}
	if left := store.Keys(media.UploadsPrefix); len(left) != 0 {
		t.Fatalf("originals left behind: %v", left)
	}
	for _, id := range ids {
		if j := jobStatus(t, svc, id); j.Status != StatusDone || j.Percent != 100 {
			t.Fatalf("job %s = %s %d%%", id.Hex(), j.Status, j.Percent)
		}
	}
}

func TestUploadLimitIsAtomic(t *testing.T) {
	svc, _, _, pid, _ := setup(t, false)
	ctx := context.Background()
	for i := range catalog.MaxProductImages {
		if _, err := svc.Enqueue(ctx, "u", KindProduct, pid, "a.png", pngBytes(t, 10, 10)); err != nil {
			t.Fatalf("upload %d: %v", i, err)
		}
	}
	_, err := svc.Enqueue(ctx, "u", KindProduct, pid, "a.png", pngBytes(t, 10, 10))
	var he *httpx.Error
	if !errors.As(err, &he) || he.Status != 422 {
		t.Fatalf("11th upload: err = %v, want 422", err)
	}
}

func TestCorruptImageFailsAndCleansUp(t *testing.T) {
	svc, _, store, pid, product := setup(t, true)
	data := pngBytes(t, 50, 50)
	data = data[:len(data)/2] // valid header, truncated pixels: passes Inspect, fails decoding
	job, err := svc.Enqueue(context.Background(), "u", KindProduct, pid, "broken.png", data)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "job failed", func() bool { return jobStatus(t, svc, job.ID).Status == StatusFailed })
	// The slot is released right after the job is marked failed.
	waitFor(t, "slot released", func() bool { return product().PendingImages == 0 })
	if store.Has(job.SourceKey) || store.Has(job.OutputKey) {
		t.Fatal("original and output must be gone")
	}
	if p := product(); p.PendingImages != 0 || len(p.Images) != 0 {
		t.Fatalf("product = %d images, %d pending", len(p.Images), p.PendingImages)
	}
}

func TestTransientFailureIsRetried(t *testing.T) {
	svc, _, store, pid, product := setup(t, true)
	store.FailPuts = 1 // the worker's first output write is throttled
	// Enqueue writes the original first, so let that one through.
	store.FailPuts = 0
	job, err := svc.Enqueue(context.Background(), "u", KindProduct, pid, "a.png", pngBytes(t, 300, 300))
	if err != nil {
		t.Fatal(err)
	}
	store.FailPuts = 1
	svc.Wake()
	// The image is attached before the job is marked done (separate writes): wait for both,
	// or a slow runner can read the job between them.
	waitFor(t, "image attached and job done after a retry", func() bool {
		return len(product().Images) == 1 && jobStatus(t, svc, job.ID).Status == StatusDone
	})
	if j := jobStatus(t, svc, job.ID); j.Attempts < 2 {
		t.Fatalf("job = %s after %d attempts, want done after a retry", j.Status, j.Attempts)
	}
}

func TestCancelAndOwnerDeletionRemoveOriginals(t *testing.T) {
	svc, cat, store, pid, product := setup(t, false) // no dispatcher: jobs stay queued
	ctx := context.Background()
	a, _ := svc.Enqueue(ctx, "u", KindProduct, pid, "a.png", pngBytes(t, 20, 20))
	b, _ := svc.Enqueue(ctx, "u", KindProduct, pid, "b.png", pngBytes(t, 20, 20))

	if err := svc.Cancel(ctx, KindProduct, pid, a.ID); err != nil {
		t.Fatal(err)
	}
	if store.Has(a.SourceKey) || jobStatus(t, svc, a.ID).Status != StatusCancelled {
		t.Fatal("cancel must delete the original and mark the job cancelled")
	}
	if p := product(); p.PendingImages != 1 {
		t.Fatalf("pendingImages = %d, want 1 after cancelling one of two", p.PendingImages)
	}

	if err := cat.DeleteProduct(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if store.Has(b.SourceKey) || jobStatus(t, svc, b.ID).Status != StatusCancelled {
		t.Fatal("deleting the product must cancel its uploads and delete their originals")
	}
}

func TestSweepRemovesStaleOriginals(t *testing.T) {
	svc, _, store, pid, _ := setup(t, false)
	ctx := context.Background()
	live, _ := svc.Enqueue(ctx, "u", KindProduct, pid, "a.png", pngBytes(t, 20, 20))
	_ = store.Put(ctx, "uploads/products/orphan.png", []byte("x"), "image/png")
	store.Age("uploads/products/orphan.png", 48*time.Hour)
	store.Age(live.SourceKey, 48*time.Hour) // old but still queued: must stay
	n, err := svc.Sweep(ctx)
	if err != nil || n != 1 {
		t.Fatalf("Sweep = %d, %v", n, err)
	}
	if store.Has("uploads/products/orphan.png") || !store.Has(live.SourceKey) {
		t.Fatal("sweep must remove only originals without a live job")
	}
}

func TestBackfillVariantsForStoredImages(t *testing.T) {
	svc, cat, store, pid, product := setup(t, false)
	svc.opts.MaxEdge = 2400 // production size: variants are below the stored image's edge
	ctx := context.Background()
	for _, img := range []struct {
		id   string
		w, h int
	}{{"legacy", 1000, 500}, {"tiny", 100, 100}} {
		key := "products/" + pid.Hex() + "/" + img.id + ".webp"
		_ = store.Put(ctx, key, pngBytes(t, img.w, img.h), "image/webp")
		if err := cat.AttachProductImage(ctx, "u", pid, catalog.Image{ID: img.id, Key: key, Width: img.w, Height: img.h}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.BackfillVariants(ctx); err != nil {
		t.Fatal(err)
	}
	got := map[string][]catalog.ImageVariant{}
	for _, img := range product().Images {
		got[img.ID] = img.Variants
	}
	if v := got["legacy"]; len(v) != 2 || v[0].Width != 400 || v[1].Width != 800 || !store.Has(v[1].Key) {
		t.Fatalf("legacy variants = %+v", v)
	}
	if v := got["tiny"]; v == nil || len(v) != 0 {
		t.Fatalf("tiny image must be marked done with no variants, got %#v", v)
	}
	if left, err := cat.ImagesWithoutVariants(ctx, 10, nil); err != nil || len(left) != 0 {
		t.Fatalf("still without variants: %v %v", left, err)
	}
	// Deleting the image deletes its variants too (by derived key).
	if _, err := cat.RemoveProductImage(ctx, "u", pid, "legacy"); err != nil {
		t.Fatal(err)
	}
	if _, err := cat.SweepOrphanedMedia(ctx, -time.Hour); err != nil {
		t.Fatal(err)
	}
	for _, k := range media.WithVariants("products/" + pid.Hex() + "/legacy.webp") {
		if store.Has(k) {
			t.Fatalf("%s left after the image was removed", k)
		}
	}
}
