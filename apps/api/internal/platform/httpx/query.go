package httpx

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type Page struct {
	Page, Size int
}

func (p Page) Skip() int64 { return int64((p.Page - 1) * p.Size) }

func (p Page) Meta(total int64) PageMeta {
	pages := int((total + int64(p.Size) - 1) / int64(p.Size))
	return PageMeta{Page: p.Page, PageSize: p.Size, Total: total, TotalPages: pages}
}

// Query is a small accumulator for parsing and validating query parameters.
type Query struct {
	v      url.Values
	fields []FieldError
}

func NewQuery(r *http.Request) *Query { return &Query{v: r.URL.Query()} }

func (q *Query) fail(field, msg string) {
	q.fields = append(q.fields, FieldError{Field: field, Message: msg})
}

// Err returns a 400 problem listing every invalid parameter, or nil.
func (q *Query) Err() error {
	if len(q.fields) == 0 {
		return nil
	}
	return &Error{Status: http.StatusBadRequest, Title: "Invalid query parameters", Detail: "one or more query parameters are invalid", Fields: q.fields}
}

func (q *Query) String(name string, maxLen int) string {
	s := q.v.Get(name)
	if len([]rune(s)) > maxLen {
		q.fail(name, "is too long")
		return ""
	}
	return s
}

func (q *Query) Int(name string, def, lo, hi int) int {
	s := q.v.Get(name)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < lo || n > hi {
		q.fail(name, "must be an integer between "+strconv.Itoa(lo)+" and "+strconv.Itoa(hi))
		return def
	}
	return n
}

// OptInt64 returns nil when the parameter is absent.
func (q *Query) OptInt64(name string, lo, hi int64) *int64 {
	s := q.v.Get(name)
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < lo || n > hi {
		q.fail(name, "is out of range")
		return nil
	}
	return &n
}

func (q *Query) OptBool(name string) *bool {
	s := q.v.Get(name)
	if s == "" {
		return nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		q.fail(name, "must be true or false")
		return nil
	}
	return &b
}

// Enum returns the parameter if it is one of allowed, otherwise def.
func (q *Query) Enum(name, def string, allowed ...string) string {
	s := q.v.Get(name)
	if s == "" {
		return def
	}
	if !slices.Contains(allowed, s) {
		q.fail(name, "is not a supported value")
		return def
	}
	return s
}

func (q *Query) OptObjectID(name string) *bson.ObjectID {
	s := q.v.Get(name)
	if s == "" {
		return nil
	}
	id, err := bson.ObjectIDFromHex(s)
	if err != nil {
		q.fail(name, "must be a valid id")
		return nil
	}
	return &id
}

func (q *Query) Page(defSize, maxSize int) Page {
	return Page{Page: q.Int("page", 1, 1, 10_000), Size: q.Int("pageSize", defSize, 1, maxSize)}
}

// ObjectIDParam parses a path value as an ObjectID, answering 404 on failure
// so malformed ids are indistinguishable from missing resources.
func ObjectIDParam(r *http.Request, name string) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(r.PathValue(name))
	if err != nil {
		return bson.ObjectID{}, ErrNotFound
	}
	return id, nil
}
