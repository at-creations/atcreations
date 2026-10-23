package uploads

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

const idlePoll = 3 * time.Second

var stageNames = map[imagingv1.Stage]string{
	imagingv1.Stage_STAGE_WAITING:     "waiting",
	imagingv1.Stage_STAGE_DOWNLOADING: "downloading",
	imagingv1.Stage_STAGE_DECODING:    "decoding",
	imagingv1.Stage_STAGE_RESIZING:    "resizing",
	imagingv1.Stage_STAGE_ENCODING:    "encoding",
	imagingv1.Stage_STAGE_UPLOADING:   "uploading",
}

// Run processes jobs until ctx ends, with Options.Concurrency loops. Safe to run on several API
// replicas: jobs are claimed atomically and leases expire if an instance dies.
func (s *Service) Run(ctx context.Context) {
	for range s.opts.Concurrency {
		go s.loop(ctx)
	}
}

func (s *Service) loop(ctx context.Context) {
	for ctx.Err() == nil {
		job, err := s.claim(ctx)
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "claim upload job failed", "err", err)
		}
		if job != nil {
			s.process(ctx, job)
			continue // look for more work straight away
		}
		select {
		case <-ctx.Done():
		case <-s.wake:
		case <-time.After(idlePoll):
		}
	}
}

// claim takes the oldest due job, or one whose lease expired (its instance died mid-job).
func (s *Service) claim(ctx context.Context) (*Job, error) {
	now := s.now().UTC()
	// A fresh token per claim: a stale attempt (even on this instance) can never touch the job
	// after it was reclaimed.
	token := s.opts.Instance + ":" + bson.NewObjectID().Hex()
	var job Job
	err := s.jobs.FindOneAndUpdate(ctx,
		bson.M{"$or": bson.A{
			bson.M{"status": StatusQueued, "nextAttemptAt": bson.M{"$lte": now}},
			bson.M{"status": StatusProcessing, "leaseUntil": bson.M{"$lt": now}},
		}},
		bson.M{
			"$set": bson.M{
				"status": StatusProcessing, "stage": "waiting", "percent": 0,
				"leaseUntil": now.Add(s.opts.Lease), "leaseOwner": token, "updatedAt": now,
			},
			"$inc": bson.M{"attempts": 1},
		},
		options.FindOneAndUpdate().SetSort(bson.D{{Key: "nextAttemptAt", Value: 1}}).SetReturnDocument(options.After),
	).Decode(&job)
	if mongox.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// owned filters updates to the job while this claim still holds the lease.
func (s *Service) owned(job *Job) bson.M {
	return bson.M{"_id": job.ID, "status": StatusProcessing, "leaseOwner": job.LeaseOwner}
}

func (s *Service) process(ctx context.Context, job *Job) {
	log := slog.With("job", job.ID.Hex(), "owner", job.OwnerKind+":"+job.OwnerID.Hex(), "attempt", job.Attempts)
	callCtx, cancel := context.WithTimeout(ctx, s.opts.AttemptTimeout)
	defer cancel()

	res, err := s.callWorker(callCtx, job)
	if errors.Is(err, errLeaseLost) {
		log.Warn("lease lost; another instance owns the job")
		return
	}
	if err != nil {
		s.handleFailure(ctx, log, job, err)
		return
	}
	s.complete(ctx, log, job, res)
}

// callWorker runs one attempt and relays the worker's progress into the job document.
func (s *Service) callWorker(ctx context.Context, job *Job) (*imagingv1.Result, error) {
	stream, err := s.worker.ProcessImage(ctx, &imagingv1.ProcessImageRequest{
		JobId: job.ID.Hex(), SourceKey: job.SourceKey, OutputKey: job.OutputKey,
		MaxEdge: uint32(s.opts.MaxEdge), Quality: uint32(s.opts.Quality), //nolint:gosec // small positive config values
	})
	if err != nil {
		return nil, err
	}
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil, status.Error(codes.Internal, "worker ended the stream without a result")
		}
		if err != nil {
			return nil, err
		}
		if res := ev.GetResult(); res != nil {
			return res, nil
		}
		if p := ev.GetProgress(); p != nil {
			// Progress also extends the lease, so slow images are not reclaimed mid-flight.
			now := s.now().UTC()
			upd, err := s.jobs.UpdateOne(ctx, s.owned(job), bson.M{"$set": bson.M{
				"stage": stageNames[p.GetStage()], "percent": int(p.GetPercent()),
				"leaseUntil": now.Add(s.opts.Lease), "updatedAt": now,
			}})
			if err == nil && upd.MatchedCount == 0 {
				return nil, errLeaseLost
			}
		}
	}
}

