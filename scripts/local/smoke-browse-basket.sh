#!/usr/bin/env bash
set -euo pipefail

# bfstore browse -> basket smoke test
#
# Purpose:
#   Exercise the current two-service local flow:
#     Catalog Service -> Basket Service
#
# What it checks:
#   - containers are visible to Docker Compose
#   - catalog and basket gRPC reflection/health respond
#   - catalog database can be seeded
#   - catalog list/get/validate endpoints work
#   - basket create/get/add/update/remove/clear endpoints work
#
# Requirements on the host:
#   docker
#   docker compose
#   make
#   mysql client
#   grpcurl
#   jq

COMPOSE_PROJECT="${COMPOSE_PROJECT:-bfstore}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yaml}"

CATALOG_ADDR="${CATALOG_ADDR:-localhost:50051}"
BASKET_ADDR="${BASKET_ADDR:-localhost:50052}"

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"

CATALOG_DB_NAME="${CATALOG_DB_NAME:-bfstore_catalog}"
CATALOG_DB_USER="${CATALOG_DB_USER:-bfstore_catalog}"
CATALOG_DB_PASSWORD="${CATALOG_DB_PASSWORD:-bfstore_catalog_password}"

CATALOG_SEED_FILE="${CATALOG_SEED_FILE:-db/catalog/seeds/002_seed_borough_catalog_large.sql}"

RUN_MIGRATIONS="${RUN_MIGRATIONS:-true}"
RUN_SEED="${RUN_SEED:-true}"

PRODUCT_ID="${PRODUCT_ID:-}"
VARIANT_ID="${VARIANT_ID:-}"

require_command() {
  local command_name="$1"

  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Missing required command: ${command_name}" >&2
    exit 1
  fi
}

section() {
  echo ""
  echo "============================================================"
  echo "$1"
  echo "============================================================"
}

step() {
  echo ""
  echo "▶ $1"
}

run_grpc() {
  grpcurl -plaintext "$@"
}

run_mysql_catalog() {
  mysql \
    --batch \
    --raw \
    --skip-column-names \
    -h "${MYSQL_HOST}" \
    -P "${MYSQL_PORT}" \
    -u"${CATALOG_DB_USER}" \
    -p"${CATALOG_DB_PASSWORD}" \
    "${CATALOG_DB_NAME}" \
    "$@"
}

extract_basket_id() {
  jq -er '.basket.basketId // .basket.basket_id'
}

extract_first_basket_item_id() {
  jq -er '.basket.basketItems[0].basketItemId // .basket.basket_items[0].basket_item_id'
}

assert_json_equals() {
  local json="$1"
  local jq_expression="$2"
  local expected="$3"
  local actual

  actual="$(jq -er "${jq_expression}" <<<"${json}")"

  if [[ "${actual}" != "${expected}" ]]; then
    echo "Assertion failed: ${jq_expression}" >&2
    echo "Expected: ${expected}" >&2
    echo "Actual:   ${actual}" >&2
    echo "JSON:" >&2
    echo "${json}" | jq . >&2
    exit 1
  fi
}

assert_json_number_gte() {
  local json="$1"
  local jq_expression="$2"
  local minimum="$3"
  local actual

  actual="$(jq -er "${jq_expression}" <<<"${json}")"

  if (( actual < minimum )); then
    echo "Assertion failed: ${jq_expression}" >&2
    echo "Expected >= ${minimum}" >&2
    echo "Actual: ${actual}" >&2
    echo "JSON:" >&2
    echo "${json}" | jq . >&2
    exit 1
  fi
}

section "Preflight"

require_command docker
require_command make
require_command mysql
require_command grpcurl
require_command jq

step "Show Compose services"
docker compose -p "${COMPOSE_PROJECT}" -f "${COMPOSE_FILE}" ps

if [[ "${RUN_MIGRATIONS}" == "true" ]]; then
  section "Database migrations"

  step "Apply catalog migrations"
  make catalog-db-migrate-up

  step "Show catalog migration version"
  make catalog-db-migrate-version

  step "Apply basket migrations"
  make basket-db-migrate-up

  step "Show basket migration version"
  make basket-db-migrate-version
else
  echo "Skipping migrations because RUN_MIGRATIONS=${RUN_MIGRATIONS}"
fi

