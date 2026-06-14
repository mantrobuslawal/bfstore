# Basket Service Requirements

## Purpose

The basket service manages a customer's active basket before checkout.

It allows clients to:

```text
create or retrieve a basket
add items to a basket
change item quantities
remove items from a basket
view the current basket
clear a basket
```

The basket service is the bridge between browsing products and placing an order.

## Scope

In scope for the first basket iteration:

```text
anonymous/session basket support
basket item add/update/remove
quantity validation
catalog product and variant references
basic price/name snapshots
service-owned MySQL schema
gRPC API
golang-migrate migration workflow
unit and repository tests
health checks
logging, tracing, metrics, and dashboard reuse
```

Out of scope for the first basket iteration:

```text
user accounts
promotions
discount codes
shipping estimates
tax calculation
payment
order placement
inventory reservation
abandoned basket workflows
multi-currency pricing
basket merge after login
```

## Functional requirements

### Create or retrieve basket

The service should allow a client to create or retrieve an active basket.

A basket may be identified by:

```text
basket_id
session_id
```

For the first iteration, a generated `basket_id` is enough.

### Add item

A client should be able to add a product variant to a basket.

Required input:

```text
basket_id
product_id
variant_id
quantity
```

The service should either create a new item or increase/update quantity if the same product/variant already exists.

### Update quantity

A client should be able to update the quantity of an existing basket item.

For the first iteration, prefer explicit item removal rather than treating quantity zero as removal.

### Remove item

A client should be able to remove an item from a basket.

Required input:

```text
basket_id
basket_item_id
```

### Get basket

A client should be able to retrieve the current basket.

The response should include:

```text
basket_id
items
item quantities
product and variant references
price/name snapshots where available
created_at
updated_at
```

### Clear basket

A client should be able to remove all items from a basket.

## Business rules

Quantity must be positive.

Recommended first iteration rule:

```text
1 <= quantity <= 99
```

The same product/variant pair should not appear as multiple active items in the same basket. Instead, quantity should be increased or updated.

The basket service does not own product truth. It stores references to catalog-owned IDs and may store snapshots for user experience and future order preparation.

For the first basket iteration, the basket may store a unit price snapshot. This is not the final pricing authority for checkout. Final pricing should be revalidated during order placement.

## Non-functional requirements

The basket service should follow the same production-shaped local standards as catalog-service:

```text
structured logging
correlation ID propagation
panic recovery
health checks
graceful shutdown
OpenTelemetry traces
request metrics
database spans
database pool metrics
Prometheus visibility
Grafana dashboard support
```

## Acceptance criteria

The first basket iteration is complete when:

```text
basket gRPC API works locally
basket schema is managed with golang-migrate
items can be added, updated, removed, and listed
duplicate product/variant items update quantity instead of duplicating rows
unit and repository tests pass
local smoke test passes
health checks work
traces and metrics appear in the local observability stack
docs explain how to run and troubleshoot the service
```

## Practical rule

```text
Basket stores customer intent.
Catalog owns product truth.
Order will later convert basket intent into a committed commercial record.
```

Keep it boring where production matters.
