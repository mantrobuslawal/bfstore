# Basket gRPC API Design

## Purpose

This document outlines the first basket-service gRPC contract.

The goal is to define a small, explicit API that supports the first local basket slice without over-designing checkout, payment, inventory, promotions, or user-account flows.

The first API supports:

```text
create basket
get basket
add item
update item quantity
remove item
clear basket
```

## Design decisions reflected in this version

This version reflects the current basket proto shape:

```text
CreateBasket is included in the first iteration.
Money values use bfstore.common.v1.Money.
All timestamps use google.protobuf.Timestamp.
Basket status is represented by a BasketStatus enum.
```

Using an enum for basket status gives the contract a clearer set of allowed lifecycle states and avoids free-form status strings drifting across services.

## Package

Proto path:

```text
proto/bfstore/basket/v1/basket_service.proto
```

Package:

```proto
syntax = "proto3";

package bfstore.basket.v1;

option go_package = "github.com/mantrobuslawal/bfstore/proto/gen/go/bfstore/basket/v1;basketv1";
```

## Imports

The basket contract imports:

```proto
import "bfstore/common/v1/money.proto";
import "google/protobuf/timestamp.proto";
```

`bfstore.common.v1.Money` is used for all API money values.

`google.protobuf.Timestamp` is used for all time fields.

## Service

First service iteration:

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

The first iteration is explicit:

```text
CreateBasket
  -> creates an empty basket
  -> returns basket_id

AddItem
  -> requires an existing basket_id
  -> adds or updates an item in that basket

GetBasket
  -> reads an existing basket
```

This is easier to test, document, and troubleshoot than making `AddItem` create baskets as a hidden side effect.

## Lifecycle enum

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

### Status meanings

| Status | Meaning |
| --- | --- |
| `BASKET_STATUS_UNSPECIFIED` | Default value. Should not be used for a valid persisted basket. |
| `BASKET_STATUS_ACTIVE` | Basket is active and can be modified. |
| `BASKET_STATUS_CLEARED` | Basket has been cleared. |
| `BASKET_STATUS_EXPIRED` | Basket has expired. Future lifecycle state. |
| `BASKET_STATUS_CHECKED_OUT` | Basket has been converted into checkout/order flow. Future lifecycle state. |

For the first implementation slice, the service will primarily use:

```text
BASKET_STATUS_ACTIVE
```

`BASKET_STATUS_CLEARED` is available if the first implementation decides to represent cleared baskets as a distinct lifecycle state.

Kuti recommendation for the first slice:

```text
ClearBasket should remove all items and keep the basket available as BASKET_STATUS_ACTIVE.
```

A later lifecycle pass can introduce stricter `CLEARED`, `EXPIRED`, and `CHECKED_OUT` semantics if needed.

## Core messages

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

Notes:

```text
basket_id is the stable external basket identifier.
items contains the current basket line items.
subtotal is the current basket subtotal snapshot.
status is a typed BasketStatus enum.
created_at and updated_at use google.protobuf.Timestamp.
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

Notes:

```text
product_id and variant_id reference catalog-owned entities.
product_name_snapshot and variant_name_snapshot are display snapshots, not catalog truth.
unit_price is a basket-time price snapshot.
line_total is normally unit_price * quantity.
added_at and updated_at use google.protobuf.Timestamp.
```

## Request and response messages

### CreateBasket

```proto
message CreateBasketRequest {
  string currency_code = 1;
}

message CreateBasketResponse {
  Basket basket = 1;
}
```

Notes:

```text
currency_code may default to GBP in local development.
The service should document defaulting behaviour clearly.
CreateBasket returns an empty ACTIVE basket.
```

For the first iteration, defaulting to `GBP` is acceptable if `currency_code` is omitted, but this should be explicit in service behaviour and tests.

### GetBasket

```proto
message GetBasketRequest {
  string basket_id = 1;
}

message GetBasketResponse {
  Basket basket = 1;
}
```

### AddItem

```proto
message AddItemRequest {
  string basket_id = 1;
  string product_id = 2;
  string variant_id = 3;
  int32 quantity = 4;
}

message AddItemResponse {
  Basket basket = 1;
}
```

Behaviour:

```text
If the product/variant pair is not already in the basket, create a new basket item.
If the same product/variant pair already exists, increase the existing quantity.
UpdateItemQuantity should replace quantity explicitly.
```

### UpdateItemQuantity

```proto
message UpdateItemQuantityRequest {
  string basket_id = 1;
  string basket_item_id = 2;
  int32 quantity = 3;
}

message UpdateItemQuantityResponse {
  Basket basket = 1;
}
```

For the first iteration:

```text
quantity must be between 1 and 99
quantity 0 should not be used as implicit remove
clients should call RemoveItem explicitly
```

### RemoveItem

```proto
message RemoveItemRequest {
  string basket_id = 1;
  string basket_item_id = 2;
}

message RemoveItemResponse {
  Basket basket = 1;
}
```

### ClearBasket

```proto
message ClearBasketRequest {
  string basket_id = 1;
}

message ClearBasketResponse {
  Basket basket = 1;
}
```

Notes for first slice:

```text
ClearBasket should remove all items and keep the basket available as BASKET_STATUS_ACTIVE.
```

A later lifecycle pass can introduce stricter `BASKET_STATUS_CLEARED` semantics if needed.

## Full first-iteration proto sketch

```proto
syntax = "proto3";

package bfstore.basket.v1;

import "bfstore/common/v1/money.proto";
import "google/protobuf/timestamp.proto";

