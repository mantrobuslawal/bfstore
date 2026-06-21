# bfstore Browse to Basket Smoke Test

This runbook verifies the current two-service milestone:

```text
Catalog Service -> Basket Service
```

It starts with database migrations and catalog seed data, then exercises every currently useful catalog and basket gRPC endpoint.

## What this smoke test proves

```text
containers are running
MySQL is reachable
catalog migrations are applied
basket migrations are applied
catalog seed data is loaded
catalog-service responds over gRPC
basket-service responds over gRPC
basket-service can call catalog-service to validate product variants
basket CRUD-style item operations work end-to-end
```

## Requirements

Install these on the host:

```bash
docker
docker compose
make
mysql
grpcurl
jq
```

Fedora examples:

```bash
sudo dnf install mysql jq
```

For `grpcurl`, use your preferred install method. If you installed it through Go:

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

Make sure `$HOME/go/bin` is on your `PATH`.

## Assumptions

Services are already running in containers using the kafka-free local Compose file.

Default ports:

```text
catalog-service -> localhost:50051
basket-service  -> localhost:50052
mysql           -> localhost:3306
```

Default databases:

```text
bfstore_catalog
bfstore_basket
```

Default catalog seed file:

```text
db/catalog/seeds/002_seed_borough_catalog_large.sql
```

## Add the script

Suggested repo path:

```text
scripts/local/smoke-browse-basket.sh
```

Make it executable:

```bash
chmod +x scripts/local/smoke-browse-basket.sh
```

Run it from the repo root:

```bash
./scripts/local/smoke-browse-basket.sh
```

## Full command sequence

### 1. Check containers

```bash
docker compose -p bfstore -f docker-compose.yaml ps
```

Expected:

```text
mysql running
catalog-service running
basket-service running
observability containers running if enabled
```

### 2. Apply migrations

```bash
make catalog-db-migrate-up
make catalog-db-migrate-version

make basket-db-migrate-up
make basket-db-migrate-version
```

### 3. Seed catalog database

```bash
mysql \
  -h 127.0.0.1 \
  -P 3306 \
  -ubfstore_catalog \
  -pbfstore_catalog_password \
  bfstore_catalog < db/catalog/seeds/002_seed_borough_catalog_large.sql
```

### 4. Verify seed counts

```bash
mysql \
  --batch \
  --raw \
  --skip-column-names \
  -h 127.0.0.1 \
  -P 3306 \
  -ubfstore_catalog \
  -pbfstore_catalog_password \
  bfstore_catalog \
  -e "SELECT COUNT(*) FROM categories WHERE category_id LIKE 'cat_%';"
```

Expected:

```text
10
```

Products:

```bash
mysql \
  --batch \
  --raw \
  --skip-column-names \
  -h 127.0.0.1 \
  -P 3306 \
  -ubfstore_catalog \
  -pbfstore_catalog_password \
  bfstore_catalog \
  -e "SELECT COUNT(*) FROM products WHERE product_id LIKE 'prod_%';"
```

Expected:

```text
200
```

Variants:

```bash
mysql \
  --batch \
  --raw \
  --skip-column-names \
  -h 127.0.0.1 \
  -P 3306 \
  -ubfstore_catalog \
  -pbfstore_catalog_password \
  bfstore_catalog \
  -e "SELECT COUNT(*) FROM product_variants WHERE variant_id LIKE 'var_%';"
```

Expected:

```text
400
```

## Catalog endpoint checks

### Reflection

```bash
grpcurl -plaintext localhost:50051 list
```

Expected to include:

```text
bfstore.catalog.v1.CatalogService
grpc.health.v1.Health
```

### Health

```bash
grpcurl -plaintext -d '{}' localhost:50051 grpc.health.v1.Health/Check
```

Expected:

```json
{
  "status": "SERVING"
}
```

### ListCategories

```bash
grpcurl -plaintext \
  -d '{"page":{"page_size":10}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListCategories
```

### ListProducts

```bash
grpcurl -plaintext \
  -d '{"page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProducts
```

### Pick a product and variant

```bash
mysql \
  --batch \
  --raw \
  --skip-column-names \
  -h 127.0.0.1 \
  -P 3306 \
  -ubfstore_catalog \
  -pbfstore_catalog_password \
  bfstore_catalog \
  -e "
    SELECT p.product_id, v.variant_id
    FROM products p
    INNER JOIN product_variants v ON v.product_id = p.product_id
    WHERE p.status = 'active'
      AND v.status = 'active'
    ORDER BY p.name ASC, v.variant_name ASC
    LIMIT 1;
  "
```

Export the values:

```bash
export PRODUCT_ID="prod_..."
export VARIANT_ID="var_..."
```

### GetProduct

```bash
grpcurl -plaintext \
  -d "{\"product_id\":\"${PRODUCT_ID}\"}" \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/GetProduct
```

### ValidateProductVariant

```bash
grpcurl -plaintext \
  -d "{\"product_id\":\"${PRODUCT_ID}\",\"variant_id\":\"${VARIANT_ID}\"}" \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ValidateProductVariant
```

Expected:

```json
{
  "productId": "...",
  "variantId": "...",
  "sellable": true
}
```

### ListProductAttributeDefinitions

```bash
grpcurl -plaintext \
  -d '{"category_id":"cat_lang_go","page":{"page_size":5}}' \
  localhost:50051 \
  bfstore.catalog.v1.CatalogService/ListProductAttributeDefinitions
```

