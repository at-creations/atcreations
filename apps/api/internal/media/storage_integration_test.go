//go:build integration

// docker compose -f docker-compose.dev.yml up -d rustfs rustfs-init
// S3_TEST_ENDPOINT=http://127.0.0.1:9000 go test -tags integration ./internal/media/
package media

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestS3StorageAgainstRustFS(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set")
	}
	s := NewS3Storage(S3Config{
		Endpoint: endpoint, Region: "us-east-1", Bucket: "atcreations-dev",
		AccessKeyID: "atcreations-dev", SecretAccessKey: "dev-rustfs-secret-key",
		PublicBaseURL: endpoint + "/atcreations-dev",
	})
	ctx := context.Background()
	prefix := "products/it-" + NewID()
	key := NewKey(prefix, "jpg")

	if err := s.Put(ctx, key, []byte("jpeg-bytes"), "image/jpeg"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Delete(context.Background(), key) })

	// The public URL serves the object anonymously (bucket policy).
	resp, err := http.Get(s.URL(key)) //nolint:gosec,noctx // test against local RustFS
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "jpeg-bytes" {
		t.Fatalf("public GET: %d %q", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("content type %q", ct)
	}

	var found []string
	err = s.Walk(ctx, prefix, func(k string, modified time.Time) error {
		if modified.IsZero() {
			t.Errorf("missing modification time for %s", k)
		}
		found = append(found, k)
		return nil
	})
	if err != nil || len(found) != 1 || found[0] != key {
		t.Fatalf("walk: %v %v", found, err)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	resp, err = http.Get(s.URL(key)) //nolint:gosec,noctx // test against local RustFS
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("object still served after delete: %d", resp.StatusCode)
	}
}

// A KeyPrefix keeps every object under its folder (shared buckets), while callers keep using
// unprefixed keys for writes, reads, listing and deletes.
func TestS3StorageKeyPrefix(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set")
	}
	cfg := S3Config{
		Endpoint: endpoint, Region: "us-east-1", Bucket: "atcreations-dev",
		AccessKeyID: "atcreations-dev", SecretAccessKey: "dev-rustfs-secret-key",
		PublicBaseURL: endpoint + "/atcreations-dev",
	}
	root := NewS3Storage(cfg)
	cfg.KeyPrefix = "it-" + NewID()
	s := NewS3Storage(cfg)
	ctx := context.Background()
	key := NewKey("products", "webp")

	if err := s.Put(ctx, key, []byte("prefixed"), "image/webp"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Delete(context.Background(), key) })

	if data, err := s.Get(ctx, key, 1<<20); err != nil || string(data) != "prefixed" {
		t.Fatalf("Get = %q, %v", data, err)
	}
	if _, err := root.Head(ctx, key); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("object must not exist at the bucket root: err = %v", err)
	}
	if _, err := root.Head(ctx, cfg.KeyPrefix+"/"+key); err != nil {
		t.Fatalf("object must exist under the prefix: %v", err)
	}
	if got, want := s.URL(key), cfg.PublicBaseURL+"/"+cfg.KeyPrefix+"/"+key; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}

	var found []string
	if err := s.Walk(ctx, "products/", func(k string, _ time.Time) error {
		found = append(found, k)
		return nil
	}); err != nil || len(found) != 1 || found[0] != key {
		t.Fatalf("walk: %v %v", found, err)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Head(ctx, cfg.KeyPrefix+"/"+key); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("object still present after delete: err = %v", err)
	}
}

func TestS3StorageGetHeadAndPrivateUploads(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set")
	}
	s := NewS3Storage(S3Config{
		Endpoint: endpoint, Region: "us-east-1", Bucket: "atcreations-dev",
		AccessKeyID: "atcreations-dev", SecretAccessKey: "dev-rustfs-secret-key",
		PublicBaseURL: endpoint + "/atcreations-dev",
	})
	ctx := context.Background()
	key := NewKey(UploadsPrefix+"it-"+NewID(), "png")
	err := s.PutObject(ctx, PutInput{Key: key, Data: []byte("original"), ContentType: "application/octet-stream",
		Metadata: map[string]string{"width": "10"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Delete(context.Background(), key) })

	data, err := s.Get(ctx, key, 1<<20)
	if err != nil || string(data) != "original" {
		t.Fatalf("Get = %q, %v", data, err)
	}
	if _, err := s.Get(ctx, key, 3); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Get over the limit: err = %v, want ErrTooLarge", err)
	}
	info, err := s.Head(ctx, key)
	if err != nil || info.Size != 8 || info.Metadata["width"] != "10" {
		t.Fatalf("Head = %+v, %v", info, err)
	}
	if _, err := s.Get(ctx, key+"-missing", 1<<20); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Get missing: err = %v", err)
	}
	if _, err := s.Head(ctx, key+"-missing"); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Head missing: err = %v", err)
	}

	// Originals in the staging area must never be publicly readable.
	resp, err := http.Get(s.URL(key)) //nolint:gosec,noctx // test against local RustFS
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("uploads/ objects must not be served publicly")
	}
}
