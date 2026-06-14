# bfstore Makefile
#
# Purpose:
# Provide a consistent local developer workflow for building, testing,
# generating Protobuf code, running the local development stack, and managing
# local service-owned databases.
#
# Usage:
#   make help
#   make proto-lint
#   make proto-generate
#   make up
#   make down
#   make local-db-fresh

SHELL := /bin/bash

PROJECT_NAME := bfstore
COMPOSE_FILE := docker-compose.yaml
DOCKER_COMPOSE := docker compose -p $(PROJECT_NAME) -f $(COMPOSE_FILE)

MYSQL_ROOT_PASSWORD := bfstore_root_password
MYSQL_INIT_PATH ?= deploy/local/mysql/init
MYSQL_VOLUME := $(PROJECT_NAME)_bfstore_mysql_data

BASKET_MIGRATIONS_PATH ?= db/basket/migrations
BASKET_DATABASE_URL ?= mysql://bfstore_basket:bfstore_basket_password@tcp(localhost:3306)/bfstore_basket?multiStatements=true&parseTime=true

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "$(PROJECT_NAME) developer commands"
	@echo "--------------------------------"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-32s %s\n", $$1, $$2}'
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

.PHONY: proto-generate-catalog
proto-generate-catalog: ## Generate Go code for catalog Protobuf contracts only
	buf generate --path proto/bfstore/catalog/v1

.PHONY: up
up: ## Start local dependencies and services
	$(DOCKER_COMPOSE) up -d

.PHONY: down
down: ## Stop local containers
	$(DOCKER_COMPOSE) down

.PHONY: down-volumes
down-volumes: ## Stop containers and remove local Compose volumes
	$(DOCKER_COMPOSE) down -v

.PHONY: logs
logs: ## Tail local container logs
	$(DOCKER_COMPOSE) logs -f

.PHONY: ps
ps: ## Show local container status
	$(DOCKER_COMPOSE) ps

.PHONY: compose-config
compose-config: ## Render and validate Docker Compose configuration
	$(DOCKER_COMPOSE) config

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
catalog-test: ## Run catalog-service unit tests
	cd services/catalog-service && go test ./...

.PHONY: catalog-integration-test
catalog-integration-test: ## Run catalog-service integration tests
	cd services/catalog-service && BFSTORE_RUN_INTEGRATION_TESTS=true go test ./test/integration/...

.PHONY: catalog-build
catalog-build: ## Build catalog-service locally
	cd services/catalog-service && go build -o bin/catalog-service ./cmd/catalog-service

.PHONY: catalog-docker-build
catalog-docker-build: ## Build catalog-service container image
	docker build -f services/catalog-service/Dockerfile -t bfstore/catalog-service:local .

.PHONY: catalog-run
catalog-run: ## Start catalog-service locally and enable gRPC reflection
	cd services/catalog-service && GRPC_REFLECTION_ENABLED=true go run ./cmd/catalog-service

.PHONY: catalog-run-telemetry
catalog-run-telemetry: ## Start catalog-service locally with OpenTelemetry enabled
	cd services/catalog-service && \
		TELEMETRY_ENABLED=true \
		OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
		OTEL_EXPORTER_OTLP_INSECURE=true \
		GRPC_REFLECTION_ENABLED=true \
		go run ./cmd/catalog-service

.PHONY: catalog-grpc-list
catalog-grpc-list: ## List catalog-service gRPC endpoints
	grpcurl -plaintext localhost:50051 list

.PHONY: catalog-health
catalog-health: ## Check catalog-service/server health
	grpcurl -plaintext -d '{}' localhost:50051 grpc.health.v1.Health/Check

.PHONY: catalog-list-products
catalog-list-products: ## List catalog-service products
	grpcurl -plaintext -d '{"page": {"page_size": 5}}' localhost:50051 bfstore.catalog.v1.CatalogService/ListProducts

.PHONY: catalog-list-categories
catalog-list-categories: ## List catalog-service product categories
	grpcurl -plaintext -d '{"page":{"page_size":5}}' localhost:50051 bfstore.catalog.v1.CatalogService/ListCategories

.PHONY: catalog-list-products-with-correlation
catalog-list-products-with-correlation: ## Test correlation ID propagation against catalog-service
	grpcurl -plaintext \
		-H 'x-correlation-id: local-dev-123' \
		-d '{"page": {"page_size":5}}' \
		localhost:50051 \
		bfstore.catalog.v1.CatalogService/ListProducts

.PHONY: catalog-load
catalog-load: ## Generate local catalog-service request traffic
	REQUESTS=100 SLEEP_SECONDS=0.05 ./scripts/local/catalog-load.sh

