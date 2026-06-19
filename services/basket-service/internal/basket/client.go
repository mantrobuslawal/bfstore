package basket

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CatalogGRPCClient struct {
	client  catalogv1.CatalogServiceClient
	logger  *slog.Logger
	timeout time.Duration
}

func NewCatalogGRPCClient(
	client catalogv1.CatalogServiceClient,
	logger *slog.Logger,
	timeout time.Duration,
) *CatalogGRPCClient {
	if logger == nil {
		logger = slog.Default()
	}

	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	return &CatalogGRPCClient{
		client:  client,
		logger:  logger,
		timeout: timeout,
	}
}

func (c *CatalogGRPCClient) ValidateProductVariant(
	ctx context.Context,
	query ValidateProductVariantQuery,
) (CatalogProductVariant, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ValidateProductVariant(ctx, &catalogv1.ValidateProductVariantRequest{
		ProductId: query.ProductID,
		VariantId: query.VariantID,
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "catalog product variant validation failed",
			"error", err,
			"product_id", query.ProductID,
			"variant_id", query.VariantID,
		)

		return CatalogProductVariant{}, mapCatalogValidationError(err)
	}

	if resp == nil {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned nil response")
	}

	if !resp.GetSellable() {
		c.logger.WarnContext(ctx, "catalog product variant is not sellable",
			"product_id", query.ProductID,
			"variant_id", query.VariantID,
		)

		return CatalogProductVariant{}, ErrProductNotSellable
	}

	unitPrice := resp.GetUnitPrice()
	if unitPrice == nil {
		return CatalogProductVariant{}, fmt.Errorf("catalog validation returned nil unit price")
	}

	result := CatalogProductVariant{
		ProductID:   resp.GetProductId(),
		VariantID:   resp.GetVariantId(),
		ProductName: GetProductName(),
		VariantName: GetVariantName(),
		UnitPrice: Money{
			CurrencyCode: CurrencyCode(unitPrice.GetCurrencyCode()),
			AmountMinor:  unitPrice.GetAmountMinor(),
		},
		Sellable: resp.GetSellable(),
	}

	if result.ProductID != query.ProductID {
		c.logger.WarnContext(ctx, "catalog validation product id mismatch",
			"requested_product_id", query.ProductID,
			"catalog_product_id", result.ProductID,
			"variant_id", query.VariantID,
		)

		return CatalogProductVariant{}, ErrProductVariantMismatch
	}

	if result.VariantID != query.VariantID {
		c.logger.WarnContext(ctx, "catalog validation variant id mismatch",
			"product_id", query.ProductID,
			"requested_variant_id", query.VariantID,
			"catalog_variant_id", result.VariantID,
		)

		return CatalogProductVariant{}, ErrProductVariantMismatch
	}

	c.logger.DebugContext(ctx, "catalog product variant validated",
		"product_id", result.ProductID,
		"variant_id", result.VariantID,
		"currenyCode", result.UnitPrice.CurrencyCode,
		"unit_price_minor_units", result.UnitPrice.AmountMinor,
	)

	return result, nil
}

func mapCatalogValidationError(err error) error {
	code := status.Code(err)

	switch code {
	case codes.NotFound:
		return ErrBasketNotFound

	case codes.InvalidArgument:
		return ErrProductVariantMismatch

	case codes.FailedPrecondition:
		return ErrProductNotSellable

	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("catalog service unavailable: %w", err)

	default:
		return fmt.Errorf("catalog validation failed: %w", err)
	}
}
