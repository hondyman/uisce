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
