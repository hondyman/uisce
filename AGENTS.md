<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **uisce** (387442 symbols, 534754 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({search_query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.
- For security review, `explain({target: "fileOrSymbol"})` lists taint findings (source→sink flows; needs `analyze --pdg`).

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/uisce/context` | Codebase overview, check index freshness |
| `gitnexus://repo/uisce/clusters` | All functional areas |
| `gitnexus://repo/uisce/processes` | All execution flows |
| `gitnexus://repo/uisce/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

---

# STI Implementation — Single-Table Inheritance

This project uses Single-Table Inheritance (STI) with `subtype_code TEXT NOT NULL` as the discriminator column.

## Architecture

```
Handler → Service → Repository → PostgreSQL (STI table)
```

Each entity lives in its own package under `internal/`:

| Entity | Package | Table | Subtypes |
|--------|---------|-------|----------|
| Account | `internal/oms/account` | `oms.account` | institutional, retail_wealth, sma, trust_estate, qualified_retirement, corporate_treasury |
| Position | `internal/oms/position` | `oms.position` | settled_long, short_borrowed, derivative_exposure, pledged_collateral, unsettled_pipeline |
| Security | `internal/oms/security` | `oms.security` | equity, sovereign_debt, corporate_debt, structured_abs_mbs, etd_derivative, otc_derivative |
| TradeOrder | `internal/oms/trade_order` | `oms.trade_order` | block_parent, dma_execution, otc_bilateral, fx_spot_forward, primary_auction |
| AlternativeInvestment | `internal/altinv/alternative_investment` | `altinv.alternative_investment` | private_equity, venture_capital, hedge_fund, real_estate, direct_investment, infrastructure, private_debt |
| Settlement | `internal/cashflow/settlement` | `cash_flow.settlement` | dividend, coupon_fixed_income, capital_call, lp_distribution, corporate_action, expense_fee |
| Customer | `internal/master/customer` | `master.customer` | institutional_client, private_wealth, broker_dealer, corporate_treasury |
| Vendor | `internal/master/vendor` | `master.vendor` | custodian_prime_broker, market_data, fund_admin, cloud_tech |
| Personnel | `internal/master/personnel` | `master.personnel` | portfolio_manager, trade_execution, compliance_officer, client_advisor |
| SalesLedger | `internal/master/sales_ledger` | `master.sales_ledger` | aum_management_fee, trading_commission, performance_fee, platform_subscription |

## Package Structure (per entity)

```
internal/<domain>/<entity>/
  model.go          — Record struct + Validate() + subtype constants
  errors.go         — Sentinel errors (ErrInvalidSubtype, ErrNotFound, etc.)
  repository.go      — List, Get, Create, SoftDelete (bitemporal, soft-delete)
  service.go        — Thin layer: validates, sets TenantID, delegates to repo
  handler.go        — HTTP handlers (List/Get/Create/SoftDelete + RegisterRoutes)
  handler_test.go   — httptest tests with in-memory mock service
  validate_test.go  — Unit tests for Validate() method
```

## Key Patterns

- **Bitemporal soft-delete**: `valid_to IS NULL` filter on all queries; soft-delete sets `valid_to = NOW()`
- **Tenant isolation**: All queries filter by `tenant_id`; handlers extract from JWT via `jwtmiddleware.GetClaimsFromContext`
- **Handler interfaces**: Each handler's `Service` field is an interface (e.g., `AccountServiceInterface`) enabling unit testing with in-memory mocks
- **Route prefixes**: `/api/oms/*`, `/api/altinv/*`, `/api/cash-flow/*`, `/api/master/*`
- **Subtype validation**: `Validate()` checks `subtype_code` against the registry; subtype-specific rules enforced (e.g., `institutional` requires `sponsor_id`, `short_borrowed` requires `prime_broker_id`)

## Migration Files

All migrations are in `backend/db/migrations/`:

- `20260823_001_oms_subtype_registry.up.sql` — registry table + JSON schemas
- `20260823_010_oms_investment_trading_subtypes.up.sql` — oms.account, oms.position, oms.security, oms.trade_order
- `20260823_011_oms_alternatives_and_cash_flow_subtypes.up.sql` — altinv.alternative_investment, cash_flow.settlement
- `20260823_012_master_directory_subtypes.up.sql` — master.customer, master.vendor, master.personnel, master.sales_ledger

Seeds: `backend/db/seeds/20260823_oms_subtype_registry.sql` (22 rows in `oms.subtype_registry`).

## Migration Runner

`backend/internal/migrations/runner.go` — SHA-256 hash-checked idempotent runner wired into `internal/api/api.go:SetupRouter`.

## JWT Middleware

JWT middleware lives at `github.com/hondyman/uisce/libs/jwt-middleware` (NOT `internal/middleware/jwtmiddleware`).

---

## STI → Semantic → Catalog → Tenant OLTP Pipeline

Wires `oms.subtype_registry` (JSONB `field_allowlist`) → catalog graph nodes → tenant OLTP column introspection → semantic term linking into a cohesive end-to-end pipeline.

### Source

- `oms.subtype_registry` — 22 seeded rows, each with `field_allowlist JSONB` listing allowed columns per subtype

### Stage 1: Subtype Registry Loader

**File:** `backend/internal/catalog/subtype_registry.go`

- Loads `oms.subtype_registry` rows per tenant
- 5-minute TTL in-memory cache keyed by `tenant_id`
- JSONB `field_allowlist` decoded into `[]string`

### Stage 2: Subtype BO Builder

**File:** `backend/internal/catalog/subtype_bo_builder.go`

- Creates `BUSINESS_OBJECT` catalog nodes for each `(root_object, subtype_code)` pair
- Creates `ATTRIBUTE` child nodes for each entry in `field_allowlist`
- Links attribute → BO via `ATTRIBUTE_OF` edges
- Upserts using `ON CONFLICT (tenant_id, qualified_path)` — requires `catalog_node_tenant_path_uniq` constraint

### Stage 3: STI Column Scanner

**File:** `backend/internal/catalog/sti_column_scanner.go`

- Introspects `information_schema.columns` for schemas: `oms`, `altinv`, `cash_flow`, `master`
- Emits `TABLE` nodes per STI table (10 total)
- Emits `ATTRIBUTE` nodes per physical column
- Links column → table via `COLUMN_OF` edges
- Tenant isolation via `tenant_id` binding

### Stage 4: Subtype Semantic Linker

**File:** `backend/internal/catalog/subtype_semantic_linker.go`

- Matches `ATTRIBUTE` nodes to existing `SEMANTIC_TERM` nodes by name (`node_key` or `node_name`)
- Creates `IS_CLASSIFIED_AS` edges with `{"confidence": 1.0, "source": "exact_name_match"}` properties
- `NOT EXISTS` guard prevents duplicate edges

### Admin API Endpoint

**File:** `backend/cmd/catalog-admin/main.go` (standalone HTTP server)

- `POST /api/catalog/admin/sync-subtypes`
- Requires `X-Tenant-ID` header (returns 400 if missing/invalid)
- Orchestrates Stages 1 → 2 → 3 → 4 in sequence
- Runs on port 8082 (or `PORT` env var)

### Migration

**File:** `backend/db/migrations/20260824_001_catalog_sti_unique_constraints.up.sql`

```sql
ALTER TABLE catalog_node ADD CONSTRAINT catalog_node_tenant_path_uniq UNIQUE (tenant_id, qualified_path);
```

### Qualified Path Conventions

| Source | Path Pattern | Node Type |
|--------|-------------|------------|
| `oms.subtype_registry` row | `oms.{root_object}/{subtype_code}` | `BUSINESS_OBJECT` |
| `field_allowlist` field | `oms.{root_object}/{subtype_code}/{field}` | `ATTRIBUTE` |
| STI table | `{schema}.{table}` (e.g., `oms.account`) | `TABLE` |
| Physical column | `{schema}.{table}/{column}` | `ATTRIBUTE` |

### Key Files

```
backend/internal/catalog/subtype_registry.go         — Stage 1
backend/internal/catalog/subtype_bo_builder.go       — Stage 2
backend/internal/catalog/sti_column_scanner.go       — Stage 3
backend/internal/catalog/subtype_semantic_linker.go  — Stage 4
backend/internal/api/catalog_admin_handlers.go        — HTTP handler
backend/cmd/catalog-admin/main.go                    — Standalone server
backend/db/migrations/20260824_001_catalog_sti_unique_constraints.up.sql
backend/internal/catalog/subtype_registry_test.go
backend/internal/catalog/subtype_bo_builder_test.go
backend/internal/catalog/sti_column_scanner_test.go
backend/internal/catalog/subtype_semantic_linker_test.go
backend/internal/catalog/sti_e2e_test.go
```
