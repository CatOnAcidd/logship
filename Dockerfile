# syntax=docker/dockerfile:1.7

########################
# Build
########################
FROM golang:1.22 AS build
WORKDIR /src

# Copy just go.mod to warm the module cache (no go.sum needed)
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Now copy the full source and tidy to produce go.sum inside the image
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags "-s -w" -o /out/logship ./cmd/logship

# create a data dir layer we can chown in the final image
RUN mkdir -p /out/var/lib/logship

########################
# Runtime (distroless, non-root)
########################
FROM gcr.io/distroless/base-debian12:latest

# Run as the predefined non-root user in distroless
USER nonroot:nonroot

# Binary and data dir (owned by nonroot)
COPY --from=build /out/logship /logship
COPY --from=build --chown=nonroot:nonroot /out/var/lib/logship /var/lib/logship

ENV DATA_DIR=/var/lib/logship
EXPOSE 8080 5514/tcp 5514/udp

ENTRYPOINT ["/logship"]
