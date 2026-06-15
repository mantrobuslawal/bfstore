// Package grpcadapter contains the gRPC transport adapter for the bfstore
// basket service.
//
// The grpcadapter package is responsible for translating between generated
// Protobuf request/response types and the basket service's internal application
// model. It should keep transport concerns at the edge of the service and avoid
// leaking gRPC-specific details into the core basket domain logic.
//
// Typical responsibilities include:
//
//   - registering the BasketService gRPC server
//   - implementing generated gRPC handler methods
//   - validating transport-level request shape
//   - mapping Protobuf requests into internal service commands
//   - mapping internal basket models back into Protobuf responses
//   - converting internal errors into appropriate gRPC status codes
//   - converting between database/domain timestamps and google.protobuf.Timestamp
//   - converting internal money values into bfstore.common.v1.Money
//
// This package should not contain database queries, persistence rules, pricing
// rules, catalog ownership logic, or checkout behaviour. Those concerns belong
// in the application, repository, catalog integration, or future checkout/order
// layers.
//
// The adapter should be intentionally thin:
//
//	gRPC request -> application command -> application result -> gRPC response
//
// Error handling should be explicit and predictable. For example:
//
//   - invalid request fields should map to codes.InvalidArgument
//   - missing baskets or basket items should map to codes.NotFound
//   - invalid basket lifecycle changes should map to codes.FailedPrecondition
//   - unexpected runtime or persistence failures should map to codes.Internal
//
// The grpcadapter package is also the right place to preserve API contract
// behaviour such as using BasketStatus enums, google.protobuf.Timestamp, and
// bfstore.common.v1.Money in outbound responses.
//
// Practical rule:
//
//	Keep transport logic at the transport edge.
//	Keep basket business rules in the basket service.
//

package grpcadapter
