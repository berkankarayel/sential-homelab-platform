// Command pulse wires together the scheduler, storage, collectors and API
// server, and runs until interrupted.
//
// TODO:
//   - Read config from environment variables (PULSE_DB_DSN, PULSE_ADDR, ...)
//   - store, err := storage.Open(dsn)
//   - sched := scheduler.New(); sched.Register(healthcheck.New(...))
//   - go sched.Run(ctx)
//   - http.ListenAndServe(addr, api.New(store).Handler())
//   - Listen for SIGINT/SIGTERM and cancel ctx for graceful shutdown
package main

func main() {
	println("pulse: skeleton running - wire scheduler/storage/api together here")
}