if [[ "${RUN_SEED}" == "true" ]]; then
  section "Catalog seed data"

  if [[ ! -f "${CATALOG_SEED_FILE}" ]]; then
    echo "Catalog seed file not found: ${CATALOG_SEED_FILE}" >&2
    exit 1
  fi

  step "Seed catalog database"
  mysql \
    -h "${MYSQL_HOST}" \
    -P "${MYSQL_PORT}" \
    -u"${CATALOG_DB_USER}" \
    -p"${CATALOG_DB_PASSWORD}" \
    "${CATALOG_DB_NAME}" < "${CATALOG_SEED_FILE}"

  step "Check seed counts"
  category_count="$(run_mysql_catalog -e "SELECT COUNT(*) FROM categories WHERE category_id LIKE 'cat_%';")"
  product_count="$(run_mysql_catalog -e "SELECT COUNT(*) FROM products WHERE product_id LIKE 'prod_%';")"
  variant_count="$(run_mysql_catalog -e "SELECT COUNT(*) FROM product_variants WHERE variant_id LIKE 'var_%';")"

  echo "categories=${category_count}"
  echo "products=${product_count}"
  echo "variants=${variant_count}"

  if [[ "${category_count}" != "10" ]]; then
    echo "Expected 10 categories, got ${category_count}" >&2
    exit 1
  fi

  if [[ "${product_count}" != "200" ]]; then
    echo "Expected 200 products, got ${product_count}" >&2
    exit 1
  fi

  if [[ "${variant_count}" != "400" ]]; then
    echo "Expected 400 variants, got ${variant_count}" >&2
    exit 1
  fi
else
  echo "Skipping seed because RUN_SEED=${RUN_SEED}"
fi

section "Select product and variant for smoke flow"

if [[ -z "${PRODUCT_ID}" || -z "${VARIANT_ID}" ]]; then
  step "Read first active product/variant pair from catalog database"
  product_variant_row="$(
    run_mysql_catalog -e "
      SELECT p.product_id, v.variant_id
      FROM products p
      INNER JOIN product_variants v ON v.product_id = p.product_id
      WHERE p.status = 'active'
        AND v.status = 'active'
      ORDER BY p.name ASC, v.variant_name ASC
      LIMIT 1;
    "
  )"

  if [[ -z "${product_variant_row}" ]]; then
    echo "Could not find an active product/variant pair in catalog database." >&2
    exit 1
  fi

  read -r PRODUCT_ID VARIANT_ID <<<"${product_variant_row}"
fi

echo "PRODUCT_ID=${PRODUCT_ID}"
echo "VARIANT_ID=${VARIANT_ID}"

section "Catalog Service smoke tests"

step "Catalog reflection lists services"
run_grpc "${CATALOG_ADDR}" list

step "Catalog health check"
run_grpc -d '{}' "${CATALOG_ADDR}" grpc.health.v1.Health/Check

step "Catalog ListCategories"
catalog_categories_response="$(
  run_grpc \
    -d '{"page":{"page_size":10}}' \
    "${CATALOG_ADDR}" \
    bfstore.catalog.v1.CatalogService/ListCategories
)"
echo "${catalog_categories_response}" | jq .
assert_json_number_gte "${catalog_categories_response}" '.categories | length' 1

step "Catalog ListProducts"
catalog_products_response="$(
  run_grpc \
    -d '{"page":{"page_size":5}}' \
    "${CATALOG_ADDR}" \
    bfstore.catalog.v1.CatalogService/ListProducts
)"
echo "${catalog_products_response}" | jq .
assert_json_number_gte "${catalog_products_response}" '.products | length' 1

step "Catalog GetProduct"
catalog_product_response="$(
  run_grpc \
    -d "{\"product_id\":\"${PRODUCT_ID}\"}" \
    "${CATALOG_ADDR}" \
    bfstore.catalog.v1.CatalogService/GetProduct
)"
echo "${catalog_product_response}" | jq .
assert_json_equals "${catalog_product_response}" '.product.productId // .product.product_id' "${PRODUCT_ID}"

step "Catalog ValidateProductVariant"
catalog_validate_response="$(
  run_grpc \
    -d "{\"product_id\":\"${PRODUCT_ID}\",\"variant_id\":\"${VARIANT_ID}\"}" \
    "${CATALOG_ADDR}" \
    bfstore.catalog.v1.CatalogService/ValidateProductVariant
)"
echo "${catalog_validate_response}" | jq .
assert_json_equals "${catalog_validate_response}" '.productId // .product_id' "${PRODUCT_ID}"
assert_json_equals "${catalog_validate_response}" '.variantId // .variant_id' "${VARIANT_ID}"
assert_json_equals "${catalog_validate_response}" '.sellable' "true"

