package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestClientIPOnlyTrustsConfiguredProxies(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	cases := []struct {
		name, remote, xff, want string
	}{
		{"direct client ignores spoofed header", "203.0.113.5:1234", "1.2.3.4", "203.0.113.5"},
		{"trusted proxy uses right-most untrusted hop", "10.0.0.2:80", "6.6.6.6, 198.51.100.7", "198.51.100.7"},
		{"skips chained trusted proxies", "10.0.0.2:80", "198.51.100.7, 10.0.0.9", "198.51.100.7"},
		{"garbage header falls back to peer", "10.0.0.2:80", "not-an-ip", "10.0.0.2"},
	}
	for _, c := range cases {
		var got string
		h := ClientIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = ClientIPFrom(r.Context()) }))
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.RemoteAddr = c.remote
		req.Header.Set("X-Forwarded-For", c.xff)
		h.ServeHTTP(httptest.NewRecorder(), req)
		if got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

type sample struct {
	Name  string `json:"name" validate:"required,max=5"`
	Count int    `json:"count" validate:"min=0"`
}

func decode(body, contentType string) error {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	var s sample
	return DecodeJSON(httptest.NewRecorder(), req, &s)
}

func statusOf(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Status
	}
	return 0
}

func TestDecodeJSON(t *testing.T) {
	cases := []struct {
		body, ct string
		want     int
	}{
		{`{"name":"ok","count":1}`, "application/json", 0},
		{`{"name":"ok","role":"admin"}`, "application/json", http.StatusUnprocessableEntity}, // mass assignment
		{`{"name":"toolong"}`, "application/json", http.StatusUnprocessableEntity},
		{`{"name":"ok"}{"name":"x"}`, "application/json", http.StatusBadRequest},
		{`{"name":`, "application/json", http.StatusBadRequest},
		{`{"name":"ok"}`, "text/plain", http.StatusUnsupportedMediaType},
		{`{"name":{"$ne":1}}`, "application/json", http.StatusUnprocessableEntity}, // operator injection
		{`{"name":"ok","count":"` + strings.Repeat("9", 2<<20) + `"}`, "application/json", http.StatusRequestEntityTooLarge},
	}
	for i, c := range cases {
		err := decode(c.body, c.ct)
		if got := statusOf(err); got != c.want {
			t.Errorf("case %d: status %d, want %d (err %v)", i, got, c.want, err)
		}
	}
}

func TestWriteErrorHidesInternalErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil), &json.SyntaxError{})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content type %q", ct)
	}
	if strings.Contains(rec.Body.String(), "SyntaxError") {
		t.Fatalf("internal error leaked: %s", rec.Body.String())
	}
}

func TestQueryValidation(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/?page=0&pageSize=500&sort=evil&featured=maybe", nil)
	q := NewQuery(req)
	q.Page(24, 100)
	q.Enum("sort", "newest", "newest", "oldest")
	q.OptBool("featured")
	err := q.Err()
	var e *Error
	if !errors.As(err, &e) || e.Status != http.StatusBadRequest || len(e.Fields) != 4 {
		t.Fatalf("expected 4 field errors, got %#v", err)
	}
}

func TestPagesStopAtListDepth(t *testing.T) {
	page := func(target string) (Page, error) {
		q := NewQuery(httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil))
		p := q.Page(25, 100)
		return p, q.Err()
	}
	if _, err := page("/?page=100&pageSize=100"); err != nil {
		t.Fatalf("last allowed page: %v", err)
	}
	if _, err := page("/?page=101&pageSize=100"); err == nil {
		t.Fatal("paging past MaxListDepth must be refused")
	}
	m := Page{Page: 1, Size: 50}.MetaCapped(MaxListDepth, true)
	if !m.TotalCapped || m.TotalPages != MaxListDepth/50 {
		t.Fatalf("meta = %+v", m)
	}
}
