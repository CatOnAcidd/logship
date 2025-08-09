# Build stage
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/logship ./cmd/logship

# Runtime (distroless)
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/logship /app/logship
COPY config.yaml /app/config.yaml
VOLUME ["/data", "/var/log/host"]
EXPOSE 8080 514/tcp 514/udp
ENTRYPOINT ["/app/logship","-config","/app/config.yaml"]
