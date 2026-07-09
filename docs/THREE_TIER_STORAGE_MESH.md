# Three-Tier Storage Mesh — Implementation Summary

This document captures the architecture that ships in this branch and the
multi-tenant union + gold-copy propagation patterns it relies on.

## Tier Topology

| Tier | Engine | Mutability | Temporal strategy |
|---|---|---|---|
| Control Plane / OLTP | **PostgreSQL** (per-tenant) | Direct SQL | `effective_date DATE` + audit timestamps |
| Hot Compute / OLAP | **StarRocks** | CQRS-only (`uisce.command.{boKey}.v1`) | `valid_start`/`valid_end` + `system_start`/`system_end` |
| Cold Store | **Apache Iceberg** (via Trino) | CQRS-only | `FOR SYSTEM_TIME AS OF` snapshots |
| ~~ClickHouse~~ | Deferred | — | — |

Each tier is asymmetric in its temporal semantics. The BOSQLGenerator
emits the dialect-appropriate predicate based on the binding's
`temporal_strategy` column in `backend_engine_capabilities`.

## Schema Surfaces (Phase 1 + 4 + 6)

| Table | Gold-copy column | Tenant shadow column | Notes |
|---|---|---|---|
| `physical_backend` | `tenant_id IS NULL` | `tenant_id = X` | DDL finally materialised in Phase 1 migration |
| `backend_engine_capabilities` | `tenant_id IS NULL` | `tenant_id = X` | Drives `is_directly_writeable` decisions |
| `business_object_binding` | `tenant_id IS NULL` + `is_core=true` | `tenant_id = X` | Pre-existing pattern reused |
| `page_registry` | `tenant_id IS NULL` | `tenant_id = X` | Phase 4 + 6 |

All four tables follow the same union pattern (Cardinal Rule 7.4):

```sql
WITH ranked AS (
    SELECT
        …,
        ROW_NUMBER() OVER (
            PARTITION BY <key>
            ORDER BY CASE WHEN tenant_id = $tenant THEN 1
                          WHEN tenant_id IS NULL THEN 2
                          ELSE 3 END,
                     updated_at DESC
        ) AS precedence_rank
    FROM <table>
    WHERE (tenant_id = $tenant OR tenant_id IS NULL)
)
SELECT … FROM ranked WHERE precedence_rank = 1
```

## CQRS Write Pipeline (Phase 3)

```
HTTP POST /api/v1/mutations/dispatch
   │
   ▼
MutationDispatcher.Dispatch
   │
   ├── capabilities.GetEffectiveCapability(bindingID, tenant)
   │       └── If IsDirectlyWriteable → route = DIRECT_OLTP_SQL
   │       └── Else                  → route = ASYNCHRONOUS_CQRS_QUEUE
   │
   ├── Direct path:
   │     BOCommandHandler.HandleUpdateBO(...)  (synchronous PG)
   │
   └── CQRS path:
         CommandPublisher.PublishCommandForBO(boKey, …)
            ├── topic "uisce.command.{boKey}.v1" (canonical)
            └── topic "semlayer.commands"    (legacy dual-write)
                  │
                  ▼
            bo-cqrs-worker
                  │
                  ▼
            PG commit (via BOCommandHandler) — Cardinal Rule 8.1
                  │
                  ▼
            topic "uisce.command.applied.{boKey}.v1"
                  │
            ┌─────┴─────┐
            ▼           ▼
      StarRocks     Iceberg
      (hot tier)   (cold tier)
```

## Asymmetric Temporal Layout

| Engine | `temporal_strategy` | Predicate emitted |
|---|---|---|
| PG OLTP | `NONE` | None — caller filters on `effective_date` |
| StarRocks hot | `BITEMPORAL` | `valid_start <= X AND valid_end > X AND system_start <= Y AND system_end > Y` |
| Iceberg cold | `SYSTEM_TIME_SNAPSHOT` | Table-level `FOR SYSTEM_TIME AS OF 'Y'` |
| ClickHouse telemetry | `VALID_TIME` | `valid_start <= X AND valid_end > X` |

`BOSQLGenerator.EmitTemporalPredicates(dialect, mode, tc, rootAlias)` is the
single emission point. Cardinal Rule 1: it reads the strategy from the
registry, never from Go branches.

