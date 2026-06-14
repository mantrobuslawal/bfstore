# ADR 0012: Product, Variant, Category, and Basket Identifier Strategy

## Status

Proposed

## Context

The basket service will be the first bfstore service outside the catalog boundary to reference catalog-owned entities. Product, variant, and category identifiers now become cross-service contract values rather than internal catalog implementation details.

These identifiers will appear in gRPC messages, basket database rows, future Kafka events, order lines, inventory checks, logs, metrics, dashboards, test fixtures, and debugging workflows.

Poor identifier choices at this stage can create avoidable coupling, confusing test data, brittle migrations, and painful troubleshooting.

## Decision

bfstore will use different identifier styles for different domain concepts.

The guiding rule is:

```text
Use opaque stable IDs for high-volume mutable entities.
Use mnemonic stable IDs where human searchability has strong operational value.
Do not use display names as identifiers.
Do not treat slugs as stable system identity.
```

## Product IDs

Products should use stable opaque external IDs.

Example:

```text
prod_01JZ8Z3K9XQJ8V7N9K2QW5Y6AB
```

Rationale:

```text
products may be numerous
product names can change
product taxonomy can change
product IDs may appear in basket, order, inventory, review, search, and recommendation data
opaque IDs avoid embedding business meaning that may later become wrong
```

## Variant IDs

Variants should use stable opaque external IDs.

Example:

```text
var_01JZ8Z4PZK8BWAMY29T5F2P7HF
```

Rationale:

```text
variants represent purchasable options
variants may affect price, inventory, fulfilment, and order lines
variant attributes may change over time
basket should reference the precise variant selected
```

## Category IDs

Categories may use stable mnemonic IDs.

Examples:

```text
cat_lang_go
cat_lang_python
cat_topic_security
cat_topic_distributed_systems
cat_room_office
cat_room_living_room
```

Rationale:

```text
categories are fewer than products
categories are often searched manually
categories are useful in logs, seed data, docs, and debugging
mnemonic IDs improve local development and admin readability
```

Category IDs must still be treated as stable contracts once published. If a display name changes, the category ID should not churn casually.

Example:

```text
category_id: cat_lang_go
name: Go
slug: go
description: Products inspired by Go, gophers, and Go community culture.
```

## SKU

SKU should be separate from product ID and variant ID.

Example:

```text
GO-MUG-BLUE-001
```

Rationale:

```text
SKU is a business/stock identifier
SKU may be meaningful to operations
SKU may map to warehouse/inventory processes
SKU should not replace system identity
```

## Slug

Slug should be separate from ID.

Example:

```text
go-gopher-mug
```

Rationale:

```text
slug is useful for URLs and human display
slug may change for SEO or naming reasons
slug should not be used as cross-service identity
```

## Internal database IDs

Each service may use internal database primary keys if useful, such as `BIGINT AUTO_INCREMENT` or binary UUIDs.

Internal database IDs should not leak across service boundaries.

Cross-service contracts should use external stable IDs such as:

```text
product_id
variant_id
category_id
basket_id
basket_item_id
```

## Basket implications

The basket service should store catalog references using external IDs.

Example basket item fields:

```text
basket_item_id
basket_id
product_id
variant_id
quantity
product_name_snapshot
variant_name_snapshot
unit_price_snapshot
currency_code
added_at
updated_at
```

The basket service should not own catalog truth. It may store snapshots for customer experience and order preparation, but product truth remains in catalog.

## Event implications

Future basket events should use stable IDs.

Example:

```text
BasketItemAdded
  basket_id
  basket_item_id
  product_id
  variant_id
  quantity
```

Events should not rely on product names, category names, or slugs as identity.

## Consequences

Positive:

```text
basket can safely reference catalog data
debugging remains readable
categories remain searchable
product and variant IDs avoid meaning drift
future order/payment/inventory flows have stable references
```

Tradeoffs:

```text
opaque product IDs are less human-readable than slugs
mnemonic category IDs require naming discipline
seed data must be curated carefully
identifier strategy must be documented and enforced in proto/database docs
```

## Rejected alternatives

### Use product names as IDs

Rejected because names can change and may not be unique.

### Use slugs as system IDs

Rejected because slugs are display/URL concerns and may change.

### Use only auto-increment database IDs everywhere

Rejected because internal storage IDs should not become cross-service contracts.

### Use mnemonic IDs for every entity

Rejected because product and variant semantics may change and high-volume entities are better represented by stable opaque IDs.

## Practical rule

```text
Product and variant IDs should be boring and stable.
Category IDs may be mnemonic, but still stable.
Slugs are for humans and URLs, not system identity.
```

Keep it boring where production matters.
