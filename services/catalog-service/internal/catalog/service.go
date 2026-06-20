package catalog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Service contains Catalog Service business logic.
type Service struct {
	repository Repository
	logger     *slog.Logger
}

// NewService creates a catalog service.
func NewService(repository Repository, logger *slog.Logger) *Service {
	if repository == nil {
		panic("catalog: nil repository")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repository: repository,
		logger:     logger.With("component", "catalog_service"),
	}
}

// CatalogQueryResult represents domain object returned by a catalog service query.
//
// Represents products, product categories and product attribute definiton objects.
type CatalogQueryResult interface {
	Product |
		Category |
		ProductAttributeDefinition
}

// ListResult represents the combination of a collection of catalog objects and
// next page token.
type ListResult[T CatalogQueryResult] struct {
	Result        []T
	NextPageToken string
}

// ListQuery represents the collection of catalog object id, search filter
// options, max page size and cursor for pagination of catalog results.
type ListQuery struct {
	ID            string
	FilterOptions []bool
	Limit         int
	Cursor        *catalogCursor
}

// catalogCursor represents object encoded to page token string for result
// pagination.
type catalogCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

// ListProducts returns customer-visible catalogue products.
func (s *Service) ListProducts(ctx context.Context, input ListProductsFilter) (ListResult[Product], error) {
	pageSize, err := normalisePageSize(input.PageSize)
	if err != nil {
		s.logger.WarnContext(ctx, "invalid product list page size",
			"page_size", input.PageSize,
		)

		return ListResult[Product]{}, ErrInvalidPageSize
	}

	var cursor *catalogCursor
	if strings.TrimSpace(input.PageToken) != "" {
		cursor, err = decodeCursor(input.PageToken)
		if err != nil {
			s.logger.WarnContext(ctx, "invalid product list page token",
				"error", err,
			)

			return ListResult[Product]{}, fmt.Errorf("invalid page token: %w", err)
		}
	}

	id, err := isValidCatalogID(input.CategoryID)
	if err != nil {
		s.logger.WarnContext(ctx, "invalid product list category id",
			"category_id", input.CategoryID,
		)

		return ListResult[Product]{}, ErrInvalidCategoryID
	}

	products, err := s.repository.ListProducts(ctx, ListQuery{
		ID:            id,
		FilterOptions: []bool{input.IncludeInactive},
		Limit:         pageSize + 1,
		Cursor:        cursor,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list products",
			"error", err,
			"category_id", input.CategoryID,
			"include_inactive", input.IncludeInactive,
		)

		return ListResult[Product]{}, fmt.Errorf("list products category id:%q :%w", input.CategoryID, err)
	}

	nextToken := ""
	hasMore := len(products) > pageSize

	if hasMore {
		products = products[:pageSize]
		last := products[len(products)-1]
		nextToken, err = encodeCursor(last)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to encode products next page token",
				"error", err,
				"category_id", input.CategoryID,
			)

			return ListResult[Product]{}, fmt.Errorf("encode page token: %w", err)
		}
	}

	s.logger.DebugContext(ctx, "listed products",
		"category_id", input.CategoryID,
		"product_count", len(products),
		"has_more", hasMore,
	)

	return ListResult[Product]{
		Result:        products,
		NextPageToken: nextToken,
	}, nil
}

// GetProduct returns a single product, its variants and attributes.
func (s *Service) GetProduct(ctx context.Context, productID ProductID) (ProductDetails, error) {
	id, err := isValidCatalogID(productID)
	if err != nil || strings.TrimSpace(id) == "" {
		s.logger.WarnContext(ctx, "invalid product id",
			"product_id", productID,
		)

		return ProductDetails{}, ErrInvalidProductID
	}

	product, err := s.repository.GetProduct(ctx, ProductID(id))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get product",
			"error", err,
			"product_id", id,
		)

		return ProductDetails{}, fmt.Errorf("get product %q: %w", id, err)
	}

	definitions, err := s.repository.ListProductAttributeDefinitions(ctx, ListQuery{
		ID:            string(product.CategoryID),
		FilterOptions: []bool{true, false},
		Limit:         500, // TODO: create ListAllProductAttributeDefinitionsForCategory.
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list attribute definitions for product category",
			"error", err,
			"product_id", product.ProductID,
			"category_id", product.CategoryID,
		)

		return ProductDetails{}, fmt.Errorf("list attribute defintions for category %q: %w", product.CategoryID, err)
	}

	details, err := hydrateProduct(product, definitions)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to hydrate product",
			"error", err,
			"product_id", product.ProductID,
		)

		return ProductDetails{}, fmt.Errorf("hydrate product %q: %w", product.ProductID, err)
	}

	s.logger.DebugContext(ctx, "got product details",
		"product_id", details.ProductID,
		"variant_count", len(details.Variants),
		"attribute_count", len(details.Attributes),
		"image_count", len(details.Images),
	)

	return details, nil
}

