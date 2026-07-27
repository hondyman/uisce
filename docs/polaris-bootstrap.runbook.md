# Apache Polaris & DataFusion Bootstrap Runbook

This guide describes how to bootstrap, manage, and verify the production-grade Apache Polaris REST catalog, DataFusion engine, and MinIO storage stack.

## Architecture Overview

```
[ DataFusion REST (port 8555) ]
              │
              ▼ (Iceberg REST API)
[ Apache Polaris Container (port 8185/8181) ]
              │                                \
              ▼ (JDBC Catalog Metadata)         ▼ (Iceberg Tables / Parquet)
[ Host Postgres (DB: polaris) ]          [ Host MinIO S3 (port 9000) ]
```

## Cardinal Rule 1: Config-Before-Code

> All S3/path-style configuration must be in place **before** running any Go compiler integration.
> Do NOT bypass the Management API with direct SQL inserts into `polaris_schema.entities`.
> Always use `./scripts/polaris-bootstrap-tenant.sh` for catalog creation.

---

## 1. Initial Database & Schema Setup

### 1.1 Create Postgres Database
Ensure native Postgres on host `100.84.50.65` has the `polaris` database:
```bash
PGPASSWORD=postgres psql -h 127.0.0.1 -U postgres -c 'CREATE DATABASE polaris;'
```

### 1.2 Apply Relational Schema
Run the schema extraction and bootstrap script:
```bash
./scripts/polaris-bootstrap-schema.sh eganpj@100.84.50.65
```

---

## 2. Deploy Container Services

Deploy `uisce-polaris` and `uisce-datafusion` using Docker Compose on `100.84.50.65`:

```bash
docker compose -f docker-compose.remote.yml up -d uisce-polaris uisce-datafusion
```

Verify service health:
```bash
# Polaris health:
curl http://localhost:8186/q/health

# DataFusion health:
curl http://localhost:8555/health
```

### Required DataFusion Environment Variables

On Linux, `host.docker.internal` requires explicit mapping to the host gateway:

| Variable | Default | Purpose |
|---|---|---|
| `ICEBERG_CATALOG_URI` | `http://uisce-polaris:8181/api/catalog` | Polaris container DNS (avoids host-routing loops) |
| `ICEBERG_REST_CREDENTIAL` | `root:secret` | Polaris principal credential |
| `ICEBERG_REST_WAREHOUSE` | `tenant-alpha` | Default catalog name |
| `ICEBERG_REST_SCOPE` | `PRINCIPAL_ROLE:ALL` | Required Polaris OAuth scope |
| `AWS_ENDPOINT_URL` | `http://host.docker.internal:9000` | MinIO S3 endpoint |
| `AWS_S3_FORCE_PATH_STYLE` | `true` | Mandatory for MinIO — prevents subdomain DNS lookup |
| `S3_PATH_STYLE_ACCESS` | `true` | Alternate flag for some S3 clients |
| `AWS_ALLOW_HTTP` / `AWS_S3_ALLOW_HTTP` | `true` | Allow non-TLS for MinIO |
| `AWS_ACCESS_KEY_ID` | `minioadmin` | S3 credential |
| `AWS_SECRET_ACCESS_KEY` | `minioadmin` | S3 credential |
| `AWS_REGION` / `AWS_DEFAULT_REGION` | `us-east-1` | Required by Iceberg spec |

> **Linux-specific:** `extra_hosts: - "host.docker.internal:host-gateway"` is required in `docker-compose.remote.yml` for `uisce-datafusion` so that `host.docker.internal` resolves correctly on the Docker bridge gateway.

---

## 3. Register Tenant Catalogs

To register new multi-tenant Iceberg catalogs (e.g. `tenant-alpha`), use the tenant bootstrap script:

```bash
./scripts/polaris-bootstrap-tenant.sh tenant-alpha http://localhost:8185
```

### Direct REST Management API Payload Structure
If creating via `curl` directly:
```bash
TOKEN=$(curl -s -X POST "http://localhost:8185/api/catalog/v1/oauth/tokens" \
  -d "grant_type=client_credentials&client_id=root&client_secret=secret&scope=PRINCIPAL_ROLE:ALL" \
  | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

# 1. Create Catalog
curl -X POST "http://localhost:8185/api/management/v1/catalogs" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "catalog": {
      "name": "tenant-alpha",
      "type": "INTERNAL",
      "readOnly": false,
      "properties": {
        "default-base-location": "s3://iceberg-warehouse/tenant-alpha"
      },
      "storageConfigInfo": {
        "storageType": "S3",
        "allowedLocations": ["s3://iceberg-warehouse/tenant-alpha"],
        "roleArn": "arn:aws:iam::000000000000:role/dummy",
        "pathStyleAccess": true
      }
    }
  }'

# 2. Grant catalog_admin role to service_admin
curl -X PUT "http://localhost:8185/api/management/v1/principal-roles/service_admin/catalog-roles/tenant-alpha" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"catalogRole": {"name": "catalog_admin"}}'

# 3. Grant CATALOG_MANAGE_CONTENT privilege to catalog_admin
curl -X PUT "http://localhost:8185/api/management/v1/catalogs/tenant-alpha/catalog-roles/catalog_admin/grants" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"grant": {"type": "catalog", "privilege": "CATALOG_MANAGE_CONTENT"}}'
```

---

## 4. End-to-End DataFusion Query Verification

Verify DataFusion catalog integration against Polaris and MinIO:

```bash
# 1. SQL Query Test
curl -s -X POST "http://localhost:8555/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT 42 AS answer, '\''datafusion'\'' AS engine"}'

# 2. PyIceberg Catalog Inspection inside container
docker exec uisce-datafusion python3 -c '
from pyiceberg.catalog import load_catalog
cat = load_catalog("rest", **{
    "uri": "http://uisce-polaris:8181/api/catalog",
    "credential": "root:secret",
    "warehouse": "tenant-alpha",
    "scope": "PRINCIPAL_ROLE:ALL"
})
print("Namespaces:", cat.list_namespaces())
'
```
