// Package httpx holds HTTP helpers shared by all handlers: JSON responses,
// RFC 9457 problem details, strict request decoding and middleware.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Error is a client-facing error. Anything that is not an *Error is reported
// as a generic 500 so internal details never leak.
type Error struct {
	Status int
	Title  string
	Detail string
	Fields []FieldError
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Title + ": " + e.Detail }

func NewError(status int, detail string) *Error {
	return &Error{Status: status, Title: http.StatusText(status), Detail: detail}
}

var (
	ErrNotFound     = NewError(http.StatusNotFound, "resource not found")
	ErrUnauthorized = NewError(http.StatusUnauthorized, "authentication required")
	ErrForbidden    = NewError(http.StatusForbidden, "you do not have permission to perform this action")
)

func Validation(fields ...FieldError) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Title: "Validation failed", Detail: "one or more fields are invalid", Fields: fields}
}

func Conflict(detail string) *Error { return NewError(http.StatusConflict, detail) }
func BadRequest(detail string) *Error {
	return NewError(http.StatusBadRequest, detail)
}

type problem struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Detail    string       `json:"detail,omitempty"`
	RequestID string       `json:"requestId,omitempty"`
	Errors    []FieldError `json:"errors,omitempty"`
}

// WriteError renders err as application/problem+json.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var e *Error
	if !errors.As(err, &e) {
		slog.ErrorContext(r.Context(), "unhandled error", "err", err, "path", r.URL.Path, "method", r.Method)
		e = NewError(http.StatusInternalServerError, "an unexpected error occurred")
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(problem{
		Type:      "about:blank",
		Title:     e.Title,
		Status:    e.Status,
		Detail:    e.Detail,
		RequestID: RequestIDFrom(r.Context()),
		Errors:    e.Fields,
	})
}

type envelope struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{Data: data})
}

func Paged(w http.ResponseWriter, data any, meta PageMeta) {
	write(w, http.StatusOK, envelope{Data: data, Meta: meta})
}

func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Handler is an http.HandlerFunc that returns an error, rendered by Wrap.
type Handler func(w http.ResponseWriter, r *http.Request) error

func Wrap(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, r, err)
		}
	}
}
