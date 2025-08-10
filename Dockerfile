# syntax=docker/dockerfile:1.7

FROM golang:1.22 AS build
WORKDIR /src

# 1) Warm cache
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# 2) Copy the rest and then TIDY (this writes go.sum in the builder)
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy

# 3) Build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags "-s -w" -o /out/logship ./cmd/logship

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/logship /app/logship
ENV LISTEN_ADDR=:8080
ENV DATA_DIR=/var/lib/logship
ENV SYSLOG_UDP_LISTEN=:5514
ENV SYSLOG_TCP_LISTEN=:5514
VOLUME ["/var/lib/logship"]
EXPOSE 8080 5514/tcp 5514/udp
ENTRYPOINT ["/app/logship"]
