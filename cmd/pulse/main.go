// Command pulse wires together the scheduler, storage, and API layers, and
// runs until interrupted.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"pulse/collectors/healthcheck"
	"pulse/internal/api"
	"pulse/internal/collector"
	"pulse/internal/scheduler"
	"pulse/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := storage.Open(ctx, requireEnv("PULSE_DB_DSN"))
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	interval := healthCheckInterval()
	sched := scheduler.New()
	for _, c := range healthCheckCollectors(interval) {
		sched.Register(c)
	}

	schedDone := make(chan error, 1)
	go func() { schedDone <- sched.Run(ctx) }()

	resultsDone := make(chan struct{})
	go func() {
		defer close(resultsDone)
		for result := range sched.Results() {
			logResult(result)
			if err := store.SaveResult(context.Background(), result); err != nil {
				log.Printf("storage: save %s: %v", result.CollectorName, err)
			}
		}
	}()

	httpServer := &http.Server{
		Addr:    listenAddr(),
		Handler: api.New(store).Handler(),
	}
	go func() {
		log.Printf("api: listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("api: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("api: shutdown: %v", err)
	}

	if err := <-schedDone; err != nil {
		log.Fatalf("scheduler: %v", err)
	}
	<-resultsDone
}

func logResult(result collector.Result) {
	if result.Err != nil {
		log.Printf("%s: %s (%v)", result.CollectorName, result.Status, result.Err)
		return
	}
	log.Printf("%s: %s %v", result.CollectorName, result.Status, result.Data)
}

// requireEnv reads an environment variable and exits with a clear error if
// it isn't set, instead of failing later with a confusing connection error.
func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s must be set", key)
	}
	return value
}

func listenAddr() string {
	if addr := os.Getenv("PULSE_ADDR"); addr != "" {
		return addr
	}
	return ":8080"
}

func healthCheckInterval() time.Duration {
	raw := os.Getenv("PULSE_HEALTHCHECK_INTERVAL")
	if raw == "" {
		return 15 * time.Second
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Fatalf("invalid PULSE_HEALTHCHECK_INTERVAL %q: %v", raw, err)
	}
	return d
}

// healthCheckCollectors builds one healthcheck.Collector per entry in
// PULSE_HEALTHCHECK_TARGETS, a comma-separated list of "name=url" pairs,
// e.g. "jenkins=http://jenkins.local:8080/login,grafana=http://grafana.local:3000".
// If it's unset, it falls back to a single self-check (PULSE_HEALTHCHECK_TARGET,
// default https://example.com) so the service is useful out of the box
// without any environment-specific configuration.
func healthCheckCollectors(interval time.Duration) []collector.Collector {
	raw := os.Getenv("PULSE_HEALTHCHECK_TARGETS")
	if raw == "" {
		target := os.Getenv("PULSE_HEALTHCHECK_TARGET")
		if target == "" {
			target = "https://example.com"
		}
		return []collector.Collector{healthcheck.New("healthcheck:self", target, interval)}
	}

	var collectors []collector.Collector
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, url, ok := strings.Cut(entry, "=")
		if !ok {
			log.Fatalf("invalid PULSE_HEALTHCHECK_TARGETS entry %q: expected name=url", entry)
		}
		collectors = append(collectors, healthcheck.New(
			"healthcheck:"+strings.TrimSpace(name),
			strings.TrimSpace(url),
			interval,
		))
	}
	return collectors
}
