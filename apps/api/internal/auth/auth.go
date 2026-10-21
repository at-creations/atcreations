// Package auth verifies Clerk session tokens and enforces role-based access.
//
// Roles live in the Clerk user's publicMetadata.role (only settable from the
// backend or the Clerk dashboard, never by the user). For speed, configure a
// session-token template that exposes them as custom claims:
//
//	{"role": "{{user.public_metadata.role}}", "email": "{{user.primary_email_address}}", "name": "{{user.full_name}}"}
//
// When the claims are absent the role is looked up through the Clerk Backend
// API and cached briefly.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/user"

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

type Role string

const (
	RoleStaff   Role = "staff"
	RoleManager Role = "manager"
	RoleAdmin   Role = "admin"
)

func (r Role) Valid() bool { return r == RoleStaff || r == RoleManager || r == RoleAdmin }

type Permission string

const (
	ProductsWrite   Permission = "products:write"
	ProductsDelete  Permission = "products:delete"
	CategoriesWrite Permission = "categories:write"
	OrdersRead      Permission = "orders:read"
	OrdersWrite     Permission = "orders:write"
	OrdersShipping  Permission = "orders:shipping"
	StoreWrite      Permission = "store:write"
	StaffManage     Permission = "staff:manage"
	AuditRead       Permission = "audit:read"
)

var rolePermissions = map[Role][]Permission{
	RoleStaff:   {ProductsWrite, OrdersRead, OrdersWrite},
	RoleManager: {ProductsWrite, ProductsDelete, CategoriesWrite, OrdersRead, OrdersWrite, OrdersShipping},
	RoleAdmin:   {ProductsWrite, ProductsDelete, CategoriesWrite, OrdersRead, OrdersWrite, OrdersShipping, StoreWrite, StaffManage, AuditRead},
}

func PermissionsOf(r Role) []Permission { return slices.Clone(rolePermissions[r]) }

func (r Role) Can(p Permission) bool { return slices.Contains(rolePermissions[r], p) }

// Principal is the authenticated staff member making the request.
type Principal struct {
	UserID    string
	SessionID string
	Email     string
	Name      string
	Role      Role
}

func (p *Principal) Can(perm Permission) bool { return p != nil && p.Role.Can(perm) }

type principalKey struct{}

func PrincipalFrom(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey{}).(*Principal)
	return p
}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Identity is what we know about a Clerk user.
type Identity struct {
	Email, Name string
	Role        Role
}

// Directory looks up identities by Clerk user id.
type Directory interface {
	Identity(ctx context.Context, userID string) (Identity, error)
}

type customClaims struct {
	Role  string `json:"role"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Middleware verifies the bearer token and attaches a Principal. Requests
// without a valid token or without a staff role are rejected.
type Middleware struct {
	verify    func(http.Handler) http.Handler
	directory Directory
}

func NewMiddleware(authorizedParties []string, dir Directory) *Middleware {
	opts := []clerkhttp.AuthorizationOption{
		clerkhttp.Leeway(5 * time.Second),
		clerkhttp.CustomClaimsConstructor(func(context.Context) any { return &customClaims{} }),
		clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpx.WriteError(w, r, httpx.ErrUnauthorized)
		})),
	}
	if len(authorizedParties) > 0 {
		// Reject tokens minted for other origins (and tokens with no azp).
		opts = append(opts, clerkhttp.AuthorizedParty(func(azp string) bool {
			return slices.Contains(authorizedParties, azp)
		}))
	}
	return &Middleware{verify: clerkhttp.WithHeaderAuthorization(opts...), directory: dir}
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return m.verify(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := clerk.SessionClaimsFromContext(r.Context())
		if !ok || claims == nil || claims.Subject == "" {
			httpx.WriteError(w, r, httpx.ErrUnauthorized)
			return
		}
		p := &Principal{UserID: claims.Subject, SessionID: claims.SessionID}
		if cc, ok := claims.Custom.(*customClaims); ok && cc != nil {
			p.Role, p.Email, p.Name = Role(cc.Role), cc.Email, cc.Name
		}
		if !p.Role.Valid() {
			id, err := m.directory.Identity(r.Context(), p.UserID)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			p.Role, p.Email, p.Name = id.Role, id.Email, id.Name
		}
		if !p.Role.Valid() {
			httpx.WriteError(w, r, httpx.NewError(http.StatusForbidden, "your account has no staff role; ask an administrator for access"))
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
	}))
}

// Require rejects principals lacking perm.
func Require(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !PrincipalFrom(r.Context()).Can(perm) {
				httpx.WriteError(w, r, httpx.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClerkDirectory resolves identities via the Clerk Backend API with a short cache.
type ClerkDirectory struct {
	users *user.Client
	ttl   time.Duration
	mu    sync.Mutex
	cache map[string]cachedIdentity
}

type cachedIdentity struct {
	id  Identity
	exp time.Time
}

func NewClerkDirectory(users *user.Client, ttl time.Duration) *ClerkDirectory {
	return &ClerkDirectory{users: users, ttl: ttl, cache: map[string]cachedIdentity{}}
}

func (d *ClerkDirectory) Identity(ctx context.Context, userID string) (Identity, error) {
	d.mu.Lock()
	c, ok := d.cache[userID]
	d.mu.Unlock()
	if ok && time.Now().Before(c.exp) {
		return c.id, nil
	}
	u, err := d.users.Get(ctx, userID)
	if err != nil {
		var apiErr *clerk.APIErrorResponse
		if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusNotFound {
			return Identity{}, httpx.ErrUnauthorized // deleted user
		}
		// Upstream outage or rate limit: report 503 so the client retries
		// instead of bouncing the user through sign-in.
		slog.WarnContext(ctx, "clerk user lookup failed", "err", err)
		return Identity{}, httpx.NewError(http.StatusServiceUnavailable, "identity provider unavailable, try again shortly")
	}
	if u.Banned || u.Locked {
		return Identity{}, httpx.ErrUnauthorized
	}
	id := IdentityOf(u)
	d.mu.Lock()
	if len(d.cache) > 10_000 {
		clear(d.cache)
	}
	d.cache[userID] = cachedIdentity{id: id, exp: time.Now().Add(d.ttl)}
	d.mu.Unlock()
	return id, nil
}

// Forget drops a cached identity (call after role changes).
func (d *ClerkDirectory) Forget(userID string) {
	d.mu.Lock()
	delete(d.cache, userID)
	d.mu.Unlock()
}

// IdentityOf extracts email, display name and role from a Clerk user.
func IdentityOf(u *clerk.User) Identity {
	var meta struct {
		Role string `json:"role"`
	}
	_ = json.Unmarshal(u.PublicMetadata, &meta)
	id := Identity{Role: Role(meta.Role)}
	for _, e := range u.EmailAddresses {
		if u.PrimaryEmailAddressID != nil && e.ID == *u.PrimaryEmailAddressID {
			id.Email = e.EmailAddress
		}
	}
	if u.FirstName != nil {
		id.Name = *u.FirstName
	}
	if u.LastName != nil && *u.LastName != "" {
		if id.Name != "" {
			id.Name += " "
		}
		id.Name += *u.LastName
	}
	if id.Name == "" {
		id.Name = id.Email
	}
	return id
}