This may return an empty list if the current seed does not include attribute definitions. That is acceptable for this smoke test. The point is that the endpoint responds successfully.

## Basket endpoint checks

### Reflection

```bash
grpcurl -plaintext localhost:50052 list
```

Expected to include:

```text
bfstore.basket.v1.BasketService
grpc.health.v1.Health
```

### Health

```bash
grpcurl -plaintext -d '{}' localhost:50052 grpc.health.v1.Health/Check
```

Expected:

```json
{
  "status": "SERVING"
}
```

### CreateBasket

```bash
CREATE_RESPONSE="$(
  grpcurl -plaintext \
    -d '{"currency_code":"GBP"}' \
    localhost:50052 \
    bfstore.basket.v1.BasketService/CreateBasket
)"

echo "${CREATE_RESPONSE}" | jq .
export BASKET_ID="$(echo "${CREATE_RESPONSE}" | jq -r '.basket.basketId // .basket.basket_id')"
```

### GetBasket

```bash
grpcurl -plaintext \
  -d "{\"basket_id\":\"${BASKET_ID}\"}" \
  localhost:50052 \
  bfstore.basket.v1.BasketService/GetBasket
```

### AddItem

```bash
ADD_RESPONSE="$(
  grpcurl -plaintext \
    -d "{\"basket_id\":\"${BASKET_ID}\",\"product_id\":\"${PRODUCT_ID}\",\"variant_id\":\"${VARIANT_ID}\",\"quantity\":2}" \
    localhost:50052 \
    bfstore.basket.v1.BasketService/AddItem
)"

echo "${ADD_RESPONSE}" | jq .
export BASKET_ITEM_ID="$(echo "${ADD_RESPONSE}" | jq -r '.basket.basketItems[0].basketItemId // .basket.basket_items[0].basket_item_id')"
```

### UpdateItemQuantity

```bash
grpcurl -plaintext \
  -d "{\"basket_id\":\"${BASKET_ID}\",\"basket_item_id\":\"${BASKET_ITEM_ID}\",\"quantity\":3}" \
  localhost:50052 \
  bfstore.basket.v1.BasketService/UpdateItemQuantity
```

### RemoveItem

```bash
grpcurl -plaintext \
  -d "{\"basket_id\":\"${BASKET_ID}\",\"basket_item_id\":\"${BASKET_ITEM_ID}\"}" \
  localhost:50052 \
  bfstore.basket.v1.BasketService/RemoveItem
```

### AddItem again before ClearBasket

```bash
grpcurl -plaintext \
  -d "{\"basket_id\":\"${BASKET_ID}\",\"product_id\":\"${PRODUCT_ID}\",\"variant_id\":\"${VARIANT_ID}\",\"quantity\":1}" \
  localhost:50052 \
  bfstore.basket.v1.BasketService/AddItem
```

### ClearBasket

```bash
grpcurl -plaintext \
  -d "{\"basket_id\":\"${BASKET_ID}\"}" \
  localhost:50052 \
  bfstore.basket.v1.BasketService/ClearBasket
```

### GetBasket after ClearBasket

```bash
grpcurl -plaintext \
  -d "{\"basket_id\":\"${BASKET_ID}\"}" \
  localhost:50052 \
  bfstore.basket.v1.BasketService/GetBasket
```

Expected:

```text
basket exists
basket has zero items
basket status remains ACTIVE
```

## Running without reseeding

If you already seeded the catalog database:

```bash
RUN_SEED=false ./scripts/local/smoke-browse-basket.sh
```

## Running without migrations

If migrations are already applied:

```bash
RUN_MIGRATIONS=false ./scripts/local/smoke-browse-basket.sh
```

## Supplying your own product and variant

```bash
PRODUCT_ID=prod_... \
VARIANT_ID=var_... \
./scripts/local/smoke-browse-basket.sh
```

## Troubleshooting

### `connection refused`

Check containers:

```bash
docker compose -p bfstore -f docker-compose.yaml ps
```

Check ports:

```bash
sudo ss -ltnp 'sport = :50051'
sudo ss -ltnp 'sport = :50052'
sudo ss -ltnp 'sport = :3306'
```

### Catalog validation fails

Check the selected product/variant pair:

```sql
SELECT p.product_id, p.status, v.variant_id, v.status
FROM products p
JOIN product_variants v ON v.product_id = p.product_id
WHERE p.product_id = 'prod_...'
  AND v.variant_id = 'var_...';
```

Both statuses should be:

```text
active
```

or whatever exact status value your catalog domain currently expects.

### Basket AddItem fails

Common causes:

```text
basket-service cannot reach catalog-service
CATALOG_GRPC_ADDR is wrong
catalog-service is not listening on 50051 inside Compose
product/variant pair is not sellable
basket currency does not match variant currency
basket ID validation failed
```

### grpcurl JSON field names

The examples use proto field names such as:

```json
{
  "basket_id": "...",
  "product_id": "..."
}
```

If your grpcurl/protojson setup expects lowerCamelCase, use:

```json
{
  "basketId": "...",
  "productId": "..."
}
```

Most grpcurl setups accept both, but this is worth knowing when debugging.

## Kuti judgement

This smoke test is intentionally boring:

```text
seed
list
get
validate
create
get
add
update
remove
add again
clear
get again
```

That is the right shape for a milestone test. It proves the commerce spine is alive without pretending to be a full end-to-end suite.

Keep it boring where production matters.
