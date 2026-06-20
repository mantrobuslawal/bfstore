# Catalog Database

This directory contains local database artefacts for catalog-service.</br>
[The catalog seed manifest is located here.](https://github.com/mantrobuslawal/bfstore/tree/basket-service-v1/docs/data/catalog-large-seed-manifest.md)

Catalog Service owns the product catalog domain for bfstore.

It is responsible for:

```text
products
categories
product variants
category-scoped product attributes
product imagery
catalog outbox events
```

It is not responsible for:

```text
stock levels
basket state
orders
payments
shipping
search indexes
recommendation models
```

## Directory layout

```text
db/catalog/
├── README.md
├── migrations/
│   ├── README.md
│   ├── 000001_create_catalog_schema.up.sql
│   └── 000001_create_catalog_schema.down.sql
└── seeds/
    └── 001_seed_borough_products.sql
```

## Migration tool

Catalog Service uses the same `golang-migrate` approach as Basket Service.

Install:

```bash
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Run catalog migrations from the repo root:

```bash
make catalog-db-migrate-up
make catalog-db-migrate-version
```

Roll back one migration:

```bash
make catalog-db-migrate-down
```

Rebuild the local catalog database from scratch:

```bash
make local-db-fresh-catalog
```

## Design summary

bfstore sells varied developer-themed homeware.

Different product categories need different attributes.

Examples:

```text
lamps need bulb type and max wattage
wall art needs size and material
lockboxes need lock type and security rating
rugs need shape and pile height
soft furnishings need fabric type and care instructions
```

The catalog schema uses:

```text
relational product core
category-scoped attribute definitions
typed product attribute values
variant support
controlled attribute options
```

This avoids:

```text
one giant products table with hundreds of nullable columns
uncontrolled schemaless JSON blobs
mixing stock/order/payment/basket data into the catalog database
```

## Initial schema migration

Initial schema migration:

```text
db/catalog/migrations/000001_create_catalog_schema.up.sql
```

Rollback migration:

```text
db/catalog/migrations/000001_create_catalog_schema.down.sql
```

The schema includes:

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

## Seed data

Seed data:

```text
db/catalog/seeds/001_seed_borough_products.sql
```

Example products:

```text
Gopher Desk Lamp
Gopher Cushion Set
Rob Pike Wall Tapestry
Rivest Super-Secure Lockbox
Dijkstra Pathfinding Rug
Grace Hopper Debugging Blanket
```

Seed data is intentionally memorable, but it exercises serious catalog modelling
concerns.

## ValidateProductVariant dependency

Basket Service now relies on Catalog Service to validate product and variant
pairs before adding items to a basket.

The catalog schema must support efficient lookup by:

```text
products.product_id
product_variants.variant_id
product_variants.product_id
```

Catalog remains the owner of product truth. Basket stores a temporary basket-line
snapshot only.

## Client-facing engineering evidence

This database foundation demonstrates:

```text
service-owned data design
least-privilege database thinking
repeatable migrations
realistic seed data
catalog modelling for varied product types
event-driven outbox readiness
cross-service validation without shared database ownership
```

## Practical rule

Catalog owns product truth.

Basket owns customer intent.

Order revalidates before payment.


