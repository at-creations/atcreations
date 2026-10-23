//go:build integration

// docker compose -f docker-compose.dev.yml up -d rustfs rustfs-init
// S3_TEST_ENDPOINT=http://127.0.0.1:9000 go test -tags integration ./internal/media/
package media

import (
	"context"
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