// ValidateProductVariant validates that a product and variant pairing exists and
// is currently sellable.
//
// Basket Service uses this method before adding a product variant to a basket.
// The returned snapshot is intentionally small and should not be treated as
// final order truth.
func (s *Service) ValidateProductVariant(
	ctx context.Context,
	query ValidateProductVariantQuery,
) (ValidatedProductVariant, error) {
	productID := strings.TrimSpace(string(query.ProductID))
	if productID == "" {
		s.logger.WarnContext(ctx, "validate product variant missing product id",
			"variant_id", query.VariantID,
		)

		return ValidatedProductVariant{}, ErrInvalidProductID
	}

	variantID := strings.TrimSpace(string(query.VariantID))
	if variantID == "" {
		s.logger.WarnContext(ctx, "validate product variant missing variant id",
			"product_id", query.ProductID,
		)

		return ValidatedProductVariant{}, ErrInvalidVariantID
	}

	validatedVariant, err := s.repository.ValidateProductVariant(ctx, ValidateProductVariantQuery{
		ProductID: ProductID(productID),
		VariantID: VariantID(variantID),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to validate product variant",
			"error", err,
			"product_id", productID,
			"variant_id", variantID,
		)

		return ValidatedProductVariant{}, fmt.Errorf("validate product variant product %q variant %q: %w", productID, variantID, err)
	}

	if string(validatedVariant.ProductStatus) != "active" {
		s.logger.WarnContext(ctx, "product is not sellable",
			"product_id", productID,
			"variant_id", variantID,
			"product_status", validatedVariant.ProductStatus,
		)

		return ValidatedProductVariant{}, ErrProductNotSellable
	}

	if string(validatedVariant.VariantStatus) != "active" {
		s.logger.WarnContext(ctx, "product variant is not sellable",
			"product_id", productID,
			"variant_id", variantID,
			"variant_status", validatedVariant.VariantStatus,
		)

		return ValidatedProductVariant{}, ErrProductVariantNotSellable
	}

	if strings.TrimSpace(validatedVariant.ProductName) == "" {
		return ValidatedProductVariant{}, fmt.Errorf("validated product variant product %q has empty product name", productID)
	}

	if strings.TrimSpace(validatedVariant.VariantName) == "" {
		return ValidatedProductVariant{}, fmt.Errorf("validated product variant %q has empty variant name", variantID)
	}

	if strings.TrimSpace(validatedVariant.UnitPrice.CurrencyCode) == "" {
		return ValidatedProductVariant{}, fmt.Errorf("validated product variant %q has empty currency code", variantID)
	}

	if validatedVariant.UnitPrice.AmountMinor < 0 {
		return ValidatedProductVariant{}, fmt.Errorf("validated product variant %q has negative unit price", variantID)
	}

	validatedVariant.Sellable = true

	s.logger.DebugContext(ctx, "validated product variant",
		"product_id", validatedVariant.ProductID,
		"variant_id", validatedVariant.VariantID,
		"currency_code", validatedVariant.UnitPrice.CurrencyCode,
		"amount_minor", validatedVariant.UnitPrice.AmountMinor,
	)

	return validatedVariant, nil
}

