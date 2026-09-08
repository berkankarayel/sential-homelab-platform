// Package storage persists collector.Result values to Postgres and lets
// the API layer query them back.
//
// TODO:
//   - Open a Postgres connection pool (pgx recommended - pure Go, no CGO)
//     from a DSN read via an environment variable (e.g. PULSE_DB_DSN)
//   - Add a migration for a results table: collector_name, timestamp,
//     status, data (jsonb), error
//   - Implement SaveResult(ctx, collector.Result) error
//   - Implement LatestAll(ctx) ([]collector.Result, error) - latest row
//     per distinct collector_name
//   - Implement LatestByName(ctx, name string) (collector.Result, error)
package storage

import (
	"context"

	"pulse/internal/collector"
)

// Store persists and retrieves collector results.
type Store struct {
	// TODO: pool *pgxpool.Pool
}

// Open connects to Postgres using the given DSN.
func Open(dsn string) (*Store, error) {
	panic("TODO: implement")
}

// SaveResult persists a single collector result.
func (s *Store) SaveResult(ctx context.Context, r collector.Result) error {
	panic("TODO: implement")
}

// LatestAll returns the most recent result for every known collector.
func (s *Store) LatestAll(ctx context.Context) ([]collector.Result, error) {
	panic("TODO: implement")
}
