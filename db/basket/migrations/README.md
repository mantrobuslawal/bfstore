# Basket Database Migrations

This directory contains basket-service database migrations.

The basket service owns the `bfstore_basket` database.

## Tool

bfstore uses `golang-migrate` for service-owned database migrations starting with the basket service.

Install the CLI with MySQL support:

```bash
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## File naming

Use paired up/down migration files:

```text
000001_create_basket_schema.up.sql
000001_create_basket_schema.down.sql
```

Rules:

```text
up file applies the change
down file rolls back the change
up and down files must be reviewed together
do not edit migrations that have already been applied outside throwaway local development
```

## Running migrations

From the repo root:

```bash
make basket-db-migrate-up
make basket-db-migrate-version
make basket-db-migrate-down
```

## Dirty migrations

If a migration fails, `golang-migrate` may mark the version as dirty.

For local development, prefer:

```text
read the error
fix the migration
reset the local database volume
rerun make local-db-fresh
```

## Boundary rule

Basket migrations may create and change tables in:

```text
bfstore_basket
```

They must not create, modify, or depend on other service databases.

Keep it boring where production matters.
