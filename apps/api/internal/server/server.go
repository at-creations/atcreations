// Package server wires handlers, middleware and rate limits into a router.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	httprateredis "github.com/go-chi/httprate-redis"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	"github.com/atcreations/atcreations/apps/api/internal/config"
	"github.com/atcreations/atcreations/apps/api/internal/orders"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
	"github.com/atcreations/atcreations/apps/api/internal/staff"
	"github.com/atcreations/atcreations/apps/api/internal/storeinfo"
)

type Deps struct {
	Config    *config.Config
	Mongo     *mongo.Client
	Redis     *redis.Client // optional; nil means in-memory limits and no cache
	Cache     *redisx.Cache
	Auth      *auth.Middleware
	Catalog   *catalog.Service
	Orders    *orders.Handlers
	StoreInfo *storeinfo.Service
	Staff     *staff.Service
	Audit     *audit.Log
}

func byClientIP(r *http.Request) (string, error) { return httpx.ClientIPFrom(r.Context()), nil }

// limiter builds named per-IP rate limiters. With Redis the counters are
// shared by every API replica and survive restarts; if Redis becomes
// unreachable the limiter falls back to local in-memory counters.
func limiter(rdb *redis.Client) func(name string, n int, window time.Duration) func(http.Handler) http.Handler {
	return func(name string, n int, window time.Duration) func(http.Handler) http.Handler {
		opts := []httprate.Option{
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				httpx.WriteError(w, r, httpx.NewError(http.StatusTooManyRequests, "too many requests, slow down"))
			}),
		}
		if rdb != nil {
			opts = append(opts, httprateredis.WithRedisLimitCounter(&httprateredis.Config{
				Client:          rdb,
				PrefixKey:       "ratelimit:" + name,
				FallbackTimeout: 200 * time.Millisecond,
				OnFallbackChange: func(active bool) {
					slog.Warn("rate limiter redis fallback changed", "limiter", name, "local_fallback", active)
				},
			}))
		}
		return httprate.LimitBy(n, window, byClientIP, opts...)
	}
}

