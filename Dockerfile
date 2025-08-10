# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22
FROM golang:${GO_VERSION} AS build
WORKDIR /src

COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy

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
EXPOSE 8080 5514/tcp 5514/udp
VOLUME ["/var/lib/logship"]
ENTRYPOINT ["/app/logship"]
