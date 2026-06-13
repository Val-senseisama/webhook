package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webhook/internal/config"
	"webhook/internal/db"
	"webhook/internal/jobs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	ctx := context.Background()

	// Workers always use the direct connection — advisory locks require session mode.
	pool, err := pgxpool.New(ctx, cfg.DatabaseDirectURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("ping failed", "error", err)
		os.Exit(1)
	}

	database := db.New(pool)

	httpClient := &http.Client{
		Timeout: 35 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	fanoutWorker := &jobs.FanoutWorker{DB: database}
	deliveryWorker := &jobs.DeliveryWorker{DB: database, HTTPClient: httpClient}

	workers := river.NewWorkers()
	river.AddWorker(workers, fanoutWorker)
	river.AddWorker(workers, deliveryWorker)

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 50},  // fanout
			"delivery":         {MaxWorkers: 500}, // outbound HTTP delivery
		},
		Workers:     workers,
		RetryPolicy: &jobs.RetryPolicy{},
		Logger:      slog.Default(),
	})
	if err != nil {
		slog.Error("create river client", "error", err)
		os.Exit(1)
	}

	// Inject the live client into the fanout worker so it can enqueue delivery jobs.
	fanoutWorker.River = riverClient

	if err := riverClient.Start(ctx); err != nil {
		slog.Error("start river", "error", err)
		os.Exit(1)
	}

	slog.Info("worker started", "fanout_workers", 50, "delivery_workers", 500)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("draining in-flight jobs")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := riverClient.Stop(shutdownCtx); err != nil {
		slog.Error("river shutdown error", "error", err)
	}
}