// complete attaches the processed image to its owner and finishes the job.
func (s *Service) complete(ctx context.Context, log *slog.Logger, job *Job, res *imagingv1.Result) {
	img := catalog.Image{ID: job.ID.Hex(), Key: res.GetOutputKey(), Width: int(res.GetWidth()), Height: int(res.GetHeight())}
	var attachErr error
	switch job.OwnerKind {
	case KindProduct:
		attachErr = s.owners.AttachProductImage(ctx, job.CreatedBy, job.OwnerID, img)
	case KindCategory:
		attachErr = s.owners.SetCategoryImageKey(ctx, job.OwnerID, img)
	}

	status, errMsg := StatusDone, ""
	switch {
	case attachErr == nil:
	case errors.Is(attachErr, catalog.ErrOwnerGone):
		status, errMsg = StatusCancelled, "deleted"
		s.deleteObjects(ctx, res.GetOutputKey())
	case errors.Is(attachErr, catalog.ErrImagesFull):
		status, errMsg = StatusFailed, "image limit reached"
		s.deleteObjects(ctx, res.GetOutputKey())
	default:
		// Database trouble: keep the processed output and retry the attachment later (the worker
		// will report the existing output, and attaching is idempotent).
		s.retryLater(ctx, log, job, attachErr)
		return
	}
	// The worker removed the original; delete again in case its cleanup failed (idempotent).
	s.deleteObjects(ctx, job.SourceKey)
	s.finish(ctx, log, job, status, errMsg)
}

func (s *Service) handleFailure(ctx context.Context, log *slog.Logger, job *Job, err error) {
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.InvalidArgument, codes.NotFound, codes.Unauthenticated, codes.PermissionDenied:
		// Permanent: not an image, the original vanished, or a misconfigured token.
		msg := st.Message()
		if st.Code() == codes.Unauthenticated || st.Code() == codes.PermissionDenied {
			log.Error("image worker rejected the API's token; check WORKER_TOKEN")
			msg = "image service unavailable"
		}
		s.deleteObjects(ctx, job.SourceKey)
		s.finish(ctx, log, job, StatusFailed, msg)
	case codes.ResourceExhausted:
		// The worker is busy: not the job's fault, so this attempt does not count.
		s.requeue(ctx, job, s.now().Add(jitter(5*time.Second)), "", true)
	default:
		s.retryLater(ctx, log, job, err)
	}
}

func (s *Service) retryLater(ctx context.Context, log *slog.Logger, job *Job, err error) {
	if job.Attempts >= s.opts.MaxAttempts {
		log.Warn("giving up on upload", "err", err)
		s.deleteObjects(ctx, job.SourceKey, job.OutputKey)
		s.finish(ctx, log, job, StatusFailed, "processing failed, please upload again")
		return
	}
	log.Info("upload will be retried", "err", err)
	s.requeue(ctx, job, s.now().Add(backoff(job.Attempts)), "retrying", false)
}

func (s *Service) requeue(ctx context.Context, job *Job, at time.Time, msg string, refundAttempt bool) {
	update := bson.M{"$set": bson.M{
		"status": StatusQueued, "nextAttemptAt": at.UTC(), "stage": "", "percent": 0,
		"leaseOwner": "", "error": msg, "updatedAt": s.now().UTC(),
	}}
	if refundAttempt {
		update["$inc"] = bson.M{"attempts": -1}
	}
	if _, err := s.jobs.UpdateOne(context.WithoutCancel(ctx), s.owned(job), update); err != nil {
		slog.WarnContext(ctx, "requeue upload job failed; its lease will expire", "err", err)
	}
}

func (s *Service) finish(ctx context.Context, log *slog.Logger, job *Job, status, msg string) {
	now := s.now().UTC()
	res, err := s.jobs.UpdateOne(context.WithoutCancel(ctx), s.owned(job), bson.M{"$set": bson.M{
		"status": status, "error": msg, "stage": "", "percent": map[bool]int{true: 100}[status == StatusDone],
		"leaseOwner": "", "updatedAt": now, "finishedAt": now,
	}})
	if err != nil {
		log.Warn("finish upload job failed; its lease will expire", "err", err)
		return
	}
	// A successful attach already released the product's reserved slot (atomically with adding
	// the image). Otherwise release it here, exactly once: only the claim that finished the job.
	if res.ModifiedCount == 1 && job.OwnerKind == KindProduct && status == StatusFailed {
		if err := s.owners.ReleaseProductImageSlot(context.WithoutCancel(ctx), job.OwnerID); err != nil {
			log.Warn("release image slot failed", "err", err)
		}
	}
	if status == StatusDone {
		log.Info("upload processed")
	} else {
		log.Info("upload finished", "status", status, "reason", strings.TrimSpace(msg))
	}
}

// backoff is exponential with full jitter: ~2s, 4s, 8s … capped at 5 minutes.
func backoff(attempt int) time.Duration {
	d := 2 * time.Second << min(max(attempt-1, 0), 8)
	return jitter(min(d, 5*time.Minute))
}

func jitter(d time.Duration) time.Duration {
	return d/2 + time.Duration(rand.Int64N(int64(d/2)+1)) //nolint:gosec // jitter, not security
}
