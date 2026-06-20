# Local MySQL Bootstrap

This document explains how local MySQL is bootstrapped for bfstore service-owned
databases.

## Bootstrap files

Local bootstrap SQL lives in:

```text
deploy/local/mysql/init/
├── 001-create-service-databases.sql
└── 002-create-service-users.sql
```

Docker Compose mounts this directory into the MySQL container for first-time
initialisation.

## Service-owned databases

The local bootstrap should create service databases such as:

```text
bfstore_catalog
bfstore_basket
```

Future service databases may include:

```text
bfstore_inventory
bfstore_order
bfstore_payment
bfstore_shipping
bfstore_notification
```

## Local service users

The local bootstrap should create users such as:

```text
bfstore_catalog
bfstore_basket
```

For the current local developer workflow, these users can be used by
`golang-migrate`.

Longer term, split local and cloud permissions into:

```text
root/bootstrap user
migrator user
runtime app user
```

Example future direction:

```text
bfstore_catalog_migrator -> schema changes
bfstore_catalog_app      -> SELECT, INSERT, UPDATE, DELETE only

bfstore_basket_migrator  -> schema changes
bfstore_basket_app       -> SELECT, INSERT, UPDATE, DELETE only
```

## Local Makefile flow

Start MySQL only:

```bash
make local-db-up
```

Wait for authenticated root queries:

```bash
make local-db-wait
```

Bootstrap databases and users:

```bash
make local-db-bootstrap
```

Check catalog user:

```bash
make local-db-check-catalog-user
```

Check basket user:

```bash
make local-db-check-basket-user
```

## Catalog migration flow

```bash
make catalog-db-migrate-up
make catalog-db-migrate-version
```

Rollback one migration:

```bash
make catalog-db-migrate-down
```

Fresh local catalog database flow:

```bash
make local-db-fresh-catalog
```

## Basket migration flow

```bash
make basket-db-migrate-up
make basket-db-migrate-version
```

Rollback one migration:

```bash
make basket-db-migrate-down
```

Fresh local basket database flow:

```bash
make local-db-fresh-basket
```

## Full local rebuild flow

For a full local reset:

```bash
make local-db-fresh
```

This should eventually:

```text
remove the local MySQL volume
start MySQL
wait for authenticated root access
run bootstrap SQL
check service users
run catalog migrations
run basket migrations
show migration versions
```

## Important local development warning

MySQL only runs `/docker-entrypoint-initdb.d` scripts when the data directory is
empty.

If you change bootstrap SQL and still have an existing MySQL volume, the changes
will not be replayed automatically.

Use:

```bash
make local-db-reset
```

or:

```bash
docker compose -p bfstore -f docker-compose.yaml down -v
```

for throwaway local rebuilds.

## Practical rule

Bootstrap creates databases and users.

Migrations create service schema objects.

Seed files create demo data.

Keep those three jobs separate.
