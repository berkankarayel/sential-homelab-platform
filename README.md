# Pulse

A small homelab observability platform written in Go — periodically checks
the health of services (and, eventually, Proxmox VMs and log files), stores
the results in Postgres, and exposes them over a REST API.

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
| `internal/storage` | done — Postgres persistence (pgx), auto-migrated schema |
| `internal/api` | done — `GET /status`, `GET /status/{name}` |
| `collectors/healthcheck` | done — HTTP GET check (status code + latency); TCP check still TODO |
| `collectors/proxmox`, `collectors/logwatch` | not started |

`cmd/pulse` runs the scheduler, persists every result to Postgres, and
serves the latest results over HTTP.

## Roadmap

- **v1** (done): scheduler + healthcheck collector + Postgres storage + REST
  API
- **v2**: Proxmox collector — VM/LXC status, CPU/RAM/disk usage via the
  Proxmox API
- **v3**: log-tail collector (regex pattern matching) + alerting (webhook /
  Telegram on repeated failures)
- **v4**: Redis cache for fast reads, optional simple status-board web UI
- **later**: Jenkins pipeline → local Nexus registry → deploy from
  Kubernetes once that stack is in place

Not a commitment — each stage is scoped once the previous one is working.

## Running

### With Docker Compose (app + Postgres)

```bash
docker compose up --build
```

This brings up Postgres and Pulse together, with no manual setup — Pulse
waits for Postgres to report healthy before starting.

### Locally

Requires a reachable Postgres instance (`docker compose up -d postgres`
works for this):

```bash
export PULSE_DB_DSN="postgres://pulse:pulse@localhost:5432/pulse?sslmode=disable"
go run ./cmd/pulse
```

### Querying the API

```bash
curl http://localhost:8080/status
curl http://localhost:8080/status/healthcheck:self
```

```json
[
  {
    "collector_name": "healthcheck:self",
    "timestamp": "2026-09-13T15:05:11Z",
    "status": "up",
    "data": { "latency_ms": 225, "status_code": 200, "target": "https://example.com" }
  }
]
```

A collector that has never reported in returns `404`; a failed check still
returns `200` with `"status": "down"` and an `"error"` field.

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `PULSE_DB_DSN` | — (required) | Postgres connection string |
| `PULSE_ADDR` | `:8080` | HTTP listen address for the REST API |
| `PULSE_HEALTHCHECK_TARGETS` | — | Comma-separated `name=url` pairs, one healthcheck collector per entry, e.g. `jenkins=http://jenkins.local:8080/login,grafana=http://grafana.local:3000` |
| `PULSE_HEALTHCHECK_TARGET` | `https://example.com` | Single target used only when `PULSE_HEALTHCHECK_TARGETS` is unset |
| `PULSE_HEALTHCHECK_INTERVAL` | `15s` | How often every healthcheck collector runs (Go duration syntax) |

There's no hardcoded target — point `PULSE_HEALTHCHECK_TARGETS` at whatever
services you actually want to watch.

## Testing

```bash
go test ./...
```

The storage layer also has an integration test against a real Postgres,
skipped unless `PULSE_TEST_DB_DSN` is set:

```bash
docker compose up -d postgres
PULSE_TEST_DB_DSN="postgres://pulse:pulse@localhost:5432/pulse?sslmode=disable" go test ./internal/storage/...
```
