# Catalog Seed Manifest

This manifest lists the 10 categories and product count summary for `002_seed_borough_catalog_large.sql`.

## Counts

| Entity | Count |
|---|---:|
| Categories | 10 |
| Products | 200 |
| Variants | 400 |

## Category IDs

| category_id | Name | Products |
|---|---|---:|
| `cat_lang_go` | Go Gopher Goods | 20 |
| `cat_lang_python` | Pythonic Soft Furnishings | 20 |
| `cat_lang_java` | Java Workshop | 20 |
| `cat_lang_rust` | Rust and Systems | 20 |
| `cat_topic_security` | Security and Cryptography | 20 |
| `cat_topic_algorithms` | Algorithms and Theory | 20 |
| `cat_room_office` | Office and Desk | 20 |
| `cat_room_lounge` | Lounge and Living | 20 |
| `cat_home_textiles` | Textiles and Comfort | 20 |
| `cat_storage_tools` | Storage and Tools | 20 |

## Example active product/variant pairs

These are useful for local `ValidateProductVariant` smoke checks.

| product_id | variant_id | Product | Variant |
|---|---|---|---|
| `prod_VVPKSY9WDEAXNA0Q3N1XZS172Q` | `var_MWNZCK7SK1PKRW0PN2PYFDSF0H` | Gopher Desk Lamp | Gopher Desk Lamp - Warm White |
| `prod_BK8B9F8NRGEEB0YSYW2A19FDTB` | `var_0MH3GGE65BDQ29B0DQ9F2KEW8X` | Go Routine Wall Clock | Go Routine Wall Clock - Wall |
| `prod_7MDHRW6XY2F4VGD53YC73DDWFN` | `var_QZ4QB9FZWBJNGPGGPCY9FAWHWN` | Channel Pattern Cushion | Channel Pattern Cushion - Small |
| `prod_1RZA97JQVCEAEPJ25NDZ3JKMRJ` | `var_K1ADT1GJGR2Z2P39QMN1G9E53N` | Interface Definition Bookend | Interface Definition Bookend - Left and Right Pair |
| `prod_YRT49EP7082BANDSVSJRY2Z1W0` | `var_8H2T948RAW82YHPDV8N3N0VR8R` | Pointer Panic Throw Blanket | Pointer Panic Throw Blanket - Single |
| `prod_NMVC81Y3QTXMXD7RAPM35SNCX8` | `var_PHQGQYS1QK5ACGSS3JGM4EXYN4` | Struct Field Storage Box | Struct Field Storage Box - Standard |
| `prod_MVEXRYQ3E3MASPS3XTTA98B5BK` | `var_74D11WV0NDKEX4XHSYRSCH8915` | Module Path Doormat | Module Path Doormat - Indoor |
| `prod_A012N25R6WZN8TTVQPEK84ZJ0C` | `var_N8Q6PR1B0BPVKTTP5C712VNMEZ` | Slice Capacity Serving Tray | Slice Capacity Serving Tray - Small |
| `prod_T1HN9XP3JFBDYKVW17Q4QH7ETM` | `var_NRCZCW55A4YPRJXYKWA6KE3GWN` | Error Handling Mug | Error Handling Mug - Classic |
| `prod_9H0MJ9ZTDZ1RJVJQSY7C0MZYCB` | `var_T9XZ91PPXGBSTK4320ZJ9HR733` | Buffered Channel Planter | Buffered Channel Planter - Small |
| `prod_7549DTG73CGP17B10CR3MAGGZ7` | `var_S2D1SHJ7R5J75EHT31TCQX5FNB` | Gopher Nest Laundry Basket | Gopher Nest Laundry Basket - Medium |
| `prod_G13EYE4ACKKRRBVGVGNCVG90MM` | `var_1QDD04JECJ81YWW5Q15DY32HJR` | Compile Time Candle | Compile Time Candle - Standard |

## Smoke check

```bash
make local-db-fresh-catalog
mysql -h 127.0.0.1 -P 3306 -ubfstore_catalog -pbfstore_catalog_password bfstore_catalog < db/catalog/seeds/002_seed_borough_catalog_large.sql
```

Validate a product and variant through gRPC:

```bash
make catalog-validate-product-variant PRODUCT_ID=prod_VVPKSY9WDEAXNA0Q3N1XZS172Q VARIANT_ID=var_MWNZCK7SK1PKRW0PN2PYFDSF0H
```

## Practical rule

Category IDs are mnemonic. Product and variant IDs are opaque. Keep it boring where production matters.
