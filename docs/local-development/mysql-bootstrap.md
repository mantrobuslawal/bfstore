# Local MySQL Bootstrap

This document explains how local MySQL is bootstrapped for bfstore.

## Goal

A developer should be able to rebuild local MySQL from a fresh Docker volume without manually logging into the MySQL container.

Target command:

```bash
make local-db-fresh
```

## Bootstrap files

Recommended path:

```text
deploy/local/mysql/init/
```

Files:

```text
001-create-service-databases.sql
002-create-service-users.sql
```

These files are mounted to:

```text
/docker-entrypoint-initdb.d
```

inside the MySQL container.

## What bootstrap creates

The bootstrap creates service-owned databases:

```text
bfstore_catalog
bfstore_basket
bfstore_inventory
bfstore_order
bfstore_payment
```

It also creates service users:

```text
bfstore_catalog
bfstore_basket
bfstore_inventory
bfstore_order
bfstore_payment
```

Each user only gets access to its own database.

## What bootstrap does not create

Bootstrap should not create long-term service tables such as:

```text
products
variants
baskets
basket_items
orders
payments
```

Those belong in service migrations.

## Why init scripts sometimes appear not to run

MySQL init scripts only run when the MySQL data directory is first initialised.

If the Docker volume already exists, the scripts are skipped.

This will not rerun init scripts:

```bash
docker compose restart mysql
```

This should rebuild from scratch when wired correctly:

```bash
make local-db-fresh
```

## Docker Compose mount

The MySQL service should include:

```yaml
volumes:
  - bfstore-mysql-data:/var/lib/mysql
  - ./deploy/local/mysql/init:/docker-entrypoint-initdb.d:ro
```

## Local users and Docker networking

Users are created as:

```sql
'bfstore_basket'@'%'
```

rather than:

```sql
'bfstore_basket'@'localhost'
```

because Docker-host connections may appear to MySQL as a bridge network IP rather than localhost.

Access is still scoped:

```sql
GRANT ALL PRIVILEGES ON bfstore_basket.* TO 'bfstore_basket'@'%';
```

## Troubleshooting

### Access denied for user

Example:

```text
Error 1045: Access denied for user 'bfstore_basket'@'172.22.0.1'
```

Likely cause:

```text
user was created only for localhost
init scripts did not run
old MySQL volume is still being reused
wrong password in DATABASE_URL
```

Fix:

```bash
make local-db-fresh
```

### Init files changed but database did not

The old volume probably still exists.

Run:

```bash
make local-db-fresh
```

### Wrong volume removed

Compose volume names depend on the Compose project name.

Check:

```bash
docker volume ls
```

Make sure `MYSQL_VOLUME` matches the actual MySQL data volume.

## Practical rule

```text
Bootstrap creates access.
Migrations create schema.
Seeds create local data.
```

Keep it boring where production matters.
