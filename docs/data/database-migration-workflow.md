# Database Migration Workflow

bfstore uses service-owned databases and repeatable local migrations.

The current local migration standard is `golang-migrate`.

## Services currently using golang-migrate locally

```text
catalog-service -> db/catalog/migrations -> bfstore_catalog
basket-service  -> db/basket/migrations  -> bfstore_basket
```

## Install migrate CLI

Install the CLI with MySQL support:

```bash
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Confirm installation:

```bash
migrate -version
```

## Local MySQL bootstrap

The local MySQL container is started with Docker Compose.

Bootstrap SQL creates service-owned databases and service users:

```text
deploy/local/mysql/init/001-create-service-databases.sql
deploy/local/mysql/init/002-create-service-users.sql
```

The bootstrap scripts should create at least:

```text
bfstore_catalog
bfstore_basket
```

and local users such as:

```text
bfstore_catalog
bfstore_basket
```

Longer term, split users into:

```text
bootstrap/root user
service migrator user
service runtime app user
```

For now, keep the local workflow simple and visible.

## Makefile variables

Recommended variables:

```makefile
CATALOG_MIGRATIONS_PATH ?= db/catalog/migrations
CATALOG_DATABASE_URL ?= mysql://bfstore_catalog:bfstore_catalog_password@tcp(localhost:3306)/bfstore_catalog?multiStatements=true&parseTime=true

BASKET_MIGRATIONS_PATH ?= db/basket/migrations
BASKET_DATABASE_URL ?= mysql://bfstore_basket:bfstore_basket_password@tcp(localhost:3306)/bfstore_basket?multiStatements=true&parseTime=true
```

## Catalog migration commands

```bash
make catalog-db-migrate-up
make catalog-db-migrate-version
make catalog-db-migrate-down
```

Use force only when recovering a local dirty migration state:

```bash
make catalog-db-migrate-force VERSION=1
```

## Basket migration commands

```bash
make basket-db-migrate-up
make basket-db-migrate-version
make basket-db-migrate-down
```

Use force only when recovering a local dirty migration state:

```bash
make basket-db-migrate-force VERSION=1
```

## Fresh local database flows

Rebuild MySQL and migrate only catalog:

```bash
make local-db-fresh-catalog
```

Rebuild MySQL and migrate only basket:

```bash
make local-db-fresh-basket
```

Rebuild MySQL and migrate all currently implemented service databases:

```bash
make local-db-fresh
```

Recommended full local flow:

```text
local-db-reset
local-db-up
local-db-wait
local-db-check-root
local-db-bootstrap
local-db-check-catalog-user
catalog-db-migrate-up
catalog-db-migrate-version
local-db-check-basket-user
basket-db-migrate-up
basket-db-migrate-version
```

## Migration safety rules

```text
do not edit committed migrations after they have been applied outside throwaway local development
always provide paired up/down files
keep service schemas isolated
do not reference another service database
do not store service runtime state in another service database
do not use FLOAT or DOUBLE for money
prefer additive migrations for early evolution
document destructive migrations before applying them
```

## Dirty migrations

If `golang-migrate` reports a dirty version:

```text
read the migration error
fix the SQL
reset local database if this is throwaway development
rerun the relevant fresh migration flow
```

Avoid `force` unless you understand exactly what schema state the database is in.

## Catalog-service rules

Catalog migrations may change:

```text
bfstore_catalog.categories
bfstore_catalog.products
bfstore_catalog.product_variants
bfstore_catalog.product_attribute_definitions
bfstore_catalog.product_attribute_options
bfstore_catalog.product_attribute_values
bfstore_catalog.product_images
bfstore_catalog.catalogue_outbox_events
```

Catalog migrations must not change:

```text
bfstore_basket.*
bfstore_inventory.*
bfstore_order.*
bfstore_payment.*
bfstore_shipping.*
bfstore_notification.*
```

## Basket-service rules

Basket migrations may change:

```text
bfstore_basket.baskets
bfstore_basket.basket_items
```

Basket migrations must not reference catalog tables directly.

Product and variant validation happens through Catalog Service APIs, not through
cross-schema SQL joins.

## Practical rule

Migrations are part of the service contract.

