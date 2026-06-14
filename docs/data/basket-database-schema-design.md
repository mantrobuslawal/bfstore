# Basket Database Schema Design

## Purpose

This document outlines the first basket-service MySQL schema.

The schema should support:

```text
basket creation
adding items
updating item quantities
removing items
clearing baskets
retrieving basket state
```

It should prepare for future checkout/order integration without becoming an order database.

## Migration tool

When the basket service starts, bfstore will introduce:

```text
golang-migrate
```

as the migration standard for service-owned database migrations.

Future migration path:

```text
basket service phase:
  use golang-migrate

after full commerce slice:
  revisit Flyway
```

Full commerce slice includes:

```text
catalog
basket
order placement
payment
inventory movement/reservation where needed
```

## Proposed migration paths

Recommended folder:

```text
db/basket/migrations/
```

Example files:

```text
000001_create_basket_schema.up.sql
000001_create_basket_schema.down.sql
```

Later:

```text
000002_add_basket_expiry.up.sql
000002_add_basket_expiry.down.sql
```

## Tables

Recommended first tables:

```text
baskets
basket_items
```

## Table: baskets

Purpose:

```text
stores basket identity and lifecycle state
```

Suggested SQL:

```sql
CREATE TABLE baskets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  basket_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  currency_code CHAR(3) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (id),
  UNIQUE KEY uk_baskets_basket_id (basket_id),
  KEY idx_baskets_status (status),
  KEY idx_baskets_updated_at (updated_at)
);
```

Recommended status values:

```text
ACTIVE
CLEARED
EXPIRED
CHECKED_OUT
```

For the first iteration, only `ACTIVE` and `CLEARED` may be used.

## Table: basket_items

Purpose:

```text
stores mutable basket item state
```

Suggested SQL:

```sql
CREATE TABLE basket_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  basket_item_id VARCHAR(64) NOT NULL,
  basket_id VARCHAR(64) NOT NULL,
  product_id VARCHAR(64) NOT NULL,
  variant_id VARCHAR(64) NOT NULL,
  product_name_snapshot VARCHAR(255) NOT NULL,
  variant_name_snapshot VARCHAR(255) NOT NULL,
  quantity INT UNSIGNED NOT NULL,
  unit_price_minor_units BIGINT NOT NULL,
  currency_code CHAR(3) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (id),
  UNIQUE KEY uk_basket_items_basket_item_id (basket_item_id),
  UNIQUE KEY uk_basket_items_basket_product_variant (basket_id, product_id, variant_id),
  KEY idx_basket_items_basket_id (basket_id),
  KEY idx_basket_items_product_id (product_id),
  KEY idx_basket_items_variant_id (variant_id),

  CONSTRAINT fk_basket_items_basket_id
    FOREIGN KEY (basket_id)
    REFERENCES baskets (basket_id)
    ON DELETE CASCADE
);
```

## Notes on foreign keys

The basket database may use a foreign key from `basket_items.basket_id` to `baskets.basket_id` because both tables are owned by basket-service.

It must not use foreign keys to catalog tables.

Cross-service relationships should be represented by stable external IDs only.

## Duplicate item rule

This unique key prevents duplicate active product/variant rows in the same basket:

```sql
UNIQUE KEY uk_basket_items_basket_product_variant (basket_id, product_id, variant_id)
```

If the same product/variant is added twice, the service should update quantity rather than insert another row.

## Quantity rule

The service layer should enforce:

```text
1 <= quantity <= 99
```

The database may also include a check constraint later if desired.

## Snapshot rule

Basket stores snapshots for display and order preparation:

```text
product_name_snapshot
variant_name_snapshot
unit_price_minor_units
currency_code
```

These are not catalog truth.

Checkout/order placement should revalidate final price and availability.

## Repository operations

Expected first repository methods:

```text
CreateBasket
GetBasket
AddItem
UpdateItemQuantity
RemoveItem
ClearBasket
```

Implementation should use context-aware SQL calls:

```text
QueryContext
QueryRowContext
ExecContext
```

This preserves tracing and cancellation propagation.

## Future schema extensions

Later additions may include:

```text
session_id
customer_id
expires_at
promotion_code
discount_minor_units
tax_estimate_minor_units
shipping_estimate_minor_units
metadata_json
```

Do not add these until needed.

## Practical rule

```text
Basket data is mutable.
Order data is committed.
Do not design basket tables as if they are order tables.
```

Keep it boring where production matters.
