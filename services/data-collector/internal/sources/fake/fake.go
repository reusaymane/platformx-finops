package fake

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/reusaymane/platformx-finops/data-collector/internal/db"
	"go.uber.org/zap"
)

// Fake generates realistic simulated cost data for local dev.
// Simulates 1 year of hourly cost data across multiple namespaces.
type Fake struct {
	logger    *zap.Logger
	seeded    bool
	lastCheck time.Time
}

type nsConfig struct {
	namespace   string
	service     string
	environment string
	team        string
	baseCost    float64 // avg hourly cost in USD
}

var namespaces = []nsConfig{
	{"payments",   "payments-api",   "prod",    "backend",  2.80},
	{"orders",     "orders-api",     "prod",    "backend",  2.20},
	{"users",      "users-api",      "prod",    "backend",  1.60},
	{"monitoring", "prometheus",     "prod",    "platform", 1.10},
	{"ml",         "ml-forecasting", "prod",    "data",     3.40},
	{"payments",   "payments-api",   "staging", "backend",  0.60},
	{"orders",     "orders-api",     "staging", "backend",  0.45},
	{"default",    "misc",           "dev",     "platform", 0.20},
}

func New(logger *zap.Logger) *Fake {
	return &Fake{logger: logger, lastCheck: time.Now().Add(-365 * 24 * time.Hour)}
}

func (f *Fake) Name() string { return "simulated" }

func (f *Fake) Collect(ctx context.Context) ([]db.CostRecord, error) {
	now := time.Now().UTC().Truncate(time.Hour)
	from := f.lastCheck.Truncate(time.Hour)

	var records []db.CostRecord
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for t := from; t.Before(now); t = t.Add(time.Hour) {
		for _, ns := range namespaces {
			cost := simulateCost(rng, t, ns.baseCost)
			records = append(records, db.CostRecord{
				Time:        t,
				Namespace:   ns.namespace,
				Service:     ns.service,
				Environment: ns.environment,
				Team:        ns.team,
				CostUSD:     cost,
				Source:      "simulated",
			})
		}
	}

	f.lastCheck = now
	f.logger.Info("fake data generated",
		zap.Int("records", len(records)),
		zap.Time("from", from),
		zap.Time("to", now),
	)
	return records, nil
}

// simulateCost generates a realistic cost value with:
// - weekly seasonality (lower on weekends)
// - daily pattern (higher during business hours)
// - long-term growth trend
// - random noise
// - occasional spikes (1% chance) to trigger anomaly detection
func simulateCost(rng *rand.Rand, t time.Time, base float64) float64 {
	// Long-term upward trend
	daysFromStart := t.Sub(time.Now().Add(-365 * 24 * time.Hour)).Hours() / 24
	trend := 1.0 + (daysFromStart/365)*0.15

	// Weekly seasonality: weekends ~40% cheaper
	weekdayFactor := 1.0
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		weekdayFactor = 0.60
	}

	// Daily pattern: business hours cost more (more traffic)
	hour := t.Hour()
	hourFactor := 0.6 + 0.4*math.Sin(math.Pi*float64(hour-6)/12)
	if hour < 6 || hour > 22 {
		hourFactor = 0.5
	}

	// Random noise ±15%
	noise := 1.0 + (rng.Float64()-0.5)*0.30

	cost := base * trend * weekdayFactor * hourFactor * noise

	// Occasional spike (1%) — simulates incidents for anomaly detection
	if rng.Float64() < 0.01 {
		cost *= 4.0 + rng.Float64()*6.0
	}

	return math.Round(cost*10000) / 10000
}
