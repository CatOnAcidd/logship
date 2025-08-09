# syntax=docker/dockerfile:1.7

# Build stage (uses the build platform)
FROM --platform=$BUILDPLATFORM golang:1.22 AS build
WORKDIR /src
ARG TARGETOS
ARG TARGETARCH

# Copy go.mod (and go.sum if present) first to leverage layer caching
COPY go.mod ./
# go.sum may not exist on the first build — that's fine
# COPY go.sum ./ 

# Bring in the rest of the source
COPY . .

# Generate go.sum / download deps and build for the target arch
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/logship ./cmd/logship

# Runtime (distroless, per target platform)
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/logship /app/logship
COPY config.yaml /app/config.yaml
VOLUME ["/data", "/var/log/host"]
EXPOSE 8080 514/tcp 514/udp
ENTRYPOINT ["/app/logship","-config","/app/config.yaml"]
