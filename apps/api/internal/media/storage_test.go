package media

import "testing"

func TestNormalizeKeyPrefix(t *testing.T) {
	for in, want := range map[string]string{
		"": "", " ": "", "/": "", "v2": "v2/", "v2/": "v2/", "/v2//": "v2/", " shop/v2 ": "shop/v2/",
	} {
		if got := NormalizeKeyPrefix(in); got != want {
			t.Errorf("NormalizeKeyPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestURLIncludesKeyPrefix(t *testing.T) {
	s := NewS3Storage(S3Config{Endpoint: "http://127.0.0.1:9", Bucket: "b", PublicBaseURL: "https://cdn.example.com/", KeyPrefix: "v2"})
	if got, want := s.URL("products/a.webp"), "https://cdn.example.com/v2/products/a.webp"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}
