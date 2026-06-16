# bfstore Identifier Strategy

## Purpose

This document defines the identifier strategy for bfstore domain entities.

The goal is to keep identifiers:

```text
stable
explicit
searchable where useful
safe to expose through APIs
easy to recognise in logs
not coupled to mutable business data
```

Identifiers are part of the platform contract. Once an identifier is exposed through APIs, events, logs, links, or operational tooling, it should be treated as stable.

## Scope

This document covers the current identifier approach for:

```text
product_id
variant_id
category_id
sku
slug
basket_id
basket_item_id
```

It also describes which service owns each identifier and how each identifier should be used.

## Summary

| Identifier | Owner | Recommended shape | Meaning |
| --- | --- | --- | --- |
| `product_id` | catalog-service | `prod_<opaque-id>` | Stable product identity |
| `variant_id` | catalog-service | `var_<opaque-id>` | Stable purchasable variant identity |
| `category_id` | catalog-service | `cat_<mnemonic-id>` | Stable category identity |
| `sku` | catalog/inventory/fulfilment boundary | Business-specific code | Stock keeping and fulfilment identifier |
| `slug` | catalog/frontend boundary | Human-readable URL text | Display, URL, and SEO identifier |
| `basket_id` | basket-service | `basket_<opaque-id>_<checksum>` | Stable basket identity |
| `basket_item_id` | basket-service | `bitem_<opaque-id>_<checksum>` | Stable basket line-item identity |

## Core rule

Use the right identifier for the right job.

```text
System identity    -> product_id, variant_id, category_id, basket_id, basket_item_id
Stock operations   -> sku
URLs/display       -> slug
Human labels       -> names and display snapshots
```

Do not use names, slugs, or SKUs as core system identity.

## Product IDs

### Recommendation

Product IDs should be opaque, stable, external identifiers.

Recommended shape:

```text
prod_<opaque-id>
```

Example:

```text
prod_01JZ7Y8K9M2Q4R6T8V0W1X2Y3Z
```

### Ownership

`product_id` is owned by catalog-service.

Other services may reference it, but should not generate it.

### Why opaque?

A product can be:

```text
renamed
recategorised
repriced
repositioned
assigned new display copy
given new slugs
linked to new variants
```

The product identity should survive those changes.

Avoid IDs such as:

```text
prod_go_mug_blue
prod_office_chair_2026
prod_rob_pike_tapestry_large
```

Those encode mutable business meaning into identity.

## Variant IDs

### Recommendation

Variant IDs should be opaque, stable, external identifiers.

Recommended shape:

```text
var_<opaque-id>
```

Example:

```text
var_01JZ7Y9ABC3DEF456GHI789JKL
```

### Ownership

`variant_id` is owned by catalog-service.

Other services may reference it, but should not generate it.

### Meaning

A variant represents a purchasable version of a product.

Examples:

```text
Go gopher mug / 350ml / blue
Rob Pike wall tapestry / medium / navy
Rivest secure lockbox / large / matte black
```

The basket service should store `product_id` and `variant_id` when referencing catalog-owned product choices.

## Category IDs

### Recommendation

Category IDs should be mnemonic, stable, and human-searchable.

Recommended shape:

```text
cat_<mnemonic-id>
```

Examples:

```text
cat_lang_go
cat_lang_python
cat_topic_security
cat_room_office
cat_room_living
```

### Ownership

`category_id` is owned by catalog-service.

### Why mnemonic?

Category IDs often appear in:

```text
admin workflows
search and filtering
logs
debugging
documentation
seed data
manual testing
```

A mnemonic ID makes category-related data easier to inspect and troubleshoot.

### Stability rule

Even though category IDs are mnemonic, they should still be treated as stable once published.

A category display name can change without changing the ID.

Example:

```text
category_id: cat_topic_security
old name: Security
new name: Security & Cryptography
```

The ID remains stable.

## SKUs

### Recommendation

SKU is a separate business and stock-keeping identifier.

Example:

```text
BFS-GO-MUG-BLU-350
```

### Meaning

SKU is useful for:

```text
warehouse operations
inventory systems
supplier feeds
fulfilment
stock counts
reporting
```

### Important rule

SKU should not replace:

```text
product_id
variant_id
basket_item_id
order_line_id
```

SKU may appear in future inventory, fulfilment, or reporting flows, but it is not the core service identity for product or variant records.

## Slugs

### Recommendation

Slug is a display and URL identifier.

Example:

```text
go-gopher-mug
rob-pike-wall-tapestry
rivest-secure-lockbox
```

### Meaning

Slugs are useful for:

```text
URLs
SEO
human-readable links
frontend routing
marketing copy
```

### Important rule

Slugs should not be used as core system identity.

Slugs may change when:

```text
product names change
marketing wording changes
SEO strategy changes
duplicate slugs need resolving
```

Changing a slug should not change the underlying product or variant identity.

## Basket IDs

### Recommendation

Basket IDs should be opaque, stable, externally safe identifiers generated by basket-service.

Recommended shape:

```text
basket_<opaque-id>_<checksum>
```

Example:

```text
basket_01JZ7Y8K9M2Q4R6T8V0W1X2Y3Z_7K9Q2M4R
```

### Ownership

`basket_id` is owned by basket-service.

Other services may reference a basket ID, but should not generate it.

### Rules

A basket ID should not encode:

```text
customer ID
session ID
device ID
product ID
variant ID
date strings
basket status
```

A basket may later become:

```text
anonymous
session-backed
customer-associated
merged
expired
converted into checkout/order flow
```

The ID should survive those lifecycle changes without leaking implementation details.

