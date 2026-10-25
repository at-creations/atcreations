// Command imgworker runs the gRPC image worker (see internal/imageworker).
//
// It needs object-storage credentials and a shared token; it has no database access.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"

	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
	"github.com/atcreations/atcreations/apps/api/internal/imageworker"
	"github.com/atcreations/atcreations/apps/api/internal/media"
)

type config struct {
	Addr        string `env:"WORKER_ADDR" envDefault:":9090"`
	Token       string `env:"WORKER_TOKEN,required,unset"`
	Concurrency int    `env:"WORKER_CONCURRENCY" envDefault:"2"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	Env         string `env:"APP_ENV" envDefault:"development"`

	S3Endpoint        string `env:"S3_ENDPOINT,required"`
	S3Region          string `env:"S3_REGION" envDefault:"auto"`
	S3Bucket          string `env:"S3_BUCKET,required"`
	S3AccessKeyID     string `env:"S3_ACCESS_KEY_ID,required,unset"`
	S3SecretAccessKey string `env:"S3_SECRET_ACCESS_KEY,required,unset"`
	S3KeyPrefix       string `env:"S3_KEY_PREFIX"`
	CDNBaseURL        string `env:"CDN_BASE_URL"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return err
	}
	if len(cfg.Token) < 32 {
		return errors.New("WORKER_TOKEN must be at least 32 characters")
	}
	if cfg.Concurrency < 1 || cfg.Concurrency > 16 {
		return errors.New("WORKER_CONCURRENCY must be 1-16")
	}
	var level slog.Level
	_ = level.UnmarshalText([]byte(cfg.LogLevel))
	var h slog.Handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	if cfg.Env == "development" {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(h).With("service", "imgworker"))

	storage := media.NewS3Storage(media.S3Config{
		Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket,
		AccessKeyID: cfg.S3AccessKeyID, SecretAccessKey: cfg.S3SecretAccessKey, PublicBaseURL: cfg.CDNBaseURL,
		KeyPrefix: cfg.S3KeyPrefix,
	})

	authUnary, authStream := imageworker.AuthInterceptors(cfg.Token)
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(imageworker.RecoverUnary, authUnary),
		grpc.ChainStreamInterceptor(imageworker.RecoverStream, authStream),
		// Requests are tiny (object keys); images travel through object storage, not gRPC.
		grpc.MaxRecvMsgSize(64<<10),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{MinTime: 10 * time.Second, PermitWithoutStream: true}),
	)
	imagingv1.RegisterImageWorkerServiceServer(srv, imageworker.New(storage, cfg.Concurrency))
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	hs.SetServingStatus(imagingv1.ImageWorkerService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.Addr)
	if err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("image worker listening", "addr", cfg.Addr, "concurrency", cfg.Concurrency)
		errCh <- srv.Serve(lis)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	slog.Info("shutting down")
	hs.Shutdown()
	done := make(chan struct{})
	go func() { srv.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-time.After(60 * time.Second): // let in-flight images finish, within reason
		srv.Stop()
	}
	return nil
}

// healthcheck lets the distroless container probe itself without grpc_health_probe.
func healthcheck() int {
	addr := os.Getenv("WORKER_ADDR")
	switch {
	case addr == "":
		addr = "127.0.0.1:9090"
	case addr[0] == ':':
		addr = "127.0.0.1" + addr
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return 1
	}
	defer func() { _ = conn.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil || res.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		fmt.Fprintln(os.Stderr, "unhealthy:", err)
		return 1
	}
	return 0
}
