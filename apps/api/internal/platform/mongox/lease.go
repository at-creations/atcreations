package mongox

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Leases provides cluster-wide mutual exclusion for scheduled jobs so that
// running several API replicas never executes the same job concurrently.
type Leases struct {
	coll  *mongo.Collection
	owner string
}

func NewLeases(db *mongo.Database, owner string) *Leases {
	return &Leases{coll: db.Collection("job_leases"), owner: owner}
}

// Acquire takes the named lease for ttl. It returns false if another owner
// holds an unexpired lease.
func (l *Leases) Acquire(ctx context.Context, name string, ttl time.Duration) (bool, error) {
	now := time.Now().UTC()
	filter := bson.M{"_id": name, "$or": bson.A{
		bson.M{"expiresAt": bson.M{"$lte": now}},
		bson.M{"owner": l.owner},
	}}
	update := bson.M{"$set": bson.M{"owner": l.owner, "expiresAt": now.Add(ttl), "acquiredAt": now}}
	_, err := l.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if IsDuplicateKey(err) {
		// The lease document exists and is held by someone else.
		return false, nil
	}
	return err == nil, err
}

func (l *Leases) Release(ctx context.Context, name string) error {
	_, err := l.coll.UpdateOne(ctx, bson.M{"_id": name, "owner": l.owner},
		bson.M{"$set": bson.M{"expiresAt": time.Now().UTC()}})
	return err
}