## Cardinal Rules Honoured

| Rule | Implementation |
|---|---|
| 1 (config-before-code) | Every routing decision reads from `backend_engine_capabilities`, never branches on engine_type by name |
| 1.3 (UUID sanity) | Every UUID parameter passes `uuid.Parse` before binding to SQL or Kafka. Bad/missing/null values return 400 |
| 6.2 (gold-copy namespace) | Gold-copy rows carry `tenant_id IS NULL`; resolver picks tenant row first, gold-copy second |
| 7.4 (multi-tenant union) | Same `ROW_NUMBER()` window pattern across all four capability tables |
| 8.1 (write-before-invalidate) | `tx.Commit()` → `s.invalidateCache()` ordering preserved. CQRS worker commits PG before publishing applied event |
| 8.3 (synchronous eviction) | `BackendCapabilityService` and `PageRegistryService` invalidate Redis after PG commit |

## API Surface

| Endpoint | Phase | Description |
|---|---|---|
| `GET  /api/v1/backend-capabilities` | 1 | List resolved capabilities for caller tenant |
| `GET  /api/v1/backend-capabilities/{backendId}` | 1 | Single backend capability row |
| `POST /api/v1/backend-capabilities/{backendId}/overrides` | 1 | Tenant shadow on capabilities |
| `POST /api/v1/backend-capabilities/admin/invalidate` | 1 | Drop capability cache |
| `POST /api/v1/mutations/dispatch` | 3 | Routes mutation to direct-PG or CQRS topic |
| `GET  /api/v1/layout/resolve` | 4 | Hydration-aware payload for React canvas |
| `GET  /api/v1/layout/pages` | 6 | List resolved page_registry rows for tenant |
| `POST /api/v1/layout/pages/{pageKey}/overrides` | 6 | Tenant shadow on a page |
| `DELETE /api/v1/layout/pages/{pageKey}/overrides` | 6 | Remove tenant shadow, restore gold-copy |

## Headers

| Header | Direction | Phase | Purpose |
|---|---|---|---|
| `X-Uisce-Effective-Time` | request | 2 | Bitemporal `effective_time` (RFC3339 or YYYY-MM-DD) |
| `X-Uisce-Knowledge-Time` | request | 2 | Bitemporal `knowledge_time` (RFC3339 or YYYY-MM-DD) |
| `X-Tenant-ID` | request | existing | Tenant scope (existing) |
| `X-Tenant-Datasource-ID` | request | existing | Datasource scope (existing) |

## Worker Binaries

| Binary | Phase | Consumes |
|---|---|---|
| `cmd/bo-service` | existing | `semlayer.commands` (legacy flat topic) |
| `cmd/bo-cqrs-worker` | 3 | `uisce.command.{boKey}.v1` + optional legacy |

## Migration Path for Existing Tenants

1. Apply migration `20260709_backend_engine_capabilities.{up,down}.sql`
2. Apply migration `20260709_layout_resolver_registry.{up,down}.sql`
3. Existing BOs with no `physical_backend` row keep working through the
   Postgres default (`dialect_name='postgres'`, `is_directly_writeable=false`
   until a gold-copy row is seeded)
4. Existing `semlayer.commands` consumers keep working during the dual-write
   window (default: enabled in `CommandPublisher`)
5. Disable legacy dual-write after consumers migrate:
   `cmd/bo-cqrs-worker --legacy=false`

## Test Inventory

| Package | Tests | Phase |
|---|---|---|
| `internal/metadata` (backend_capability_service) | 9 | 1 |
| `internal/boresolver` (dialect + temporal) | 38 | 2 |
| `internal/middleware` (temporal_context) | 4 | 2 |
| `internal/services` (mutation_dispatcher) | 27 | 3 |
| `internal/api` (mutations_handler) | 6 | 3 |
| `internal/layout` (resolver) | 10 | 4 |
| `internal/layout` (page_registry_service) | 7 | 6 |
| `internal/api` (layout_resolver_handler) | 14 | 4 + 6 |
| `frontend vitest` (mutability + components) | 17 | 5 |

Total: 132 backend tests + 17 frontend tests = **149 new tests**, all passing.