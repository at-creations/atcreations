package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/cors"
)

// Headers browsers may read from cross-origin responses.
var exposedHeaders = []string{"X-Request-ID", "Retry-After"}

const adminPrefix = "/v1/admin"

func isAdminPath(path string) bool {
	return path == adminPrefix || strings.HasPrefix(path, adminPrefix+"/")
}

// corsPolicies applies a separate CORS policy to the admin API and to the
// public API. Preflight requests never reach the router, so the policy is
// chosen by path before routing:
//
//   - /v1/admin/*: admin origins only; bearer token (Authorization) allowed.
//   - everything else: storefront and admin origins; GET/POST, no
//     Authorization header (public endpoints never authenticate).
//
// Credentials (cookies) are never allowed: the API only uses bearer tokens.
func corsPolicies(publicOrigins, adminOrigins []string) func(http.Handler) http.Handler {
	public := cors.Handler(cors.Options{
		// The admin dashboard also reads public endpoints (e.g. store info).
		AllowedOrigins:   slices.Compact(slices.Sorted(slices.Values(append(slices.Clone(publicOrigins), adminOrigins...)))),
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Content-Type", "X-Request-ID"},
		ExposedHeaders:   exposedHeaders,
		AllowCredentials: false,
		MaxAge:           600,
	})
	admin := cors.Handler(cors.Options{
		AllowedOrigins:   adminOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   exposedHeaders,
		AllowCredentials: false,
		MaxAge:           600,
	})
	return func(next http.Handler) http.Handler {
		pub, adm := public(next), admin(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAdminPath(r.URL.Path) {
				adm.ServeHTTP(w, r)
				return
			}
			pub.ServeHTTP(w, r)
		})
	}
}
