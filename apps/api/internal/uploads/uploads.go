// Package uploads owns image uploads: it stores originals in the private staging area, keeps a
// durable job queue in MongoDB, dispatches jobs to the image worker over gRPC with leases and
// retries, attaches finished images to products and categories, and removes originals in every
// outcome.
//
// Job lifecycle:
//
//	queued ──claim──▶ processing ──ok──────────▶ done
//	   ▲                 │ ├──invalid image──▶ failed     (original deleted by the worker)
//	   └──backoff────────┘ ├──retries spent──▶ failed     (original deleted here)
//	                       └──owner deleted──▶ cancelled  (output deleted here)
//	queued ──cancel/owner deleted──▶ cancelled             (original deleted here)
//
// Any original that still slips through (crash between steps) is removed by Sweep.
package uploads

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"
	"unicode"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

const (
	KindProduct  = "product"
	KindCategory = "category"

	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"

	// Finished jobs are kept a week for the admin UI and troubleshooting.
	finishedTTL = 7 * 24 * time.Hour
	// Originals older than this with no live job are removed by Sweep.
	staleOriginal = 24 * time.Hour
)

// Job is one uploaded file on its way to becoming a catalogue image.
type Job struct {
	ID            bson.ObjectID `bson:"_id"`
	OwnerKind     string        `bson:"ownerKind"`
	OwnerID       bson.ObjectID `bson:"ownerId"`
	SourceKey     string        `bson:"sourceKey"`
	OutputKey     string        `bson:"outputKey"`
	FileName      string        `bson:"fileName"`
	Size          int64         `bson:"size"`
	Status        string        `bson:"status"`
	Stage         string        `bson:"stage,omitempty"`
	Percent       int           `bson:"percent"`
	Attempts      int           `bson:"attempts"`
	NextAttemptAt time.Time     `bson:"nextAttemptAt"`
	LeaseUntil    time.Time     `bson:"leaseUntil"`
	LeaseOwner    string        `bson:"leaseOwner,omitempty"`
	Error         string        `bson:"error,omitempty"`
	CreatedBy     string        `bson:"createdBy"`
	CreatedAt     time.Time     `bson:"createdAt"`
	UpdatedAt     time.Time     `bson:"updatedAt"`
	FinishedAt    *time.Time    `bson:"finishedAt,omitempty"`
}

