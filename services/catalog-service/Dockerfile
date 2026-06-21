# syntax=docker/dockerfile:1.7

# Catalog Service container image.
#
# Build context:
#   repo root
#
# Example:
#   docker build \
#     -f services/catalog-service/Dockerfile \
#     --build-arg SERVICE_VERSION=0.1.0 \
#     -t bfstore/catalog-service:0.1.0 \
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
COPY services/catalog-service/go.mod services/catalog-service/go.sum ./services/catalog-service/
COPY services/basket-service/go.mod ./services/basket-service/go.mod

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && \
    cd services/catalog-service && go mod download

# Copy only the source needed to build the catalog service.
COPY gen/ ./gen/
COPY pkg/ ./pkg/
COPY services/catalog-service/ ./services/catalog-service/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd services/catalog-service && \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/catalog-service \
      ./cmd/catalog-service

FROM gcr.io/distroless/static-debian${DEBIAN_VERSION}:nonroot AS runtime

ARG SERVICE_VERSION=0.1.0

LABEL org.opencontainers.image.title="bfstore catalog-service" \
      org.opencontainers.image.description="bfstore Catalog Service gRPC API" \
      org.opencontainers.image.version="${SERVICE_VERSION}" \
      org.opencontainers.image.source="https://github.com/mantrobuslawal/bfstore" \
      org.opencontainers.image.vendor="Borough Furniture Store" \
      org.opencontainers.image.licenses="Apache-2.0"

WORKDIR /

COPY --from=build --chown=nonroot:nonroot /out/catalog-service /catalog-service

USER nonroot:nonroot

EXPOSE 50051

ENTRYPOINT ["/catalog-service"]
