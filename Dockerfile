# syntax=docker/dockerfile:1.7

############################
# Build stage
############################
FROM --platform=$BUILDPLATFORM golang:1.22 AS build
WORKDIR /src

# Build args for cross-compile and edition tagging
ARG TARGETOS
ARG TARGETARCH
ARG EDITION=base

# Copy module files first (better caching)
COPY go.mod ./
# go.sum might not exist on first build — that's OK
# COPY go.sum ./

# Copy the rest of the source
COPY . .

# Resolve deps and build a static binary for the target OS/ARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w" \
      -o /out/logship ./cmd/logship

############################
# Runtime stage
############################
FROM gcr.io/distroless/base-debian12
WORKDIR /app

# Optional: record edition in the image env (your code can read this later)
ARG EDITION=base
ENV LOGSHIP_EDITION=${EDITION}

# App binary
COPY --from=build /out/logship /app/logship

# Default config (if you keep one in the repo)
# Safe even if the file doesn't exist in your repo — comment this line out if you prefer mounting it only.
COPY config.yaml /app/config.yaml

# Modern UI static assets (served by the app)
# If your server expects a different path, tweak this.
COPY internal/ui/static /app/static

# Data and host log mounts
VOLUME ["/data", "/var/log/host"]

# Ports: HTTP + Syslog TCP/UDP
EXPOSE 8080 514/tcp 514/udp

# Run
ENTRYPOINT ["/app/logship","-config","/app/config.yaml"]
