# Basket API v1

This directory contains the version 1 Protobuf contract for the bfstore basket service.

```text
proto/bfstore/basket/v1/
  basket_service.proto
  README.md
```

## Purpose

The basket service manages a customer's mutable basket before checkout.

It owns temporary customer intent:

```text
create basket
get basket
add item
update item quantity
remove item
clear basket
```

It does not own product, variant, category, inventory, payment, shipping, or order truth.

## Package

```proto
package bfstore.basket.v1;

option go_package = "github.com/mantrobuslawal/bfstore/proto/gen/go/bfstore/basket/v1;basketv1";
```

## File

```text
basket_service.proto
```

## Imports

The basket contract uses shared bfstore and Google Protobuf types:

```proto
import "bfstore/common/v1/money.proto";
import "google/protobuf/timestamp.proto";
```

Use:

```text
bfstore.common.v1.Money
```

for all API money values.

Use:

```text
google.protobuf.Timestamp
```

for all API time values.

## Service

```proto
service BasketService {
  rpc CreateBasket(CreateBasketRequest) returns (CreateBasketResponse);
  rpc GetBasket(GetBasketRequest) returns (GetBasketResponse);
  rpc AddItem(AddItemRequest) returns (AddItemResponse);
  rpc UpdateItemQuantity(UpdateItemQuantityRequest)
      returns (UpdateItemQuantityResponse);
  rpc RemoveItem(RemoveItemRequest) returns (RemoveItemResponse);
  rpc ClearBasket(ClearBasketRequest) returns (ClearBasketResponse);
}
```

## First-slice API behaviour

### CreateBasket

Creates an empty active basket and returns the generated `basket_id`.

The request accepts an optional `currency_code`.

For local development, the service may default omitted currency to:

```text
GBP
```

The defaulting behaviour should be explicit in service code and tests.

### GetBasket

Returns the current state of an existing basket.

### AddItem

Adds a catalog product variant to an existing basket.

Required identifiers:

```text
basket_id
product_id
variant_id
```

If the same product/variant pair already exists in the basket, the first implementation should increase the existing quantity.

### UpdateItemQuantity

Replaces the quantity of an existing basket item.

This operation should not be used as implicit removal.

Quantity `0` should be rejected; clients should call `RemoveItem`.

### RemoveItem

Removes one existing basket item.

### ClearBasket

Removes all basket items.

For the first slice, `ClearBasket` should leave the basket available as an active empty basket.

A later lifecycle pass may introduce stricter cleared/expired/checked-out semantics.

## Core types

### Basket

```proto
message Basket {
  string basket_id = 1;
  repeated BasketItem items = 2;
  bfstore.common.v1.Money subtotal = 3;
  BasketStatus status = 4;
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}
```

### BasketItem

```proto
message BasketItem {
  string basket_item_id = 1;
  string product_id = 2;
  string variant_id = 3;
  string product_name_snapshot = 4;
  string variant_name_snapshot = 5;
  int32 quantity = 6;
  bfstore.common.v1.Money unit_price = 7;
  bfstore.common.v1.Money line_total = 8;
  google.protobuf.Timestamp added_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}
```

### BasketStatus

```proto
enum BasketStatus {
  BASKET_STATUS_UNSPECIFIED = 0;
  BASKET_STATUS_ACTIVE = 1;
  BASKET_STATUS_CLEARED = 2;
  BASKET_STATUS_EXPIRED = 3;
  BASKET_STATUS_CHECKED_OUT = 4;
}
```

The service should not return `BASKET_STATUS_UNSPECIFIED` for a valid persisted basket.

## Identifier rules

The basket API uses stable identifiers:

```text
basket_id
basket_item_id
product_id
variant_id
```

Ownership:

```text
basket_id       -> basket-service
basket_item_id  -> basket-service
product_id      -> catalog-service
variant_id      -> catalog-service
```

The API should not use these as system identity:

```text
product name
variant name
category name
slug
SKU
```

SKU may appear later as inventory or snapshot data, but it should not replace `product_id` or `variant_id`.

## Money rules

The API contract uses:

```proto
bfstore.common.v1.Money
```

Database storage may still use simple columns such as:

```text
unit_price_minor_units BIGINT
line_total_minor_units BIGINT
currency_code CHAR(3)
```

The service layer maps database values to Protobuf `Money`.

Practical rule:

```text
Use Money in Protobuf contracts.
Use minor-unit BIGINT plus currency_code in MySQL.
Convert at the service boundary.
```

## Timestamp rules

The API contract uses:

```proto
google.protobuf.Timestamp
```

Database storage may use:

```sql
TIMESTAMP(6)
```

The service layer maps database timestamp values to Protobuf timestamps.

## Status mapping

The Protobuf API uses `BasketStatus`.

The first database schema may store status as:

```sql
status VARCHAR(32)
```

Recommended mapping:

| Database value | Protobuf value |
| --- | --- |
| `ACTIVE` | `BASKET_STATUS_ACTIVE` |
| `CLEARED` | `BASKET_STATUS_CLEARED` |
| `EXPIRED` | `BASKET_STATUS_EXPIRED` |
| `CHECKED_OUT` | `BASKET_STATUS_CHECKED_OUT` |

## Validation rules

### CreateBasketRequest

```text
currency_code may be omitted if the service has a documented default
currency_code must be a valid ISO-style 3-letter currency code when provided
```

### GetBasketRequest

```text
basket_id is required
```

### AddItemRequest

```text
basket_id is required
product_id is required
variant_id is required
quantity must be between 1 and 99
```

### UpdateItemQuantityRequest

```text
basket_id is required
basket_item_id is required
quantity must be between 1 and 99
```

### RemoveItemRequest

```text
basket_id is required
basket_item_id is required
```

### ClearBasketRequest

```text
basket_id is required
```

## Error model

Recommended gRPC status codes:

| Condition | gRPC code |
| --- | --- |
| Missing or invalid required field | `InvalidArgument` |
| Basket not found | `NotFound` |
| Basket item not found | `NotFound` |
| Basket cannot be modified in current state | `FailedPrecondition` |
| Unexpected persistence/runtime failure | `Internal` |

Examples:

```text
missing basket_id -> InvalidArgument
unknown basket_id -> NotFound
quantity less than 1 -> InvalidArgument
quantity greater than 99 -> InvalidArgument
unknown basket_item_id -> NotFound
```

## Boundary rules

The basket service must not query catalog database tables directly.

It should reference catalog-owned entities by ID and, where necessary, use catalog APIs or future read models.

Basket state is mutable customer intent.

Order state is committed commercial history.

```text
basket item = customer intent
order line = commercial record
```

## Generating code

From the repo root:

```bash
make proto-lint
make proto-generate
```

Or generate only catalog contracts when needed:

```bash
make proto-generate-catalog
```

A basket-specific generation target may be added later:

```bash
buf generate --path proto/bfstore/basket/v1
```

## Future extensions

Future versions may add:

```text
MergeBasket
ExpireBasket
ApplyPromotion
RemovePromotion
ValidateBasketForCheckout
GetBasketBySession
AssociateBasketWithCustomer
```

These are intentionally excluded from the first slice.

## Practical rule

Keep the first basket contract small, explicit, typed, and consistent with shared bfstore contract types.

Keep it boring where production matters.