## Basket item IDs

### Recommendation

Basket item IDs should be opaque, stable, externally safe identifiers generated by basket-service.

Recommended shape:

```text
bitem_<opaque-id>_<checksum>
```

Example:

```text
bitem_01JZ7Y9ABC3DEF456GHI789JKL_4T8V0W1X
```

### Ownership

`basket_item_id` is owned by basket-service.

### Meaning

A basket item ID identifies a line item inside a basket.

A basket item references:

```text
basket_id
product_id
variant_id
quantity
price snapshot
name snapshot
```

but should not derive its identity from those values.

### Deterministic IDs

Do not make basket item IDs deterministic in the first implementation.

Avoid:

```text
basket_item_id = hash(basket_id + product_id + variant_id)
```

This creates avoidable coupling and makes future use cases harder, such as:

```text
customised items
gift wrap options
promotion-specific lines
subscription vs one-off lines
seller or fulfilment differences
multiple line types for the same variant
```

Generate a new `basket_item_id` when a new basket line is created.

If `AddItem` finds an existing product/variant pair in the same basket, update that existing line instead of creating a new ID.

## Recommended basket uniqueness rules

Use globally unique external IDs:

```text
basket_id should be globally unique
basket_item_id should be globally unique
```

For the first basket slice, enforce one active line per basket/product/variant pair:

```text
basket_id + product_id + variant_id should be unique for active basket items
```

Example database constraint:

```sql
UNIQUE KEY uq_basket_items_basket_product_variant (
  basket_id,
  product_id,
  variant_id
)
```

This supports the intended first-slice behaviour:

```text
AddItem first time       -> creates basket_item_id
AddItem same variant     -> increases existing quantity
UpdateItemQuantity       -> targets basket_item_id
RemoveItem               -> targets basket_item_id
```

## Basket checksum approach

Basket IDs and basket item IDs include a checksum suffix.

Recommended shapes:

```text
basket_<26-char-payload>_<8-char-checksum>
bitem_<26-char-payload>_<8-char-checksum>
```

The checksum exists to catch:

```text
copy/paste mistakes
manual typing mistakes
truncated identifiers
accidental corruption
wrong prefix/payload combinations
```

The checksum is for integrity and typo detection.

It is not an authentication or authorisation mechanism.

## Recommended opaque payload approach

For basket-owned IDs, use a ULID-style opaque payload:

```text
48-bit millisecond timestamp
80-bit cryptographic randomness
Crockford Base32 encoding
```

Benefits:

```text
roughly sortable by creation time
compact
URL-friendly
log-friendly
does not require database round trips to allocate IDs
does not encode business data
```

The checksum should be derived from the prefix and payload, for example:

```text
checksum = first 40 bits of SHA-256(prefix + "_" + payload)
```

Then encode those 40 bits as 8 Crockford Base32 characters.

## API usage

The basket API should expose:

```proto
message Basket {
  string basket_id = 1;
  repeated BasketItem items = 2;
}

message BasketItem {
  string basket_item_id = 1;
  string product_id = 2;
  string variant_id = 3;
}
```

The basket service owns:

```text
basket_id
basket_item_id
```

The catalog service owns:

```text
product_id
variant_id
category_id
```

Inventory and fulfilment may use:

```text
sku
```

Frontend and SEO flows may use:

```text
slug
```

## Logging guidance

Identifiers should be logged consistently using stable structured field names.

Recommended fields:

```text
product_id
variant_id
category_id
sku
slug
basket_id
basket_item_id
```

Avoid inconsistent alternatives such as:

```text
productID
product
prod
id
cart_id
line_id
```

Consistent field names make logs easier to search across services.

## Data safety

Current identifiers are generally safe to include in logs:

```text
product_id
variant_id
category_id
sku
slug
basket_id
basket_item_id
```

Do not encode sensitive data into identifiers.

Never encode:

```text
customer email
customer name
address data
payment data
session token
auth token
```

into any identifier.

## Service ownership summary

| Service | Owns | References |
| --- | --- | --- |
| catalog-service | `product_id`, `variant_id`, `category_id`, `slug`, product/variant display data | none initially |
| basket-service | `basket_id`, `basket_item_id`, basket state | `product_id`, `variant_id` |
| inventory-service | future stock records, SKU-related stock state | `product_id`, `variant_id`, `sku` |
| order-service | future `order_id`, `order_line_id` | `basket_id`, `product_id`, `variant_id`, `sku` where needed |

## Naming conventions

Recommended prefixes:

| Entity | Prefix |
| --- | --- |
| Product | `prod_` |
| Variant | `var_` |
| Category | `cat_` |
| Basket | `basket_` |
| Basket item | `bitem_` |

SKU format may follow a different business-readable convention, for example:

```text
BFS-GO-MUG-BLU-350
```

Slug format should be URL-friendly lower-case text:

```text
go-gopher-mug
```

## Anti-patterns

Avoid:

```text
using product names as IDs
using variant names as IDs
using SKU as the primary product or variant ID
using slug as the primary product or variant ID
encoding customer/session data into basket IDs
deriving basket_item_id from product_id and variant_id
changing category IDs casually because the display name changed
using inconsistent log field names for the same identifier
```

## Practical rules

```text
Use opaque IDs for product and variant identity.
Use mnemonic IDs for category searchability and debugging.
Use SKU for stock and fulfilment operations.
Use slug for display and URLs.
Use basket_<payload>_<checksum> for basket identity.
Use bitem_<payload>_<checksum> for basket item identity.
Keep customer, session, payment, and mutable business data out of IDs.
```

Keep it boring where production matters.
