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
