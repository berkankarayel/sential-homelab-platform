// Package scheduler runs a set of collector.Collector implementations on
// their own interval and forwards each Result onward (to storage, for now
// just to a channel the caller reads).
package scheduler

import (
	"context"
	"sync"
	"time"

	"pulse/internal/collector"
)

// Scheduler runs registered collectors on their own schedule.
type Scheduler struct {
	collectors []collector.Collector
	results    chan collector.Result
}

// New creates an empty Scheduler.
func New() *Scheduler {
	return &Scheduler{
		results: make(chan collector.Result),
	}
}

// Register adds a collector to be run periodically once Run starts.
func (s *Scheduler) Register(c collector.Collector) {
	s.collectors = append(s.collectors, c)
}

// Results returns the channel every collector's output is sent on. It is
// closed once Run returns, after all collector goroutines have stopped.
func (s *Scheduler) Results() <-chan collector.Result {
	return s.results
}

// Run starts all registered collectors and blocks until ctx is cancelled.
// Each collector runs on its own goroutine and ticker, so a slow or failing
// Collect call on one collector never blocks or takes down the others.
func (s *Scheduler) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	for _, c := range s.collectors {
		wg.Add(1)
		go func(c collector.Collector) {
			defer wg.Done()
			s.runCollector(ctx, c)
		}(c)
	}

	wg.Wait()
	close(s.results)
	return nil
}

// runCollector calls Collect once immediately, then again on every tick of
// c.Interval(), until ctx is cancelled.
func (s *Scheduler) runCollector(ctx context.Context, c collector.Collector) {
	s.collect(ctx, c)

	ticker := time.NewTicker(c.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collect(ctx, c)
		}
	}
}

// collect runs one Collect call and forwards the result, unless ctx is
// cancelled before the result can be delivered.
func (s *Scheduler) collect(ctx context.Context, c collector.Collector) {
	result, err := c.Collect(ctx)
	if err != nil && result.Err == nil {
		result.Err = err
	}
	if result.CollectorName == "" {
		result.CollectorName = c.Name()
	}
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}

	select {
	case s.results <- result:
	case <-ctx.Done():
	}
}
