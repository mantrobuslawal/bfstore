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
  KEY idx_baskets_updated_at (updated_at)
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
  UNIQUE KEY uk_basket_items_basket_product_variant (basket_id, product_id, variant_id),
  KEY idx_basket_items_basket_id (basket_id),
  KEY idx_basket_items_product_id (product_id),
  KEY idx_basket_items_variant_id (variant_id),

  CONSTRAINT fk_basket_items_basket_id
    FOREIGN KEY (basket_id)
    REFERENCES baskets (basket_id)
    ON DELETE CASCADE
);
