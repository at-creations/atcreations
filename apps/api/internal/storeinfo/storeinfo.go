// Package storeinfo manages the singleton store contact/hours document.
package storeinfo

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

const docID = "store"

var days = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

type BusinessHours struct {
	Day    string `bson:"day" json:"day" validate:"required,oneof=mon tue wed thu fri sat sun"`
	Closed bool   `bson:"closed" json:"closed"`
	Open   string `bson:"open,omitempty" json:"open,omitempty" validate:"omitempty,len=5"`
	Close  string `bson:"close,omitempty" json:"close,omitempty" validate:"omitempty,len=5"`
}

type StoreInfo struct {
	Email         string          `bson:"email" json:"email"`
	Phone         string          `bson:"phone" json:"phone"`
	Address       string          `bson:"address" json:"address"`
	MapURL        string          `bson:"mapUrl,omitempty" json:"mapUrl,omitempty"`
	BusinessHours []BusinessHours `bson:"businessHours" json:"businessHours"`
	UpdatedAt     time.Time       `bson:"updatedAt" json:"updatedAt"`
}

type Input struct {
	Email         string          `json:"email" validate:"omitempty,email,max=254"`
	Phone         string          `json:"phone" validate:"max=40"`
	Address       string          `json:"address" validate:"max=300"`
	MapURL        string          `json:"mapUrl" validate:"omitempty,http_url,max=500"`
	BusinessHours []BusinessHours `json:"businessHours" validate:"max=7,dive"`
}

type Service struct {
	coll  *mongo.Collection
	audit *audit.Log
}

func New(db *mongo.Database, a *audit.Log) *Service {
	return &Service{coll: db.Collection("store_info"), audit: a}
}

func (s *Service) Get(ctx context.Context) (StoreInfo, error) {
	var info StoreInfo
	err := s.coll.FindOne(ctx, bson.M{"_id": docID}).Decode(&info)
	if mongox.IsNotFound(err) {
		// An unconfigured store returns an empty, well-formed document.
		return StoreInfo{BusinessHours: []BusinessHours{}}, nil
	}
	return info, err
}

func validTime(s string) bool {
	if len(s) != 5 || s[2] != ':' {
		return false
	}
	h, m := s[:2], s[3:]
	return h >= "00" && h <= "23" && m >= "00" && m <= "59" && isDigits(h) && isDigits(m)
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (in *Input) validate() error {
	var fields []httpx.FieldError
	if in.MapURL != "" {
		if u, err := url.Parse(in.MapURL); err != nil || u.Scheme != "https" {
			fields = append(fields, httpx.FieldError{Field: "mapUrl", Message: "must be an https URL"})
		}
	}
	seen := map[string]bool{}
	for i := range in.BusinessHours {
		h := &in.BusinessHours[i]
		if seen[h.Day] {
			fields = append(fields, httpx.FieldError{Field: "businessHours", Message: "each day may appear only once"})
		}
		seen[h.Day] = true
		if h.Closed {
			h.Open, h.Close = "", ""
			continue
		}
		if !validTime(h.Open) || !validTime(h.Close) || h.Open >= h.Close {
			fields = append(fields, httpx.FieldError{Field: "businessHours." + h.Day, Message: "needs valid HH:MM open and close times with open before close"})
		}
	}
	if len(fields) > 0 {
		return httpx.Validation(fields...)
	}
	return nil
}

// sortHours orders entries Monday..Sunday.
func sortHours(in []BusinessHours) []BusinessHours {
	out := make([]BusinessHours, 0, len(in))
	for _, d := range days {
		for _, h := range in {
			if h.Day == d {
				out = append(out, h)
			}
		}
	}
	return out
}

func (s *Service) HandleGet(w http.ResponseWriter, r *http.Request) error {
	info, err := s.Get(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, info)
	return nil
}

func (s *Service) HandlePut(w http.ResponseWriter, r *http.Request) error {
	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	info := StoreInfo{
		Email: strings.TrimSpace(in.Email), Phone: strings.TrimSpace(in.Phone), Address: strings.TrimSpace(in.Address),
		MapURL: in.MapURL, BusinessHours: sortHours(in.BusinessHours), UpdatedAt: time.Now().UTC(),
	}
	_, err := s.coll.ReplaceOne(r.Context(), bson.M{"_id": docID}, info, options.Replace().SetUpsert(true))
	if err != nil {
		return err
	}
	s.audit.Record(r.Context(), "store.update", "store", docID)
	httpx.JSON(w, http.StatusOK, info)
	return nil
}
