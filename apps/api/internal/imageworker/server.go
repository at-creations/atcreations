// Package imageworker implements the gRPC image worker: it turns uploaded originals into
// downscaled WebP images. It is stateless; the durable queue and retries live in the API
// (internal/uploads), which calls ProcessImage once per job attempt and relays the streamed
// progress to the admin.
package imageworker

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

const (
	minEdge = 64
	maxEdge = 4096
	// How long a call may wait for a free processing slot before reporting RESOURCE_EXHAUSTED.
	slotWait = 20 * time.Second
	// Object metadata on outputs, so a retry can report an already-written result.
	metaWidth  = "width"
	metaHeight = "height"
)

// Rough share of total time per stage, for the progress percentage.
var stagePercent = map[imagingv1.Stage]uint32{
	imagingv1.Stage_STAGE_WAITING:     0,
	imagingv1.Stage_STAGE_DOWNLOADING: 10,
	imagingv1.Stage_STAGE_DECODING:    30,
	imagingv1.Stage_STAGE_RESIZING:    55,
	imagingv1.Stage_STAGE_ENCODING:    70,
	imagingv1.Stage_STAGE_UPLOADING:   90,
}

// Server implements imagingv1.ImageWorkerServiceServer.
type Server struct {
	imagingv1.UnimplementedImageWorkerServiceServer
	storage media.Storage
	slots   chan struct{}
}

// New returns a worker that processes at most `concurrency` images at once.
func New(storage media.Storage, concurrency int) *Server {
	concurrency = max(1, concurrency)
	media.SetConcurrency(concurrency)
	return &Server{storage: storage, slots: make(chan struct{}, concurrency)}
}

func (s *Server) ProcessImage(req *imagingv1.ProcessImageRequest, stream imagingv1.ImageWorkerService_ProcessImageServer) error {
	ctx := stream.Context()
	if err := validate(req); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	log := slog.With("job", req.GetJobId(), "source", req.GetSourceKey())
	// Progress is best effort: a lost event must never fail the job.
	progress := func(st imagingv1.Stage) {
		_ = stream.Send(&imagingv1.ProcessImageResponse{Event: &imagingv1.ProcessImageResponse_Progress{
			Progress: &imagingv1.Progress{Stage: st, Percent: stagePercent[st]},
		}})
	}

	// Bounded concurrency: wait briefly for a slot, then tell the caller to back off.
	progress(imagingv1.Stage_STAGE_WAITING)
	waitCtx, cancel := context.WithTimeout(ctx, slotWait)
	select {
	case s.slots <- struct{}{}:
		cancel()
		defer func() { <-s.slots }()
	case <-waitCtx.Done():
		cancel()
		if ctx.Err() != nil {
			return status.FromContextError(ctx.Err()).Err()
		}
		return status.Error(codes.ResourceExhausted, "worker busy")
	}

	progress(imagingv1.Stage_STAGE_DOWNLOADING)
	raw, err := s.storage.Get(ctx, req.GetSourceKey(), media.MaxUploadBytes)
	switch {
	case errors.Is(err, media.ErrObjectNotFound):
		// A retry after a lost response: the original was already consumed.
		if res, ok := s.existingOutput(ctx, req); ok {
			log.Info("original already processed; reporting existing output")
			return sendResult(stream, res)
		}
		return status.Error(codes.NotFound, "original not found")
	case errors.Is(err, media.ErrTooLarge):
		s.discard(ctx, log, req.GetSourceKey())
		return status.Error(codes.InvalidArgument, "file too large")
	case err != nil:
		log.Warn("read original failed", "err", err)
		return transient(ctx, err)
	}

	started := time.Now()
	out, err := media.Process(ctx, raw, media.Options{
		MaxEdge: int(req.GetMaxEdge()), Quality: int(req.GetQuality()),
		OnStage: func(st media.Stage) {
			switch st {
			case media.StageDecoding:
				progress(imagingv1.Stage_STAGE_DECODING)
			case media.StageResizing:
				progress(imagingv1.Stage_STAGE_RESIZING)
			case media.StageEncoding:
				progress(imagingv1.Stage_STAGE_ENCODING)
			}
		},
	})
	if err != nil {
		var invalid *httpx.Error
		if errors.As(err, &invalid) {
			// Not a usable image: it can never succeed, so remove the original now.
			s.discard(ctx, log, req.GetSourceKey())
			return status.Error(codes.InvalidArgument, invalid.Detail)
		}
		log.Warn("processing failed", "err", err)
		return transient(ctx, err)
	}

	progress(imagingv1.Stage_STAGE_UPLOADING)
	err = s.storage.PutObject(ctx, media.PutInput{
		Key: req.GetOutputKey(), Data: out.Data, ContentType: out.ContentType,
		Metadata: map[string]string{metaWidth: strconv.Itoa(out.Width), metaHeight: strconv.Itoa(out.Height)},
	})
	if err != nil {
		log.Warn("write output failed", "err", err)
		return transient(ctx, err)
	}
	// The output is safe; the original is no longer needed. A failed delete is not fatal: the
	// API deletes it again when it records the result, and the staging sweep catches the rest.
	s.discard(ctx, log, req.GetSourceKey())

	log.Info("image processed", "output", req.GetOutputKey(), "width", out.Width, "height", out.Height,
		"bytes", len(out.Data), "ms", time.Since(started).Milliseconds())
	return sendResult(stream, &imagingv1.Result{
		OutputKey: req.GetOutputKey(), Width: uint32(out.Width), Height: uint32(out.Height), //nolint:gosec // bounded by maxEdge
		SizeBytes: uint64(len(out.Data)), ContentType: out.ContentType,
	})
}

