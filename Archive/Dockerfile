# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22

# -------- builder --------
FROM golang:${GO_VERSION} AS builder
WORKDIR /src

# Cache modules
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source
COPY . .

# Build (pure static, no CGO)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/logship ./cmd/logship

# -------- runtime --------
FROM gcr.io/distroless/static:nonroot
WORKDIR /
ENV DATA_DIR=/var/lib/logship \
    HTTP_LISTEN=:8080 \
    SYSLOG_UDP_LISTEN=:5514 \
    SYSLOG_TCP_LISTEN=:5514

EXPOSE 8080 5514/tcp 5514/udp
VOLUME ["/var/lib/logship"]

COPY --from=builder /out/logship /logship
USER nonroot:nonroot
ENTRYPOINT ["/logship"]
