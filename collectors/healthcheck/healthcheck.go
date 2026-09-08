// Package healthcheck implements collector.Collector for a simple HTTP or
// TCP reachability check against a target service.
//
// TODO:
//   - Fields: name, target (URL for HTTP, host:port for TCP), timeout
//   - Collect(): HTTP GET (or net.Dial for TCP) using ctx, map the outcome
//     to collector.StatusUp / collector.StatusDown, put latency/status code
//     into Result.Data
//   - New(name, target string, interval time.Duration) *HealthCheck
package healthcheck

import (
	"context"
	"time"

	"pulse/internal/collector"
)

// HealthCheck pings a single HTTP or TCP target on an interval.
type HealthCheck struct {
	// TODO: name, target string
	// TODO: interval     time.Duration
	// TODO: httpClient   *http.Client
}

// New creates a HealthCheck collector for the given target.
func New(name, target string, interval time.Duration) *HealthCheck {
	panic("TODO: implement")
}

func (h *HealthCheck) Name() string {
	panic("TODO: implement")
}

func (h *HealthCheck) Interval() time.Duration {
	panic("TODO: implement")
}

func (h *HealthCheck) Collect(ctx context.Context) (collector.Result, error) {
	panic("TODO: implement")
}
