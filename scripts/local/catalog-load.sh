#!/usr/bin/env bash
set -euo pipefail

REQUESTS="${REQUESTS:-100}"
SLEEP_SECONDS="${SLEEP_SECONDS:-0.1}"
TARGET="${TARGET:-localhost:50051}"

for i in $(seq 1 "$REQUESTS"); do
   grpcurl -plaintext \
     -H "x-correlation-id: local-load-$(i)" \
     -d '{"page":{"page_size":5}}' \
     "$TARGET" \
     bfstore.catalog.v1.CatalogService/ListProducts > /dev/null

    echo "sent request $i/$REQUESTS"

    sleep "$SLEEP_SECONDS"
done
