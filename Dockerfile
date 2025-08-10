# syntax=docker/dockerfile:1.7

# ===== Build stage =====
FROM golang:1.22 AS build
WORKDIR /src

# Cache modules by go.mod content only
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Now copy the rest and tidy to generate go.sum inside image
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/logship ./cmd/logship

# ===== Runtime =====
FROM gcr.io/distroless/base-debian12@sha256:1951bedd9ab20dd71a5ab11b3f5a624863d7af4109f299d62289928b9e311d5d

# We run as root so volume mounts are writable everywhere; switch to nonroot later if desired.
WORKDIR /app
COPY --from=build /out/logship /logship

EXPOSE 8080/tcp
EXPOSE 5514/tcp
EXPOSE 5514/udp

VOLUME ["/var/lib/logship"]

ENV DATA_DIR=/var/lib/logship \
    HTTP_ADDR=:8080 \
    SYSLOG_UDP_LISTEN=:5514 \
    SYSLOG_TCP_LISTEN=

ENTRYPOINT ["/logship"]
