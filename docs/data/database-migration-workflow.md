# Database Migration Workflow

This document describes the local bfstore database lifecycle.

The goal is a clean, automated, repeatable workflow that can rebuild a fresh local database from an empty Docker volume.

## Target command

```bash
make local-db-fresh
```

Expected result:

```text
old MySQL volume removed
fresh MySQL starts
service databases are created
service users are created
permissions are granted
basket migrations run
schema_migrations table exists
baskets table exists
basket_items table exists
```

No manual login to the MySQL container should be required for the normal path.

## Three database layers

### Layer 1: MySQL bootstrap

Bootstrap creates:

```text
databases
users
grants
```

Recommended path:

```text
deploy/local/mysql/init/
```

Files:

```text
001-create-service-databases.sql
002-create-service-users.sql
```

These are mounted into:

```text
/docker-entrypoint-initdb.d
```

They run automatically when the MySQL data directory is initialised for the first time.

### Layer 2: service migrations

Migrations create and evolve application schema.

Recommended paths:

```text
db/catalog/migrations/
db/basket/migrations/
```

Starting with basket-service, bfstore uses:

```text
golang-migrate
```

Service schema belongs in migrations, not in MySQL bootstrap files.

### Layer 3: local seed data

Seed scripts insert local/demo data.

Recommended paths:

```text
db/catalog/seeds/
db/basket/seeds/
```

## Why MySQL init does not own service schema

MySQL init scripts only run for a fresh data directory.

If long-term service schema evolution is placed in MySQL init scripts, existing volumes skip new scripts and local environments drift.

Therefore:

```text
mysql init = databases, users, grants
migrations = service tables and schema changes
seeds = local/demo data
```

## Local MySQL users

For Docker local development, service users are created with host `%`.

Example:

```sql
CREATE USER IF NOT EXISTS 'bfstore_basket'@'%'
  IDENTIFIED BY 'bfstore_basket_password';

GRANT ALL PRIVILEGES ON bfstore_basket.*
  TO 'bfstore_basket'@'%';
```

The user remains scoped to its own database.

## golang-migrate

Install:

```bash
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Basket database URL:

```text
mysql://bfstore_basket:bfstore_basket_password@tcp(localhost:3306)/bfstore_basket?multiStatements=true&parseTime=true
```

`multiStatements=true` is needed because migration files may contain more than one SQL statement.

## Make targets

Recommended targets:

```bash
make local-db-reset
make local-db-up
make local-db-wait
make basket-db-migrate-up
make basket-db-migrate-version
make local-db-fresh
```

## Dirty migrations

If a migration fails part-way through, `golang-migrate` may mark the migration version as dirty.

Do not casually run `force`.

For local development, prefer:

```text
read the error
fix the migration
reset the local database volume
rerun make local-db-fresh
```

## Revisit Flyway later

Current decision:

```text
basket service phase:
  use golang-migrate

after full commerce slice:
  revisit Flyway
```

## Verification

After:

```bash
make local-db-fresh
```

Expected tables:

```text
baskets
basket_items
schema_migrations
```

## Practical rule

```text
A database workflow is not complete until a fresh volume can rebuild automatically.
```

Keep it boring where production matters.
