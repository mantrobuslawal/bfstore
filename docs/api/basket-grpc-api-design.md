# Basket gRPC API Design

## Purpose

This document outlines the first basket-service gRPC contract.

The goal is to define a small, useful API that supports the first local basket slice without over-designing checkout, payment, inventory, or promotions.

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

## Service

First service:

```proto
service BasketService {
  rpc GetBasket(GetBasketRequest) returns (GetBasketResponse);
  rpc AddItem(AddItemRequest) returns (AddItemResponse);
  rpc UpdateItemQuantity(UpdateItemQuantityRequest) returns (UpdateItemQuantityResponse);
  rpc RemoveItem(RemoveItemRequest) returns (RemoveItemResponse);
  rpc ClearBasket(ClearBasketRequest) returns (ClearBasketResponse);
}
```

## Core messages

Prefer `google.protobuf.Timestamp` in the real proto. String timestamps below are shown only for readability.

```proto
message Basket {
  string                     basket_id = 1;
  repeated BasketItem        items = 2;
  string                     currency_code = 3;
  int64                      subtotal_minor_units = 4;
  google.protobuf.Timestamp  created_at = 5;
  google.protobuf.Timestamp  updated_at = 6;
}

message BasketItem {
  string                    basket_item_id = 1;
  string                    product_id = 2;
  string                    variant_id = 3;
  string                    product_name_snapshot = 4;
  string                    variant_name_snapshot = 5;
  int32                     quantity = 6;
  int64                     unit_price_minor_units = 7;
  int64                     line_total_minor_units = 8;
  string                    currency_code = 9;
  google.protobuf.Timestamp added_at = 10;
  google.protobuf.Timestamp updated_at = 11;
}
```

## Requests and responses

```proto
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

```text
basket_id is required
product_id is required when adding item
variant_id is required when adding item
quantity must be between 1 and 99
basket_item_id is required for update/remove item
```

## Error model

gRPC codes:

```text
InvalidArgument
  missing or invalid required fields

NotFound
  basket or basket item not found

FailedPrecondition
  item cannot be modified in its current state

Internal
  unexpected persistence/runtime error
```

## Identifier rules

Basket API should use:

```text
basket_id
basket_item_id
product_id
variant_id
```

It should not use product names, slugs, or SKUs as identity.

## Future extensions

Later versions may add:

```text
CreateBasket
MergeBasket
ExpireBasket
ApplyPromotion
RemovePromotion
ValidateBasketForCheckout
```

Do not add these in the first slice unless they are needed.

## Practical rule

```text
Keep the first BasketService contract small enough to implement completely.
```

