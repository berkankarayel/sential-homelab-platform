// Package scheduler runs a set of collector.Collector implementations on
// their own interval and forwards each Result onward (to storage, for now).
//
// TODO:
//   - Hold a registry of collector.Collector
//   - For each registered collector, run a goroutine with a time.Ticker
//     set to its Interval()
//   - Respect context cancellation for graceful shutdown (stop all tickers,
//     wait for in-flight Collect calls with sync.WaitGroup)
//   - Send each collector.Result somewhere useful (a channel the caller
//     reads, or call into storage directly)
//   - A failing Collect() must not take down the other collectors' goroutines
package scheduler

import (
	"context"

	"pulse/internal/collector"
)

// Scheduler runs registered collectors on their own schedule.
type Scheduler struct {
	// TODO: collectors []collector.Collector
	// TODO: results    chan collector.Result
}

// New creates an empty Scheduler.
func New() *Scheduler {
	panic("TODO: implement")
}

// Register adds a collector to be run periodically once Run starts.
func (s *Scheduler) Register(c collector.Collector) {
	panic("TODO: implement")
}

// Run starts all registered collectors and blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	panic("TODO: implement")
}
