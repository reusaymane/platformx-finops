package collector

import (
	"context"
	"time"

	"github.com/reusaymane/platformx-finops/data-collector/internal/config"
	"github.com/reusaymane/platformx-finops/data-collector/internal/db"
	"github.com/reusaymane/platformx-finops/data-collector/internal/sources/awsce"
	"github.com/reusaymane/platformx-finops/data-collector/internal/sources/fake"
	"github.com/reusaymane/platformx-finops/data-collector/internal/sources/prometheus"
	"go.uber.org/zap"
)

// Source is implemented by every data source (AWS CE, Prometheus, Kubecost, fake).
type Source interface {
	Name() string
	Collect(ctx context.Context) ([]db.CostRecord, error)
}

type Collector struct {
	cfg     *config.Config
	db      *db.DB
	sources []Source
	logger  *zap.Logger
}

func New(cfg *config.Config, database *db.DB, logger *zap.Logger) (*Collector, error) {
	var sources []Source

	if cfg.FakeMode {
		// Local dev — generate realistic simulated data
		logger.Info("fake mode enabled — using simulated cost data")
		sources = append(sources, fake.New(logger))
	} else {
		// Real AWS Cost Explorer
		awsSource, err := awsce.New(cfg.AWSRegion, logger)
		if err != nil {
			return nil, err
		}
		sources = append(sources, awsSource)

		// Prometheus (Kubecost metrics)
		promSource := prometheus.New(cfg.PrometheusURL, logger)
		sources = append(sources, promSource)
	}

	return &Collector{
		cfg:     cfg,
		db:      database,
		sources: sources,
		logger:  logger,
	}, nil
}

// Collect runs all sources once and stores results in TimescaleDB.
func (c *Collector) Collect(ctx context.Context) error {
	c.logger.Info("starting collection cycle")
	start := time.Now()
	total := 0

	for _, src := range c.sources {
		records, err := src.Collect(ctx)
		if err != nil {
			c.logger.Error("source collection failed",
				zap.String("source", src.Name()),
				zap.Error(err),
			)
			continue
		}

		if err := c.db.InsertCostRecords(ctx, records); err != nil {
			c.logger.Error("failed to insert records",
				zap.String("source", src.Name()),
				zap.Error(err),
			)
			continue
		}

		c.logger.Info("collected",
			zap.String("source", src.Name()),
			zap.Int("records", len(records)),
		)
		total += len(records)
	}

	c.logger.Info("collection cycle complete",
		zap.Int("total_records", total),
		zap.Duration("duration", time.Since(start)),
	)
	return nil
}

// Start runs Collect on a ticker loop until ctx is cancelled.
func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("collector stopped")
			return
		case <-ticker.C:
			if err := c.Collect(ctx); err != nil {
				c.logger.Error("collection error", zap.Error(err))
			}
		}
	}
}
