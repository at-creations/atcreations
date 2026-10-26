package mail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUseSendSendsEmail(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/emails" {
			t.Errorf("request = %s %s, want POST /api/v1/emails", r.Method, r.URL.Path)
		}
		if a := r.Header.Get("Authorization"); a != "Bearer us_key" {
			t.Errorf("Authorization = %q", a)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"emailId":"e1"}`))
	}))
	defer srv.Close()

	s := NewUseSend(srv.URL+"/", "us_key", "orders@example.com", "AT Creations")
	err := s.Send(context.Background(), Message{To: "a@b.c", Subject: "Hi", HTML: "<p>x</p>", Text: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got["from"] != "AT Creations <orders@example.com>" || got["subject"] != "Hi" ||
		got["html"] != "<p>x</p>" || got["text"] != "x" {
		t.Fatalf("body = %v", got)
	}
	if to, ok := got["to"].([]any); !ok || len(to) != 1 || to[0] != "a@b.c" {
		t.Fatalf("to = %v", got["to"])
	}
}

func TestUseSendReportsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"domain not verified"}`))
	}))
	defer srv.Close()

	err := NewUseSend(srv.URL, "k", "a@b.c", "").Send(context.Background(), Message{To: "x@y.z"})
	if err == nil || !strings.Contains(err.Error(), "usesend status 403") || !strings.Contains(err.Error(), "domain not verified") {
		t.Fatalf("err = %v", err)
	}
}