func sendResult(stream imagingv1.ImageWorkerService_ProcessImageServer, res *imagingv1.Result) error {
	return stream.Send(&imagingv1.ProcessImageResponse{Event: &imagingv1.ProcessImageResponse_Result{Result: res}})
}

// existingOutput reports a previously written output (from its metadata), if any.
func (s *Server) existingOutput(ctx context.Context, req *imagingv1.ProcessImageRequest) (*imagingv1.Result, bool) {
	info, err := s.storage.Head(ctx, req.GetOutputKey())
	if err != nil {
		return nil, false
	}
	w, errW := strconv.Atoi(info.Metadata[metaWidth])
	h, errH := strconv.Atoi(info.Metadata[metaHeight])
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return nil, false
	}
	return &imagingv1.Result{
		OutputKey: req.GetOutputKey(), Width: uint32(w), Height: uint32(h), //nolint:gosec // parsed from our own metadata
		SizeBytes: uint64(max(info.Size, 0)), ContentType: "image/webp",
	}, true
}

func (s *Server) discard(ctx context.Context, log *slog.Logger, key string) {
	// Use a fresh deadline: the caller's may be nearly spent, and cleanup should still happen.
	dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	if err := s.storage.Delete(dctx, key); err != nil {
		log.Warn("delete original failed; the staging sweep will remove it", "err", err)
	}
}

// transient maps an unexpected failure to a retryable status.
func transient(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return status.FromContextError(ctx.Err()).Err()
	}
	return status.Error(codes.Unavailable, fmt.Sprintf("temporary failure: %v", err))
}

func validate(req *imagingv1.ProcessImageRequest) error {
	src, dst := req.GetSourceKey(), req.GetOutputKey()
	switch {
	case !strings.HasPrefix(src, media.UploadsPrefix) || strings.Contains(src, ".."):
		return errors.New("source_key must be in the uploads area")
	case dst == "" || strings.HasPrefix(dst, media.UploadsPrefix) || strings.Contains(dst, "..") || !strings.HasSuffix(dst, ".webp"):
		return errors.New("output_key must be a .webp key outside the uploads area")
	case req.GetMaxEdge() < minEdge || req.GetMaxEdge() > maxEdge:
		return fmt.Errorf("max_edge must be %d-%d", minEdge, maxEdge)
	case req.GetQuality() < 1 || req.GetQuality() > 100:
		return errors.New("quality must be 1-100")
	}
	return nil
}

// ---------- Interceptors ----------

// HealthMethodPrefix is reachable without the token (container health checks).
const HealthMethodPrefix = "/grpc.health.v1.Health/"

func authorized(ctx context.Context, want []byte) bool {
	md, _ := metadata.FromIncomingContext(ctx)
	got := md.Get("authorization")
	return len(got) == 1 && subtle.ConstantTimeCompare([]byte(got[0]), want) == 1
}

// AuthInterceptors require "authorization: Bearer <token>" on every call except health checks.
func AuthInterceptors(token string) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	want := []byte("Bearer " + token)
	unary := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !strings.HasPrefix(info.FullMethod, HealthMethodPrefix) && !authorized(ctx, want) {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return handler(ctx, req)
	}
	stream := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !strings.HasPrefix(info.FullMethod, HealthMethodPrefix) && !authorized(ss.Context(), want) {
			return status.Error(codes.Unauthenticated, "invalid token")
		}
		return handler(srv, ss)
	}
	return unary, stream
}

func recovered(ctx context.Context, method string, err *error) {
	if r := recover(); r != nil {
		slog.ErrorContext(ctx, "panic in handler", "method", method, "panic", r, "stack", string(debug.Stack()))
		*err = status.Error(codes.Internal, "internal error")
	}
}

// RecoverUnary and RecoverStream turn panics into INTERNAL errors instead of crashing the worker.
func RecoverUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer recovered(ctx, info.FullMethod, &err)
	return handler(ctx, req)
}

func RecoverStream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer recovered(ss.Context(), info.FullMethod, &err)
	return handler(srv, ss)
}
