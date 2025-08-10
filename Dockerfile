# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build     CGO_ENABLED=0 GOOS=linux     go build -trimpath -ldflags "-s -w" -o /out/logship ./cmd/logship

FROM gcr.io/distroless/base-debian12:latest
WORKDIR /
COPY --from=build /out/logship /logship
EXPOSE 8080 5514/tcp 5514/udp
VOLUME ["/var/lib/logship"]
USER 65532:65532
ENTRYPOINT ["/logship"]
