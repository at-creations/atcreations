package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/config"
)

type noDirectory struct{}

func (noDirectory) Identity(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, nil
}

func testRouter() http.Handler {
	cfg := &config.Config{
		CORSPublicOrigins: []string{"https://shop.example.com"},
		CORSAdminOrigins:  []string{"https://admin.example.com"},
	}
	return New(Deps{Config: cfg, Auth: auth.NewMiddleware(nil, noDirectory{})})
}

func TestAdminRoutesRequireToken(t *testing.T) {
	h := testRouter()
	for _, path := range []string{"/v1/admin/me", "/v1/admin/products", "/v1/admin/staff", "/v1/admin/audit"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without token: got %d, want 401", path, rec.Code)
		}
	}
}

func TestAdminRoutesRejectGarbageToken(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/admin/me", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
}

func TestSecurityHeadersAndUnknownRoutes(t *testing.T) {
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d", rec.Code)
	}
	for _, h := range []string{"X-Content-Type-Options", "Content-Security-Policy", "X-Frame-Options", "X-Request-ID"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing header %s", h)
		}
	}
}

func preflight(h http.Handler, path, origin, method, headers string) http.Header {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, path, nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", method)
	if headers != "" {
		req.Header.Set("Access-Control-Request-Headers", headers)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Header()
}

func TestCORSPolicies(t *testing.T) {
	h := testRouter()
	const shop, admin, evil = "https://shop.example.com", "https://admin.example.com", "https://evil.example"
	cases := []struct {
		name                         string
		path, origin, method, header string
		allowed                      bool
	}{
		{"admin origin → admin API", "/v1/admin/products", admin, http.MethodPut, "authorization,content-type", true},
		{"storefront origin → admin API", "/v1/admin/me", shop, http.MethodGet, "authorization", false},
		{"unknown origin → admin API", "/v1/admin/me", evil, http.MethodGet, "authorization", false},
		{"storefront origin → public view counter", "/v1/products/x/views", shop, http.MethodPost, "", true},
		{"admin origin → public store info", "/v1/store-info", admin, http.MethodGet, "", true},
		{"unknown origin → public API", "/v1/products", evil, http.MethodGet, "", false},
		{"public API never accepts Authorization", "/v1/products", shop, http.MethodGet, "authorization", false},
		{"public API refuses DELETE", "/v1/products/x", shop, http.MethodDelete, "", false},
		{"lookalike path is public, not admin", "/v1/administrator", admin, http.MethodPut, "authorization", false},
	}
	for _, c := range cases {
		hdr := preflight(h, c.path, c.origin, c.method, c.header)
		got := hdr.Get("Access-Control-Allow-Origin") == c.origin
		if got != c.allowed {
			t.Errorf("%s: allowed=%v, want %v", c.name, got, c.allowed)
		}
		if hdr.Get("Access-Control-Allow-Credentials") != "" {
			t.Errorf("%s: credentials must never be allowed", c.name)
		}
	}
}
