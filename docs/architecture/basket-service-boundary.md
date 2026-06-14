# Basket Service Boundary

## Overview

The basket service owns temporary customer basket state.

It is responsible for tracking what a customer intends to buy before checkout.

It is not the source of truth for products, prices, inventory, payments, or orders.

## Service responsibilities

The basket service owns:

```text
basket identity
basket lifecycle
basket items
basket item quantities
basket item snapshots for display/order preparation
basket updated timestamps
```

The basket service does not own:

```text
product truth
variant truth
category truth
inventory availability
payment authorisation
order records
shipping calculations
notification workflows
```

## Relationship with catalog-service

The basket service references catalog-owned entities using stable external IDs:

```text
product_id
variant_id
```

It may store snapshots such as:

```text
product_name_snapshot
variant_name_snapshot
unit_price_snapshot
currency_code
```

These snapshots are not catalog truth. They are local basket state used for display and later order preparation.

## Relationship with inventory-service

Inventory reservation is out of scope for the first basket iteration.

Later, checkout/order placement may call inventory to reserve stock.

Basket should not attempt to become inventory truth.

## Relationship with order-service

The order service will later convert basket intent into an order.

Basket is mutable.

Order is committed history.

```text
basket item = customer intent
order line = commercial record
```

## Data ownership

The basket service should have its own database schema.

It should not read catalog tables directly.

It should not write catalog, inventory, order, or payment tables.

## Communication style

First iteration:

```text
gRPC API for basket commands and queries
no Kafka events required initially unless intentionally added
```

Future events may include:

```text
BasketCreated
BasketItemAdded
BasketItemQuantityChanged
BasketItemRemoved
BasketCleared
BasketExpired
```

## Identifier usage

Basket should use:

```text
basket_id
basket_item_id
product_id
variant_id
```

Product and variant IDs come from catalog.

Basket IDs and basket item IDs are owned by basket-service.

## Boundary rules

### Rule 1: Basket does not validate product truth by reading catalog DB

Basket must not directly query catalog tables.

If validation is needed, it should happen through catalog APIs or future read model contracts.

### Rule 2: Basket should store enough snapshot data for user experience

A basket page should not become unusable just because catalog display data changes.

### Rule 3: Basket should not make final pricing promises

Prices shown in basket are provisional.

Checkout/order placement should revalidate price before committing an order.

### Rule 4: Basket should be easy to discard

Basket state is temporary.

The model should support future expiry/cleanup without affecting committed orders.

## Practical rule

```text
Basket is not a mini order service.
Basket is mutable customer intent before checkout.
```

Keep it boring where production matters.
