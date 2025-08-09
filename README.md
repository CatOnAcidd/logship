# logship

A lightweight log collector + buffer + transformer + viewer + forwarder in one Dockerable box.

## Features (MVP)
- Inputs: HTTP `/ingest`, Syslog TCP/UDP 514, file tail of mounted folder
- Storage: SQLite (WAL) with retention
- Transform: Config-driven JQ (via gojq)
- Forward: Batched HTTP with headers
- Viewer: Zero-JS-build static page at `/` and JSON API `/logs`

## Quick start

```bash
docker compose up --build -d
# send a test log
curl -XPOST "http://localhost:8080/ingest" -H "content-type: application/json" \
  -d '{"ts": 1723180000000, "host":"testbox","level":"info","message":"hello world","user":"a@b.com"}'
# open http://localhost:8080/
```

## Config
See `config.yaml` for defaults. Mount your own at `/app/config.yaml` or pass `-config`.

## Notes
- SQLite driver is pure Go (modernc.org/sqlite), so the image is CGO-free and small.
- Forwarder currently supports a single HTTP destination; extend `internal/forward` for more sinks.
