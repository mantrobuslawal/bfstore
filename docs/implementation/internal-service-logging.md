# Internal Service Logging Approach

## Purpose

This document defines the internal logging approach for bfstore services.

It complements request-level gRPC interceptor logging by describing what should be logged inside service-layer and repository-layer code.

The goal is to keep logs useful, structured, consistent, and safe.

Good logs should help answer questions such as:

```text
What business action happened?
Which entity was affected?
Was the operation accepted or rejected?
Did persistence fail?
Was an internal decision made that explains the result?
Can this log be correlated with traces, metrics, and request logs?
```

Logs should not simply record every function entry and exit.

## Scope

This guidance applies to bfstore service implementations, starting with:

```text
catalog-service
basket-service
```

It should also be reused by future services such as:

```text
inventory-service
order-service
payment-service
shipping-service
notification-service
search-service
review-service
recommendation-service
```

## Core principle

Do not choose log level based on whether a function is public or private.

Use this rule instead:

```text
Log meaningful events, decisions, failures, and boundaries.
Do not log functions merely because they exist.
```

Function visibility is not the right logging boundary.

A public function can be too noisy for `Info`.

A private function can make a business-significant decision worth logging.

## Logging responsibilities by layer

### gRPC interceptors

The gRPC interceptor layer logs generic transport boundaries.

It should usually log:

```text
gRPC method
status code
duration
correlation ID
request ID, if present
peer information, if useful
panic recovery, if applicable
```

Example log event:

```text
grpc request completed
```

Example attributes:

```text
method=/bfstore.basket.v1.BasketService/AddItem
code=OK
duration_ms=12
correlation_id=...
```

The interceptor should not try to understand basket, catalog, order, or payment business rules.

### gRPC adapter layer

The gRPC adapter layer translates between generated Protobuf types and internal service types.

It may log:

```text
request validation failures, when useful
transport-to-service mapping failures
unexpected enum or timestamp mapping issues
```

It should not log every successful request at `Info`, because the interceptor already handles request completion logging.

### Service layer

The service layer should log business-significant events and decisions.

For basket-service, useful `Info` events include:

```text
basket created
basket item added
basket item quantity updated
basket item removed
basket cleared
```

Useful `Warn` events include:

```text
basket modification rejected because the basket status is not modifiable
request rejected due to suspicious but recoverable business state
```

Useful `Debug` events include:

```text
currency omitted and defaulted to GBP
existing product/variant pair found in basket
basket subtotal recalculated
database status mapped to BasketStatus enum
```

The service layer should avoid logging every successful read at `Info`.

For example, `GetBasket` success is usually covered by interceptor logs and traces.

### Repository layer

The repository layer should usually be quieter than the service layer.

It should log:

```text
database failures
transaction rollback failures
unexpected missing rows
unexpected duplicate rows
data shape anomalies
slow query warnings, if not already handled elsewhere
```

It should not usually log every successful SQL query at `Info`.

Successful repository operations may be logged at `Debug` if helpful during local development or an incident investigation.

## Recommended log levels

| Level | Use for |
| --- | --- |
| `Debug` | Internal decisions, mapping details, calculated values, successful repository diagnostics |
| `Info` | Successful business events and high-level lifecycle outcomes |
| `Warn` | Recoverable but suspicious conditions, rejected state transitions, retries, degraded behaviour |
| `Error` | Failed operation that prevents the requested work from completing |
| `Fatal` | Process cannot start or continue safely; usually only in `main` |

## What to log at Info

Use `Info` for successful business events that matter.

Examples:

```text
basket created
basket item added
basket item quantity updated
basket item removed
basket cleared
product created
product updated
catalog category created
```

Example:

```go
logger.InfoContext(ctx, "basket item added",
	"basket_id", basketID,
	"basket_item_id", basketItemID,
	"product_id", productID,
	"variant_id", variantID,
	"quantity", quantity,
)
```

## What to log at Debug

Use `Debug` for internal implementation details that are useful when troubleshooting but too noisy for ordinary production logs.

Examples:

```text
currency omitted and defaulted to GBP
subtotal recalculated
status mapped from database value
existing basket item found for product/variant pair
repository loaded basket with item count
```

Example:

```go
logger.DebugContext(ctx, "basket subtotal recalculated",
	"basket_id", basketID,
	"item_count", len(items),
	"subtotal_minor_units", subtotalMinorUnits,
	"currency_code", currencyCode,
)
```

## What to log at Warn

Use `Warn` for recoverable but suspicious or operationally important conditions.

Examples:

```text
basket modification rejected because basket is checked out
request rejected because basket has expired
retry scheduled after transient dependency failure
fallback path used
```

Example:

```go
logger.WarnContext(ctx, "basket modification rejected",
	"basket_id", basketID,
	"status", status,
	"operation", "AddItem",
)
```

## What to log at Error

Use `Error` when the requested operation failed and the service could not complete the work.

Examples:

```text
failed to insert basket
failed to update basket item quantity
failed to commit transaction
failed to load basket
failed to map persisted basket state
```

Example:

```go
logger.ErrorContext(ctx, "failed to insert basket",
	"error", err,
	"basket_id", basketID,
)
```

Do not log the same error repeatedly at every layer unless each log adds meaningful context.

A practical approach is:

```text
repository logs persistence failure details
service logs business operation failure if it adds business context
adapter maps final error to gRPC status
interceptor logs final request status
```

## Basket-service examples

### CreateBasket

Recommended logs:

| Event | Level |
| --- | --- |
| Currency omitted and defaulted to GBP | `Debug` |
| Basket created | `Info` |
| Basket insert failed | `Error` |

Example:

