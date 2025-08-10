# syntax=docker/dockerfile:1.7

########################
# Build
########################
FROM golang:1.22 AS build
WORKDIR /src

# Preload deps for better Docker layer caching
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# App source
COPY . .

# Build static binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags "-s -w" -o /out/logship ./cmd/logship

########################
# Runtime (distroless)
########################
FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=build /out/logship /app/logship

# Defaults (can be overridden by envs / compose)
ENV LISTEN_ADDR=:8080
ENV DATA_DIR=/var/lib/logship

VOLUME ["/var/lib/logship"]
EXPOSE 8080 5514/tcp 5514/udp

# Run as root so we can create DATA_DIR by default (you can drop to nonroot via compose)
ENTRYPOINT ["/app/logship"]
