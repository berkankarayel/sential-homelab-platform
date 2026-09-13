// Package storage persists collector.Result values to Postgres and lets
// the API layer query them back.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulse/internal/collector"
)

// schema creates the results table and its lookup index if they don't
// already exist. Run once on every Open so a fresh database is usable
// immediately, without a separate migration step.
const schema = `
CREATE TABLE IF NOT EXISTS results (
	id             BIGSERIAL PRIMARY KEY,
	collector_name TEXT NOT NULL,
	ts             TIMESTAMPTZ NOT NULL,
	status         TEXT NOT NULL,
	data           JSONB,
	error          TEXT
);

CREATE INDEX IF NOT EXISTS idx_results_collector_name_ts
	ON results (collector_name, ts DESC);
`

// Store persists and retrieves collector results in Postgres.
type Store struct {
	pool *pgxpool.Pool
}

// Open connects to Postgres using the given DSN and ensures the schema
// exists. The returned Store is safe for concurrent use.
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: ping: %w", err)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: migrate: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Close releases the underlying connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// SaveResult persists a single collector result.
func (s *Store) SaveResult(ctx context.Context, r collector.Result) error {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return fmt.Errorf("storage: marshal data: %w", err)
	}

	var errText *string
	if r.Err != nil {
		text := r.Err.Error()
		errText = &text
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO results (collector_name, ts, status, data, error)
		 VALUES ($1, $2, $3, $4, $5)`,
		r.CollectorName, r.Timestamp, string(r.Status), data, errText,
	)
	if err != nil {
		return fmt.Errorf("storage: insert: %w", err)
	}
	return nil
}

// LatestAll returns the most recent result for every distinct collector
// name that has ever reported in, ordered by collector name.
func (s *Store) LatestAll(ctx context.Context) ([]collector.Result, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (collector_name)
			collector_name, ts, status, data, error
		 FROM results
		 ORDER BY collector_name, ts DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: query latest: %w", err)
	}
	defer rows.Close()

	var results []collector.Result
	for rows.Next() {
		r, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: query latest: %w", err)
	}

	return results, nil
}

// ErrNotFound is returned by LatestByName when the named collector has
// never reported a result.
var ErrNotFound = errors.New("storage: no result for collector")

// LatestByName returns the most recent result for a single collector.
// It returns ErrNotFound if name has never reported in.
func (s *Store) LatestByName(ctx context.Context, name string) (collector.Result, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT collector_name, ts, status, data, error
		 FROM results
		 WHERE collector_name = $1
		 ORDER BY ts DESC
		 LIMIT 1`,
		name,
	)

	r, err := scanResult(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return collector.Result{}, ErrNotFound
		}
		return collector.Result{}, err
	}
	return r, nil
}

// rowScanner is satisfied by both pgx.Rows and pgx.Row, so scanResult
// works for both the single-row and multi-row query paths above.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanResult(row rowScanner) (collector.Result, error) {
	var (
		r       collector.Result
		status  string
		ts      time.Time
		data    []byte
		errText *string
	)

	if err := row.Scan(&r.CollectorName, &ts, &status, &data, &errText); err != nil {
		return collector.Result{}, fmt.Errorf("storage: scan: %w", err)
	}

	r.Timestamp = ts
	r.Status = collector.Status(status)
	if errText != nil {
		r.Err = errors.New(*errText)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &r.Data); err != nil {
			return collector.Result{}, fmt.Errorf("storage: unmarshal data: %w", err)
		}
	}

	return r, nil
}
