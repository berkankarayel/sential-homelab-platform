package scheduler_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"pulse/internal/collector"
	"pulse/internal/scheduler"
)

type fakeCollector struct {
	name     string
	interval time.Duration
	calls    atomic.Int32
}

func (f *fakeCollector) Name() string            { return f.name }
func (f *fakeCollector) Interval() time.Duration { return f.interval }

func (f *fakeCollector) Collect(ctx context.Context) (collector.Result, error) {
	f.calls.Add(1)
	return collector.Result{Status: collector.StatusUp}, nil
}

func TestRun_TicksAndFillsMissingFields(t *testing.T) {
	c := &fakeCollector{name: "test", interval: 20 * time.Millisecond}

	sched := scheduler.New()
	sched.Register(c)

	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- sched.Run(ctx) }()

	var results []collector.Result
	for r := range sched.Results() {
		results = append(results, r)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("got %d results, want at least 2 (one immediate + one on tick)", len(results))
	}
	if c.calls.Load() != int32(len(results)) {
		t.Errorf("Collect called %d times, but got %d results", c.calls.Load(), len(results))
	}

	first := results[0]
	if first.CollectorName != "test" {
		t.Errorf("CollectorName = %q, want %q", first.CollectorName, "test")
	}
	if first.Timestamp.IsZero() {
		t.Error("Timestamp not filled in")
	}
}

type erroringCollector struct{ name string }

func (e *erroringCollector) Name() string            { return e.name }
func (e *erroringCollector) Interval() time.Duration { return time.Hour }

func (e *erroringCollector) Collect(ctx context.Context) (collector.Result, error) {
	return collector.Result{}, errors.New("boom")
}

func TestRun_PropagatesCollectError(t *testing.T) {
	c := &erroringCollector{name: "err"}

	sched := scheduler.New()
	sched.Register(c)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- sched.Run(ctx) }()

	var results []collector.Result
	for r := range sched.Results() {
		results = append(results, r)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (interval is 1h, only the immediate call should fire)", len(results))
	}
	if results[0].Err == nil || results[0].Err.Error() != "boom" {
		t.Errorf("Err = %v, want %q", results[0].Err, "boom")
	}
}
