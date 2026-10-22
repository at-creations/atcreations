// Package staff manages admin-dashboard users through the Clerk Backend API.
// Accounts are invitation-only; roles live in Clerk publicMetadata.role.
package staff

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/invitation"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/redis/go-redis/v9"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
)

type Member struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	ImageURL     string     `json:"imageUrl,omitempty"`
	Role         string     `json:"role"`
	Banned       bool       `json:"banned"`
	LastSignInAt *time.Time `json:"lastSignInAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type Invite struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Service struct {
	// Guarded mutations are serialized so the last-admin check and the change
	// it protects cannot interleave (e.g. two admins demoting each other).
	// With Redis the lock is shared by all API replicas; without it, mu is used.
	mu          sync.Mutex
	rdb         *redis.Client
	users       *user.Client
	invitations *invitation.Client
	directory   *auth.ClerkDirectory
	audit       *audit.Log
	redirectURL string
}

func New(users *user.Client, invitations *invitation.Client, dir *auth.ClerkDirectory, a *audit.Log, rdb *redis.Client, redirectURL string) *Service {
	return &Service{rdb: rdb, users: users, invitations: invitations, directory: dir, audit: a, redirectURL: redirectURL}
}

func ms(v int64) time.Time { return time.UnixMilli(v).UTC() }

func toMember(u *clerk.User) Member {
	id := auth.IdentityOf(u)
	role := string(id.Role)
	if !id.Role.Valid() {
		role = "none"
	}
	m := Member{ID: u.ID, Email: id.Email, Name: id.Name, Role: role, Banned: u.Banned, CreatedAt: ms(u.CreatedAt)}
	if u.ImageURL != nil {
		m.ImageURL = *u.ImageURL
	}
	if u.LastSignInAt != nil {
		t := ms(*u.LastSignInAt)
		m.LastSignInAt = &t
	}
	return m
}

func inviteRole(meta json.RawMessage) string {
	var v struct {
		Role string `json:"role"`
	}
	_ = json.Unmarshal(meta, &v)
	return v.Role
}

// clerkError maps Clerk API failures to client-safe problems.
func clerkError(err error) error {
	var apiErr *clerk.APIErrorResponse
	if !errors.As(err, &apiErr) {
		return err
	}
	msg := "the identity provider rejected the request"
	if len(apiErr.Errors) > 0 && apiErr.Errors[0].LongMessage != "" {
		msg = apiErr.Errors[0].LongMessage
	}
	switch apiErr.HTTPStatusCode {
	case http.StatusNotFound:
		return httpx.ErrNotFound
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return httpx.Validation(httpx.FieldError{Field: "email", Message: msg})
	case http.StatusConflict:
		return httpx.Conflict(msg)
	default:
		slog.Error("clerk api error", "status", apiErr.HTTPStatusCode, "trace", apiErr.TraceID)
		return httpx.NewError(http.StatusBadGateway, "identity provider unavailable, try again later")
	}
}

// activeAdmins counts non-banned admins across all users.
func (s *Service) activeAdmins(ctx context.Context) (int, error) {
	n := 0
	for offset := int64(0); offset < 5000; offset += 500 {
		list, err := s.users.List(ctx, &user.ListParams{ListParams: clerk.ListParams{Limit: clerk.Int64(500), Offset: clerk.Int64(offset)}})
		if err != nil {
			return 0, clerkError(err)
		}
		for _, u := range list.Users {
			if !u.Banned && auth.IdentityOf(u).Role == auth.RoleAdmin {
				n++
			}
		}
		if len(list.Users) < 500 {
			break
		}
	}
	return n, nil
}

// GuardTarget enforces the safety rules for mutating another account:
// nobody may modify themselves, and the last active admin cannot be removed.
func GuardTarget(actorID, targetID string, targetIsActiveAdmin bool, removesAdmin bool, activeAdmins int) error {
	if actorID == targetID {
		return httpx.Conflict("you cannot change your own account here")
	}
	if targetIsActiveAdmin && removesAdmin && activeAdmins <= 1 {
		return httpx.Conflict("this is the last active administrator")
	}
	return nil
}

func (s *Service) guard(ctx context.Context, target *clerk.User, removesAdmin bool) error {
	p := auth.PrincipalFrom(ctx)
	isAdmin := !target.Banned && auth.IdentityOf(target).Role == auth.RoleAdmin
	admins := 2
	if isAdmin && removesAdmin {
		var err error
		if admins, err = s.activeAdmins(ctx); err != nil {
			return err
		}
	}
	return GuardTarget(p.UserID, target.ID, isAdmin, removesAdmin, admins)
}

func (s *Service) target(ctx context.Context, id string) (*clerk.User, error) {
	if !strings.HasPrefix(id, "user_") || len(id) > 64 {
		return nil, httpx.ErrNotFound
	}
	u, err := s.users.Get(ctx, id)
	if err != nil {
		return nil, clerkError(err)
	}
	return u, nil
}

// lock serializes staff mutations across replicas.
func (s *Service) lock(ctx context.Context) (func(), error) {
	if s.rdb == nil {
		s.mu.Lock()
		return s.mu.Unlock, nil
	}
	unlock, err := redisx.Lock(ctx, s.rdb, "staff-mutation", 30*time.Second, 10*time.Second)
	if errors.Is(err, redisx.ErrLockHeld) {
		return nil, httpx.Conflict("another staff change is in progress; try again")
	}
	if err != nil {
		return nil, err
	}
	return unlock, nil
}

// ---------- handlers ----------

func (s *Service) HandleList(w http.ResponseWriter, r *http.Request) error {
	q := httpx.NewQuery(r)
	page := q.Page(25, 100)
	text := q.String("q", 100)
	if err := q.Err(); err != nil {
		return err
	}
	params := &user.ListParams{
		ListParams: clerk.ListParams{Limit: clerk.Int64(int64(page.Size)), Offset: clerk.Int64(page.Skip())},
		OrderBy:    clerk.String("-created_at"),
	}
	if text != "" {
		params.Query = clerk.String(text)
	}
	list, err := s.users.List(r.Context(), params)
	if err != nil {
		return clerkError(err)
	}
	out := make([]Member, len(list.Users))
	for i, u := range list.Users {
		out[i] = toMember(u)
	}
	httpx.Paged(w, out, page.Meta(list.TotalCount))
	return nil
}

func (s *Service) HandleListInvitations(w http.ResponseWriter, r *http.Request) error {
	list, err := s.invitations.List(r.Context(), &invitation.ListParams{
		ListParams: clerk.ListParams{Limit: clerk.Int64(100)},
		Statuses:   []string{"pending"},
	})
	if err != nil {
		return clerkError(err)
	}
	out := make([]Invite, len(list.Invitations))
	for i, inv := range list.Invitations {
		out[i] = Invite{ID: inv.ID, Email: inv.EmailAddress, Role: inviteRole(inv.PublicMetadata), Status: inv.Status, CreatedAt: ms(inv.CreatedAt)}
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (s *Service) HandleInvite(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email string    `json:"email" validate:"required,email,max=254"`
		Role  auth.Role `json:"role" validate:"required,oneof=staff manager admin"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]string{"role": string(in.Role)})
	raw := json.RawMessage(meta)
	params := &invitation.CreateParams{
		EmailAddress:   strings.ToLower(strings.TrimSpace(in.Email)),
		PublicMetadata: &raw,
		Notify:         clerk.Bool(true),
		ExpiresInDays:  clerk.Int64(7),
	}
	if s.redirectURL != "" {
		params.RedirectURL = clerk.String(s.redirectURL)
	}
	inv, err := s.invitations.Create(r.Context(), params)
	if err != nil {
		return clerkError(err)
	}
	s.audit.Record(r.Context(), "staff.invite."+string(in.Role), "invitation", inv.ID)
	httpx.JSON(w, http.StatusCreated, Invite{ID: inv.ID, Email: inv.EmailAddress, Role: string(in.Role), Status: inv.Status, CreatedAt: ms(inv.CreatedAt)})
	return nil
}

