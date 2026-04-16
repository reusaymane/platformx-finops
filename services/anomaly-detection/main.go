package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/reusaymane/platformx-finops/anomaly-detection/internal/db"
	"github.com/reusaymane/platformx-finops/anomaly-detection/internal/detector"
	"github.com/reusaymane/platformx-finops/anomaly-detection/internal/notifier"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("starting anomaly-detection service")

	database, err := db.New(os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Fatal("failed to connect to db", zap.Error(err))
	}

	threshold := 2.5
	if t := os.Getenv("ZSCORE_THRESHOLD"); t != "" {
		if v, err := strconv.ParseFloat(t, 64); err == nil {
			threshold = v
		}
	}

	slack := notifier.NewSlack(os.Getenv("SLACK_WEBHOOK_URL"))

	detect := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		points, err := database.GetHourlyCosts(ctx, 8760) // last 7 days
		if err != nil {
			logger.Error("failed to get costs", zap.Error(err))
			return
		}

		// Group by namespace
		series := make(map[string][]float64)
		for _, p := range points {
			series[p.Namespace] = append(series[p.Namespace], p.TotalCost)
		}

		totalAnomalies := 0
		for ns, values := range series {
			anomalies := detector.DetectAnomalies(values, threshold)
			for _, a := range anomalies {
				logger.Warn("anomaly detected",
					zap.String("namespace", ns),
					zap.String("severity", a.Severity),
					zap.Float64("actual", a.Actual),
					zap.Float64("expected", a.Expected),
					zap.Float64("zscore", a.ZScore),
				)

				if err := database.InsertAnomaly(ctx, ns, "prod",
					a.Actual, a.Expected, a.ZScore, a.Severity); err != nil {
					logger.Error("failed to insert anomaly", zap.Error(err))
				}

				if err := slack.Notify(ns, a.Severity, a.Actual, a.Expected, a.ZScore); err != nil {
					logger.Error("slack notification failed", zap.Error(err))
				}

				totalAnomalies++
			}
		}

		logger.Info("detection cycle complete",
			zap.Int("namespaces", len(series)),
			zap.Int("anomalies", totalAnomalies),
		)
	}

	// Run immediately on startup
	detect()

	// Then every 15 minutes
	c := cron.New()
	c.AddFunc("*/15 * * * *", detect)
	c.Start()

	// Health endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Info("listening", zap.String("port", port))
	http.ListenAndServe(":"+port, nil)
}
