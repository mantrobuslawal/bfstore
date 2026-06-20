package grpcadapter

import (
	"context"
	"log/slog"
	"strings"

	"github.com/mantrobuslawal/bfstore/services/catalog-service/internal/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/catalog/v1"
	commonv1 "github.com/mantrobuslawal/bfstore/gen/go/bfstore/common/v1"
)

type productVariantValidator interface {
	ValidateProductVariant(
		ctx context.Context,
		query catalog.ValidateProductVariantQuery,
	) (catalog.ValidatedProductVariant, error)
}

// CatalogHandler implements the generated Catalog Service gRPC interface.
type CatalogHandler struct {
	catalogv1.UnimplementedCatalogServiceServer

	catalogService          *catalog.Service
	productVariantValidator productVariantValidator
	logger                  *slog.Logger
}

// NewCatalogHandler creates a Catalog Service gRPC handler.
func NewCatalogHandler(catalogService *catalog.Service, logger *slog.Logger) *CatalogHandler {
	if catalogService == nil {
		panic("grpcadapter: nil catalog service")
	}

	if logger == nil {
		logger = slog.Default()
	}

	logger = logger.With("component", "catalog_grpc_handler")

	return &CatalogHandler{
		catalogService:          catalogService,
		productVariantValidator: catalogService,
		logger:                  logger,
	}
}

// ListProducts returns collection of products matching filter criteria.
//
// Products are not fully hydrated. Images, variant data, product attributes,
// and other detailed product data are intentionally not provided by this
// endpoint.
func (h *CatalogHandler) ListProducts(
	ctx context.Context,
	req *catalogv1.ListProductsRequest,
) (*catalogv1.ListProductsResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "list products request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := h.catalogService.ListProducts(ctx, catalog.ListProductsFilter{
		CategoryID:      catalog.CategoryID(req.GetCategoryId()),
		IncludeInactive: req.GetIncludeInactive(),
		PageSize:        int(req.GetPage().GetPageSize()),
		PageToken:       req.GetPage().GetPageToken(),
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list products",
			"error", err,
			"category_id", req.GetCategoryId(),
			"include_inactive", req.GetIncludeInactive(),
		)

		return nil, mapServiceError(err)
	}

	products := result.Result

	response := &catalogv1.ListProductsResponse{
		Products: make([]*catalogv1.Product, 0, len(products)),
		Page: &commonv1.PageResponse{
			NextPageToken: result.NextPageToken,
			TotalCount:    0, // Not calculated. Set to 0 as default.
		},
	}

	for _, product := range products {
		protoProduct, err := listProductToProto(&product)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to map listed product to proto",
				"error", err,
				"product_id", product.ProductID,
			)

			return nil, mapServiceError(err)
		}

		response.Products = append(response.Products, protoProduct)
	}

	h.logger.DebugContext(ctx, "listed products",
		"product_count", len(response.GetProducts()),
		"next_page_token_present", response.GetPage().GetNextPageToken() != "",
	)

	return response, nil
}

// GetProduct returns single product matching requested productID.
//
// This Product instance is fully hydrated, including images, variants,
// attributes, and related product detail.
func (h *CatalogHandler) GetProduct(
	ctx context.Context,
	req *catalogv1.GetProductRequest,
) (*catalogv1.GetProductResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "get product request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	productID := strings.TrimSpace(req.GetProductId())
	if productID == "" {
		h.logger.WarnContext(ctx, "get product request missing product id")
		return nil, status.Error(codes.InvalidArgument, "missing product id")
	}

	product, err := h.catalogService.GetProduct(ctx, catalog.ProductID(productID))
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get product",
			"error", err,
			"product_id", productID,
		)

		return nil, mapServiceError(err)
	}

	protoProduct, err := productDetailsToProto(product)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to map product details to proto",
			"error", err,
			"product_id", productID,
		)

		return nil, mapServiceError(err)
	}

	h.logger.DebugContext(ctx, "got product",
		"product_id", productID,
	)

	return &catalogv1.GetProductResponse{
		Product: protoProduct,
	}, nil
}

// ValidateProductVariant validates a product and variant pairing for internal
// service-to-service consumers such as basket-service.
//
// Basket Service uses this endpoint when adding an item to a basket. Catalog
// Service returns a small authoritative snapshot without handing over ownership
// of product truth.
func (h *CatalogHandler) ValidateProductVariant(
	ctx context.Context,
	req *catalogv1.ValidateProductVariantRequest,
) (*catalogv1.ValidateProductVariantResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "validate product variant request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	productID := strings.TrimSpace(req.GetProductId())
	if productID == "" {
		h.logger.WarnContext(ctx, "validate product variant request missing product id",
			"variant_id", req.GetVariantId(),
		)

		return nil, status.Error(codes.InvalidArgument, "missing product id")
	}

	variantID := strings.TrimSpace(req.GetVariantId())
	if variantID == "" {
		h.logger.WarnContext(ctx, "validate product variant request missing variant id",
			"product_id", productID,
		)

		return nil, status.Error(codes.InvalidArgument, "missing variant id")
	}

	if h.productVariantValidator == nil {
		h.logger.ErrorContext(ctx, "product variant validator is not configured",
			"product_id", productID,
			"variant_id", variantID,
		)

		return nil, status.Error(codes.Internal, "product variant validator is not configured")
	}

	validatedVariant, err := h.productVariantValidator.ValidateProductVariant(
		ctx,
		catalog.ValidateProductVariantQuery{
			ProductID: catalog.ProductID(productID),
			VariantID: catalog.VariantID(variantID),
		},
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to validate product variant",
			"error", err,
			"product_id", productID,
			"variant_id", variantID,
		)

		return nil, mapServiceError(err)
	}

	response, err := validatedProductVariantToProto(validatedVariant)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to map validated product variant to proto",
			"error", err,
			"product_id", productID,
			"variant_id", variantID,
		)

		return nil, status.Error(codes.Internal, "failed to map validated product variant")
	}

	h.logger.DebugContext(ctx, "validated product variant",
		"product_id", response.GetProductId(),
		"variant_id", response.GetVariantId(),
		"sellable", response.GetSellable(),
		"currency_code", response.GetUnitPrice().GetCurrencyCode(),
	)

	return response, nil
}

