# Catalog Database Migrations

This directory contains catalog-service database migrations.

The catalog service owns the `bfstore_catalog` database. Only catalog-service
migrations should create, change, or drop objects in this schema.

## Tool

bfstore uses `golang-migrate` for service-owned database migrations.

Install the CLI with MySQL support:

```bash
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Confirm the CLI is available:

```bash
migrate -version
```

## Current migration files

The current catalog schema is represented by paired up/down migration files:

```text
db/catalog/migrations/
├── README.md
├── 000001_create_catalog_schema.up.sql
└── 000001_create_catalog_schema.down.sql
```

The initial schema creates the local catalog foundation, including:

```text
categories
products
product_variants
product_attribute_definitions
product_attribute_options
product_attribute_values
product_images
catalogue_outbox_events
```

## File naming

Use paired up/down migration files:

```text
000001_create_catalog_schema.up.sql
000001_create_catalog_schema.down.sql
000002_add_catalog_feature.up.sql
000002_add_catalog_feature.down.sql
```

Rules:

```text
up file applies the change
down file rolls back the change
up and down files must be reviewed together
do not edit migrations that have already been applied outside throwaway local development
new schema changes should get a new migration version
```

## Running migrations

From the repo root:

```bash
make catalog-db-migrate-up
make catalog-db-migrate-version
make catalog-db-migrate-down
```

For local throwaway development, rebuild MySQL and reapply catalog migrations:

```bash
make local-db-fresh-catalog
```

If the shared local reset flow should migrate both catalog and basket databases:

```bash
make local-db-fresh
```

## Makefile variables

```makefile
CATALOG_MIGRATIONS_PATH ?= db/catalog/migrations
CATALOG_DATABASE_URL ?= mysql://bfstore_catalog:bfstore_catalog_password@tcp(localhost:3306)/bfstore_catalog?multiStatements=true&parseTime=true
```

## Makefile targets

```makefile
.PHONY: catalog-db-migrate-up
catalog-db-migrate-up: ## Run catalog-service database migrations
	migrate -path $(CATALOG_MIGRATIONS_PATH) -database "$(CATALOG_DATABASE_URL)" up

.PHONY: catalog-db-migrate-down
catalog-db-migrate-down: ## Roll back one catalog-service database migration
	migrate -path $(CATALOG_MIGRATIONS_PATH) -database "$(CATALOG_DATABASE_URL)" down 1

.PHONY: catalog-db-migrate-version
catalog-db-migrate-version: ## Show catalog-service database migration version
	migrate -path $(CATALOG_MIGRATIONS_PATH) -database "$(CATALOG_DATABASE_URL)" version

.PHONY: catalog-db-migrate-force
catalog-db-migrate-force: ## Force catalog-service migration version. Usage: make catalog-db-migrate-force VERSION=1
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required. Example: make catalog-db-migrate-force VERSION=1"; \
		exit 1; \
	fi
	migrate -path $(CATALOG_MIGRATIONS_PATH) -database "$(CATALOG_DATABASE_URL)" force $(VERSION)
```

## Dirty migrations

If a migration fails, `golang-migrate` may mark the version as dirty.

For local development, prefer:

```text
read the migration error
fix the migration
reset the local database volume
rerun make local-db-fresh-catalog
```

Do not use `force` casually. It changes the migration version marker without
actually applying or rolling back schema changes.

## Boundary rule

Catalog migrations may create and change tables in:

```text
bfstore_catalog
```

They must not create, modify, or depend on other service databases.

Catalog may store:

```text
product identity
product names and descriptions
category taxonomy
product variants
category-scoped attribute definitions
product attribute values
catalog price snapshots
catalog outbox events
```

Catalog must not store:

```text
basket contents
stock reservation state
orders
payments
shipments
search ranking state
recommendation model state
```

## Seed data

Seed data lives outside the migration history:

```text
db/catalog/seeds/001_seed_borough_products.sql
```

Seed scripts are local/demo convenience artefacts. Do not hide schema changes in
seed files.

Local sequence:

```bash
make local-db-fresh-catalog
mysql -h 127.0.0.1 -P 3306 -ubfstore_catalog -pbfstore_catalog_password bfstore_catalog < db/catalog/seeds/001_seed_borough_products.sql
```

## Testing expectations

Catalog migrations should be validated by tests or local checks for:

```text
migrations apply cleanly
migrations roll back cleanly
catalog tables exist after migration up
catalog tables are removed after migration down
products can be inserted and queried
product variants can be validated by product_id and variant_id
inactive products and variants can be represented
money is stored in minor units
catalog tables do not reference other service schemas
```

## ValidateProductVariant readiness

The basket-service `AddItem` flow depends on catalog being able to validate:

```text
product_id
variant_id
product/variant ownership
product status
variant status
variant price
currency
```

The migration must therefore preserve indexed access to:

```text
products.product_id
product_variants.variant_id
product_variants.product_id
```

Supporting indexes:

```text
UNIQUE KEY uk_products_product_id (product_id)
UNIQUE KEY uk_product_variants_variant_id (variant_id)
KEY idx_product_variants_product_id (product_id)
```

## Practical rule

Keep catalog migrations boring, reversible, and service-owned.
