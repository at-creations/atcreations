package mongox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// RunOnce runs a data migration once per database, recording it in app_migrations so later
// starts skip it without scanning anything. fn must be idempotent: replicas starting together
// may both run it.
func RunOnce(ctx context.Context, db *mongo.Database, name string, fn func(context.Context) error) error {
	coll := db.Collection("app_migrations")
	n, err := coll.CountDocuments(ctx, bson.M{"_id": name}, options.Count().SetLimit(1))
	if err != nil || n > 0 {
		return err
	}
	start := time.Now()
	if err := fn(ctx); err != nil {
		return fmt.Errorf("migration %s: %w", name, err)
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": name}, bson.M{"$setOnInsert": bson.M{"at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
	slog.InfoContext(ctx, "data migration done", "name", name, "duration_ms", time.Since(start).Milliseconds())
	return err
}

// Backfill updates, in batches, every document matching filter: set returns the fields to $set
// for one decoded document. The filter must stop matching once a document is updated.
func Backfill[T any](ctx context.Context, coll *mongo.Collection, filter bson.M, projection bson.M, id func(*T) bson.ObjectID, set func(*T) bson.M) error {
	const batch = 500
	for {
		cur, err := coll.Find(ctx, filter, options.Find().SetProjection(projection).SetLimit(batch))
		if err != nil {
			return err
		}
		var docs []T
		if err := cur.All(ctx, &docs); err != nil {
			return err
		}
		if len(docs) == 0 {
			return nil
		}
		models := make([]mongo.WriteModel, len(docs))
		for i := range docs {
			models[i] = mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": id(&docs[i])}).SetUpdate(bson.M{"$set": set(&docs[i])})
		}
		if _, err := coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
			return err
		}
		if len(docs) < batch {
			return nil
		}
	}
}
