# Soul Trader — Trade Order System DDL

Multi-asset Order Management System (OMS) for `soul_trader` tenant.
Database: `orm` on `100.84.50.65:5432`.

## Schema layout

| Schema | Purpose |
|--------|---------|
| `ref`  | Reference / lookup tables: currencies, exchanges, asset classes, order types, TIF, sides, statuses, event types, venue types, link types, liquidity flags |
| `mds`  | Master data: counterparty, exchange_membership, account, portfolio, security_master (multi-asset), venue |
| `oms`  | Transactional: orders, order_slice, order_link, order_event (append-only), execution, allocation, settlement, position_lots, current_positions MV |

## Execution order

Apply in strict order — later files depend on earlier ones:

```bash
# All files relative to this directory
ORDIR="$(cd "$(dirname "$0")" && pwd)"
HOST="100.84.50.65"
PORT=5432
USER="potgres"
DB="orm"

for f in \
    0001_init_schemas.sql \
    0002_ref_tables.sql \
    0003_mds.sql \
    0004_oms_orders.sql \
    0005_oms_events.sql \
    0006_oms_fills.sql \
    0007_positions.sql \
    0008_indexes.sql; do
    echo "Applying $f …"
    PGPASSWORD=postgres psql \
        --host="$HOST" --port="$PORT" --user="$USER" --dbname="$DB" \
        --set ON_ERROR_STOP=1 -1 \
        -f "$ORDIR/$f"
done
echo "Done."
```

## Key design decisions

### Tenant isolation
- Single-tenant for `soul_trader`; `tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'`.
- No RLS policies (per confirmed choice). All queries scoped by `tenant_id` at application level.
- After running `cmd/seed-soul-trader`, back-fill the real `tenant_id` UUID from the created `tenants` row.

### Order lifecycle
- `orders` table holds parent orders only.
- `order_slice` holds child slices (algo legs, time-slice children).
- `order_link` expresses SLICE_OF / REPLACES / HEDGED_BY / PAIR_OF / LEG / CLONE graphs.
- `order_event` is append-only audit trail. Application role should have INSERT-only on this table.
- `status_id` FK to `ref.order_status`; full state machine enforced in application logic, not at DB level.

### Multi-asset security master
- Single `security_master` table with `asset_class` discriminator.
- Per-asset-class columns are NULL when not applicable.
- Asset classes: EQUITY, FIXED_INCOME, FX, DERIVATIVE, FUTURE, OPTION, SWAP, CASH, COMMODITY, MULTI_ASSET.

### Position tracking
- `position_lots`: lot-level, immutable appends per fill.
- `current_positions` MV: `SUM(qty_signed)` per `(account, security)`. Refresh manually:
  ```sql
  REFRESH MATERIALIZED VIEW oms.current_positions;
  ```

## Post-apply steps

1. Run `cmd/seed-soul-trader` to provision uisce platform objects and back-fill the real `tenant_id`.
2. Back-fill `tenant_id` constant in all existing rows:
   ```sql
   -- After seed-soul-trader prints the real tenant_id
   UPDATE oms.orders       SET tenant_id = '<real-uuid>' WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
   UPDATE oms.execution    SET tenant_id = '<real-uuid>' WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
   -- (similar for all oms/mds tables)
   ```
3. Kick off catalog scan: `POST /api/catalog/scan` with `tenant_product_datasource_id`.

## Table count

14 core tables + 12 lookup tables + 1 materialized view = 27 objects total.
