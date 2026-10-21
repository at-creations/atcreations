// Package audit records who changed what through the admin API.
package audit

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

type Entry struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ActorID    string        `bson:"actorId" json:"actorId"`
	ActorEmail string        `bson:"actorEmail,omitempty" json:"actorEmail,omitempty"`
	Action     string        `bson:"action" json:"action"`
	Resource   string        `bson:"resource" json:"resource"`
	ResourceID string        `bson:"resourceId,omitempty" json:"resourceId,omitempty"`
	RequestID  string        `bson:"requestId,omitempty" json:"requestId,omitempty"`
	IP         string        `bson:"ip,omitempty" json:"-"`
	At         time.Time     `bson:"at" json:"at"`
}

type Log struct{ coll *mongo.Collection }

func New(db *mongo.Database) *Log { return &Log{coll: db.Collection("audit_log")} }

func (l *Log) EnsureIndexes(ctx context.Context) error {
	return mongox.EnsureIndexes(ctx, l.coll,
		mongox.Index{Keys: bson.D{{Key: "at", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "resource", Value: 1}, {Key: "at", Value: -1}}},
		mongox.Index{Keys: bson.D{{Key: "actorId", Value: 1}, {Key: "at", Value: -1}}},
		// Keep two years of history.
		mongox.Index{Keys: bson.D{{Key: "at", Value: 1}}, TTL: mongox.Int32(2 * 365 * 24 * 3600), Name: "at_ttl"},
	)
}

// Record stores an audit entry. Failures are logged, never surfaced: the
// mutation has already happened and must not be reported as failed.
func (l *Log) Record(ctx context.Context, action, resource, resourceID string) {
	e := Entry{
		Action: action, Resource: resource, ResourceID: resourceID,
		RequestID: httpx.RequestIDFrom(ctx), IP: httpx.ClientIPFrom(ctx), At: time.Now().UTC(),
	}
	if p := auth.PrincipalFrom(ctx); p != nil {
		e.ActorID, e.ActorEmail = p.UserID, p.Email
	}
	if _, err := l.coll.InsertOne(context.WithoutCancel(ctx), e); err != nil {
		slog.ErrorContext(ctx, "audit write failed", "err", err, "action", action)
	}
}

func (l *Log) HandleList(w http.ResponseWriter, r *http.Request) error {
	q := httpx.NewQuery(r)
	page := q.Page(50, 100)
	filter := bson.M{}
	if res := q.String("resource", 40); res != "" {
		filter["resource"] = res
	}
	if actor := q.String("actorId", 64); actor != "" {
		filter["actorId"] = actor
	}
	if err := q.Err(); err != nil {
		return err
	}
	total, err := l.coll.CountDocuments(r.Context(), filter)
	if err != nil {
		return err
	}
	cur, err := l.coll.Find(r.Context(), filter, options.Find().
		SetSort(bson.D{{Key: "at", Value: -1}}).SetSkip(page.Skip()).SetLimit(int64(page.Size)))
	if err != nil {
		return err
	}
	entries := []Entry{}
	if err := cur.All(r.Context(), &entries); err != nil {
		return err
	}
	httpx.Paged(w, entries, page.Meta(total))
	return nil
}
