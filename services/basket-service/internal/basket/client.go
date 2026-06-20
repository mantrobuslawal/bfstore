package basket

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CatalogClient defines the catalog behaviour basket-service needs.
type CatalogClient interface {
	ValidateProductVariant(ctx context.Context, query ValidateProductVariantQuery) (CatalogProductVariant, error)
}

// CatalogGRPCClient adapts the generated catalog gRPC client to the basket
// service's small catalog validation interface.
type CatalogGRPCClient struct {
	client  catalogv1.CatalogServiceClient
	logger  *slog.Logger
	timeout time.Duration
}

var _ CatalogClient = (*CatalogGRPCClient)(nil)

func NewCatalogGRPCClient(
	client catalogv1.CatalogServiceClient,
	logger *slog.Logger,
	timeout time.Duration,
) *CatalogGRPCClient {
	if client == nil {
		panic("basket: nil catalog gRPC client")
	}

	if logger == nil {
		logger = slog.Default()
	}

	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	return &CatalogGRPCClient{
		client:  client,
		logger:  logger.With("component", "catalog_grpc_client"),
		timeout: timeout,
	}
}

func (c *CatalogGRPCClient) ValidateProductVariant(
	ctx context.Context,
	query ValidateProductVariantQuery,
) (CatalogProductVariant, error) {
	productID := strings.TrimSpace(query.ProductID)
	if productID == "" {
		return CatalogProductVariant{}, ErrMissingProductID
	}

	variantID := strings.TrimSpace(query.VariantID)
	if variantID == "" {
		return CatalogProductVariant{}, ErrMissingVariantID
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ValidateProductVariant(ctx, &catalogv1.ValidateProductVariantRequest{
		ProductId: productID,
		VariantId: variantID,
	})
	if err != nil {
		mappedErr := mapCatalogValidationError(err)

		if isExpectedCatalogValidationFailure(mappedErr) {
			c.logger.DebugContext(ctx, "catalog product variant validation rejected request",
				"error", mappedErr,
				"product_id", productID,
				"variant_id", variantID,
			)
		} else {
			c.logger.ErrorContext(ctx, "catalog product variant validation failed",
				"error", err,
				"mapped_error", mappedErr,
				"product_id", productID,
				"variant_id", variantID,
			)
		}

		return CatalogProductVariant{}, mappedErr
	}

	if resp == nil {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned nil response")
	}

	unitPrice := resp.GetUnitPrice()
	if unitPrice == nil {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned nil unit price")
	}

	result := CatalogProductVariant{
		ProductID:   strings.TrimSpace(resp.GetProductId()),
		VariantID:   strings.TrimSpace(resp.GetVariantId()),
		ProductName: strings.TrimSpace(resp.GetProductName()),
		VariantName: strings.TrimSpace(resp.GetVariantName()),
		UnitPrice: Money{
			CurrencyCode: CurrencyCode(strings.ToUpper(strings.TrimSpace(unitPrice.GetCurrencyCode()))),
			AmountMinor:  unitPrice.GetAmountMinor(),
		},
		Sellable: resp.GetSellable(),
	}

	if result.ProductID != productID {
		c.logger.WarnContext(ctx, "catalog validation product id mismatch",
			"requested_product_id", productID,
			"catalog_product_id", result.ProductID,
			"variant_id", variantID,
		)

		return CatalogProductVariant{}, ErrProductVariantMismatch
	}

	if result.VariantID != variantID {
		c.logger.WarnContext(ctx, "catalog validation variant id mismatch",
			"product_id", productID,
			"requested_variant_id", variantID,
			"catalog_variant_id", result.VariantID,
		)

		return CatalogProductVariant{}, ErrProductVariantMismatch
	}

	if result.ProductName == "" {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned empty product name")
	}

	if result.VariantName == "" {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned empty variant name")
	}

	if result.UnitPrice.CurrencyCode == "" {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned empty currency code")
	}

	if result.UnitPrice.AmountMinor < 0 {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned negative unit price")
	}

	if !result.Sellable {
		c.logger.WarnContext(ctx, "catalog product variant is not sellable",
			"product_id", productID,
			"variant_id", variantID,
		)

		return CatalogProductVariant{}, ErrProductNotSellable
	}

	c.logger.DebugContext(ctx, "catalog product variant validated",
		"product_id", result.ProductID,
		"variant_id", result.VariantID,
		"currency_code", result.UnitPrice.CurrencyCode,
		"unit_price_minor_units", result.UnitPrice.AmountMinor,
	)

	return result, nil
}

func mapCatalogValidationError(err error) error {
	code := status.Code(err)
	message := strings.ToLower(status.Convert(err).Message())

	switch code {
	case codes.NotFound:
		switch {
		case strings.Contains(message, "variant"):
			return ErrVariantNotFound
		case strings.Contains(message, "product"):
			return ErrProductNotFound
		default:
			return ErrProductNotFound
		}
	case codes.InvalidArgument:
		return ErrProductVariantMismatch
	case codes.FailedPrecondition:
		if strings.Contains(message, "basket") {
			return ErrBasketNotModifiable
		}

		return ErrProductNotSellable
	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("%w: %v", ErrCatalogServiceUnavailable, err)
	default:
		return fmt.Errorf("catalog validation failed: %w", err)
	}
}

func isExpectedCatalogValidationFailure(err error) bool {
	switch err {
	case ErrProductNotFound,
		ErrVariantNotFound,
		ErrProductVariantMismatch,
		ErrProductNotSellable:
		return true
	default:
		return false
	}
}
