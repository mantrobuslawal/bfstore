# bfstore Makefile
#
# Purpose:
# Provide a consistent local developer workflow for building, testing,
# generating Protobuf code, and running the local development stack.
#
# Usage:
#   make help
#   make proto-lint
#   make proto-generate
#   make up
#   make down

SHELL := /bin/bash

PROJECT_NAME := bfstore
COMPOSE_FILE := docker-compose.yaml

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "$(PROJECT_NAME) developer commands"
	@echo "--------------------------------"
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-24s %s\n", $$1, $$2}'
	@echo ""

.PHONY: proto-lint
proto-lint: ## Lint Protobuf contracts with Buf
	buf lint

.PHONY: proto-breaking
proto-breaking: ## Run Buf breaking-change checks
	buf breaking

.PHONY: proto-generate
proto-generate: ## Generate Go code from Protobuf contracts
	buf generate

.PHONY: proto
proto: proto-lint proto-generate ## Lint and generate Protobuf contracts

.PHONY: up
up: ## Start local dependencies and services
	docker compose -f $(COMPOSE_FILE) up -d

.PHONY: down
down: ## Stop local containers
	docker compose -f $(COMPOSE_FILE) down

.PHONY: down-volumes
down-volumes: ## Stop containers and remove local volumes
	docker compose -f $(COMPOSE_FILE) down -v

.PHONY: logs
logs: ## Tail local container logs
	docker compose -f $(COMPOSE_FILE) logs -f

.PHONY: ps
ps: ## Show local container status
	docker compose -f $(COMPOSE_FILE) ps

.PHONY: test
test: ## Run Go tests
	go test ./...

.PHONY: tidy
tidy: ## Tidy Go modules
	go mod tidy

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: check
check: fmt tidy proto-lint test ## Run local quality checks

.PHONY: clean
clean: ## Remove generated build artefacts
	rm -rf gen

.PHONY: catalog-test
catalog-test: ## Run Catalogue Service unit tests
	cd services/catalog-service && go test ./...

.PHONY: catalog-integration-test
catalog-integration-test: ## Run Catalogue Service integration tests
	cd services/catalog-service && BFSTORE_RUN_INTEGRATION_TESTS=true go test ./test/integration/...

.PHONY: catalog-build
catalog-build: ## Build Catalogue Service locally
	cd services/catalog-service && go build -o bin/catalog-service ./cmd/catalog-service

.PHONY: catalog-docker-build
catalog-docker-build: ## Build Catalogue Service container image
	docker build -f services/catalog-service/Dockerfile -t bfstore/catalog-service:local .

.PHONY: proto-generate-catalog
proto-generate-catalog: ## Generate Go code for Catalogue Protobuf contracts only
	buf generate --path proto/bfstore/catalog/v1

.PHONY: catalog-run
catalog-run: ## Start catalog service locally and enable gRPC reflection
	cd services/catalog-service && GRPC_REFLECTION_ENABLED=true go run ./cmd/catalog-service

.PHONY: catalog-grpc-list
catalog-grpc-list: ## List server endpoints
	grpcurl -plaintext localhost:50051 list

.PHONY: catalog-health
catalog-health: ## Check overall server health
	grpcurl -plaintext -d '{}' localhost:50051 grpc.health.v1.Health/Check 

.PHONY: catalog-list-products
catalog-list-products: ## List catalog service products
	grpcurl -plaintext -d '{"page": {"page_size": 5}}' localhost:50051 bfstore.catalog.v1.CatalogService/ListProducts

.PHONY: catalog-list-categories
catalog-list-categories: ## List catalog service product categories
	grpcurl -plaintext -d '{"page":{"page_size":5}}' localhost:50051 bfstore.catalog.v1.CatalogService/ListCategories

.PHONY: catalog-list-products-with-correlation
catalog-list-products-with-correlation: ## Test correlation id propagation
	grpcurl -plaintext \
	   -H 'x-correlation-id: local-dev-123' \
	   -d '{"page": {"page_size":5}}' \
	   localhost:50051 \
	   bfstore.catalog.v1.CatalogService/ListProducts

.PHONY: otel-up
otel-up:   ## Start docker compose otel-collector
	docker compose -f ${COMPOSE_FILE} up -d otel-collector

.PHONY: otel-logs
otel-logs:	## Follow docker compose otel-collector logs
	docker compose -f $(COMPOSE_FILE) logs -f otel-collector

.PHONY: catalog-run-telemetry
catalog-run-telemetry:
	cd services/catalog-service && \
	   TELEMETRY_ENABLED=true \
	   OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
	   OTEL_EXPORTER_OTLP_INSECURE=true \
	   GRPC_REFLECTION_ENABLED=true \
	   go run ./cmd/catalog-service

.PHONY: metrics-up
metrics-up:
	docker compose -f $(COMPOSE_FILE) up -d otel-collector prometheus

.PHONY: metrics-logs
metrics-logs:
	docker compose -f $(COMPOSE_FILE) logs -f otel-collector prometheus

.PHONY: observability-up
observability-up:
	docker compose -f $(COMPOSE_FILE) up -d otel-collector jaeger prometheus grafana

.PHONY: observability-logs
observability-logs:
	docker compose -f $(COMPOSE_FILE) logs -f otel-collector jaeger prometheus grafana

.PHONY: catalog-load
catalog-load: ## Load test catalog service
	REQUESTS=100 SLEEP_SECONDS=0.05 ./scripts/local/catalog-load.sh 





