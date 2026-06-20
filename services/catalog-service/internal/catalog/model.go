package catalog

import (
	"errors"
	"time"
)

var (
	ErrInvalidVariantID              = errors.New("invalid variant id")
	ErrProductVariantNotFound        = errors.New("product variant not found")
	ErrProductVariantProductMismatch = errors.New("product variant does not belong to product")
	ErrProductNotSellable            = errors.New("product is not sellable")
	ErrProductVariantNotSellable     = errors.New("product variant is not sellable")
)

// Money represents a monetary value in minor units.
//
// This mirrors the Protobuf Money contract.
type Money struct {
	AmountMinor  int64
	CurrencyCode string
}

// ProductID represents a catalog product identifier.
type ProductID string

// VariantID represents the identifier of a catalog product variant.
type VariantID string

// ImageID represents a catalog product image identifier.
type ImageID string

// ProductAttributeValueID represents the identifier of a product attribute value.
type ProductAttributeValueID string

// Sku represents the a product's SKU.
type Sku string

// CategoryID represents a catalog category identifier.
type CategoryID string

// AttributeID represents a catalog product attribute identifier.
type AttributeID string

// OptionID represents a catalog product attribute option identifier.
type OptionID string

// CatalogID interface represents various CatalogID types.
type CatalogID interface {
	ProductID |
		CategoryID |
		ImageID |
		VariantID |
		AttributeID |
		OptionID |
		Sku |
		ProductAttributeValueID
}

// Product is the internal domain representation of a catalogue product.
type Product struct {
	ProductID   ProductID
	CategoryID  CategoryID
	Name        string
	Slug        string
	Description string
	Brand       string
	Status      ProductStatus
	BasePrice   Money
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Variants    []*ProductVariant
	Attributes  []*ProductAttributeValue
	Images      []*ProductImage
}

// Category is the internal domain representation of a product category.
type Category struct {
	CategoryID       CategoryID
	ParentCategoryID *CategoryID
	Name             string
	Slug             string
	Description      string
	Status           CategoryStatus
	DisplayOrder     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ProductVariant represents a purchaseable product variation.
type ProductVariant struct {
	VariantID   VariantID
	ProductID   ProductID
	Sku         Sku
	VariantName string
	Status      ProductVariantStatus
	Price       Money
	Attributes  []*ProductAttributeValue
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProductAttributeDefinition defines an attribute that is valid for a category.
type ProductAttributeDefinition struct {
	AttributeID       AttributeID
	CategoryID        CategoryID
	Code              string
	DisplayName       string
	Description       string
	DataType          ProductAttributeDataType
	Unit              string
	IsRequired        bool
	IsFilterable      bool
	IsVariantDefining bool
	Options           []*ProductAttributeOption
	Status            ProductAttributeDefinitionStatus
	CreatedAt         time.Time // used for pagination
}

// ProductAttributeOption represents a controlled allowed value for an attribute
// definition.
type ProductAttributeOption struct {
	OptionID     OptionID
	Value        string
	DisplayName  string
	DisplayOrder int
	Status       ProductAttributeOptionStatus
}

// ProductAttributeValue represents a product-specific or a variant-specific
// value for a defined attribute.
type ProductAttributeValue struct {
	ProductAttributeValueID ProductAttributeValueID
	ProductID               ProductID
	VariantID               *VariantID
	AttributeID             AttributeID

	ValueString  string
	ValueNumber  string
	ValueBoolean *bool
	ValueJSON    []byte
	Unit         string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProductImage represents customer-facing catalog imagery.
type ProductImage struct {
	ImageID      ImageID
	ProductID    ProductID
	Url          string
	AltText      string
	DisplayOrder int
}

// ProductDetails represents a fully hydrated version of a catalog product.
//
// This product view is provided when a call to GetProduct is made.
type ProductDetails struct {
	ProductID   ProductID
	CategoryID  CategoryID
	Name        string
	Slug        string
	Description string
	Brand       string
	Status      ProductStatus
	BasePrice   Money
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Variants   []*ProductVariantDetails
	Attributes []*ProductAttributeValueDetails
	Images     []*ProductImage
}

// ProductVariantDetails represents a fully hydrated version of a catalog product
// variant.
//
// This view is provided when a call to GetProduct is made.
type ProductVariantDetails struct {
	VariantID   VariantID
	ProductID   ProductID
	Sku         Sku
	VariantName string
	Status      ProductVariantStatus
	Price       Money
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Attributes []*ProductAttributeValueDetails
}

// ProductAttributeValueDetails represents a fully hydrated version of a product
// or variant attribute value.
type ProductAttributeValueDetails struct {
	ProductAttributeValueID ProductAttributeValueID
	ProductID               ProductID
	VariantID               *VariantID
	AttributeID             AttributeID

	Code        string
	DisplayName string
	DataType    ProductAttributeDataType
	Options     []*ProductAttributeOption

	ValueString  string
	ValueNumber  string
	ValueBoolean *bool
	ValueJSON    []byte
	Unit         string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ValidateProductVariantQuery identifies a product and variant pair that another
// internal service wants to use.
//
// Basket Service uses this query when adding an item to a basket.
type ValidateProductVariantQuery struct {
	ProductID ProductID
	VariantID VariantID
}

// ValidatedProductVariant is the small catalog-owned snapshot returned to
// service-to-service callers such as Basket Service.
//
// Basket Service may store these values as a basket-line snapshot, but Catalog
// Service remains the owner of product and variant truth. Order placement should
// revalidate before payment.
type ValidatedProductVariant struct {
	ProductID   ProductID
	VariantID   VariantID
	ProductName string
	VariantName string

	ProductStatus ProductStatus
	VariantStatus ProductVariantStatus

	UnitPrice Money
	Sellable  bool
}

// ListProductsFilter defines filter for product listing.
type ListProductsFilter struct {
	CategoryID      CategoryID
	IncludeInactive bool
	PageSize        int
	PageToken       string
}

// ListCategoriesFilter defines filter for category listing.
type ListCategoriesFilter struct {
	ParentCategoryID CategoryID
	IncludeInactive  bool
	PageSize         int
	PageToken        string
}

// ListProductAttributeDefinitionsFilter defines filter for product attribute
// definitions.
type ListProductAttributeDefinitionsFilter struct {
	// Required category ID.
	CategoryID      CategoryID
	IsFilterable    bool
	IncludeInactive bool
	PageSize        int
	PageToken       string
}
