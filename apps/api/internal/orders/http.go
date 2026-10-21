package orders

import (
	"net/http"
	"strings"
	"time"

	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

type Handlers struct {
	svc            *Service
	publicCheckout bool
}

func NewHandlers(svc *Service, publicCheckout bool) *Handlers {
	return &Handlers{svc: svc, publicCheckout: publicCheckout}
}

func actor(r *http.Request) string {
	if p := auth.PrincipalFrom(r.Context()); p != nil {
		return p.UserID
	}
	return "system"
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) error {
	if !h.publicCheckout {
		return httpx.NewError(http.StatusNotFound, "online ordering is not available")
	}
	var in CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	o, code, err := h.svc.Create(r.Context(), &in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"number": o.Number, "accessCode": code, "totalCents": o.TotalCents})
	return nil
}

func (h *Handlers) Track(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Number     string `json:"number" validate:"required,max=32"`
		Email      string `json:"email" validate:"required,max=254"`
		AccessCode string `json:"accessCode" validate:"required,max=32"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	t, err := h.svc.Track(r.Context(), in.Number, in.Email, in.AccessCode)
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, t)
	return nil
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) error {
	q := httpx.NewQuery(r)
	lq := ListQuery{
		Status: q.Enum("status", "", "pending", "confirmed", "processing", "shipped", "delivered", "completed", "cancelled", "refunded"),
		Text:   q.String("q", 100),
		Sort:   q.Enum("sort", "newest", "newest", "oldest", "total_desc", "total_asc"),
		Page:   q.Page(25, 100),
	}
	from, to := q.String("from", 10), q.String("to", 10)
	if err := q.Err(); err != nil {
		return err
	}
	for _, d := range []struct {
		raw string
		dst **time.Time
	}{{from, &lq.From}, {to, &lq.To}} {
		if d.raw == "" {
			continue
		}
		t, err := time.Parse(time.DateOnly, d.raw)
		if err != nil {
			return httpx.BadRequest("from/to must be YYYY-MM-DD dates")
		}
		*d.dst = &t
	}
	items, meta, err := h.svc.List(r.Context(), lq)
	if err != nil {
		return err
	}
	httpx.Paged(w, items, meta)
	return nil
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	o, err := h.svc.Get(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}

func (h *Handlers) Transition(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	var in TransitionInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	o, err := h.svc.Transition(r.Context(), actor(r), id, &in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}

func (h *Handlers) SetNote(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		StaffNote string `json:"staffNote" validate:"max=5000"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	o, err := h.svc.SetStaffNote(r.Context(), actor(r), id, in.StaffNote)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}

func (h *Handlers) SetShipping(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	var in Address
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	in.Country = strings.ToUpper(in.Country)
	o, err := h.svc.SetShipping(r.Context(), actor(r), id, in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}
