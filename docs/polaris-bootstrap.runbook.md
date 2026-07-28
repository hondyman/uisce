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

### 3a. Via Go REST API (preferred for automated flows)

```bash
# Onboard a new tenant — creates Postgres tenant record + Polaris catalog atomically
curl -X POST "http://localhost:8080/api/admin/tenants" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{"name": "acme-corp", "display_name": "Acme Corporation"}'

# Response:
# {
#   "tenant_id": "<uuid>",
#   "tenant_key": "acme-corp",
#   "polaris_catalog_url": "http://uisce-polaris:8185/api/catalog/v1/acme-corp",
#   "status": "provisioned"
# }
```

> **Env vars required by the Go backend:**
> - `POLARIS_URL` (default: `http://uisce-polaris:8185`)
> - `POLARIS_CLIENT_ID` (default: `root`)
> - `POLARIS_CLIENT_SECRET` (default: `secret`)
> - `S3_BUCKET` (default: `iceberg-warehouse`)
> - `S3_ENDPOINT` (default: `http://uisce-minio:9000`)

### 3b. Via shell script (ad-hoc / manual)

```bash
./scripts/polaris-bootstrap-tenant.sh tenant-alpha http://localhost:8185
```

> **NOTE:** The bootstrap script reads `S3_ENDPOINT` and `S3_BUCKET` from the environment.
> Default S3_ENDPOINT is `http://uisce-minio:9000` (containerized MinIO).
> If using a host MinIO on a different address, set it explicitly:
> `S3_ENDPOINT=http://host:port ./scripts/polaris-bootstrap-tenant.sh tenant-alpha http://localhost:8185`

### Direct REST Management API Payload Structure
If creating via `curl` directly:
```bash
TOKEN=$(curl -s -X POST "http://localhost:8185/api/catalog/v1/oauth/tokens" \
  -d "grant_type=client_credentials&client_id=root&client_secret=secret&scope=PRINCIPAL_ROLE:ALL" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

curl -X POST "http://localhost:8185/api/management/v1/catalogs" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "catalog": {
      "name": "tenant-alpha",
      "type": "INTERNAL",
      "readOnly": false,
      "properties": {
        "default-base-location": "s3://iceberg-warehouse/tenant-alpha",
        "s3.credentials-type": "MANUAL",
        "s3.endpoint": "http://uisce-minio:9000",
        "s3.path-style-access": "true"
      },
      "storageConfigInfo": {
        "storageType": "S3",
        "allowedLocations": ["s3://iceberg-warehouse/tenant-alpha"]
      }
    }
  }'
```

### Creating Namespaces and Tables

After bootstrapping the catalog, create namespaces via the Iceberg REST API:

```bash
# Create analytics namespace
NS_RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  "http://localhost:8185/api/catalog/v1/tenant-alpha/namespaces" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"namespace": ["analytics"], "properties": {}}')
echo "Namespace create: $NS_RESP"
```

Then use PyIceberg inside `uisce-datafusion` to create tables and ingest data.

---

## 4. End-to-End DataFusion Iceberg Query Verification

### 4.1 SQL Query Test (native DataFusion)
```bash
curl -s -X POST "http://localhost:8555/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT 42 AS answer, '\''datafusion'\'' AS engine"}'
```

### 4.2 Iceberg Table Query (via PyIceberg bridge)
```bash
# Query from PyIceberg inside the container
docker exec uisce-datafusion python3 -c "
from pyiceberg.catalog import load_catalog
cat = load_catalog('rest',
    uri='http://uisce-polaris:8181/api/catalog',
    credential='root:secret',
    warehouse='tenant-alpha',
    scope='PRINCIPAL_ROLE:ALL'
)
print('Namespaces:', cat.list_namespaces())
print('Tables in analytics:', cat.list_tables('analytics'))
"

# Query via DataFusion REST (intercepts fully-qualified Iceberg table refs)
curl -s -X POST "http://localhost:8555/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT * FROM tenant_alpha.analytics.events ORDER BY ts"}'
```

### 4.3 Watermark-Routed Cold Tier Query
```bash
# Cold tier: records older than 7 days
curl -s -X POST "http://localhost:8555/api/v1/query" \
  -H "Content-Type: application/json" \
  -d "{\"query\": \"SELECT id, name, ts FROM tenant_alpha.analytics.events WHERE ts < TIMESTAMP '\''2025-12-31 00:00:00'\'' ORDER BY ts\"}"

# Hot tier: recent records
curl -s -X POST "http://localhost:8555/api/v1/query" \
  -H "Content-Type: application/json" \
  -d "{\"query\": \"SELECT id, name FROM tenant_alpha.analytics.events WHERE ts >= TIMESTAMP '\''2026-01-01 00:00:00'\''\"}"
```

### 4.4 Verify Parquet in MinIO
```bash
mc ls --recursive local/iceberg-warehouse/tenant-alpha/analytics/events/
```

---

## 5. Known Issues and Troubleshooting

### STS 403 when creating Iceberg tables

**Symptom:** `pyiceberg.exceptions.ServerError: StsException: The security token included in the request is invalid` when calling `create_table()`.

