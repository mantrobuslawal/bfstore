# Catalog Seed Data

This directory contains local/demo seed data for the bfstore catalog-service.

Seed data is used to make local development, manual testing, smoke checks, demos,
and portfolio walkthroughs easier. It is not part of the schema migration
history.

## Directory

```text
db/catalog/seeds/
├── README.md
└── 002_seed_borough_catalog_large.sql
```

## Current seed file

```text
002_seed_borough_catalog_large.sql
```

This seed file creates:

```text
10 categories
200 products
400 product variants
```

It is designed to support:

```text
catalog browsing
product listing
category filtering
product/variant validation
basket-service AddItem testing
portfolio demonstrations
```

## Identifier strategy

The seed data follows the current bfstore identifier approach.

### Categories

Category IDs are mnemonic and human-searchable:

```text
cat_lang_go
cat_lang_python
cat_lang_java
cat_lang_rust
cat_topic_security
cat_topic_algorithms
cat_room_office
cat_room_lounge
cat_home_textiles
cat_storage_tools
```

This keeps debugging and querying easier when working with category data.

### Products

Product IDs are opaque external identifiers:

```text
prod_<26 Crockford Base32 characters>
```

Example:

```text
prod_VVPKSY9WDEAXNA0Q3N1XZS172Q
```

### Variants

Variant IDs are opaque external identifiers:

```text
var_<26 Crockford Base32 characters>
```

Example:

```text
var_MWNZCK7SK1PKRW0PN2PYFDSF0H
```

## Relationship to migrations

Migrations create and change database structure.

Seed files insert example/demo data.

Keep those responsibilities separate:

```text
db/catalog/migrations/ -> schema
db/catalog/seeds/      -> local/demo data
```

Do not put demo seed data into migration files.

## Prerequisites

Before running catalog seed data, the catalog database must exist and catalog
migrations must have been applied.

From the repo root:

```bash
make local-db-fresh-catalog
```

Or, if MySQL is already running:

```bash
make catalog-db-migrate-up
make catalog-db-migrate-version
```

## Running the seed

From the repo root:

```bash
mysql \
  -h 127.0.0.1 \
  -P 3306 \
  -ubfstore_catalog \
  -pbfstore_catalog_password \
  bfstore_catalog < db/catalog/seeds/002_seed_borough_catalog_large.sql
```

One-line version:

```bash
mysql -h 127.0.0.1 -P 3306 -ubfstore_catalog -pbfstore_catalog_password bfstore_catalog < db/catalog/seeds/002_seed_borough_catalog_large.sql
```

## Verifying seed data

Check category count:

```sql
SELECT COUNT(*) AS category_count
FROM categories
WHERE category_id LIKE 'cat_%';
```

Check product count:

```sql
SELECT COUNT(*) AS product_count
FROM products
WHERE product_id LIKE 'prod_%';
```

Check variant count:

```sql
SELECT COUNT(*) AS variant_count
FROM product_variants
WHERE variant_id LIKE 'var_%';
```

Expected result:

```text
categories: 10
products:   200
variants:   400
```

## ValidateProductVariant smoke check

The large catalog seed includes active product/variant pairs that can be used by
basket-service.

Example:

```bash
make catalog-validate-product-variant \
  PRODUCT_ID=prod_VVPKSY9WDEAXNA0Q3N1XZS172Q \
  VARIANT_ID=var_MWNZCK7SK1PKRW0PN2PYFDSF0H
```

Expected behaviour:

```text
Catalog Service confirms the product exists
Catalog Service confirms the variant exists
Catalog Service confirms the variant belongs to the product
Catalog Service returns product name, variant name, unit price, currency, and sellable status
```

Basket Service can then store that response as a temporary basket-line snapshot.

## Seed manifest

The human-readable manifest for the large seed lives in:

```text
docs/data/catalog-large-seed-manifest.md
```

Use the manifest when you need:

```text
category summary
example product IDs
example variant IDs
demo smoke-check IDs
seed data counts
```

## Seed data rules

Use seed files for:

```text
local development data
manual testing data
demo data
portfolio walkthroughs
repeatable smoke checks
```

Do not use seed files for:

```text
schema changes
database constraints
production data changes
cross-service data ownership
runtime application state
```

## Service ownership rule

Catalog seed data may insert into catalog-owned tables only.

Allowed:

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

Not allowed:

```text
basket tables
inventory tables
order tables
payment tables
shipping tables
notification tables
search indexes
recommendation data stores
```

Basket and other services must interact with catalog data through service APIs,
not direct cross-schema database access.

## Practical rule

Migrations create structure.

Seeds create local/demo data.

Catalog owns product truth.

Basket owns customer intent.

Order revalidates before payment.

Keep it boring where production matters.