// ListCategories returns list of product categories matching filter criteria.
func (h *CatalogHandler) ListCategories(
	ctx context.Context,
	req *catalogv1.ListCategoriesRequest,
) (*catalogv1.ListCategoriesResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "list categories request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := h.catalogService.ListCategories(ctx, catalog.ListCategoriesFilter{
		ParentCategoryID: catalog.CategoryID(req.GetParentCategoryId()),
		IncludeInactive:  req.GetIncludeInactive(),
		PageSize:         int(req.GetPage().GetPageSize()),
		PageToken:        req.GetPage().GetPageToken(),
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list categories",
			"error", err,
			"parent_category_id", req.GetParentCategoryId(),
			"include_inactive", req.GetIncludeInactive(),
		)

		return nil, mapServiceError(err)
	}

	categories := result.Result
	token := result.NextPageToken

	response := &catalogv1.ListCategoriesResponse{
		Categories: make([]*catalogv1.Category, 0, len(categories)),
		Page: &commonv1.PageResponse{
			NextPageToken: token,
			TotalCount:    0, // Not calculated. Set to 0 as default.
		},
	}

	for _, category := range categories {
		categoryProto, err := categoryToProto(&category)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to map category to proto",
				"error", err,
				"category_id", category.CategoryID,
			)

			return nil, mapServiceError(err)
		}

		response.Categories = append(response.Categories, categoryProto)
	}

	h.logger.DebugContext(ctx, "listed categories",
		"category_count", len(response.GetCategories()),
		"next_page_token_present", response.GetPage().GetNextPageToken() != "",
	)

	return response, nil
}

// ListProductAttributeDefinitions returns list of product attribute definitions
// matching filter criteria.
func (h *CatalogHandler) ListProductAttributeDefinitions(
	ctx context.Context,
	req *catalogv1.ListProductAttributeDefinitionsRequest,
) (*catalogv1.ListProductAttributeDefinitionsResponse, error) {
	if req == nil {
		h.logger.WarnContext(ctx, "list product attribute definitions request is nil")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	categoryID := strings.TrimSpace(req.GetCategoryId())
	if categoryID == "" {
		h.logger.WarnContext(ctx, "list product attribute definitions request missing category id")
		return nil, status.Error(codes.InvalidArgument, "missing category id")
	}

	result, err := h.catalogService.ListProductAttributeDefinitions(ctx,
		catalog.ListProductAttributeDefinitionsFilter{
			CategoryID:      catalog.CategoryID(categoryID),
			IsFilterable:    req.GetFilterableOnly(),
			IncludeInactive: req.GetIncludeInactive(),
			PageSize:        int(req.GetPage().GetPageSize()),
			PageToken:       req.GetPage().GetPageToken(),
		})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list product attribute definitions",
			"error", err,
			"category_id", categoryID,
			"filterable_only", req.GetFilterableOnly(),
			"include_inactive", req.GetIncludeInactive(),
		)

		return nil, mapServiceError(err)
	}

	productAttributeDefinitions := result.Result

	response := &catalogv1.ListProductAttributeDefinitionsResponse{
		AttributeDefinitions: make([]*catalogv1.ProductAttributeDefinition, 0, len(productAttributeDefinitions)),
		Page: &commonv1.PageResponse{
			NextPageToken: result.NextPageToken,
			TotalCount:    0, // Not calculated. Set to 0 as default.
		},
	}

	for _, attributeDefinition := range productAttributeDefinitions {
		attributeDefinitionProto, err := productAttributeDefinitionToProto(&attributeDefinition)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to map product attribute definition to proto",
				"error", err,
				"attribute_id", attributeDefinition.AttributeID,
				"category_id", attributeDefinition.CategoryID,
			)

			return nil, mapServiceError(err)
		}

		response.AttributeDefinitions = append(response.AttributeDefinitions, attributeDefinitionProto)
	}

	h.logger.DebugContext(ctx, "listed product attribute definitions",
		"category_id", categoryID,
		"attribute_definition_count", len(response.GetAttributeDefinitions()),
		"next_page_token_present", response.GetPage().GetNextPageToken() != "",
	)

	return response, nil
}
