package prometheus

import (
	"context"
	"github.com/reusaymane/platformx-finops/data-collector/internal/db"
	"go.uber.org/zap"
)

type Prometheus struct {
	url    string
	logger *zap.Logger
}

func New(url string, logger *zap.Logger) *Prometheus {
	return &Prometheus{url: url, logger: logger}
}

func (p *Prometheus) Name() string { return "prometheus" }

func (p *Prometheus) Collect(ctx context.Context) ([]db.CostRecord, error) {
	p.logger.Info("prometheus collection not yet implemented")
	return nil, nil
}