func hydrateProduct(product Product, definitions []ProductAttributeDefinition) (ProductDetails, error) {
	definitionsByID := make(map[AttributeID]ProductAttributeDefinition, len(definitions))

	for _, definition := range definitions {
		definitionsByID[definition.AttributeID] = definition
	}

	details := ProductDetails{
		ProductID:   product.ProductID,
		CategoryID:  product.CategoryID,
		Name:        product.Name,
		Slug:        product.Slug,
		Description: product.Description,
		Brand:       product.Brand,
		Status:      product.Status,
		BasePrice:   product.BasePrice,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
		Images:      product.Images,
	}

	for _, value := range product.Attributes {
		if value.VariantID != nil {
			continue
		}

		hydrated, err := hydrateProductAttributeValue(value, definitionsByID)
		if err != nil {
			return ProductDetails{}, err
		}

		details.Attributes = append(details.Attributes, hydrated)
	}

	for _, variant := range product.Variants {
		variantDetails := &ProductVariantDetails{
			VariantID:   variant.VariantID,
			ProductID:   variant.ProductID,
			Sku:         variant.Sku,
			VariantName: variant.VariantName,
			Status:      variant.Status,
			Price:       variant.Price,
			CreatedAt:   variant.CreatedAt,
			UpdatedAt:   variant.UpdatedAt,
		}

		for _, value := range product.Attributes {
			if value.VariantID == nil {
				continue
			}

			if variant.VariantID != *value.VariantID {
				continue
			}

			hydrated, err := hydrateProductAttributeValue(value, definitionsByID)
			if err != nil {
				return ProductDetails{}, err
			}

			variantDetails.Attributes = append(variantDetails.Attributes, hydrated)
		}

		details.Variants = append(details.Variants, variantDetails)
	}

	return details, nil
}

func hydrateProductAttributeValue(
	value *ProductAttributeValue,
	definitionsByID map[AttributeID]ProductAttributeDefinition,
) (*ProductAttributeValueDetails, error) {
	if value == nil {
		return nil, fmt.Errorf("nil product attribute value")
	}

	definition, ok := definitionsByID[value.AttributeID]
	if !ok {
		return nil, fmt.Errorf("missing attribute definition %q", value.AttributeID)
	}

	return &ProductAttributeValueDetails{
		ProductAttributeValueID: value.ProductAttributeValueID,
		ProductID:               value.ProductID,
		VariantID:               value.VariantID,
		AttributeID:             value.AttributeID,

		Code:        definition.Code,
		DisplayName: definition.DisplayName,
		DataType:    definition.DataType,
		Options:     definition.Options,

		ValueString:  value.ValueString,
		ValueNumber:  value.ValueNumber,
		ValueBoolean: value.ValueBoolean,
		ValueJSON:    value.ValueJSON,
		Unit:         value.Unit,

		CreatedAt: value.CreatedAt,
		UpdatedAt: value.UpdatedAt,
	}, nil
}

// ListCategories returns customer-visible catalog categories.
func (s *Service) ListCategories(ctx context.Context, input ListCategoriesFilter) (ListResult[Category], error) {
	pageSize, err := normalisePageSize(input.PageSize)
	if err != nil {
		s.logger.WarnContext(ctx, "invalid category list page size",
			"page_size", input.PageSize,
		)

		return ListResult[Category]{}, ErrInvalidPageSize
	}

	var cursor *catalogCursor
	if strings.TrimSpace(input.PageToken) != "" {
		cursor, err = decodeCursor(input.PageToken)
		if err != nil {
			s.logger.WarnContext(ctx, "invalid category list page token",
				"error", err,
			)

			return ListResult[Category]{}, fmt.Errorf("invalid page token: %w", err)
		}
	}

	id, err := isValidCatalogID(input.ParentCategoryID)
	if err != nil {
		s.logger.WarnContext(ctx, "invalid parent category id",
			"parent_category_id", input.ParentCategoryID,
		)

		return ListResult[Category]{}, ErrInvalidCategoryID
	}

	categories, err := s.repository.ListCategories(ctx, ListQuery{
		ID:            id,
		FilterOptions: []bool{input.IncludeInactive},
		Limit:         pageSize + 1,
		Cursor:        cursor,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list categories",
			"error", err,
			"parent_category_id", input.ParentCategoryID,
			"include_inactive", input.IncludeInactive,
		)

		return ListResult[Category]{}, fmt.Errorf("list categories: %w", err)
	}

	nextToken := ""
	hasMore := len(categories) > pageSize

	if hasMore {
		categories = categories[:pageSize]
		last := categories[len(categories)-1]
		nextToken, err = encodeCursor(last)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to encode categories next page token",
				"error", err,
				"parent_category_id", input.ParentCategoryID,
			)

			return ListResult[Category]{}, fmt.Errorf("create next page token: %w", err)
		}
	}

	s.logger.DebugContext(ctx, "listed categories",
		"parent_category_id", input.ParentCategoryID,
		"category_count", len(categories),
		"has_more", hasMore,
	)

	return ListResult[Category]{
		Result:        categories,
		NextPageToken: nextToken,
	}, nil
}

