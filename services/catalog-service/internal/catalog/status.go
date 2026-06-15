package catalog

import "strings"

// LifecycleStatus interface represents life phases of catalog objects such as
// products, product categories and product variants.
type LifecycleStatus interface {
	ProductStatus |
		CategoryStatus |
		ProductVariantStatus |
		ProductAttributeDefinitionStatus |
		ProductAttributeOptionStatus
}

func isKnownLifecycleStatus[S LifecycleStatus](status S) bool {
	switch string(status) {
	case "draft", "active", "inactive", "archived":
		return true
	default:
		return false
	}
}

// ProductStatus is the domain representation of a
// product's lifecycle.
type ProductStatus string

// ParseToProductStatus converts a valid string value to a LifecycleStatus.
// Returns an ErrInvalidLifecycleStatus error when the input string cannot
// be matched to a valid LifecycleStatus.
func ParseToProductStatus(status string) (ProductStatus, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "draft":
		return ProductStatusDraft, nil
	case "active":
		return ProductStatusActive, nil
	case "inactive":
		return ProductStatusInactive, nil
	case "archived":
		return ProductStatusArchived, nil

	default:
		return "", ErrInvalidLifecycleStatus
	}
}

// Valid ProductStatus values.
const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusArchived ProductStatus = "archived"
)

// IsValid returns true if the status is valid and false otherwise.
func (s ProductStatus) IsValid() bool {
	return isKnownLifecycleStatus(s)
}

// String converts a status from a LifecycleStatus to a string.
func (s ProductStatus) String() string {
	return string(s)
}

// CategoryStatus is the domain representation of the
// product category lifecycle.
type CategoryStatus string

// Valid CategoryStatus values.
const (
	CategoryStatusDraft    CategoryStatus = "draft"
	CategoryStatusActive   CategoryStatus = "active"
	CategoryStatusInactive CategoryStatus = "inactive"
	CategoryStatusArchived CategoryStatus = "archived"
)

// IsValid returns true if the status is valid and false otherwise.
func (c CategoryStatus) IsValid() bool {
	return isKnownLifecycleStatus(c)
}

// String converts a status from a LifecycleStatus to a string.
func (c CategoryStatus) String() string {
	return string(c)
}

// ParseToCategoryStatus converts a valid string value to a LifecycleStatus.
// Returns an ErrInvalidLifecycleStatus error when the input string cannot
// be matched to a valid LifecycleStatus.
func ParseToCategoryStatus(status string) (CategoryStatus, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "draft":
		return CategoryStatusDraft, nil
	case "active":
		return CategoryStatusActive, nil
	case "inactive":
		return CategoryStatusInactive, nil
	case "archived":
		return CategoryStatusArchived, nil

	default:
		return "", ErrInvalidLifecycleStatus
	}
}

// ProductVariantStatus is the domain representation of a
// products variant's lifecycle.
type ProductVariantStatus string

// Valid ProductVariantStatus values.
const (
	ProductVariantStatusDraft    ProductVariantStatus = "draft"
	ProductVariantStatusActive   ProductVariantStatus = "active"
	ProductVariantStatusInactive ProductVariantStatus = "inactive"
	ProductVariantStatusArchived ProductVariantStatus = "archived"
)

// ParseToProductVariantStatus converts a valid string value to a LifecycleStatus.
// Returns an ErrInvalidLifecycleStatus error when the input string cannot
// be matched to a valid LifecycleStatus.
func ParseToProductVariantStatus(status string) (ProductVariantStatus, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "draft":
		return ProductVariantStatusDraft, nil
	case "active":
		return ProductVariantStatusActive, nil
	case "inactive":
		return ProductVariantStatusInactive, nil
	case "archived":
		return ProductVariantStatusArchived, nil

	default:
		return "", ErrInvalidLifecycleStatus
	}
}

// IsValid returns true if the status is valid and false otherwise.
func (v ProductVariantStatus) IsValid() bool {
	return isKnownLifecycleStatus(v)
}

// String converts a status from a LifecycleStatus to a string.
func (v ProductVariantStatus) String() string {
	return string(v)
}

// ProductAttributeDefinitionStatus is the domain representation of a
// product attribute's lifecycle.
type ProductAttributeDefinitionStatus string

// Valid ProductAttributeDefintionStatus values.
const (
	ProductAttributeDefinitionStatusDraft    ProductAttributeDefinitionStatus = "draft"
	ProductAttributeDefinitionStatusActive   ProductAttributeDefinitionStatus = "active"
	ProductAttributeDefinitionStatusInactive ProductAttributeDefinitionStatus = "inactive"
	ProductAttributeDefinitionStatusArchived ProductAttributeDefinitionStatus = "archived"
)

// ParseToProductAttributeDefinitionStatus converts a valid string value to a LifecycleStatus.
// Returns an ErrInvalidLifecycleStatus error when the input string cannot
// be matched to a valid LifecycleStatus.
func ParseToProductAttributeDefinitionStatus(status string) (ProductAttributeDefinitionStatus, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "draft":
		return ProductAttributeDefinitionStatusDraft, nil
	case "active":
		return ProductAttributeDefinitionStatusActive, nil
	case "inactive":
		return ProductAttributeDefinitionStatusInactive, nil
	case "archived":
		return ProductAttributeDefinitionStatusArchived, nil

	default:
		return "", ErrInvalidLifecycleStatus
	}
}

// IsValid returns true if the status is valid and false otherwise.
func (s ProductAttributeDefinitionStatus) IsValid() bool {
	return isKnownLifecycleStatus(s)
}

// String converts a status from a LifecycleStatus to a string.
func (s ProductAttributeDefinitionStatus) String() string {
	return string(s)
}

// ProductAttributeOptionStatus is the domain representation of a
// product attribute option's lifecycle.
type ProductAttributeOptionStatus string

// Valid ProductAttrubuteOptions values.
const (
	ProductAttributeOptionsStatusDraft    ProductAttributeOptionStatus = "draft"
	ProductAttributeOptionsStatusActive   ProductAttributeOptionStatus = "active"
	ProductAttributeOptionsStatusInactive ProductAttributeOptionStatus = "inactive"
	ProductAttributeOptionsStatusArchived ProductAttributeOptionStatus = "archived"
)

// ParseToProductAttributeOptionStatus converts a valid string value to a LifecycleStatus.
// Returns an ErrInvalidLifecycleStatus error when the input string cannot
// be matched to a valid LifecycleStatus.
func ParseToProductAttributeOptionStatus(status string) (ProductAttributeOptionStatus, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "draft":
		return ProductAttributeOptionsStatusDraft, nil
	case "active":
		return ProductAttributeOptionsStatusActive, nil
	case "inactive":
		return ProductAttributeOptionsStatusInactive, nil
	case "archived":
		return ProductAttributeOptionsStatusArchived, nil

	default:
		return "", ErrInvalidLifecycleStatus
	}
}

// IsValid returns true if the status is valid and false otherwise.
func (o ProductAttributeOptionStatus) IsValid() bool {
	return isKnownLifecycleStatus(o)
}

// String converts a status from a LifecycleStatus to a string.
func (o ProductAttributeOptionStatus) String() string {
	return string(o)
}
