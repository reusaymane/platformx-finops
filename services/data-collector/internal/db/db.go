package db

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	*sqlx.DB
}

// CostRecord is a single cost data point stored in TimescaleDB.
type CostRecord struct {
	Time        time.Time `db:"time"`
	Namespace   string    `db:"namespace"`
	Service     string    `db:"service"`
	Environment string    `db:"environment"`
	Team        string    `db:"team"`
	CostUSD     float64   `db:"cost_usd"`
	Source      string    `db:"source"`
}

func New(dsn string) (*DB, error) {
	sqlxDB, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	sqlxDB.SetMaxOpenConns(10)
	sqlxDB.SetMaxIdleConns(5)
	sqlxDB.SetConnMaxLifetime(5 * time.Minute)
	return &DB{sqlxDB}, nil
}

// InsertCostRecords bulk-inserts cost records into TimescaleDB.
func (d *DB) InsertCostRecords(ctx context.Context, records []CostRecord) error {
	if len(records) == 0 {
		return nil
	}

	const batchSize = 1000
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		query := `
			INSERT INTO cost_records (time, namespace, service, environment, team, cost_usd, source)
			VALUES (:time, :namespace, :service, :environment, :team, :cost_usd, :source)
			ON CONFLICT DO NOTHING`
		if _, err := d.NamedExecContext(ctx, query, batch); err != nil {
			return err
		}
	}
	return nil
}

// LastCollectionTime returns the most recent record time for a given source.
func (d *DB) LastCollectionTime(ctx context.Context, source string) (time.Time, error) {
	var t time.Time
	err := d.GetContext(ctx, &t,
		`SELECT COALESCE(MAX(time), NOW() - INTERVAL '1 year')
		 FROM cost_records WHERE source = $1`, source)
	return t, err
}

// HealthCheck pings the database.
func (d *DB) HealthCheck(ctx context.Context) error {
	return d.PingContext(ctx)
}
