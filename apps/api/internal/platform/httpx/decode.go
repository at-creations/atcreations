package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

const maxJSONBody = 1 << 20 // 1 MiB

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

// Validator returns the shared validator, configured to report JSON field names.
func Validator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		validate.RegisterTagNameFunc(func(f reflect.StructField) string {
			name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	})
	return validate
}

// DecodeJSON strictly decodes a JSON body into dst and validates it.
// Unknown fields are rejected, which prevents mass-assignment of fields the
// endpoint does not expect.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "application/json" {
			return NewError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		}
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return BadRequest("request body must contain a single JSON object")
	}
	return Validate(dst)
}

// Validate runs struct validation and converts failures into a 422 problem.
func Validate(v any) error {
	err := Validator().Struct(v)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err
	}
	fields := make([]FieldError, 0, len(verrs))
	for _, fe := range verrs {
		// Namespace is "Type.field.sub"; drop the root type name.
		ns := fe.Namespace()
		if i := strings.IndexByte(ns, '.'); i >= 0 {
			ns = ns[i+1:]
		}
		fields = append(fields, FieldError{Field: ns, Message: message(fe)})
	}
	return Validation(fields...)
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "max":
		return "must be at most " + fe.Param()
	case "min":
		return "must be at least " + fe.Param()
	case "oneof":
		return "must be one of: " + fe.Param()
	case "url", "http_url":
		return "must be a valid URL"
	default:
		return "is invalid (" + fe.Tag() + ")"
	}
}

func decodeError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var maxErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxErr):
		return NewError(http.StatusRequestEntityTooLarge, "request body too large")
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return BadRequest("malformed JSON")
	case errors.As(err, &typeErr):
		return Validation(FieldError{Field: typeErr.Field, Message: fmt.Sprintf("must be of type %s", typeErr.Type)})
	case errors.Is(err, io.EOF):
		return BadRequest("request body is required")
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return Validation(FieldError{Field: strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`), Message: "is not allowed"})
	default:
		return BadRequest("invalid request body")
	}
}
