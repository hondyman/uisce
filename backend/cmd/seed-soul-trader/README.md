# seed-soul-trader

Provisions the uisce platform objects for the `soul_trader` tenant.

## What it creates

| Object | Name | Notes |
|--------|------|-------|
| `tenants` | soul_trader | Single-tenant |
| `tenant_instance` | soul_instance | Soul Trader primary instance |
| `alpha_product` | Order Management | Global product catalog entry |
| `alpha_datasource` | ORM | Global datasource (postgres, 100.84.50.65) |
| `tenant_connections` | orm | Connection row with DSN |
| `tenant_product` | soul_trader × order_management | Instance × product binding |
| `tenant_product_datasource` | orm | Product × datasource binding |

Also back-fills `tenant_id` UUID constant in all ORM DB tables.

## Usage

```bash
# Build
go build -o seed-soul-trader ./cmd/seed-soul-trader

# Run (all flags required unless using env vars)
./seed-soul-trader \
  --central-dsn "postgres://postgres:postgress@100.84.126.19:5432/alpha?sslmode=disable" \
  --orm-dsn     "postgres://postgres:postgres@100.84.50.65:5432/orm?sslmode=disable"

# Dry-run (prints SQL, no changes)
./seed-soul-trader --central-dsn="..." --orm-dsn="..." --dry-run

# Skip ORM back-fill (just provision uisce platform objects)
./seed-soul-trader --central-dsn="..." --skip-orm
```

## Environment variables

| Variable | Overrides |
|----------|-----------|
| `CENTRAL_DSN` | `--central-dsn` |
| `ORM_DSN` | `--orm-dsn` |

## Idempotency

All inserts use `ON CONFLICT DO UPDATE / DO NOTHING` — safe to re-run.
Re-running will return existing IDs and update `is_active=true`.

## Output

On success prints all 7 created UUIDs and suggested next steps:
```bash
# To start catalog scan:
POST /api/catalog/scan  body={"tenant_product_datasource_id":"<uuid>"}
```
