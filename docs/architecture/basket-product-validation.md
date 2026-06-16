# Basket Product and Variant Validation

## Purpose

This document defines when and where bfstore validates `product_id` and `variant_id` references used by basket-service.

The goal is to prevent baskets from accepting invalid catalog references while still recognising that final commercial validation must happen again before order placement and payment.

## Summary

Product and variant validation happens in more than one place:

```text
AddItem
  -> lightweight catalog validation before storing the basket item

Checkout / CreateOrder
  -> authoritative revalidation before creating committed order lines

Payment
  -> charges a validated order amount; it should not validate product identity
```

Practical rule:

```text
Validate product_id and variant_id when adding to basket.
Revalidate them when creating or placing the order.
Do not wait until payment.
```

## Why payment is too late

Payment should not be the first point where the system discovers:

```text
the product does not exist
the variant does not exist
the variant does not belong to the product
the product is no longer sellable
the variant is inactive
the price has changed
the basket total is no longer valid
```

Payment should receive a valid payable amount for an already validated order/payment intent.

Payment concerns include:

```text
amount
currency
payment method
payment provider reference
idempotency
payment status
provider errors
```

Product identity belongs to catalog validation and order validation before money moves.

## AddItem validation

When basket-service receives an `AddItem` request, it should validate enough to protect basket quality.

Inputs:

```text
basket_id
product_id
variant_id
quantity
```

Basket-service should validate:

```text
basket exists
basket is modifiable
quantity is between 1 and 99
product_id is present
variant_id is present
product exists in catalog
variant exists in catalog
variant belongs to product
product and variant are sellable enough to add to basket
```

After validation, basket-service should snapshot useful catalog information:

```text
product_name_snapshot
variant_name_snapshot
unit_price
currency_code
```

The basket snapshot is not catalog truth. It is what basket-service understood at the time the item was added or updated.

## Checkout / order validation

Even if `AddItem` validated successfully, time may pass before checkout.

During that time:

```text
product could be discontinued
variant could be disabled
price could change
stock could disappear
promotion could expire
basket could become stale
```

Therefore checkout/order creation must revalidate authoritatively before creating committed order lines.

Order or checkout flow should revalidate:

```text
product still exists
variant still exists
variant still belongs to product
product and variant are still sellable
current price
currency consistency
stock or reservation rules
basket totals
promotion/tax/shipping rules when introduced
```

Only after this revalidation should the system create or confirm an order amount for payment.

## Recommended service responsibilities

| Concern | Owner |
| --- | --- |
| Product existence | catalog-service |
| Variant existence | catalog-service |
| Variant belongs to product | catalog-service |
| Product/variant sellability | catalog-service |
| Basket existence and status | basket-service |
| Basket item quantity rule | basket-service |
| Basket subtotal snapshot | basket-service |
| Final order line validation | order-service or checkout orchestration |
| Payment authorisation/capture | payment-service |

## AddItem flow

Recommended first-slice flow:

```text
1. Validate request shape.
2. Load basket.
3. Confirm basket is modifiable.
4. Ask catalog-service to validate product_id + variant_id.
5. Receive product/variant display snapshots and unit price.
6. Add item or increase existing item quantity.
7. Recalculate basket subtotal.
8. Persist basket item changes transactionally.
9. Return updated basket.
```

## Catalog validation result

The basket service does not need full catalog product details.

It needs a small validation result:

```text
product_id
variant_id
product_name
variant_name
unit_price
sellable
```

If catalog says the product/variant pair is invalid, basket-service should reject `AddItem`.

## Error model

Recommended errors for `AddItem`:

| Condition | gRPC status |
| --- | --- |
| Missing `basket_id` | `InvalidArgument` |
| Missing `product_id` | `InvalidArgument` |
| Missing `variant_id` | `InvalidArgument` |
| Quantity less than 1 or greater than 99 | `InvalidArgument` |
| Basket not found | `NotFound` |
| Product not found | `NotFound` |
| Variant not found | `NotFound` |
| Variant does not belong to product | `InvalidArgument` or `FailedPrecondition` |
| Product/variant not sellable | `FailedPrecondition` |
| Basket not modifiable | `FailedPrecondition` |
| Catalog-service unavailable | `Unavailable` or mapped dependency error |
| Unexpected persistence failure | `Internal` |

## Snapshot rules

Basket-service may store:

```text
product_name_snapshot
variant_name_snapshot
unit_price_minor_units
currency_code
line_total_minor_units
```

These snapshots are useful for displaying a basket without needing to call catalog on every read.

However, snapshots are not final commercial truth.

At checkout/order creation, current catalog/inventory/pricing rules must be checked again.

## Relationship with identifier strategy

The basket service uses:

```text
basket_id
basket_item_id
product_id
variant_id
```

It must not use these as identity:

```text
product name
variant name
category name
slug
sku
```

`product_id` and `variant_id` are catalog-owned identifiers.

`basket_id` and `basket_item_id` are basket-owned identifiers.

## Anti-patterns

Avoid:

```text
storing arbitrary product_id and variant_id without catalog validation
waiting until payment to discover invalid catalog references
using SKU instead of product_id or variant_id
using product name as identity
using slug as identity
trusting basket price snapshots as final checkout price
calling payment before order validation has confirmed the payable amount
```

## Future extensions

Later versions may introduce:

```text
catalog validation cache
catalog read model owned by basket-service
basket stale warnings
price changed notifications
inventory reservation
promotion validation
checkout validation endpoint
order preflight validation
```

These should be added after the first basket and catalog flow is stable.

## Practical rule

```text
Basket validates enough to keep basket state clean.
Order validates authoritatively before committing commercial history.
Payment charges only after order/payment amount is valid.
```

Keep it boring where production matters.
