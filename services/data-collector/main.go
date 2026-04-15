package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reusaymane/platformx-finops/data-collector/internal/collector"
	"github.com/reusaymane/platformx-finops/data-collector/internal/config"
	"github.com/reusaymane/platformx-finops/data-collector/internal/db"
	"github.com/reusaymane/platformx-finops/data-collector/internal/server"
	"go.uber.org/zap"
)

func main() {
	// ── Logger ────────────────────────────────────────────────────────────────
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("starting platformx data-collector")

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// ── Database ──────────────────────────────────────────────────────────────
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to timescaledb", zap.Error(err))
	}
	defer database.Close()
	logger.Info("connected to timescaledb")

	// ── Collector ─────────────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	col, err := collector.New(cfg, database, logger)
	if err != nil {
		logger.Fatal("failed to create collector", zap.Error(err))
	}

	// Run first collection immediately
	if err := col.Collect(ctx); err != nil {
		logger.Error("initial collection failed", zap.Error(err))
	}

	// Schedule recurring collections
	go col.Start(ctx)

	// ── HTTP server (health + metrics) ────────────────────────────────────────
	srv := server.New(cfg.Port, database, logger)
	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("server error", zap.Error(err))
		}
	}()

	logger.Info("data-collector running",
		zap.String("port", cfg.Port),
		zap.String("interval", cfg.CollectInterval.String()),
		zap.Bool("fake_mode", cfg.FakeMode),
	)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	logger.Info("data-collector stopped")
}
