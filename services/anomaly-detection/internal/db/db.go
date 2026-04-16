package db

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct{ *sqlx.DB }

type CostPoint struct {
	Bucket    time.Time `db:"bucket"`
	Namespace string    `db:"namespace"`
	TotalCost float64   `db:"total_cost"`
}

func New(dsn string) (*DB, error) {
	d, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(5)
	return &DB{d}, nil
}

// GetHourlyCosts returns the last N hours of costs for all namespaces
func (d *DB) GetHourlyCosts(ctx context.Context, hours int) ([]CostPoint, error) {
	var points []CostPoint
	err := d.SelectContext(ctx, &points, `
		SELECT
			time_bucket('1 hour', time) AS bucket,
			namespace,
			SUM(cost_usd) AS total_cost
		FROM cost_records
		WHERE time > NOW() - ($1 || ' hours')::interval
		GROUP BY bucket, namespace
		ORDER BY namespace, bucket ASC
	`, hours)
	return points, err
}

// InsertAnomaly stores a detected anomaly
func (d *DB) InsertAnomaly(ctx context.Context, namespace, environment string, actual, expected, zscore float64, severity string) error {
	_, err := d.ExecContext(ctx, `
		INSERT INTO anomalies (namespace, environment, cost_actual, cost_expected, zscore, severity)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, namespace, environment, actual, expected, zscore, severity)
	return err
}
