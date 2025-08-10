# syntax=docker/dockerfile:1.7

########################
# Build
########################
FROM golang:1.22 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# build once per arch; your GH Actions will handle matrix/multi-arch
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
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