// JobDTO is the admin's view of a job.
type JobDTO struct {
	ID        string    `json:"id"`
	FileName  string    `json:"fileName"`
	Size      int64     `json:"size"`
	Status    string    `json:"status"`
	Stage     string    `json:"stage,omitempty"`
	Percent   int       `json:"percent"`
	Attempts  int       `json:"attempts"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (j *Job) DTO() JobDTO {
	return JobDTO{
		ID: j.ID.Hex(), FileName: j.FileName, Size: j.Size, Status: j.Status, Stage: j.Stage,
		Percent: j.Percent, Attempts: j.Attempts, Error: j.Error, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt,
	}
}

// Owners is what the queue needs from the catalogue.
type Owners interface {
	ReserveProductImageSlot(ctx context.Context, id bson.ObjectID) error
	ReleaseProductImageSlot(ctx context.Context, id bson.ObjectID) error
	AttachProductImage(ctx context.Context, actor string, id bson.ObjectID, img catalog.Image) error
	SetCategoryImageKey(ctx context.Context, id bson.ObjectID, img catalog.Image) error
	CategoryExists(ctx context.Context, id bson.ObjectID) (bool, error)
}

type Options struct {
	// Jobs processed at once by this API instance.
	Concurrency int
	// Attempts before a job fails for good.
	MaxAttempts int
	// Longest edge and WebP quality of the stored image.
	MaxEdge, Quality int
	// How long a claimed job is reserved; extended on every progress event.
	Lease time.Duration
	// Per-attempt deadline for the worker call.
	AttemptTimeout time.Duration
	// Identifies this instance in leases.
	Instance string
}

func (o *Options) defaults() {
	if o.Concurrency <= 0 {
		o.Concurrency = 3
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 6
	}
	if o.MaxEdge <= 0 {
		o.MaxEdge = media.DefaultMaxEdge
	}
	if o.Quality <= 0 {
		o.Quality = media.DefaultQuality
	}
	if o.AttemptTimeout <= 0 {
		o.AttemptTimeout = 2 * time.Minute
	}
	if o.Lease <= o.AttemptTimeout {
		o.Lease = o.AttemptTimeout + time.Minute
	}
	if o.Instance == "" {
		o.Instance = bson.NewObjectID().Hex()
	}
}

type Service struct {
	jobs    *mongo.Collection
	storage media.Storage
	owners  Owners
	worker  imagingv1.ImageWorkerServiceClient
	opts    Options
	wake    chan struct{}
	now     func() time.Time
}

func New(db *mongo.Database, storage media.Storage, owners Owners, worker imagingv1.ImageWorkerServiceClient, opts Options) *Service {
	opts.defaults()
	return &Service{
		jobs: db.Collection("image_jobs"), storage: storage, owners: owners, worker: worker, opts: opts,
		wake: make(chan struct{}, 1), now: time.Now,
	}
}

func (s *Service) EnsureIndexes(ctx context.Context) error {
	return mongox.EnsureIndexes(ctx, s.jobs,
		mongox.Index{Keys: bson.D{{Key: "status", Value: 1}, {Key: "nextAttemptAt", Value: 1}}},
		mongox.Index{Keys: bson.D{{Key: "status", Value: 1}, {Key: "leaseUntil", Value: 1}}},
		mongox.Index{Keys: bson.D{{Key: "ownerKind", Value: 1}, {Key: "ownerId", Value: 1}, {Key: "createdAt", Value: 1}}},
		mongox.Index{Keys: bson.D{{Key: "sourceKey", Value: 1}}},
		mongox.Index{Keys: bson.D{{Key: "finishedAt", Value: 1}}, TTL: mongox.Int32(int32(finishedTTL / time.Second)), Name: "finished_ttl"},
	)
}

func prefixFor(kind string) string {
	if kind == KindCategory {
		return catalog.CategoryImagePrefix
	}
	return catalog.ProductImagePrefix
}

// Enqueue validates an uploaded original from its header, reserves room for it, stores it in the
// staging area and queues it for processing.
func (s *Service) Enqueue(ctx context.Context, actor, kind string, ownerID bson.ObjectID, fileName string, data []byte) (*Job, error) {
	format, _, err := media.Inspect(data)
	if err != nil {
		return nil, err
	}
	switch kind {
	case KindProduct:
		if err := s.owners.ReserveProductImageSlot(ctx, ownerID); err != nil {
			return nil, err
		}
	case KindCategory:
		ok, err := s.owners.CategoryExists(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, httpx.ErrNotFound
		}
	default:
		return nil, fmt.Errorf("unknown owner kind %q", kind)
	}
	release := func() {
		if kind == KindProduct {
			if err := s.owners.ReleaseProductImageSlot(context.WithoutCancel(ctx), ownerID); err != nil {
				slog.WarnContext(ctx, "release image slot failed", "err", err)
			}
		}
	}

	now := s.now().UTC()
	job := &Job{
		ID: bson.NewObjectID(), OwnerKind: kind, OwnerID: ownerID, FileName: cleanFileName(fileName),
		Size: int64(len(data)), Status: StatusQueued, NextAttemptAt: now, CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
	}
	job.SourceKey = media.NewKey(media.UploadsPrefix+kind+"s/"+ownerID.Hex(), format)
	job.OutputKey = prefixFor(kind) + ownerID.Hex() + "/" + job.ID.Hex() + ".webp"

	// The storage client retries throttling (429/503/SlowDown) with adaptive backoff; if it
	// still fails, the admin's uploader retries the whole request after Retry-After.
	if err := s.storage.Put(ctx, job.SourceKey, data, "application/octet-stream"); err != nil {
		release()
		slog.WarnContext(ctx, "store original failed", "err", err)
		return nil, httpx.NewError(503, "storage is busy, please retry")
	}
	if _, err := s.jobs.InsertOne(ctx, job); err != nil {
		release()
		s.deleteObjects(ctx, job.SourceKey)
		return nil, err
	}
	s.Wake()
	return job, nil
}

// Wake nudges the dispatcher to look for work now instead of at its next poll.
func (s *Service) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// List returns an owner's uploads that the admin should see: in progress, failed (until
// dismissed) and those finished in the last minute (so the UI can show them completing).
func (s *Service) List(ctx context.Context, kind string, ownerID bson.ObjectID) ([]JobDTO, error) {
	recent := s.now().UTC().Add(-time.Minute)
	cur, err := s.jobs.Find(ctx, bson.M{
		"ownerKind": kind, "ownerId": ownerID,
		"$or": bson.A{
			bson.M{"status": bson.M{"$in": bson.A{StatusQueued, StatusProcessing, StatusFailed}}},
			bson.M{"status": StatusDone, "finishedAt": bson.M{"$gte": recent}},
		},
	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetLimit(50))
	if err != nil {
		return nil, err
	}
	var jobs []Job
	if err := cur.All(ctx, &jobs); err != nil {
		return nil, err
	}
	out := make([]JobDTO, len(jobs))
	for i := range jobs {
		out[i] = jobs[i].DTO()
	}
	return out, nil
}

// Cancel removes a queued upload (deleting its original) or dismisses a failed one. Jobs being
// processed cannot be cancelled; they finish within a couple of minutes.
func (s *Service) Cancel(ctx context.Context, kind string, ownerID, jobID bson.ObjectID) error {
	now := s.now().UTC()
	var before Job
	err := s.jobs.FindOneAndUpdate(ctx,
		bson.M{"_id": jobID, "ownerKind": kind, "ownerId": ownerID, "status": bson.M{"$in": bson.A{StatusQueued, StatusFailed}}},
		bson.M{"$set": bson.M{"status": StatusCancelled, "updatedAt": now, "finishedAt": now}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&before)
	if mongox.IsNotFound(err) {
		n, cerr := s.jobs.CountDocuments(ctx, bson.M{"_id": jobID, "ownerKind": kind, "ownerId": ownerID})
		if cerr == nil && n > 0 {
			return httpx.Conflict("this upload is being processed and can no longer be cancelled")
		}
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if before.Status == StatusQueued {
		s.deleteObjects(ctx, before.SourceKey)
		if kind == KindProduct {
			_ = s.owners.ReleaseProductImageSlot(ctx, ownerID)
		}
	}
	return nil
}

// CancelOwner is called when a product or category is deleted: queued uploads are cancelled and
// their originals removed. Jobs already processing notice the missing owner when they finish.
func (s *Service) CancelOwner(ctx context.Context, kind string, ownerID bson.ObjectID) {
	cur, err := s.jobs.Find(ctx, bson.M{"ownerKind": kind, "ownerId": ownerID, "status": StatusQueued})
	if err != nil {
		slog.WarnContext(ctx, "cancel uploads for deleted owner failed", "err", err)
		return
	}
	var jobs []Job
	if err := cur.All(ctx, &jobs); err != nil {
		slog.WarnContext(ctx, "cancel uploads for deleted owner failed", "err", err)
		return
	}
	now := s.now().UTC()
	keys := make([]string, 0, len(jobs))
	for _, j := range jobs {
		res, err := s.jobs.UpdateOne(ctx, bson.M{"_id": j.ID, "status": StatusQueued},
			bson.M{"$set": bson.M{"status": StatusCancelled, "error": "deleted", "updatedAt": now, "finishedAt": now}})
		if err == nil && res.ModifiedCount == 1 {
			keys = append(keys, j.SourceKey)
		}
	}
	s.deleteObjects(ctx, keys...)
}

// Sweep removes originals left in the staging area by crashes: anything older than a day that no
// queued or processing job still needs.
func (s *Service) Sweep(ctx context.Context) (int, error) {
	cutoff := s.now().Add(-staleOriginal)
	var stale []string
	err := s.storage.Walk(ctx, media.UploadsPrefix, func(key string, modified time.Time) error {
		if modified.Before(cutoff) {
			stale = append(stale, key)
		}
		return nil
	})
	if err != nil || len(stale) == 0 {
		return 0, err
	}
	live := map[string]bool{}
	cur, err := s.jobs.Find(ctx, bson.M{"sourceKey": bson.M{"$in": stale}, "status": bson.M{"$in": bson.A{StatusQueued, StatusProcessing}}},
		options.Find().SetProjection(bson.M{"sourceKey": 1}))
	if err != nil {
		return 0, err
	}
	var rows []struct {
		SourceKey string `bson:"sourceKey"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, err
	}
	for _, r := range rows {
		live[r.SourceKey] = true
	}
	var del []string
	for _, k := range stale {
		if !live[k] {
			del = append(del, k)
		}
	}
	if len(del) == 0 {
		return 0, nil
	}
	return len(del), s.storage.Delete(ctx, del...)
}

func (s *Service) deleteObjects(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if err := s.storage.Delete(dctx, keys...); err != nil {
		slog.WarnContext(ctx, "delete objects failed; the sweep will retry", "err", err, "count", len(keys))
	}
}

// cleanFileName keeps a short, printable base name for display only (keys never use it).
func cleanFileName(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, name)
	if r := []rune(name); len(r) > 120 {
		name = string(r[:120])
	}
	if name == "" || name == "." || name == "/" {
		return "image"
	}
	return name
}

var errLeaseLost = errors.New("lease lost")
