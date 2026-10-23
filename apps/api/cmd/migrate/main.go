// Command migrate imports the legacy catalogue (categories, products, images,
// store info) into the new database and object storage.
//
// It is a dry run unless -apply is given. The legacy database is only read
// (secondary-preferred), and legacy images are fetched from the public CDN.
//
//	OLD_MONGO_URI='mongodb+srv://…/store_ts_production?…&tlsCertificateKeyFile=/path/db-key.pem' \
//	MONGO_URI='mongodb://127.0.0.1:27017/?replicaSet=rs0&directConnection=true' \
//	S3_ENDPOINT=http://127.0.0.1:9000 S3_BUCKET=atcreations-dev \
//	S3_ACCESS_KEY_ID=… S3_SECRET_ACCESS_KEY=… CDN_BASE_URL=… \
//	go run ./cmd/migrate            # dry run
//	go run ./cmd/migrate -apply     # write
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/caarlos0/env/v11"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"

	"github.com/atcreations/atcreations/apps/api/internal/audit"
	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/catalog"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/migrate"
	"github.com/atcreations/atcreations/apps/api/internal/platform/mongox"
	"github.com/atcreations/atcreations/apps/api/internal/platform/redisx"
	"github.com/atcreations/atcreations/apps/api/internal/storeinfo"
)

type config struct {
	OldMongoURI string `env:"OLD_MONGO_URI,required,unset"`
	OldDatabase string `env:"OLD_MONGO_DATABASE"` // defaults to the URI's database
	OldCDN      string `env:"OLD_CDN_URL" envDefault:"https://cdn.atcreations.ca"`

	MongoURI      string `env:"MONGO_URI,required,unset"`
	MongoDatabase string `env:"MONGO_DATABASE" envDefault:"atcreations"`
	RedisURL      string `env:"REDIS_URL,unset"` // optional: invalidates the API's response cache

	S3Endpoint        string `env:"S3_ENDPOINT,required"`
	S3Region          string `env:"S3_REGION" envDefault:"us-east-1"`
	S3Bucket          string `env:"S3_BUCKET,required"`
	S3AccessKeyID     string `env:"S3_ACCESS_KEY_ID,required,unset"`
	S3SecretAccessKey string `env:"S3_SECRET_ACCESS_KEY,required,unset"`
	CDNBaseURL        string `env:"CDN_BASE_URL,required"`
}

func main() {
	apply := flag.Bool("apply", false, "write to the target database and storage (default: dry run)")
	concurrency := flag.Int("concurrency", 4, "parallel product imports")
	overwrite := flag.Bool("overwrite", false, "replace categories and products that already exist (before go-live only: discards admin edits, uploaded images and stock changes)")
	flag.Parse()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if err := run(*apply, *overwrite, *concurrency); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}
}

func run(apply, overwrite bool, concurrency int) error {
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Legacy database: reads only, from a secondary when available.
	oldOpts := options.Client().ApplyURI(cfg.OldMongoURI).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetAppName("atcreations-migrate").SetTimeout(60 * time.Second)
	oldClient, err := mongo.Connect(oldOpts)
	if err != nil {
		return fmt.Errorf("connect legacy db: %w", err)
	}
	defer func() { _ = oldClient.Disconnect(context.Background()) }()
	oldDBName := cfg.OldDatabase
	if oldDBName == "" {
		cs, err := connStringDatabase(cfg.OldMongoURI)
		if err != nil {
			return err
		}
		oldDBName = cs
	}
	oldDB := oldClient.Database(oldDBName)
	if err := oldClient.Ping(ctx, readpref.SecondaryPreferred()); err != nil {
		return fmt.Errorf("ping legacy db: %w", err)
	}

	targetClient, targetDB, err := mongox.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase, 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { _ = targetClient.Disconnect(context.Background()) }()

	var cache *redisx.Cache
	if cfg.RedisURL != "" {
		rdb, err := redisx.Connect(ctx, cfg.RedisURL)
		if err != nil {
			slog.Warn("redis unavailable; the API's cached catalogue may stay stale for up to 5 minutes", "err", err)
		} else {
			defer func() { _ = rdb.Close() }()
			cache = redisx.NewCache(rdb)
		}
	}

	storage := media.NewS3Storage(media.S3Config{
		Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket,
		AccessKeyID: cfg.S3AccessKeyID, SecretAccessKey: cfg.S3SecretAccessKey, PublicBaseURL: cfg.CDNBaseURL,
	})
	auditLog := audit.New(targetDB)
	cat := catalog.NewService(targetDB, storage, auditLog, cache, "migration-does-not-hash-anything-0000")
	store := storeinfo.New(targetDB, auditLog, cache)
	if apply {
		if err := cat.EnsureIndexes(ctx); err != nil {
			return err
		}
	}

	slog.Info("starting", "mode", map[bool]string{true: "APPLY", false: "dry run"}[apply], "legacy_db", oldDBName, "target_db", cfg.MongoDatabase)
	runner := migrate.NewRunner(oldDB, targetDB, cat, store, storage, migrate.Options{Apply: apply, Overwrite: overwrite, OldCDN: cfg.OldCDN, Concurrency: concurrency})
	rep, err := runner.Run(ctx)
	if err != nil {
		return err
	}
	if apply {
		actx := auth.WithPrincipal(ctx, &auth.Principal{UserID: "migration", Email: "migration"})
		auditLog.Record(actx, "migration.import", "migration", time.Now().UTC().Format(time.RFC3339))
	}

	fmt.Printf("\n%s summary\n", map[bool]string{true: "Migration", false: "Dry-run"}[apply])
	fmt.Printf("  categories:            %d (%d already there, kept)\n", rep.Categories, rep.CategoriesKept)
	fmt.Printf("  products:              %d (%d without price → \"contact for price\"; %d already there, kept)\n", rep.Products, rep.ProductsWithoutPrice, rep.ProductsKept)
	fmt.Printf("  images:                %d (%d already migrated, reused)\n", rep.Images, rep.ImagesReused)
	fmt.Printf("  store info:            %d\n", rep.StoreInfo)
	fmt.Printf("  warnings:              %d\n", len(rep.Warnings))
	for _, w := range rep.Warnings {
		fmt.Printf("    - %s\n", w)
	}
	if !apply {
		fmt.Println("\nNothing was written. Re-run with -apply to migrate.")
	}
	return nil
}

// connStringDatabase extracts the database name from a MongoDB URI.
func connStringDatabase(uri string) (string, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse OLD_MONGO_URI: %w", err)
	}
	if cs.Database == "" {
		return "", fmt.Errorf("OLD_MONGO_URI has no database; set OLD_MONGO_DATABASE")
	}
	return cs.Database, nil
}
