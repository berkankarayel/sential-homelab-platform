// Command pulse wires together the scheduler and its collectors, and runs
// until interrupted.
//
// TODO:
//   - store, err := storage.Open(os.Getenv("PULSE_DB_DSN")); sched results
//     get written to it instead of just logged to stdout
//   - http.ListenAndServe(os.Getenv("PULSE_ADDR"), api.New(store).Handler())
//   - Register more collectors (proxmox, logwatch) once they exist
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pulse/collectors/healthcheck"
	"pulse/internal/scheduler"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sched := scheduler.New()
	sched.Register(healthcheck.New("healthcheck:self", healthCheckTarget(), 15*time.Second))

	done := make(chan error, 1)
	go func() {
		done <- sched.Run(ctx)
	}()

	for result := range sched.Results() {
		if result.Err != nil {
			log.Printf("%s: %s (%v)", result.CollectorName, result.Status, result.Err)
			continue
		}
		log.Printf("%s: %s %v", result.CollectorName, result.Status, result.Data)
	}

	if err := <-done; err != nil {
		log.Fatalf("scheduler: %v", err)
	}
}

// healthCheckTarget returns the URL the default healthcheck collector pings,
// overridable via PULSE_HEALTHCHECK_TARGET until real config loading exists.
func healthCheckTarget() string {
	if target := os.Getenv("PULSE_HEALTHCHECK_TARGET"); target != "" {
		return target
	}
	return "https://example.com"
}
