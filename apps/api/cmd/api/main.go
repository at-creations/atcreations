// Command api runs the AT Creations HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/invitation"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/redis/go-redis/v9"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	"github.com/atcreations/atcreations/apps/api/internal/config"
	"github.com/atcreations/atcreations/apps/api/internal/jobs"
	"github.com/atcreations/atcreations/apps/api/internal/mail"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/orders"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
	"github.com/atcreations/atcreations/apps/api/internal/server"
	"github.com/atcreations/atcreations/apps/api/internal/staff"
	"github.com/atcreations/atcreations/apps/api/internal/storeinfo"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// healthcheck lets the distroless container probe itself without a shell or curl.
func healthcheck() int {
	addr := os.Getenv("HTTP_ADDR")
	switch {
	case addr == "":
		addr = "127.0.0.1:8080"
	case addr[0] == ':':
		addr = "127.0.0.1" + addr
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/healthz", nil) //nolint:gosec // G704: address comes from our own HTTP_ADDR
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: address comes from our own HTTP_ADDR
	if err != nil {
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoClient, db, err := mongox.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTimeout)
	if err != nil {
		return err
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()

	var rdb *redis.Client
	if cfg.RedisURL != "" {
		if rdb, err = redisx.Connect(ctx, cfg.RedisURL); err != nil {
			return err
		}
		defer func() { _ = rdb.Close() }()
	} else {
		slog.Warn("REDIS_URL not set: using in-memory rate limits, no response cache")
	}
	cache := redisx.NewCache(rdb)

	clerk.SetKey(cfg.ClerkSecretKey)
	clerkCfg := &clerk.ClientConfig{}
	clerkCfg.Key = clerk.String(cfg.ClerkSecretKey)
	users := user.NewClient(clerkCfg)
	invitations := invitation.NewClient(clerkCfg)
	directory := auth.NewClerkDirectory(users, rdb, 30*time.Second)

	storage := media.NewS3Storage(media.S3Config{
		Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket,
		AccessKeyID: cfg.S3AccessKeyID, SecretAccessKey: cfg.S3SecretAccessKey, PublicBaseURL: cfg.CDNBaseURL,
	})

	var sender mail.Sender = mail.LogSender{}
	if cfg.MailProvider == "sendgrid" {
		sender = mail.NewSendGrid(cfg.SendGridKey, cfg.MailFrom, cfg.MailFromName)
	}

	auditLog := audit.New(db)
	catalogSvc := catalog.NewService(db, storage, auditLog, cache, cfg.HashPepper)
	orderSvc := orders.NewService(mongoClient, db, storage, orders.NewMailer(sender, cfg.StoreName, cfg.StorefrontURL), auditLog, cache, cfg.HashPepper)
	storeSvc := storeinfo.New(db, auditLog, cache)
	staffSvc := staff.New(users, invitations, directory, auditLog, rdb, cfg.ClerkInviteRedirectURL)

	idxCtx, cancel := context.WithTimeout(ctx, time.Minute)
	for _, ensure := range []func(context.Context) error{auditLog.EnsureIndexes, catalogSvc.EnsureIndexes, orderSvc.EnsureIndexes} {
		if err := ensure(idxCtx); err != nil {
			cancel()
			return err
		}
	}
	cancel()

	if cfg.JobsEnabled {
		host, _ := os.Hostname()
		jobs.NewScheduler(mongox.NewLeases(db, fmt.Sprintf("%s-%d", host, os.Getpid())),
			jobs.Job{Name: "recent-views", Every: time.Hour, Timeout: 5 * time.Minute, Run: catalogSvc.RefreshRecentViews},
			jobs.Job{Name: "media-sweep", Every: 24 * time.Hour, Timeout: 30 * time.Minute, Run: func(ctx context.Context) error {
				n, err := catalogSvc.SweepOrphanedMedia(ctx, 24*time.Hour)
				if n > 0 {
					slog.InfoContext(ctx, "removed orphaned media", "count", n)
				}
				return err
			}},
		).Start(ctx)
	}

	handler := server.New(server.Deps{
		Config: cfg, Mongo: mongoClient, Redis: rdb, Cache: cache, Auth: auth.NewMiddleware(cfg.ClerkAuthorizedParties, directory),
		Catalog: catalogSvc, Orders: orders.NewHandlers(orderSvc, cfg.PublicCheckoutEnabled),
		StoreInfo: storeSvc, Staff: staffSvc, Audit: auditLog,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second, // allows image uploads on slow links
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.Env, "version", version)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelShutdown()
	return srv.Shutdown(shutdownCtx)
}

func setupLogging(cfg *config.Config) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if cfg.Env == "development" {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}