option go_package = "github.com/mantrobuslawal/bfstore/proto/gen/go/bfstore/basket/v1;basketv1";

service BasketService {
  rpc CreateBasket(CreateBasketRequest) returns (CreateBasketResponse);
  rpc GetBasket(GetBasketRequest) returns (GetBasketResponse);
  rpc AddItem(AddItemRequest) returns (AddItemResponse);
  rpc UpdateItemQuantity(UpdateItemQuantityRequest)
      returns (UpdateItemQuantityResponse);
  rpc RemoveItem(RemoveItemRequest) returns (RemoveItemResponse);
  rpc ClearBasket(ClearBasketRequest) returns (ClearBasketResponse);
}

enum BasketStatus {
  BASKET_STATUS_UNSPECIFIED = 0;
  BASKET_STATUS_ACTIVE = 1;
  BASKET_STATUS_CLEARED = 2;
  BASKET_STATUS_EXPIRED = 3;
  BASKET_STATUS_CHECKED_OUT = 4;
}

message Basket {
  string basket_id = 1;
  repeated BasketItem items = 2;
  bfstore.common.v1.Money subtotal = 3;
  BasketStatus status = 4;
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}

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

message CreateBasketRequest {
  string currency_code = 1;
}

message CreateBasketResponse {
  Basket basket = 1;
}

message GetBasketRequest {
  string basket_id = 1;
}

message GetBasketResponse {
  Basket basket = 1;
}

message AddItemRequest {
  string basket_id = 1;
  string product_id = 2;
  string variant_id = 3;
  int32 quantity = 4;
}

message AddItemResponse {
  Basket basket = 1;
}

message UpdateItemQuantityRequest {
  string basket_id = 1;
  string basket_item_id = 2;
  int32 quantity = 3;
}

message UpdateItemQuantityResponse {
  Basket basket = 1;
}

message RemoveItemRequest {
  string basket_id = 1;
  string basket_item_id = 2;
}

message RemoveItemResponse {
  Basket basket = 1;
}

message ClearBasketRequest {
  string basket_id = 1;
}

message ClearBasketResponse {
  Basket basket = 1;
}
```

## Validation rules

### CreateBasketRequest

```text
currency_code may be omitted if the service has a documented default
currency_code must be a valid ISO-style 3-letter currency code when provided
```

First iteration default:

```text
GBP
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

gRPC status codes:

```text
InvalidArgument
  missing or invalid required fields

NotFound
  basket or basket item not found

FailedPrecondition
  basket cannot be modified in its current state

Internal
  unexpected persistence/runtime error
```

Examples:

```text
missing basket_id -> InvalidArgument
unknown basket_id -> NotFound
quantity less than 1 -> InvalidArgument
quantity greater than 99 -> InvalidArgument
removing unknown basket_item_id -> NotFound
```

## Identifier rules

Basket API uses:

```text
basket_id
basket_item_id
product_id
variant_id
```

It does not use these as identity:

```text
product name
variant name
category name
slug
SKU
```

SKU may appear later as snapshot or inventory-related data, but it does not replace `product_id` or `variant_id`.

## Money rules

API contracts use:

```proto
bfstore.common.v1.Money
```

Database storage still uses:

```text
unit_price_minor_units BIGINT
line_total_minor_units BIGINT
currency_code CHAR(3)
```

The service maps between database columns and Protobuf `Money`.

Practical rule:

```text
Use Money in Protobuf contracts.
Use minor-unit BIGINT plus currency_code in MySQL.
Convert at the service boundary.
```

## Timestamp rules

API contracts use:

```proto
google.protobuf.Timestamp
```

for all time fields.

Database storage uses:

```sql
TIMESTAMP(6)
```

or another agreed timestamp standard.

The service maps between database timestamp values and Protobuf timestamps.

## Status rules

API contracts use:

```proto
BasketStatus
```

rather than a raw string.

Database storage may continue to use:

```sql
status VARCHAR(32)
```

for the first implementation, but repository/service mapping should convert database strings to the Protobuf enum.

Recommended local mapping:

| Database value | Protobuf value |
| --- | --- |
| `ACTIVE` | `BASKET_STATUS_ACTIVE` |
| `CLEARED` | `BASKET_STATUS_CLEARED` |
| `EXPIRED` | `BASKET_STATUS_EXPIRED` |
| `CHECKED_OUT` | `BASKET_STATUS_CHECKED_OUT` |

The service should not return `BASKET_STATUS_UNSPECIFIED` for a valid persisted basket.

## Future extensions

Later versions may add:

```text
MergeBasket
ExpireBasket
ApplyPromotion
RemovePromotion
ValidateBasketForCheckout
GetBasketBySession
AssociateBasketWithCustomer
```

These should not be added in the first slice unless needed.

## Design notes

### Why CreateBasket is explicit

Explicit lifecycle operations make the API easier to reason about.

```text
CreateBasket creates state.
AddItem modifies known state.
GetBasket reads known state.
```

This is cleaner than making `AddItem` create a basket as a hidden side effect.

### Why Money is shared

Money appears across catalog, basket, order, payment, refund, and future reporting contracts.

A shared type avoids schema drift.

### Why Timestamp is used

Typed timestamps make the Protobuf contract clearer and easier to map consistently across services.

### Why BasketStatus is an enum

Basket lifecycle has a small, known set of states.

Using an enum makes the allowed values explicit, improves generated-code ergonomics, and avoids accidental string drift such as:

```text
active
ACTIVE
Active
basket_active
```

## Practical rule

```text
Keep the first BasketService contract small, explicit, typed, and consistent with existing common types.
```
