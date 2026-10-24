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
	_ "time/tzdata" // the distroless image has no zoneinfo; ANALYTICS_TIMEZONE needs it

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/invitation"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/atcreations/atcreations/apps/api/internal/analytics"
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
	"github.com/atcreations/atcreations/apps/api/internal/uploads"

	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
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
	for _, w := range cfg.Warnings {
		slog.Warn("configuration", "warning", w)
	}

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
	// Cached responses embed config-derived values (e.g. image URLs from
	// CDN_BASE_URL), so start every process with a fresh cache generation.
	cache.Invalidate(ctx, redisx.NSCatalog, redisx.NSStore)

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

	var sender mail.Sender = mail.LogSender{Verbose: cfg.Env == "development" || cfg.Env == "test"}
	if cfg.MailProvider == "sendgrid" {
		sender = mail.NewSendGrid(cfg.SendGridKey, cfg.MailFrom, cfg.MailFromName)
	}

	auditLog := audit.New(db)
	catalogSvc := catalog.NewService(db, storage, auditLog, cache, cfg.HashPepper)
	orderSvc := orders.NewService(mongoClient, db, storage, orders.NewMailer(sender, cfg.StoreName, cfg.StorefrontURL), auditLog, cache, cfg.HashPepper)
	storeSvc := storeinfo.New(db, auditLog, cache)
	staffSvc := staff.New(users, invitations, directory, auditLog, rdb, cfg.ClerkInviteRedirectURL)
	analyticsSvc := analytics.New(db, rdb, cfg.HashPepper, analytics.Options{
		Location: cfg.AnalyticsLocation, StorefrontURL: cfg.StorefrontURL, ImageURL: storage.URL,
	})
	catalogSvc.OnProductView(analyticsSvc.RecordProductView)

	// The image worker runs on the internal network; every call carries the shared token.
	workerConn, err := grpc.NewClient(cfg.ImageWorkerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(tokenCreds(cfg.ImageWorkerToken)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: 30 * time.Second, Timeout: 10 * time.Second}),
	)
	if err != nil {
		return fmt.Errorf("image worker client: %w", err)
	}
	defer func() { _ = workerConn.Close() }()
	host, _ := os.Hostname()
	uploadSvc := uploads.New(db, storage, catalogSvc, imagingv1.NewImageWorkerServiceClient(workerConn), uploads.Options{
		Concurrency: cfg.UploadConcurrency, Instance: fmt.Sprintf("%s-%d", host, os.Getpid()),
	})
	catalogSvc.OnOwnerDeleted(uploadSvc.CancelOwner)

	idxCtx, cancel := context.WithTimeout(ctx, time.Minute)
	for _, ensure := range []func(context.Context) error{auditLog.EnsureIndexes, catalogSvc.EnsureIndexes, orderSvc.EnsureIndexes, analyticsSvc.EnsureIndexes, uploadSvc.EnsureIndexes} {
		if err := ensure(idxCtx); err != nil {
			cancel()
			return err
		}
	}
	// One-time data migrations, in the background so startup is not delayed. Until they finish,
	// search does not find the documents they have not reached yet.
	go func() {
		for name, fn := range map[string]func(context.Context) error{
			"product-search-words-v1": catalogSvc.BackfillSearchWords,
			"order-search-words-v1":   orderSvc.BackfillSearchWords,
		} {
			if err := mongox.RunOnce(ctx, db, name, fn); err != nil {
				slog.Error("data migration failed; it runs again on the next start", "err", err)
			}
		}
	}()
	cancel()

	uploadSvc.Run(ctx)

	if cfg.JobsEnabled {
		jobs.NewScheduler(mongox.NewLeases(db, fmt.Sprintf("%s-%d", host, os.Getpid())),
			jobs.Job{Name: "recent-views", Every: time.Hour, Timeout: 5 * time.Minute, Run: catalogSvc.RefreshRecentViews},
			// Gives images stored before responsive copies existed their copies, through the worker
			// behind uploads. Hourly until it has finished once (progress is kept per image).
			jobs.Job{Name: "image-variants", Every: time.Hour, Timeout: 55 * time.Minute, Run: func(ctx context.Context) error {
				return mongox.RunOnce(ctx, db, "image-variants-v1", uploadSvc.BackfillVariants)
			}},
			// Moves page/product view counts buffered in Redis to MongoDB.
			jobs.Job{Name: "analytics-flush", Every: 30 * time.Second, Timeout: 25 * time.Second, Run: analyticsSvc.Flush, Quiet: true},
			// Repairs drift in the stored category product counts (and backfills them on first run).
			jobs.Job{Name: "category-counts", Every: 6 * time.Hour, Timeout: 10 * time.Minute, Run: catalogSvc.RecountCategoryProducts},
			jobs.Job{Name: "upload-sweep", Every: time.Hour, Timeout: 10 * time.Minute, Run: func(ctx context.Context) error {
				n, err := uploadSvc.Sweep(ctx)
				if n > 0 {
					slog.InfoContext(ctx, "removed stale upload originals", "count", n)
				}
				return err
			}},
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
		Catalog: catalogSvc, Orders: orders.NewHandlers(orderSvc, cfg.PublicCheckoutEnabled, cfg.AnalyticsLocation),
		StoreInfo: storeSvc, Staff: staffSvc, Audit: auditLog,
		Analytics: analytics.NewHandlers(analyticsSvc, cfg.TrustedProxyPrefixes),
		Uploads:   uploadSvc,
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
	err = srv.Shutdown(shutdownCtx)
	// Emails for orders placed just before the signal still go out (before Mongo disconnects).
	if werr := orderSvc.Wait(shutdownCtx); werr != nil {
		slog.Warn("shutdown", "err", werr)
	}
	return err
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

// tokenCreds sends the image worker's shared token on every call. The worker is only reachable
// on the internal network, so transport security is not required.
type tokenCreds string

func (t tokenCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + string(t)}, nil
}

func (tokenCreds) RequireTransportSecurity() bool { return false }
