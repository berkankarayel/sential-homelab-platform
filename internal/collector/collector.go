// Package collector defines the shared contract that every data source
// (health check, Proxmox, log watch, ...) must implement so the scheduler
// can run them all the same way.
package collector

import (
	"context"
	"time"
)

// Status is the outcome of a single Collect call.
type Status string

const (
	StatusUp      Status = "up"
	StatusDown    Status = "down"
	StatusUnknown Status = "unknown"
)

// Result is what a Collector produces on each run.
type Result struct {
	CollectorName string
	Timestamp     time.Time
	Status        Status
	Data          map[string]any
	Err           error
}

// Collector is implemented by every data source the scheduler can run
// (healthcheck, proxmox, logwatch, ...).
type Collector interface {
	// Name identifies this collector instance (e.g. "healthcheck:jenkins").
	Name() string

	// Collect runs one check/fetch and returns the outcome. Must respect
	// ctx cancellation/timeout.
	Collect(ctx context.Context) (Result, error)

	// Interval is how often the scheduler should call Collect.
	Interval() time.Duration
}

