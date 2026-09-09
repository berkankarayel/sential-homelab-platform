// Package healthcheck implements collector.Collector for a simple HTTP
// reachability check against a target service.
//
// TODO:
//   - TCP port check variant (net.Dial) for non-HTTP targets
//   - Configurable retry before flagging a target down
package healthcheck

import (
	"context"
	"net/http"
	"time"

	"pulse/internal/collector"
)

// HealthCheck pings a single HTTP target on an interval.
type HealthCheck struct {
	name       string
	target     string
	interval   time.Duration
	httpClient *http.Client
}

// New creates a HealthCheck collector for the given target URL. name
// identifies this instance (e.g. "healthcheck:jenkins"); target is a full
// URL (e.g. "http://jenkins.local:8080/login").
func New(name, target string, interval time.Duration) *HealthCheck {
	return &HealthCheck{
		name:     name,
		target:   target,
		interval: interval,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (h *HealthCheck) Name() string {
	return h.name
}

func (h *HealthCheck) Interval() time.Duration {
	return h.interval
}

// Collect issues a single HTTP GET against the target. Any 2xx/3xx response
// is StatusUp; anything else (bad status code, timeout, connection refused)
// is StatusDown.
func (h *HealthCheck) Collect(ctx context.Context) (collector.Result, error) {
	result := collector.Result{
		CollectorName: h.name,
		Timestamp:     time.Now(),
		Data:          map[string]any{"target": h.target},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.target, nil)
	if err != nil {
		result.Status = collector.StatusUnknown
		result.Err = err
		return result, err
	}

	start := time.Now()
	resp, err := h.httpClient.Do(req)
	latency := time.Since(start)
	result.Data["latency_ms"] = latency.Milliseconds()

	if err != nil {
		result.Status = collector.StatusDown
		result.Err = err
		return result, nil
	}
	defer resp.Body.Close()

	result.Data["status_code"] = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = collector.StatusUp
	} else {
		result.Status = collector.StatusDown
	}

	return result, nil
}
