// Package mongox wraps MongoDB connection setup and small shared helpers.
package mongox

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

// CountForPage counts the matches of a paginated list without reading a large collection in
// full: an empty filter uses the collection's metadata (constant time); any other filter stops
// counting at httpx.MaxListDepth, which is as deep as lists can be paged anyway. capped reports
// that there are more matches than total.
func CountForPage(ctx context.Context, coll *mongo.Collection, filter bson.M) (total int64, capped bool, err error) {
	if len(filter) == 0 {
		total, err = coll.EstimatedDocumentCount(ctx)
		return total, false, err
	}
	total, err = coll.CountDocuments(ctx, filter, options.Count().SetLimit(httpx.MaxListDepth+1))
	if total > httpx.MaxListDepth {
		return httpx.MaxListDepth, true, err
	}
	return total, false, err
}

func Connect(ctx context.Context, uri, database string, timeout time.Duration) (*mongo.Client, *mongo.Database, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetAppName("atcreations-api").
		SetTimeout(timeout).
		SetServerSelectionTimeout(timeout).
		SetWriteConcern(writeconcern.Majority()).
		SetRetryWrites(true)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("mongo ping: %w", err)
	}
	return client, client.Database(database), nil
}

// IsDuplicateKey reports whether err is a unique-index violation.
func IsDuplicateKey(err error) bool { return mongo.IsDuplicateKeyError(err) }

func IsNotFound(err error) bool { return errors.Is(err, mongo.ErrNoDocuments) }

// ContainsPattern returns a case-insensitive regex that matches q literally.
// User input is always quoted so it cannot inject regex syntax (ReDoS).
func ContainsPattern(q string) bson.Regex {
	return bson.Regex{Pattern: regexp.QuoteMeta(q), Options: "i"}
}

// Index is a compact index declaration.
type Index struct {
	Keys    bson.D
	Unique  bool
	TTL     *int32
	Partial bson.M
	Name    string
}

func EnsureIndexes(ctx context.Context, coll *mongo.Collection, idx ...Index) error {
	models := make([]mongo.IndexModel, 0, len(idx))
	for _, i := range idx {
		o := options.Index()
		if i.Unique {
			o.SetUnique(true)
		}
		if i.TTL != nil {
			o.SetExpireAfterSeconds(*i.TTL)
		}
		if i.Partial != nil {
			o.SetPartialFilterExpression(i.Partial)
		}
		if i.Name != "" {
			o.SetName(i.Name)
		}
		models = append(models, mongo.IndexModel{Keys: i.Keys, Options: o})
	}
	if _, err := coll.Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create indexes on %s: %w", coll.Name(), err)
	}
	return nil
}

// DropIndexes removes indexes that a newer EnsureIndexes call superseded. Missing ones are
// ignored, so it is safe to run on every start.
func DropIndexes(ctx context.Context, coll *mongo.Collection, names ...string) error {
	for _, name := range names {
		err := coll.Indexes().DropOne(ctx, name)
		var cmdErr mongo.CommandError
		if err != nil && (!errors.As(err, &cmdErr) || cmdErr.Name != "IndexNotFound") {
			return fmt.Errorf("drop index %s on %s: %w", name, coll.Name(), err)
		}
	}
	return nil
}

func Int32(v int32) *int32 { return &v }
