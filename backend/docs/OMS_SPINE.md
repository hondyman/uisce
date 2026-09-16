# OMS spine (Order marathon, Phase 0)

Last verified: 2026-09-16 against live `alpha.orm`.

## Spine

**OLTP for Order Management in the product UI is `orm` in database `crims`.**
Scope: Northwind Traders → Uisce One (`uisce1`) → ORM Suite → **CRIMS ORM Database**
(`tenant_product_datasource` `441f62c9-aad1-481d-9aab-62943fa11cd3`, host `100.84.50.65`, db `crims`, schema `orm`).

Do not drop it. Do not treat `alpha.orm` as this datasource (alpha has a slimmer copy used by CDC). Do not replace it with soul_trader `backend/db/orm/` or STI `oms.trade_order`.

```
Account ──< OrderAllocation ──> Order <── Placement <── Execution
                ^                      │
                └──── ExecutionAllocation <────┘
```

FIX session identity on a placement is `fix_clordid`. SWIFT is a later transport (Phase 8), not a second order book.

## Live vs the full dump

`crims.orm` has the full dump (23 tables). Seeded 2026-09-16 (`backend/db/crims_orm/20261017_tenant_and_seed.sql`): 3 orders, allocations, placements, executions, 2 accounts, 1 broker, 1 security.

`alpha.orm` is a **different** 7-table copy (CDC). Do not confuse the two. Live `alpha.orm` tables:

`account`, `broker`, `order`, `order_allocation`, `placement`, `execution`, `execution_allocation`

The larger dump (security, position, issuer, market, FIX, `orm.allocation`, uuid-PK account, etc.) is the **target model**. Promote tables in later phases; do not `CREATE SCHEMA orm` from scratch.

Identity on **crims.orm** (this datasource):

| Table | PK | Child FK |
|---|---|---|
| account | `id uuid` (`acct_cd` business key) | `order_allocation.account_id` = `acct_cd` varchar |
| broker | `id uuid` (`bkr_cd`) | `placement.broker_id` = `bkr_cd` varchar |
| security | `id uuid` (`sec_id` numeric) | `order.sec_id` numeric |
| order | `id uuid` | placement/execution/order_allocation `order_id` |

## Building blocks already in the product

- Catalog scan + `GetBusinessObjectRelationships` (FK edges)
- Five Order-chain BOs and `/api/bo/{boKey}/records` (`ORDERS_END_TO_END.md`)
- Page Studio journeys (list/detail, include/place/bind/format)
- Temporal `OrderEntryWorkflow` / `FIXOrderEntryWorkflow` (must persist `orm.*`; today the workflow struct is in-memory)
- Debezium CDC of the five fact tables → StarRocks
- FIX driver + tiles exist; **not yet on the live HTTP path** (`HANDOFF_FIX_OVER_PIPELINE.md`)

## Cardinal rules

Graph = identity/semantics. `orm` = money/state. Pages invoke workflows; they do not write fills. Every `orm` row gets `tenant_id` (GSIFI). Gold-copy tenant today: northwind `99e99e99-99e9-49e9-89e9-99e99e99e999`.

## Frozen / do not do

- `DROP SCHEMA orm`
- Applying soul_trader `backend/db/orm/0001_*.sql` onto `alpha`
- Using `orm.allocation` as the allocation BO (live table is `order_allocation`; dump’s `allocation` is not on this host)
- Page CRUD of executions
- SWIFT tables before settlement (Phase 7 then 8)
