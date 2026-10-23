//go:build integration

// docker compose -f docker-compose.dev.yml up -d rustfs rustfs-init
// S3_TEST_ENDPOINT=http://127.0.0.1:9000 go test -tags integration ./internal/imageworker/
package imageworker

import (
	"context"
	"os"
	"testing"

	"google.golang.org/grpc/metadata"

	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
	"github.com/atcreations/atcreations/apps/api/internal/media"
)

// TestHEICAgainstRustFS converts a real iPhone-style HEIC file through S3-compatible storage.
func TestHEICAgainstRustFS(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set")
	}
	store := media.NewS3Storage(media.S3Config{
		Endpoint: endpoint, Region: "us-east-1", Bucket: "atcreations-dev",
		AccessKeyID: "atcreations-dev", SecretAccessKey: "dev-rustfs-secret-key", PublicBaseURL: endpoint + "/atcreations-dev",
	})
	raw, err := os.ReadFile("../media/testdata/sample.heic")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id := media.NewID()
	src, dst := media.UploadsPrefix+"products/it-"+id+"/a.heic", "products/it-"+id+"/out.webp"
	if err := store.Put(ctx, src, raw, "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), src, dst) })

	srv := New(store, 1)
	c := startWith(t, srv)
	ctxAuth := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	_, res, err := run(t, c, ctxAuth, &imagingv1.ProcessImageRequest{JobId: "it", SourceKey: src, OutputKey: dst, MaxEdge: 800, Quality: 80})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetContentType() != "image/webp" || max(res.GetWidth(), res.GetHeight()) > 800 {
		t.Fatalf("result = %+v", res)
	}
	if _, err := store.Head(ctx, dst); err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if _, err := store.Head(ctx, src); err == nil {
		t.Fatal("original must be deleted")
	}
}
