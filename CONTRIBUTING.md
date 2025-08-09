# Contributing to logship

## Dev setup
- Go 1.22+
- `make run` or `docker compose up`

## Commit/PR
- Create a feature branch: `git checkout -b feature/<short-desc>`
- Run `go build ./...` and `go vet ./...` before pushing
- Open a PR with a clear description and testing notes

## Code style
- Idiomatic Go; small packages
- Keep the Docker image minimal (distroless)