.PHONY: otel-up
otel-up: ## Start OpenTelemetry Collector
	$(DOCKER_COMPOSE) up -d otel-collector

.PHONY: otel-logs
otel-logs: ## Follow OpenTelemetry Collector logs
	$(DOCKER_COMPOSE) logs -f otel-collector

.PHONY: metrics-up
metrics-up: ## Start metrics dependencies
	$(DOCKER_COMPOSE) up -d otel-collector prometheus

.PHONY: metrics-logs
metrics-logs: ## Follow metrics dependency logs
	$(DOCKER_COMPOSE) logs -f otel-collector prometheus

.PHONY: observability-up
observability-up: ## Start local observability stack
	$(DOCKER_COMPOSE) up -d otel-collector jaeger prometheus grafana

.PHONY: observability-logs
observability-logs: ## Follow local observability stack logs
	$(DOCKER_COMPOSE) logs -f otel-collector jaeger prometheus grafana

.PHONY: local-db-reset
local-db-reset: ## Stop containers and remove the local MySQL data volume
	$(DOCKER_COMPOSE) down
	-docker volume rm $(MYSQL_VOLUME)

.PHONY: local-db-up
local-db-up: ## Start local MySQL only
	$(DOCKER_COMPOSE) up -d mysql

.PHONY: local-db-wait
local-db-wait: ## Wait for local MySQL to accept authenticated root queries
	@echo "Waiting for MySQL to accept root connections..."
	@until $(DOCKER_COMPOSE) exec -T mysql mysql \
		-uroot \
		-p$(MYSQL_ROOT_PASSWORD) \
		-e "SELECT 1;" >/dev/null 2>&1; do \
		sleep 2; \
	done
	@echo "MySQL is ready for authenticated queries."

.PHONY: local-db-check-root
local-db-check-root: ## Check root login works with configured local MySQL password
	$(DOCKER_COMPOSE) exec -T mysql mysql -uroot -p$(MYSQL_ROOT_PASSWORD) -e "SELECT VERSION();"

.PHONY: local-db-bootstrap
local-db-bootstrap: ## Create local service databases, users, and grants
	$(DOCKER_COMPOSE) exec -T mysql mysql -uroot -p$(MYSQL_ROOT_PASSWORD) < $(MYSQL_INIT_PATH)/001-create-service-databases.sql
	$(DOCKER_COMPOSE) exec -T mysql mysql -uroot -p$(MYSQL_ROOT_PASSWORD) < $(MYSQL_INIT_PATH)/002-create-service-users.sql

.PHONY: local-db-check-basket-user
local-db-check-basket-user: ## Check basket database user and grants
	$(DOCKER_COMPOSE) exec -T mysql mysql -uroot -p$(MYSQL_ROOT_PASSWORD) -e "SELECT user, host FROM mysql.user WHERE user = 'bfstore_basket';"
	$(DOCKER_COMPOSE) exec -T mysql mysql -uroot -p$(MYSQL_ROOT_PASSWORD) -e "SHOW GRANTS FOR 'bfstore_basket'@'%';"

.PHONY: basket-db-migrate-up
basket-db-migrate-up: ## Run basket-service database migrations
	migrate -path $(BASKET_MIGRATIONS_PATH) -database "$(BASKET_DATABASE_URL)" up

.PHONY: basket-db-migrate-down
basket-db-migrate-down: ## Roll back one basket-service database migration
	migrate -path $(BASKET_MIGRATIONS_PATH) -database "$(BASKET_DATABASE_URL)" down 1

.PHONY: basket-db-migrate-version
basket-db-migrate-version: ## Show basket-service database migration version
	migrate -path $(BASKET_MIGRATIONS_PATH) -database "$(BASKET_DATABASE_URL)" version

.PHONY: basket-db-migrate-force
basket-db-migrate-force: ## Force basket-service migration version. Usage: make basket-db-migrate-force VERSION=1
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required. Example: make basket-db-migrate-force VERSION=1"; \
		exit 1; \
	fi
	migrate -path $(BASKET_MIGRATIONS_PATH) -database "$(BASKET_DATABASE_URL)" force $(VERSION)

.PHONY: local-db-fresh
local-db-fresh: ## Rebuild local MySQL from scratch and run basket migrations
	$(MAKE) local-db-reset
	$(MAKE) local-db-up
	$(MAKE) local-db-wait
	$(MAKE) local-db-check-root
	$(MAKE) local-db-bootstrap
	$(MAKE) local-db-check-basket-user
	$(MAKE) basket-db-migrate-up
	$(MAKE) basket-db-migrate-version
