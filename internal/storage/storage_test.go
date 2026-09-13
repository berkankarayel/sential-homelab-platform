package storage_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"pulse/internal/collector"
	"pulse/internal/storage"
)

// TestStore exercises Store against a real Postgres instance. It's an
// integration test, not a unit test: it needs PULSE_TEST_DB_DSN pointing at
// a throwaway database (e.g. the one docker-compose.yml starts) and is
// skipped otherwise, so `go test ./...` still passes without Postgres
// running.
func TestStore(t *testing.T) {
	dsn := os.Getenv("PULSE_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("PULSE_TEST_DB_DSN not set, skipping Postgres integration test")
	}

	ctx := context.Background()
	store, err := storage.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(store.Close)

	name := "healthcheck:storage-test-" + time.Now().Format("150405.000000000")

	err = store.SaveResult(ctx, collector.Result{
		CollectorName: name,
		Timestamp:     time.Now(),
		Status:        collector.StatusUp,
		Data:          map[string]any{"latency_ms": int64(12)},
	})
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	result, err := store.LatestByName(ctx, name)
	if err != nil {
		t.Fatalf("LatestByName: %v", err)
	}
	if result.Status != collector.StatusUp {
		t.Errorf("Status = %v, want %v", result.Status, collector.StatusUp)
	}
	if latency, _ := result.Data["latency_ms"].(float64); latency != 12 {
		t.Errorf("Data[latency_ms] = %v, want 12", result.Data["latency_ms"])
	}

	all, err := store.LatestAll(ctx)
	if err != nil {
		t.Fatalf("LatestAll: %v", err)
	}
	if !containsName(all, name) {
		t.Errorf("LatestAll() missing collector %q: %+v", name, all)
	}

	_, err = store.LatestByName(ctx, "does-not-exist-"+name)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("err = %v, want storage.ErrNotFound", err)
	}
}

func containsName(results []collector.Result, name string) bool {
	for _, r := range results {
		if r.CollectorName == name {
			return true
		}
	}
	return false
}
