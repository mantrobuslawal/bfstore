package basket

import (
	"context"
	"log/slog"
	"time"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"
	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
)

type CatalogGRPCClient struct {
	client  catalogv1.CatalogServiceClient
	logger  *slog.Logger
	timeout time.Duration
}

func newCatalogGRPCClient(
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
	query basket.ValidateProductVariantQuery,
) (basket.CatalogProductVariant, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ValidateProductVariant(ctx, &catalogv1.ValidateProductVariantRequest{
		ProductId: query.ProductID,
		VariantId: query.VariantID,
	})
	if err != nil {

	}
}