func New(d Deps) http.Handler {
	limit := limiter(d.Redis)
	r := chi.NewRouter()
	r.Use(
		httpx.RequestID,
		httpx.ClientIP(d.Config.TrustedProxyPrefixes),
		httpx.AccessLog,
		httpx.Recover,
		httpx.SecurityHeaders,
		cors.Handler(cors.Options{
			AllowedOrigins:   d.Config.CORSOrigins,
			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
			ExposedHeaders:   []string{"X-Request-ID"},
			AllowCredentials: false, // bearer tokens only; no cookies are ever used
			MaxAge:           600,
		}),
		limit("global", 300, time.Minute), // per-IP ceiling
	)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { httpx.WriteError(w, r, httpx.ErrNotFound) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, r, httpx.NewError(http.StatusMethodNotAllowed, "method not allowed"))
	})

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", httpx.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := d.Mongo.Ping(ctx, readpref.Primary()); err != nil {
			return httpx.NewError(http.StatusServiceUnavailable, "not ready")
		}
		if d.Redis != nil {
			if err := d.Redis.Ping(ctx).Err(); err != nil {
				return httpx.NewError(http.StatusServiceUnavailable, "not ready")
			}
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
		return nil
	}))

	w := httpx.Wrap
	c, o, st := d.Catalog, d.Orders, d.Staff

	r.Route("/v1", func(r chi.Router) {
		// ---- Public catalogue ----
		r.Group(func(r chi.Router) {
			r.Use(httpx.PublicCache(60 * time.Second))
			catalogCache := d.Cache.Middleware(redisx.NSCatalog, 5*time.Minute)
			r.With(catalogCache).Get("/categories", w(c.HandlePublicCategories))
			r.With(catalogCache).Get("/categories/{slug}", w(c.HandlePublicCategory))
			r.With(catalogCache).Get("/products", w(c.HandlePublicProducts))
			r.With(catalogCache).Get("/products/{slug}", w(c.HandlePublicProduct))
			r.With(d.Cache.Middleware(redisx.NSStore, time.Hour)).Get("/store-info", w(d.StoreInfo.HandleGet))
		})
		r.With(limit("views", 30, time.Minute)).Post("/products/{id}/views", w(c.HandleRecordView))
		r.With(limit("orders", 5, 10*time.Minute)).Post("/orders", w(o.Create))
		r.With(limit("track", 10, 15*time.Minute)).Post("/orders/track", w(o.Track))

		// ---- Admin (Clerk-authenticated) ----
		r.Route("/admin", func(r chi.Router) {
			r.Use(httpx.NoStore, d.Auth.Handler, limit("admin", 600, time.Minute))
			can := auth.Require

			r.Get("/me", w(handleMe))

			r.Get("/products", w(c.HandleAdminProducts))
			r.Get("/products/{id}", w(c.HandleAdminProduct))
			r.With(can(auth.ProductsWrite)).Post("/products", w(c.HandleCreateProduct))
			r.With(can(auth.ProductsWrite)).Put("/products/{id}", w(c.HandleUpdateProduct))
			r.With(can(auth.ProductsDelete)).Delete("/products/{id}", w(c.HandleDeleteProduct))
			r.With(can(auth.ProductsWrite), limit("uploads", 60, time.Minute)).Post("/products/{id}/images", w(c.HandleAddProductImage))
			r.With(can(auth.ProductsWrite)).Put("/products/{id}/images/order", w(c.HandleReorderProductImages))
			r.With(can(auth.ProductsWrite)).Delete("/products/{id}/images/{imageId}", w(c.HandleRemoveProductImage))

			r.Get("/categories", w(c.HandlePublicCategories))
			r.With(can(auth.CategoriesWrite)).Post("/categories", w(c.HandleCreateCategory))
			r.With(can(auth.CategoriesWrite)).Put("/categories/{id}", w(c.HandleUpdateCategory))
			r.With(can(auth.CategoriesWrite)).Delete("/categories/{id}", w(c.HandleDeleteCategory))
			r.With(can(auth.CategoriesWrite), limit("uploads", 60, time.Minute)).Put("/categories/{id}/image", w(c.HandleSetCategoryImage))
			r.With(can(auth.CategoriesWrite)).Delete("/categories/{id}/image", w(c.HandleRemoveCategoryImage))

			r.With(can(auth.StoreWrite)).Put("/store-info", w(d.StoreInfo.HandlePut))

			r.With(can(auth.OrdersRead)).Get("/orders", w(o.List))
			r.With(can(auth.OrdersRead)).Get("/orders/{id}", w(o.Get))
			r.With(can(auth.OrdersWrite)).Post("/orders/{id}/status", w(o.Transition))
			r.With(can(auth.OrdersWrite)).Put("/orders/{id}/notes", w(o.SetNote))
			r.With(can(auth.OrdersShipping)).Put("/orders/{id}/shipping", w(o.SetShipping))

			r.Group(func(r chi.Router) {
				r.Use(can(auth.StaffManage), limit("staff", 60, time.Minute))
				r.Get("/staff", w(st.HandleList))
				r.Get("/staff/invitations", w(st.HandleListInvitations))
				r.Post("/staff/invitations", w(st.HandleInvite))
				r.Delete("/staff/invitations/{invitationId}", w(st.HandleRevokeInvitation))
				r.Put("/staff/{userId}/role", w(st.HandleSetRole))
				r.Post("/staff/{userId}/ban", w(st.HandleBan))
				r.Post("/staff/{userId}/unban", w(st.HandleUnban))
				r.Delete("/staff/{userId}", w(st.HandleDelete))
			})

			r.With(can(auth.AuditRead)).Get("/audit", w(d.Audit.HandleList))
		})
	})
	return r
}

func handleMe(w http.ResponseWriter, r *http.Request) error {
	p := auth.PrincipalFrom(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"id": p.UserID, "email": p.Email, "name": p.Name, "role": p.Role, "permissions": auth.PermissionsOf(p.Role),
	})
	return nil
}
