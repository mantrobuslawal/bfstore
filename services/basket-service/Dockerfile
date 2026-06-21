# syntax=docker/dockerfile:1.7

# Basket Service container image.
#
# Build context:
#   repo root
#
# Example:
#   docker build \
#     -f services/basket-service/Dockerfile \
#     --build-arg SERVICE_VERSION=0.1.0 \
#     -t bfstore/basket-service:0.1.0 \
#     .

ARG GO_VERSION=1.26.3
ARG DEBIAN_VERSION=12

FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Copy workspace and module files first to improve dependency caching.
COPY go.work go.mod go.sum ./

# go.work references both catalog-service and basket-service modules.
# Copy both service go.mod files so workspace resolution succeeds during basket builds.
# This does not mean basket-service imports catalog-service directly.
COPY services/catalog-service/go.mod services/catalog-service/go.sum ./services/catalog-service/
COPY services/basket-service/go.mod ./services/basket-service/go.mod

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && \
    cd services/catalog-service && go mod download && \
    cd ../basket-service && go mod download

# Copy only the source needed to build the basket service.
COPY gen/ ./gen/
COPY pkg/ ./pkg/
COPY services/basket-service/ ./services/basket-service/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd services/basket-service && \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/basket-service \
      ./cmd/basket-service

FROM gcr.io/distroless/static-debian${DEBIAN_VERSION}:nonroot AS runtime

ARG SERVICE_VERSION=0.1.0

LABEL org.opencontainers.image.title="bfstore basket-service" \
      org.opencontainers.image.description="bfstore Basket Service gRPC API" \
      org.opencontainers.image.version="${SERVICE_VERSION}" \
      org.opencontainers.image.source="https://github.com/mantrobuslawal/bfstore" \
      org.opencontainers.image.vendor="Borough Furniture Store" \
      org.opencontainers.image.licenses="Apache-2.0"

WORKDIR /

COPY --from=build --chown=nonroot:nonroot /out/basket-service /basket-service

USER nonroot:nonroot

EXPOSE 50052

ENTRYPOINT ["/basket-service"]
