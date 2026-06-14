# Basket gRPC API Design

## Purpose

This document outlines the first basket-service gRPC contract.

The goal is to define a small, explicit API that supports the first local basket slice without over-designing checkout, payment, inventory, promotions, or user-account flows.

The first API should support:

```text
create basket
get basket
add item
update item quantity
remove item
clear basket
```

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
import "google/protobuf/timestamp.proto";
import "bfstore/common/v1/money.proto";
```

`google.protobuf.Timestamp` will be used for all time fields.

`bfstore.common.v1.Money` will be used for all API money values.

## Service

First service iteration:

```proto
service BasketService {
  rpc CreateBasket(CreateBasketRequest) returns (CreateBasketResponse);
  rpc GetBasket(GetBasketRequest) returns (GetBasketResponse);
  rpc AddItem(AddItemRequest) returns (AddItemResponse);
  rpc UpdateItemQuantity(UpdateItemQuantityRequest) returns (UpdateItemQuantityResponse);
  rpc RemoveItem(RemoveItemRequest) returns (RemoveItemResponse);
  rpc ClearBasket(ClearBasketRequest) returns (ClearBasketResponse);
}
```

The first iteration is be explicit:

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

This is easier to test, document, and troubleshoot.

## Core messages

### Basket

```proto
message Basket {
  string basket_id = 1;
  repeated BasketItem items = 2;
  bfstore.common.v1.Money subtotal = 3;
  string status = 4;
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}
```

Notes:

```text
basket_id is the stable external basket identifier.
items contains the current basket line items.
subtotal is the current basket subtotal snapshot.
status is included for lifecycle visibility.
created_at and updated_at use google.protobuf.Timestamp.
```

Recommended first status values:

```text
ACTIVE
CLEARED
EXPIRED
CHECKED_OUT
```

For the first iteration, only `ACTIVE` and `CLEARED` may be used.

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
If the same product/variant pair already exists, increase or update quantity according to service rules.
AddItem should increase quantity for an existing product/variant pair.
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
ClearBasket should remove all items and keep the basket available as ACTIVE.
```

A later lifecycle pass can introduce stricter `CLEARED` semantics if needed.

## Full first-iteration proto sketch

```proto
syntax = "proto3";

package bfstore.basket.v1;

import "google/protobuf/timestamp.proto";
import "bfstore/common/v1/money.proto";

option go_package = "github.com/mantrobuslawal/bfstore/proto/gen/go/bfstore/basket/v1;basketv1";

service BasketService {
  rpc CreateBasket(CreateBasketRequest) returns (CreateBasketResponse);
  rpc GetBasket(GetBasketRequest) returns (GetBasketResponse);
  rpc AddItem(AddItemRequest) returns (AddItemResponse);
  rpc UpdateItemQuantity(UpdateItemQuantityRequest) returns (UpdateItemQuantityResponse);
  rpc RemoveItem(RemoveItemRequest) returns (RemoveItemResponse);
  rpc ClearBasket(ClearBasketRequest) returns (ClearBasketResponse);
}

message Basket {
  string basket_id = 1;
  repeated BasketItem items = 2;
  bfstore.common.v1.Money subtotal = 3;
  string status = 4;
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

Basket API will use:

```text
basket_id
basket_item_id
product_id
variant_id
```

It will not use these as identity:

```text
product name
variant name
category name
slug
SKU
```

SKU may appear later as snapshot or inventory-related data, but will not replace `product_id` or `variant_id`.

## Money rules

API contracts will use:

```proto
bfstore.common.v1.Money
```

Database storage will still use:

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

API contracts will use:

```proto
google.protobuf.Timestamp
```

for all time fields.

Database storage will use:

```sql
TIMESTAMP(6)
```

or another agreed timestamp standard.

The service maps between database timestamp values and Protobuf timestamps.

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

Will not add these in the first slice unless needed.

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

## Practical rule

```text
Keep the first BasketService contract small, explicit, and consistent with existing common types.
```