func (s *Service) HandleRevokeInvitation(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("invitationId")
	if !strings.HasPrefix(id, "inv_") || len(id) > 64 {
		return httpx.ErrNotFound
	}
	if _, err := s.invitations.Revoke(r.Context(), id); err != nil {
		return clerkError(err)
	}
	s.audit.Record(r.Context(), "staff.invite.revoke", "invitation", id)
	httpx.NoContent(w)
	return nil
}

func (s *Service) HandleSetRole(w http.ResponseWriter, r *http.Request) error {
	unlock, err := s.lock(r.Context())
	if err != nil {
		return err
	}
	defer unlock()
	var in struct {
		Role auth.Role `json:"role" validate:"required,oneof=staff manager admin"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	target, err := s.target(r.Context(), r.PathValue("userId"))
	if err != nil {
		return err
	}
	if err := s.guard(r.Context(), target, in.Role != auth.RoleAdmin); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]string{"role": string(in.Role)})
	raw := json.RawMessage(meta)
	u, err := s.users.UpdateMetadata(r.Context(), target.ID, &user.UpdateMetadataParams{PublicMetadata: &raw})
	if err != nil {
		return clerkError(err)
	}
	s.directory.Forget(target.ID)
	s.audit.Record(r.Context(), "staff.role."+string(in.Role), "user", target.ID)
	httpx.JSON(w, http.StatusOK, toMember(u))
	return nil
}

func (s *Service) HandleBan(w http.ResponseWriter, r *http.Request) error {
	unlock, err := s.lock(r.Context())
	if err != nil {
		return err
	}
	defer unlock()
	target, err := s.target(r.Context(), r.PathValue("userId"))
	if err != nil {
		return err
	}
	if err := s.guard(r.Context(), target, true); err != nil {
		return err
	}
	// Banning also revokes every active session in Clerk.
	u, err := s.users.Ban(r.Context(), target.ID)
	if err != nil {
		return clerkError(err)
	}
	s.directory.Forget(target.ID)
	s.audit.Record(r.Context(), "staff.ban", "user", target.ID)
	httpx.JSON(w, http.StatusOK, toMember(u))
	return nil
}

func (s *Service) HandleUnban(w http.ResponseWriter, r *http.Request) error {
	unlock, err := s.lock(r.Context())
	if err != nil {
		return err
	}
	defer unlock()
	target, err := s.target(r.Context(), r.PathValue("userId"))
	if err != nil {
		return err
	}
	if err := s.guard(r.Context(), target, false); err != nil {
		return err
	}
	u, err := s.users.Unban(r.Context(), target.ID)
	if err != nil {
		return clerkError(err)
	}
	s.audit.Record(r.Context(), "staff.unban", "user", target.ID)
	httpx.JSON(w, http.StatusOK, toMember(u))
	return nil
}

func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) error {
	unlock, err := s.lock(r.Context())
	if err != nil {
		return err
	}
	defer unlock()
	target, err := s.target(r.Context(), r.PathValue("userId"))
	if err != nil {
		return err
	}
	if err := s.guard(r.Context(), target, true); err != nil {
		return err
	}
	if _, err := s.users.Delete(r.Context(), target.ID); err != nil {
		return clerkError(err)
	}
	s.directory.Forget(target.ID)
	s.audit.Record(r.Context(), "staff.delete", "user", target.ID)
	httpx.NoContent(w)
	return nil
}
