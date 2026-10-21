// Package jobs runs periodic maintenance. Each run takes a MongoDB lease so
// that with several API replicas only one executes a given job at a time.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
)

type Job struct {
	Name    string
	Every   time.Duration
	Timeout time.Duration
	Run     func(ctx context.Context) error
}

type Scheduler struct {
	leases *mongox.Leases
	jobs   []Job
}

func NewScheduler(leases *mongox.Leases, jobs ...Job) *Scheduler {
	return &Scheduler{leases: leases, jobs: jobs}
}

// Start launches every job loop; they stop when ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	for _, j := range s.jobs {
		go s.loop(ctx, j)
	}
}

func (s *Scheduler) loop(ctx context.Context, j Job) {
	// Stagger first runs a little after boot.
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		s.runOnce(ctx, j)
		timer.Reset(j.Every)
	}
}

func (s *Scheduler) runOnce(ctx context.Context, j Job) {
	log := slog.With("job", j.Name)
	ok, err := s.leases.Acquire(ctx, j.Name, j.Every-time.Second)
	if err != nil {
		log.Error("lease acquire failed", "err", err)
		return
	}
	if !ok {
		log.Debug("lease held elsewhere; skipping")
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, j.Timeout)
	defer cancel()
	start := time.Now()
	if err := j.Run(runCtx); err != nil {
		log.Error("job failed", "err", err, "duration_ms", time.Since(start).Milliseconds())
		_ = s.leases.Release(context.WithoutCancel(ctx), j.Name)
		return
	}
	log.Info("job finished", "duration_ms", time.Since(start).Milliseconds())
}
