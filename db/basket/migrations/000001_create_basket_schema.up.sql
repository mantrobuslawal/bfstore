CREATE TABLE baskets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  basket_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  currency_code CHAR(3) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (id),

  UNIQUE KEY uk_baskets_basket_id (basket_id),
  KEY idx_baskets_status (status),
  KEY idx_baskets_updated_at (updated_at),

  CONSTRAINT chk_baskets_basket_id_format
    CHECK (basket_id REGEXP '^basket_[0-9A-HJKMNP-TV-Z]{26}_[0-9A-HJKMNP-TV-Z]{8}$'),

  CONSTRAINT chk_baskets_status
    CHECK (status IN ('ACTIVE', 'CLEARED', 'EXPIRED', 'CHECKED_OUT')),

  CONSTRAINT chk_baskets_currency_code
    CHECK (currency_code REGEXP '^[A-Z]{3}$')
);

CREATE TABLE basket_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  basket_item_id VARCHAR(64) NOT NULL,
  basket_id VARCHAR(64) NOT NULL,
  product_id VARCHAR(64) NOT NULL,
  variant_id VARCHAR(64) NOT NULL,
  product_name_snapshot VARCHAR(255) NOT NULL,
  variant_name_snapshot VARCHAR(255) NOT NULL,
  quantity INT UNSIGNED NOT NULL,
  unit_price_minor_units BIGINT NOT NULL,
  line_total_minor_units BIGINT NOT NULL,
  currency_code CHAR(3) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (id),

  UNIQUE KEY uk_basket_items_basket_item_id (basket_item_id),
  UNIQUE KEY uk_basket_items_basket_product_variant (
    basket_id,
    product_id,
    variant_id
  ),

  KEY idx_basket_items_basket_id (basket_id),
  KEY idx_basket_items_product_id (product_id),
  KEY idx_basket_items_variant_id (variant_id),

  CONSTRAINT fk_basket_items_basket_id
    FOREIGN KEY (basket_id)
    REFERENCES baskets (basket_id)
    ON DELETE CASCADE,

  CONSTRAINT chk_basket_items_basket_item_id_format
    CHECK (basket_item_id REGEXP '^bitem_[0-9A-HJKMNP-TV-Z]{26}_[0-9A-HJKMNP-TV-Z]{8}$'),

  CONSTRAINT chk_basket_items_basket_id_format
    CHECK (basket_id REGEXP '^basket_[0-9A-HJKMNP-TV-Z]{26}_[0-9A-HJKMNP-TV-Z]{8}$'),

  CONSTRAINT chk_basket_items_product_id_format
    CHECK (product_id REGEXP '^prod_[0-9A-HJKMNP-TV-Z]{26}$'),

  CONSTRAINT chk_basket_items_variant_id_format
    CHECK (variant_id REGEXP '^var_[0-9A-HJKMNP-TV-Z]{26}$'),

  CONSTRAINT chk_basket_items_quantity
    CHECK (quantity BETWEEN 1 AND 99),

  CONSTRAINT chk_basket_items_unit_price_non_negative
    CHECK (unit_price_minor_units >= 0),

  CONSTRAINT chk_basket_items_line_total_non_negative
    CHECK (line_total_minor_units >= 0),

  CONSTRAINT chk_basket_items_currency_code
    CHECK (currency_code REGEXP '^[A-Z]{3}$')
);
