# logship Roadmap

## v0.1 (MVP)
- HTTP /ingest, Syslog TCP/UDP, File tail
- SQLite WAL storage, retention GC
- gojq-based transforms
- HTTP forwarder (batched), retry & backoff
- Minimal viewer UI and /logs API
- Docker image & compose, GH Actions build

## v0.2
- Basic auth for UI & API
- Backpressure: watermarks & 429 on /ingest
- Configurable rate limits per input
- Structured log levels & mapping from syslog severity
- Health/metrics endpoint (Prometheus)

## v0.3
- Loki / Elasticsearch / S3 sinks
- OpenTelemetry OTLP input/output
- GROK + GeoIP transforms
- Live tail via Server-Sent Events

## v0.4
- Multi-tenant API keys & quotas
- UI filters, pagination, dark/light themes
- Export/download query results

## v1.0
- HA notes & backup/restore docs
- Benchmarks and soak tests