// ListProductAttributeDefinitions returns catalog product attribute definitions.
func (s *Service) ListProductAttributeDefinitions(
	ctx context.Context,
	input ListProductAttributeDefinitionsFilter,
) (ListResult[ProductAttributeDefinition], error) {
	pageSize, err := normalisePageSize(input.PageSize)
	if err != nil {
		s.logger.WarnContext(ctx, "invalid product attribute definitions page size",
			"page_size", input.PageSize,
			"category_id", input.CategoryID,
		)

		return ListResult[ProductAttributeDefinition]{}, ErrInvalidPageSize
	}

	var cursor *catalogCursor
	if strings.TrimSpace(input.PageToken) != "" {
		cursor, err = decodeCursor(input.PageToken)
		if err != nil {
			s.logger.WarnContext(ctx, "invalid product attribute definitions page token",
				"error", err,
				"category_id", input.CategoryID,
			)

			return ListResult[ProductAttributeDefinition]{}, fmt.Errorf("invalid page token: %w", err)
		}
	}

	id, err := isValidCatalogID(input.CategoryID)
	if err != nil {
		s.logger.WarnContext(ctx, "invalid product attribute definition category id",
			"category_id", input.CategoryID,
		)

		return ListResult[ProductAttributeDefinition]{}, ErrInvalidCategoryID
	}

	attributeDefinitions, err := s.repository.ListProductAttributeDefinitions(ctx, ListQuery{
		ID:            id,
		FilterOptions: []bool{input.IncludeInactive, input.IsFilterable},
		Limit:         pageSize + 1,
		Cursor:        cursor,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list product attribute definitions",
			"error", err,
			"category_id", input.CategoryID,
			"include_inactive", input.IncludeInactive,
			"is_filterable", input.IsFilterable,
		)

		return ListResult[ProductAttributeDefinition]{}, fmt.Errorf("list product attribute definitions: %w", err)
	}

	nextToken := ""
	hasMore := len(attributeDefinitions) > pageSize

	if hasMore {
		attributeDefinitions = attributeDefinitions[:pageSize]
		last := attributeDefinitions[len(attributeDefinitions)-1]
		nextToken, err = encodeCursor(last)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to encode product attribute definitions next page token",
				"error", err,
				"category_id", input.CategoryID,
			)

			return ListResult[ProductAttributeDefinition]{}, fmt.Errorf("create next page token: %w", err)
		}
	}

	s.logger.DebugContext(ctx, "listed product attribute definitions",
		"category_id", input.CategoryID,
		"attribute_definition_count", len(attributeDefinitions),
		"has_more", hasMore,
	)

	return ListResult[ProductAttributeDefinition]{
		Result:        attributeDefinitions,
		NextPageToken: nextToken,
	}, nil
}

// Helper functions for PageSize and PageToken.

func normalisePageSize(size int) (int, error) {
	if size <= 0 {
		return defaultPageSize, nil
	}

	if size > maxPageSize {
		return 0, ErrInvalidPageSize
	}

	return size, nil
}

func decodeCursor(pageToken string) (*catalogCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(pageToken)
	if err != nil {
		return nil, err
	}

	var token catalogCursor
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	if token.ID == "" {
		return nil, errors.New("invalid field in catalog cursor")
	}

	if token.CreatedAt.IsZero() {
		return nil, errors.New("invalid field in catalog cursor")
	}

	return &token, nil
}

func encodeCursor[T CatalogQueryResult](c T) (string, error) {
	var token catalogCursor

	product, ok := any(c).(Product)
	if ok {
		token.CreatedAt = product.CreatedAt
		token.ID = string(product.ProductID)
	}

	category, ok := any(c).(Category)
	if ok {
		token.CreatedAt = category.CreatedAt
		token.ID = string(category.CategoryID)
	}

	pad, ok := any(c).(ProductAttributeDefinition)
	if ok {
		token.CreatedAt = pad.CreatedAt
		token.ID = string(pad.AttributeID)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("marshal struct to json: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

// TODO: Implement proper validation.
func isValidCatalogID[T CatalogID](id T) (string, error) {
	return strings.TrimSpace(string(id)), nil
}