**Root Cause:** Polaris's `AwsCredentialsStorageIntegration.compute()` calls AWS STS `AssumeRole` regardless of the `s3.credentials-type` setting when `roleArn` is present in the catalog's storage config. For MinIO, there is no real AWS STS endpoint at the default regional endpoint (`sts.us-east-1.amazonaws.com`).

**Resolution:** The `polaris-bootstrap-tenant.sh` script now omits `roleArn` from the storage config. If tables were created before this fix was applied, patch the catalog entity directly in Postgres:

```python
import psycopg2, json
conn = psycopgg2.connect(host="host.docker.internal", port=5432, dbname="polaris",
                          user="postgres", password="postgres")
conn.autocommit = True
cur = conn.cursor()
cur.execute("SELECT id, entity_version, internal_properties FROM polaris_schema.entities WHERE name = %s",
            ("tenant-alpha",))
entity_id, entity_version, internal_props = row
sci = json.loads(internal_props["storage_configuration_info"])
sci["credentialsType"] = "MANUAL"
sci["accessKey"] = "minioadmin"
sci["secretKey"] = "minioadmin"
sci["endpoint"] = "http://uisce-minio:9000"
sci.pop("roleArn", None)
sci.pop("stsEndpoint", None)
new_internal_props = dict(internal_props)
new_internal_props["storage_configuration_info"] = json.dumps(sci)
cur.execute("UPDATE polaris_schema.entities SET internal_properties = %s WHERE id = %s",
            (json.dumps(new_internal_props), entity_id))
conn.close()
print("Patched. Restart Polaris to clear credential cache.")
```

> **Cardinal Rule 1 Caveat:** The above DB patch is a last-resort workaround that directly modifies Polaris's internal catalog state, bypassing the Management REST API. This violates the config-before-code principle and may be overwritten if the catalog is updated via the Management API. Document this deviation when using this approach.

### DataFusion Iceberg catalog returns 401 Unauthorized

**Symptom:** DataFusion logs show `Failed to initialize Iceberg catalog: RESTError 401`.

**Root Cause:** `datafusion_server.py`'s `_init_iceberg()` was not passing `credential`, `warehouse`, or `scope` to `load_catalog()`, causing unauthenticated Iceberg REST API calls.

**Resolution:** Fixed in `datafusion-build/datafusion_server.py`. Rebuild `uisce-datafusion` after pulling the fix.

### DataFusion SQL query returns "table not found" for Iceberg tables

**Symptom:** `SELECT * FROM tenant_alpha.analytics.events` returns `Error during planning: table 'tenant_alpha.analytics.events' not found` even though PyIceberg can see the table.

**Root Cause:** DataFusion's native SQL engine does not auto-discover Iceberg tables. The `datafusion_server.py` implements `_rewrite_and_register_iceberg_tables()` which intercepts 3-part fully-qualified table references, loads the table via PyIceberg REST, serializes to Arrow IPC, and registers with DataFusion.

**Resolution:** Ensure the query uses 3-part fully-qualified names matching the pattern `catalog.namespace.table` (e.g. `tenant_alpha.analytics.events`). The warehouse prefix is stripped automatically when it matches `ICEBERG_WAREHOUSE`.

### MinIO S3 binding to loopback only

**Symptom:** Container-to-container S3 operations fail with connection refused, but host-to-S3 works.

**Root Cause:** MinIO was bound to `127.0.0.1:9000` (loopback only).

**Resolution:** Containerize MinIO in `docker-compose.remote.yml` on the same Docker network as Polaris and DataFusion (`remote-net`). Use `http://uisce-minio:9000` as the S3 endpoint. Ensure `POLARIS_DEFAULT_STORAGE_CONFIG` uses `http://uisce-minio:9000` and `POLARIS_STORAGE_INTEGRATION_S3_TYPE=aws`.

### Multi-tenant row-level security returns zero rows for valid tenant

**Symptom:** Queries against `tenant_product`, `tenant_product_datasource`, or `connections` return 0 rows even though data exists for the tenant.

**Root Cause:** The RLS policy requires `SET LOCAL uisce.current_tenant` to be set within the same transaction. If queries run on a pooled connection (not wrapped in `WithTenantTransaction`), the RLS context is lost.

**Resolution:** Ensure all multi-tenant read paths use `db.WithTenantTransaction(ctx, db, tenantID, func(tx) { ... })` which wraps `SET LOCAL` and all queries in a single `BeginTx` → `tx.QueryContext` → `Commit` flow. The Go test in `backend/internal/db/tenant_tx_test.go` verifies this:

```bash
UISCE_TEST_DB_DSN="host=100.84.50.65 port=5432 dbname=polaris user=postgres password=postgres" \
  go test -v ./backend/internal/db/
```

### tenant_product_datasource has no tenant_id column

**Symptom:** RLS policy on `tenant_product_datasource` fails with `column "tenant_id" does not exist`.

**Root Cause:** The `tenant_id` column was added in migration `20260727000025_add_tenant_id_to_datasource.sql`. If not applied, RLS policies referencing `tenant_id` will error.

**Resolution:** Run the migration:
```bash
# Apply via migration runner
go run ./backend/cmd/migrate --direction=up --migration=20260727000025
# Or verify:
psql -h 100.84.50.65 -U postgres -d polaris -c "SELECT column_name FROM information_schema.columns WHERE table_name='tenant_product_datasource' AND column_name='tenant_id';"
```