```go
logger.InfoContext(ctx, "basket created",
	"basket_id", basket.ID,
	"currency_code", basket.CurrencyCode,
)
```

### GetBasket

Recommended logs:

| Event | Level |
| --- | --- |
| Basket loaded successfully | `Debug`, optional |
| Basket not found | Usually returned as `NotFound`; log only if operationally useful |
| Basket load failed due to database error | `Error` |

Avoid logging every successful `GetBasket` at `Info`.

### AddItem

Recommended logs:

| Event | Level |
| --- | --- |
| Existing product/variant pair found | `Debug` |
| Basket item added | `Info` |
| Basket item quantity increased | `Info` |
| Subtotal recalculated | `Debug` |
| Basket cannot be modified in current state | `Warn` |
| Persistence failure | `Error` |

### UpdateItemQuantity

Recommended logs:

| Event | Level |
| --- | --- |
| Quantity replaced | `Info` |
| Quantity validation rejected | Usually no service log unless suspicious |
| Basket item not found | Usually returned as `NotFound`; log only if useful |
| Persistence failure | `Error` |

### RemoveItem

Recommended logs:

| Event | Level |
| --- | --- |
| Basket item removed | `Info` |
| Basket item not found | Usually returned as `NotFound`; log only if useful |
| Persistence failure | `Error` |

### ClearBasket

Recommended logs:

| Event | Level |
| --- | --- |
| Basket cleared | `Info` |
| Clear rejected due to basket lifecycle state | `Warn` |
| Persistence failure | `Error` |

## Catalog-service examples

### Product reads

Successful product reads should usually rely on:

```text
gRPC interceptor logs
traces
metrics
```

Avoid logging every successful catalog read at `Info`.

### Product or category mutations

For future write operations, log meaningful business events:

```text
product created
product updated
category created
category updated
```

Use `Info` for successful business changes and `Error` for failed persistence.

## Structured logging fields

Use stable, searchable field names.

Recommended common fields:

```text
correlation_id
request_id
service
operation
grpc_method
duration_ms
error
```

Recommended basket fields:

```text
basket_id
basket_item_id
product_id
variant_id
quantity
currency_code
status
```

Recommended catalog fields:

```text
product_id
variant_id
category_id
sku
currency_code
status
```

Do not invent new field names for the same concept in different packages.

For example, prefer:

```text
basket_id
```

over inconsistent variants such as:

```text
basketID
basket
id
cart_id
```

## Context-aware logging

Use context-aware logging where available:

```go
logger.InfoContext(ctx, "basket created",
	"basket_id", basketID,
)
```

The same `context.Context` should flow through:

```text
gRPC adapter
service layer
repository layer
dependency clients
```

This helps connect logs with request metadata, correlation IDs, traces, and future deadline/cancellation handling.

## Relationship with tracing and metrics

Logs are not the only observability signal.

Use each signal for the right job:

| Signal | Best for |
| --- | --- |
| Logs | Discrete events, decisions, failures, and human-readable context |
| Traces | Request flow across functions, services, and dependencies |
| Metrics | Rates, latency, error counts, saturation, and trends |

Do not use logs as a substitute for metrics.

Do not use logs as a substitute for traces.

For example:

```text
request duration -> metric and interceptor log
database query span -> trace
basket item added -> service-layer log
database insert failure -> repository error log
```

## Data safety

Do not log sensitive or high-risk data.

Avoid logging:

```text
auth tokens
session tokens
passwords
payment details
full addresses
email addresses, unless explicitly required and approved
raw request bodies
large payloads
sensitive headers
```

Generally safe for current bfstore local development:

```text
basket_id
basket_item_id
product_id
variant_id
category_id
quantity
currency_code
status
minor-unit amounts
```

As bfstore adds customer and payment flows, logging rules should be reviewed again.

## Anti-patterns

Avoid:

```text
logging every function entry and exit
logging every successful repository query at Info
logging raw request payloads by default
logging the same error at every layer without adding context
using inconsistent field names
using logs instead of metrics for counts and latency
using logs instead of traces for request flow
choosing log level based only on public/private function visibility
```

## Implementation guidance

Inject a logger into service and repository structs:

```go
type Service struct {
	repo   Repository
	logger *slog.Logger
}
```

Prefer operation-specific log messages:

```go
logger.InfoContext(ctx, "basket item added",
	"basket_id", basketID,
	"basket_item_id", basketItemID,
	"product_id", productID,
	"variant_id", variantID,
	"quantity", quantity,
)
```

Avoid generic messages:

```go
logger.InfoContext(ctx, "AddItem called")
logger.InfoContext(ctx, "repository method finished")
```

The first message explains what happened.

The second only says that code ran.

## Recommended first implementation steps

For basket-service, add structured logs to write paths first:

```text
CreateBasket
AddItem
UpdateItemQuantity
RemoveItem
ClearBasket
```

Then add repository error logs around:

```text
insert basket
load basket
insert basket item
update basket item quantity
remove basket item
clear basket items
transaction commit/rollback
```

For catalog-service, review existing logging after basket-service has the same pattern.

## Repo location

Recommended path:

```text
docs/implementation/internal-service-logging.md
```

This sits alongside the existing implementation-level observability and service guidance.

If the repo later gets a broader observability section, this document can also be linked from:

```text
docs/implementation/local-observability.md
docs/implementation/request-metrics.md
services/catalog-service/README.md
services/basket-service/README.md
```

## Practical rule

```text
Interceptors log transport boundaries.
Service layer logs meaningful business events.
Repository layer logs persistence failures and unusual data conditions.
Debug logs explain internal decisions.
Info logs record successful business outcomes.
```

Keep it boring where production matters.