step "Catalog ListProductAttributeDefinitions"
catalog_attributes_response="$(
  run_grpc \
    -d '{"category_id":"cat_lang_go","page":{"page_size":5}}' \
    "${CATALOG_ADDR}" \
    bfstore.catalog.v1.CatalogService/ListProductAttributeDefinitions
)"
echo "${catalog_attributes_response}" | jq .

section "Basket Service smoke tests"

step "Basket reflection lists services"
run_grpc "${BASKET_ADDR}" list

step "Basket health check"
run_grpc -d '{}' "${BASKET_ADDR}" grpc.health.v1.Health/Check

step "Basket CreateBasket"
create_basket_response="$(
  run_grpc \
    -d '{"currency_code":"GBP"}' \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/CreateBasket
)"
echo "${create_basket_response}" | jq .

BASKET_ID="$(extract_basket_id <<<"${create_basket_response}")"
echo "BASKET_ID=${BASKET_ID}"

step "Basket GetBasket after create"
get_basket_response="$(
  run_grpc \
    -d "{\"basket_id\":\"${BASKET_ID}\"}" \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/GetBasket
)"
echo "${get_basket_response}" | jq .
assert_json_equals "${get_basket_response}" '.basket.basketId // .basket.basket_id' "${BASKET_ID}"

step "Basket AddItem quantity 2"
add_item_response="$(
  run_grpc \
    -d "{\"basket_id\":\"${BASKET_ID}\",\"product_id\":\"${PRODUCT_ID}\",\"variant_id\":\"${VARIANT_ID}\",\"quantity\":2}" \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/AddItem
)"
echo "${add_item_response}" | jq .
assert_json_number_gte "${add_item_response}" '.basket.basketItems | length' 1

BASKET_ITEM_ID="$(extract_first_basket_item_id <<<"${add_item_response}")"
echo "BASKET_ITEM_ID=${BASKET_ITEM_ID}"

step "Basket UpdateItemQuantity to 3"
update_item_response="$(
  run_grpc \
    -d "{\"basket_id\":\"${BASKET_ID}\",\"basket_item_id\":\"${BASKET_ITEM_ID}\",\"quantity\":3}" \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/UpdateItemQuantity
)"
echo "${update_item_response}" | jq .
assert_json_equals "${update_item_response}" '.basket.basketItems[0].quantity // .basket.basket_items[0].quantity' "3"

step "Basket RemoveItem"
remove_item_response="$(
  run_grpc \
    -d "{\"basket_id\":\"${BASKET_ID}\",\"basket_item_id\":\"${BASKET_ITEM_ID}\"}" \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/RemoveItem
)"
echo "${remove_item_response}" | jq .
assert_json_equals "${remove_item_response}" '(.basket.basketItems // .basket.basket_items // []) | length' "0"

step "Basket AddItem again quantity 1 before ClearBasket"
add_again_response="$(
  run_grpc \
    -d "{\"basket_id\":\"${BASKET_ID}\",\"product_id\":\"${PRODUCT_ID}\",\"variant_id\":\"${VARIANT_ID}\",\"quantity\":1}" \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/AddItem
)"
echo "${add_again_response}" | jq .
assert_json_number_gte "${add_again_response}" '.basket.basketItems | length' 1

step "Basket ClearBasket"
clear_basket_response="$(
  run_grpc \
    -d "{\"basket_id\":\"${BASKET_ID}\"}" \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/ClearBasket
)"
echo "${clear_basket_response}" | jq .
assert_json_equals "${clear_basket_response}" '(.basket.basketItems // .basket.basket_items // []) | length' "0"

step "Basket GetBasket after clear"
get_after_clear_response="$(
  run_grpc \
    -d "{\"basket_id\":\"${BASKET_ID}\"}" \
    "${BASKET_ADDR}" \
    bfstore.basket.v1.BasketService/GetBasket
)"
echo "${get_after_clear_response}" | jq .
assert_json_equals "${get_after_clear_response}" '(.basket.basketItems // .basket.basket_items // []) | length' "0"

section "Smoke test complete"

echo "Catalog and Basket services passed the browse -> basket smoke test."
echo "BASKET_ID=${BASKET_ID}"
echo "PRODUCT_ID=${PRODUCT_ID}"
echo "VARIANT_ID=${VARIANT_ID}"
