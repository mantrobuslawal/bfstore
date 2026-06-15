// Package basket contains the core application logic for the bfstore basket
// service.
//
// The basket service manages a customer's mutable basket before checkout. It
// owns temporary customer intent, including creating baskets, adding items,
// updating item quantities, removing items, clearing baskets, and calculating
// basket totals.
//
// Basket state is deliberately separate from catalog, inventory, order,
// payment, and shipping state. The basket service references catalog-owned
// products and variants by stable external identifiers, but it does not own
// product truth, price truth, stock truth, or order history.
//
// The first implementation slice keeps the service intentionally small:
//
//   - create an empty basket
//   - retrieve an existing basket
//   - add a catalog product variant to a basket
//   - update an existing basket item quantity
//   - remove an item from a basket
//   - clear all items from a basket
//
// Basket item names and prices are stored as snapshots. They represent what
// the basket service understood at the time an item was added or updated. Final
// pricing, stock availability, promotions, and checkout validation should be
// rechecked later by the appropriate services during order placement.
//
// The package should keep business rules explicit and boring. Validation rules
// such as required identifiers, valid quantities, basket status transitions,
// and subtotal calculation belong close to the service layer rather than being
// hidden in transport or database code.
//
// Practical rule:
//
//	Basket state is customer intent.
//	Order state is commercial history.
//
package basket
