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
	cfg := &config.Config{CORSOrigins: []string{"https://admin.example.com"}}
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

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	h := testRouter()
	for origin, allowed := range map[string]bool{"https://admin.example.com": true, "https://evil.example": false} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/v1/admin/me", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		req.Header.Set("Access-Control-Request-Headers", "authorization")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		got := rec.Header().Get("Access-Control-Allow-Origin") == origin
		if got != allowed {
			t.Errorf("origin %s: allowed=%v, want %v", origin, got, allowed)
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Errorf("credentials must never be allowed")
		}
	}
}
