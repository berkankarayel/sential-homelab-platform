# Pulse

A small homelab observability platform written in Go — periodically checks
the health of services (and, eventually, Proxmox VMs and log files), stores
the results, and exposes them over a REST API.

Pulse isn't meant to replace Prometheus/Grafana or Uptime Kuma for real
homelab monitoring — it's a long-running learning project for Go, backend
design, and the DevOps tooling around it (Docker, and later Jenkins/Nexus/
Kubernetes).

## Architecture

Everything is built around one interface. Any data source — an HTTP ping, a
Proxmox API poll, a log tail — implements it the same way:

```go
type Collector interface {
    Name() string
    Collect(ctx context.Context) (Result, error)
    Interval() time.Duration
}
```

A central `Scheduler` holds a list of registered collectors and runs each
one on its own goroutine, ticking at its own `Interval()`. A slow or failing
collector can't block or take down the others. Results flow through a
channel to whatever consumes them next:

```
Collector (healthcheck, proxmox, logwatch, ...)
        │  Collect(ctx) -> Result
        ▼
   Scheduler  (one goroutine + ticker per collector)
        │  Result channel
        ▼
   Storage (Postgres)  ──▶  API (REST)
```

Adding a new data source is just writing a new `Collector` implementation —
the scheduler, storage, and API layers don't change.

## Layout

```
cmd/pulse/              entrypoint - wires scheduler + storage + api together
internal/collector/      the Collector interface + Result/Status types
internal/scheduler/      runs registered collectors on their own interval
internal/storage/        Postgres persistence for collector results
internal/api/            REST API to query results
collectors/healthcheck/  HTTP health check Collector implementation
```

`internal/` is used deliberately: it's a Go compiler rule, not a visibility
setting. Anyone can read this public repo, but nothing outside this module
can import these packages — the only supported way in is through
`cmd/pulse`.

## Status

| Piece | State |
|---|---|
| `internal/collector` | done — interface + `Result`/`Status` types |
| `internal/scheduler` | done — per-collector goroutine, ticker, graceful shutdown via context |
| `collectors/healthcheck` | done — HTTP GET check (status code + latency); TCP check still TODO |
| `internal/storage` | stub — Postgres persistence not implemented yet |
| `internal/api` | stub — REST endpoints not implemented yet |
| `collectors/proxmox`, `collectors/logwatch` | not started |

Right now `cmd/pulse` runs the scheduler with a healthcheck collector and
logs each result to stdout — nothing is persisted or queryable over HTTP
yet.

## Roadmap

- **v1** (in progress): scheduler + healthcheck collector + Postgres storage
  + REST API (`GET /status`, `GET /status/{name}`)
- **v2**: Proxmox collector — VM/LXC status, CPU/RAM/disk usage via the
  Proxmox API
- **v3**: log-tail collector (regex pattern matching) + alerting (webhook /
  Telegram on repeated failures)
- **v4**: Redis cache for fast reads, optional simple status-board web UI
- **later**: Jenkins pipeline → local Nexus registry → deploy from
  Kubernetes once that stack is in place

Not a commitment — each stage is scoped once the previous one is working.

## Running

### Locally

```bash
go run ./cmd/pulse
```

### With Docker Compose (app + Postgres)

```bash
docker compose up --build
```

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `PULSE_HEALTHCHECK_TARGET` | `https://example.com` | URL the built-in healthcheck collector pings |
| `PULSE_DB_DSN` | — | Postgres DSN (used once `internal/storage` is wired in) |
| `PULSE_ADDR` | — | HTTP listen address (used once `internal/api` is wired in) |
