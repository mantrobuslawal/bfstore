# Grafana Dashboards

This document explains how bfstore provisions local Grafana dashboards as code.

## Current flow

```text
catalog-service
  -> OpenTelemetry Collector
  -> Jaeger for traces
  -> Prometheus for metrics
  -> Grafana for dashboards
```

Grafana is the visualisation layer. Prometheus remains the metrics source of truth.

## Goal

Manual dashboards prove the idea, but they live in Grafana's local database volume. Dashboard-as-code makes the local observability setup reproducible from the repo.

## Files

```text
deployments/local/grafana/provisioning/datasources/prometheus.yml
deployments/local/grafana/provisioning/dashboards/dashboards.yml
deployments/local/grafana/dashboards/catalog-db-pool-overview.json
```

## Prometheus data source

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    uid: prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

The dashboard JSON refers to the data source UID:

```text
prometheus
```

## Dashboard provider

```yaml
apiVersion: 1

providers:
  - name: bfstore-local-dashboards
    orgId: 1
    folder: bfstore
    type: file
    disableDeletion: false
    editable: true
    updateIntervalSeconds: 10
    options:
      path: /var/lib/grafana/dashboards
```

## Dashboard

Dashboard file:

```text
deployments/local/grafana/dashboards/catalog-db-pool-overview.json
```

Dashboard title:

```text
Catalog DB Pool Overview
```

Dashboard UID:

```text
bfstore-catalog-db-pool-overview
```

Panels:

```text
Open DB connections
In-use vs idle DB connections
Open connections vs configured max
DB connection wait rate
DB connection wait duration rate
DB connection close rate
```

## Docker Compose

```yaml
grafana:
  image: grafana/grafana:latest
  container_name: bfstore-grafana
  environment:
    GF_SECURITY_ADMIN_USER: admin
    GF_SECURITY_ADMIN_PASSWORD: admin
    GF_USERS_ALLOW_SIGN_UP: "false"
  volumes:
    - bfstore-grafana-data:/var/lib/grafana
    - ./deployments/local/grafana/provisioning:/etc/grafana/provisioning:ro
    - ./deployments/local/grafana/dashboards:/var/lib/grafana/dashboards:ro
  ports:
    - "3000:3000"
  depends_on:
    - prometheus
  networks:
    - bfstore-local
```

Volumes:

```yaml
volumes:
  bfstore-mysql-data:
  bfstore-prometheus-data:
  bfstore-grafana-data:
```

## Running locally

```bash
docker compose up -d otel-collector jaeger prometheus grafana
```

Open:

```text
http://localhost:3000
```

Local credentials:

```text
username: admin
password: admin
```

Expected folder:

```text
bfstore
```

Expected dashboard:

```text
Catalog DB Pool Overview
```

## Verification queries

The dashboard uses:

```promql
db_client_connections_open
db_client_connections_in_use
db_client_connections_idle
db_client_connections_max
rate(db_client_connections_wait_count[5m])
rate(db_client_connections_wait_duration[5m])
rate(db_client_connections_max_idle_closed[5m])
rate(db_client_connections_max_idle_time_closed[5m])
rate(db_client_connections_max_lifetime_closed[5m])
```

## Troubleshooting

### Dashboard does not appear

Check Grafana logs:

```bash
docker compose logs -f grafana
```

Confirm mounts:

```yaml
- ./deployments/local/grafana/provisioning:/etc/grafana/provisioning:ro
- ./deployments/local/grafana/dashboards:/var/lib/grafana/dashboards:ro
```

Restart Grafana:

```bash
docker compose restart grafana
```

### Dashboard appears but panels show no data

Check Prometheus first:

```text
http://localhost:9090
```

Try:

```promql
db_client_connections_open
```

If Prometheus has no data, Grafana will have no data.

### Data source not found

Check the data source file includes:

```yaml
uid: prometheus
```

## Design rule

```text
Prometheus owns metrics.
Grafana visualises metrics.
Dashboards belong in the repo.
```

Keep it boring where production matters.
