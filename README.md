# Pulse

Homelab observability platform. Full roadmap and rationale: see `pulse.txt`.

## Layout

```
cmd/pulse/              entrypoint - wires scheduler + storage + api together
internal/collector/     the Collector interface every data source implements
internal/scheduler/     runs registered collectors on their own interval
internal/storage/       Postgres persistence for collector results
internal/api/           REST API to query results
collectors/healthcheck/ first Collector implementation (HTTP/TCP ping)
```

Everything outside `internal/collector/collector.go` is a stub (`panic("TODO: implement")`) - the shape is defined, the logic isn't written yet.

## Run (once implemented)

```bash
go run ./cmd/pulse
```

## Run with Postgres via Docker

```bash
docker compose up --build
```
