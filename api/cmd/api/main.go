package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webhook/internal/config"
	"webhook/internal/db"
	"webhook/internal/handlers"
	"webhook/internal/jobs"
	"webhook/internal/middleware"
	"webhook/internal/migrations"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	ctx := context.Background()

	// Pooler connection — transaction mode (port 6543); must use simple protocol
	// because PgBouncer/Supavisor can't share named prepared statements across connections.
	poolerCfg, err := pgxpool.ParseConfig(cfg.DatabasePoolerURL)
	if err != nil {
		slog.Error("parse pooler url", "error", err)
		os.Exit(1)
	}
	poolerCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	poolerPool, err := pgxpool.NewWithConfig(ctx, poolerCfg)
	if err != nil {
		slog.Error("connect pooler", "error", err)
		os.Exit(1)
	}
	defer poolerPool.Close()

	// Direct connection — River client for job insertion (advisory locks need session mode)
	directPool, err := pgxpool.New(ctx, cfg.DatabaseDirectURL)
	if err != nil {
		slog.Error("connect direct", "error", err)
		os.Exit(1)
	}
	defer directPool.Close()

	if err := poolerPool.Ping(ctx); err != nil {
		slog.Error("ping failed", "error", err)
		os.Exit(1)
	}

	// Migrations via goose (needs database/sql interface)
	sqlDB := stdlib.OpenDBFromPool(directPool)
	if err := runMigrations(sqlDB); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	// River schema migration (river_job, river_queue, etc.)
	riverMigrator, err := rivermigrate.New(riverpgxv5.New(directPool), nil)
	if err != nil {
		slog.Error("river migrator init", "error", err)
		os.Exit(1)
	}
	if _, err := riverMigrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		slog.Error("river migrations failed", "error", err)
		os.Exit(1)
	}

	database := db.New(poolerPool)

	riverClient, err := river.NewClient(riverpgxv5.New(directPool), &river.Config{})
	if err != nil {
		slog.Error("create river client", "error", err)
		os.Exit(1)
	}

	defaultTenantID := uuid.MustParse(os.Getenv("DEFAULT_TENANT_ID"))

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Event-Type", "Idempotency-Key"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	ingestH := &handlers.IngestHandler{
		DB:              database,
		River:           riverClient,
		DefaultTenantID: defaultTenantID,
	}
	r.Post("/ingest/{source}", ingestH.Handle)

	r.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(database))

		endpointsH := &handlers.EndpointsHandler{DB: database}
		eventsH := &handlers.EventsHandler{DB: database, River: riverClient}
		deliveriesH := &handlers.DeliveriesHandler{DB: database, River: riverClient}
		apikeysH := &handlers.APIKeysHandler{DB: database}

		r.Route("/v1/endpoints", endpointsH.Routes())
		r.Route("/v1/events", eventsH.Routes())
		r.Route("/v1/deliveries", deliveriesH.Routes())
		r.Route("/v1/apikeys", apikeysH.Routes())
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("api server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func runMigrations(sqlDB *sql.DB) error {
	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(sqlDB, ".")
}

var _ = jobs.FanoutArgs{} // keep jobs package linked
