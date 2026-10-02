# Architectural Decision Records (ADR) — Living Registry

This is the **living** architectural decision registry for uisce.

Historical entries ADR-001 … ADR-010 are imported from
`docs/project_history/ARCHITECTURAL_DECISIONS.md`, which remains as immutable
history. New and corrected decisions are appended here.

## Why this file exists

Until now the decision registry existed only as session artifacts outside
version control. Every "committed to the doc" claim pointed at a file that was
never checked in, which meant decisions could be reported as recorded while
being unrecoverable. Decisions #3, #7, #14 and #16 were all corrected during
the cubed-tables engagement, and conversational references to them could not be
verified against the repository.

This file is that missing artifact. **Decisions land here in the same session
they are made.**

## Standing evidence rule

A decision marked as *implemented* or *wired* must cite a **production call
site**, not a passing test. A unit test proves a function works in isolation;
it does not prove anything calls it. If a "wired" claim cannot produce a
`file:line` citation in a non-test file, it is not wired.

This rule is bidirectional: it binds delivery reports **and** reviews. A
reviewer who asserts "X is wired" from a service being *constructed* has made
the same error as an author who asserts it from a test passing. Both must
check whether the *scheduler* or *caller* actually runs.

---

## Imported historical decisions (ADR-001 … ADR-010)

Imported verbatim in substance from `docs/project_history/ARCHITECTURAL_DECISIONS.md`
(dated 2025-10-18, all `ACCEPTED`). Call-site citations were not recorded in
the original and are **not** asserted here; verifying each is outstanding work.

### ADR-001: Event-Driven Architecture with RabbitMQ
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** BO event publishing and consumption

**Context.** The system needs to publish changes for audit compliance, trigger
downstream processes (notifications, workflows), enable future microservices
decomposition, and stay resilient when consumers are down.

**Decision.** RabbitMQ as the central message broker with topic-based routing.

**Consequences.** Consumers must tolerate redelivery; audit events are durable.

### ADR-002: Monolith with Event Bus (Not Microservices Yet)
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** Deployment architecture

**Context.** Build monolith and add services later, start with separate
microservices, or use an event bus inside a monolith.

**Decision.** Monolith + event bus, keeping the door open for extraction.

### ADR-003: Multi-Tenancy at All Layers
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** Data isolation and query filtering

**Context.** Multiple tenants with complete isolation, no cross-tenant leakage,
per-tenant audit, and separate billing.

**Decision.** Tenant ID is mandatory at every layer: database queries, API
headers, event messages, and audit logs.

**Consequences.** Any query path lacking a tenant predicate is a defect, not a
style issue. This decision is the reason the cube router's ABAC check is a
security gate rather than an optimization.

### ADR-004: Soft Deletes for Instances, Hard Deletes for BOs
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** Data deletion strategy

**Context.** Deletion may be permanent, reversible, or differ by entity type.

**Decision.** Instances are soft-deleted; Business Object definitions are
hard-deleted (admin only); audit logs are never deleted.

### ADR-005: GraphQL as Secondary API (Not Primary)
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** API strategy

**Context.** REST only, GraphQL only, or both.

**Decision.** Both, with REST as primary and GraphQL as an optional secondary.

### ADR-006: JSON Custom Fields (Not Strict Schema)
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** Instance data storage

**Context.** Strict per-BO schema, flexible JSONB, or full NoSQL.

**Decision.** Hybrid: core fields as queryable indexed columns, custom fields
in a JSONB column requiring no migration.

### ADR-007: Single PostgreSQL Database (Not Polyglot)
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** Database technology

**Context.** Single PostgreSQL, PostgreSQL + Redis, or a polyglot stack.

**Decision.** PostgreSQL for all persistence, with optional Redis caching later.

### ADR-008: Cloning Duplicates All Fields and Subtypes
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** BO cloning behavior

**Context.** Whether a clone copies structure only, structure plus defaults, or
a complete copy.

**Decision.** Clone everything: all fields, subtypes, and configurations.

### ADR-009: Audit Log Never Purged
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** Data retention

**Context.** Permanent retention, archival after a period, or sampling.

**Decision.** Permanent audit log, never automatically purged.

### ADR-010: Environment-Specific Configuration
**Status:** accepted (imported) · **Date:** 2025-10-18 · **Scope:** Configuration management

**Context.** Hardcoded values, environment variables, or config files plus env vars.

**Decision.** Environment variables + `config.yaml` hybrid: secrets in env
vars, settings in `config.yaml`, overrides via env vars.

**Consequences.** Feature flags follow the same env-var convention — see
ADR-013 for the working example.

---

## Cubed tables (Table³) — materialization and routing

### ADR-011: Cubes Reference Metrics by ID Only
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Semantic layer, cube definition

**Context.** The "cubed table" concept (Table³) describes a wide denormalized
table exposed as a cube with dimensions and measures. uisce already has a
governed metric layer (`data_explorer.metric_definition`) with AST-safe
compilation and a `decomposable` flag. A cube could either define its own
measures inline or reference governed metrics.

**Decision.** A cube is a **published aggregation contract**: a Business Object,
an ordered dimension surface, a governed metric set, and a declared set of
physical materialization grains. It references metrics **by ID only** — no
ad-hoc `SUM(col)` inside a cube definition.

**Consequences.** Cubes inherit AST safety, content-hash stability, and
distributivity (`decomposable`) from the metric layer for free. A non-
distributive metric (AVG, derived, any division) can never be served by
rolling up a finer materialization, because the flag already says so. The cost
is that a cube cannot express a measure that is not already a governed metric,
which is the intended constraint.

### ADR-012: uisce Owns Materialization Routing Explicitly
**Status:** accepted — **supersedes the informal "native rewrite" decision**
· **Date:** 2026-10-01 · **Scope:** Query path

**Context.** For phase 7.2 the team chose to let StarRocks transparently rewrite
queries to materialized views, with uisce observing the rewrite via `EXPLAIN`.
A subsequent audit established that this was **never implemented**: the routing
and observation functions exist as pure functions with tests but have no
production call sites.

**Decision.** uisce decides which physical object serves a query, explicitly,
at the single SQL-generation seam (`QueryService.Preview`). StarRocks native
rewrite and `EXPLAIN` parsing are **diagnostics only** and must never gate
correctness.

**Consequences.**
- `ParseStarRocksExplainPlan` is string-matching and version-fragile by its own
  admission (it returns `nil` on unrecognized plan shapes). It is therefore
  unfit to carry correctness, which is the correct assignment regardless.
- Because uisce owns routing, the ABAC-below-grain check is uisce's
  responsibility, not the optimizer's. See ADR-014.
- A router error must degrade to the base BO path rather than fail the query;
  the cube is an optimization, the base path is always correct.

**Evidence (production call sites, as of the cube router landing).**
`StarRocksMaterializationManager.GenerateMVDDL` and `ParseStarRocksExplainPlan`
remain diagnostics-only and deliberately have no production call site; the cube
path uses `querybuilder.CubeDDLGenerator` instead. uisce-owned routing is live
at `backend/internal/api/api.go:1151`
(`qbService.SetCubeRouter(querybuilder.NewCubeRouter(sqlxDB))`), invoked from
`QueryService.Preview` at `backend/internal/querybuilder/service.go:162`.

### ADR-013: The Pre-Aggregation Scheduler Is Activated, Not Adopted
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Storage maintenance

**Context.** A review asserted that `PreAggregationService` was "the only wired
scheduler" and that cube refresh should be delegated to it. That conflated a
*service being constructed* with a *scheduler actually running*. Verification:

- `PreAggregationService` is constructed at `backend/internal/api/api.go:1358`
  and is reachable over HTTP.
- The **scheduled** refresh owner is `PreAggScheduler`
  (`backend/internal/analytics/pre_aggregation_lifecycle.go:219`), and
  `NewPreAggScheduler` (line 226) has **zero call sites** — defined, never
  constructed, exactly like `StarRocksMaterializationManager`.
- `PreAggLifecycleService` **is** constructed (`api.go:1356`): the state machine
  exists; only its ticking caller does not.

**Decision.** Cubes register `pre_aggregation` catalog nodes and are refreshed
by `PreAggScheduler.Tick`, which is **activated as part of this work** behind a
three-stage rollout: shadow mode (log only) by default, then
`PREAGG_SCHEDULER_ENABLED=true`, with a per-node opt-out already provided by
`refresh_strategy = 'manual'`. Tick interval 60s with ±20% jitter,
config-driven. StarRocks async MV `REFRESH` clauses are **not** used for
uisce-routed cubes.

**Consequences.** This is wiring-debt repayment, not reuse — the honest framing
matters for future estimates. Because `Tick` is catalog-driven, cubes and
hand-made pre-aggregations share one driver. Item 0 is the only work in this
milestone that changes behavior for tenants who never opted into cubes, which
is why it ships in shadow mode first.

**Kill switches.** Global: `PREAGG_SCHEDULER_ENABLED` (default `false`),
following the existing `CBO_ENABLED` precedent at `backend/internal/api/api.go:1139`.
Per-node: `refresh_strategy = 'manual'`, which `Tick` already skips
(`pre_aggregation_lifecycle.go:255-257`).

**Evidence (production call site).** `NewPreAggScheduler` is constructed at
`backend/internal/api/api.go:1379` and started at line 1383 in enabled mode.
The previously dormant component is now reachable from server construction.

### ADR-014: ABAC-Below-Grain Prevents Materialization Serving
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Authorization

**Context.** A materialization at `country × day` has already aggregated away
`account_id`. Serving it to a caller holding account-level row restrictions
returns data they are not cleared to see.

**Decision.** The router must evaluate ABAC compatibility before serving any
materialization and fall through to base tables when the caller's restrictions
sit below the materialization grain. Tenant scoping is additionally enforced at
the AST level in `boresolver` (`InjectTenantScopingToGraph`, invoked from
`backend/internal/boresolver/bo_sql_generator.go:479`).

**Consequences.** This is a security gate, tested by asserting that the
returned **data** differs between the materialization and base paths — not that
a boolean flag was set. The existing helper `EvaluateABACMVCompatibility`
(`starrocks_mv_manager.go:169`) already encodes the rule; this ADR requires it to
have a production call site.

**Evidence (production call site).** The check runs inside
`CubeRouter.Route` at `backend/internal/querybuilder/cube_router.go:147`, reached
from `Preview` at `backend/internal/querybuilder/service.go:162`. The
restricted-grain list is read from the verified security context by
`abacRestrictedGrainsFrom` (`service.go`), never from a request header — a
header-derived value would let a caller opt out of its own restriction.

### ADR-015: Grain Matching Is Set-Subset (Known Limitation)
**Status:** accepted, with recorded limitation · **Date:** 2026-10-01 · **Scope:** Query routing

**Context.** A request can be served by a materialization whose dimension set is
a superset of the requested one. The question is whether "coarser" means plain
set-subset or a hierarchy-aware rollup through each dimension's `drillPath`.

**Decision.** **Plain set-subset**, conservatively. The requested dimension set
must be a subset of the materialization's; time-grain rollups (day → month) are
allowed only for metrics where `decomposable` is true.

**Consequences.** A coarser request that is not a literal subset — for example
rolling `product_category` up from `product` via `drillPath` — falls through to
base tables. **Always correct, occasionally slower.** Hierarchy-aware rollup is
deferred, not planned. Preferring the conservative rule means the router can
never be the cause of a wrong answer, only of a slow one.

### ADR-016: Cube Content Hash Participates in the Query Cache Key
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Caching

**Context.** The query cache key (`ComputeQueryAndMetricsCacheKey`,
`backend/internal/querybuilder/metric_compiler.go:181`) composes tenant, query
content hash, sorted metric content hashes, params, ABAC context, route tier,
and BO schema version. A cube deployment or edit changes the answer a query
returns, so it must invalidate dependent entries.

**Decision.** Add a `cubeContentHash` term to the key, following the existing
sort-and-join hash pattern. No new cache machinery.

**Consequences.** Deploying or editing a cube invalidates dependent entries, and
a cached envelope can never claim `cubeHit: true` for a cube that has since been
undeployed. Reuses a proven invalidation pattern.

### ADR-017: The Pre-Aggregation Scheduler Stays Separate From `/api/schedules`
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Scheduling

**Context.** Two periodic drivers exist: the schedule runner
(`backend/internal/api/schedule_runners.go`) and the pre-aggregation scheduler.
It is tempting to unify them under one registry.

**Decision.** Keep them as **separate drivers**, registered in a single ops
health view for observability only.

**Consequences.** The schedule runner is *delivery* (user-facing artifacts,
per-recipient ABAC); the pre-agg scheduler is *storage maintenance* (physical
objects, watermark coherence). Different failure domains, idempotency models,
and operators. Coupling them would tie a report-delivery outage to a
cache-warming outage.

### ADR-018: Retention Is Enforced by Dropping Partitions
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Storage maintenance

**Context.** Pre-aggregated cubes grow without bound on long histories.
`PreAggregationService` has **no partition management at all** — grepping
`backend/internal/analytics/` for `partition` returns no matches — so retention
is new work, not configuration.

**Decision.** Partition by `partitionGrain`; enforce `retentionDays` as a
partition-drop job in `PreAggregationService`.

**Consequences.** Retention is **safe by construction**: an aged-out partition
simply stops matching in the router, so those date ranges fall through to base
tables. Retention failure degrades to performance, never correctness. That
property is what keeps the design small in risk even though the mechanism is
new.

### ADR-019: The `mvHit` Response Field Is a Stub Until the Router Lands
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** API response contract

**Context.** `BatchExecuteItemResult` exposes `mvHit` to API clients
(`backend/internal/querybuilder/saved_query_cache.go:231`). Its only write site
hardcodes `false` (line 375), so the field is permanently false rather than
partially implemented.

**Decision.** The field is either sourced from the cube router or removed. It
must never report `true` from a hardcoded default, and it must not be left
claiming a capability the server does not have.

**Consequences.** Shipping a documented-but-always-false capability flag is its
own kind of misleading contract. Recorded here so the stub is not mistaken for a
working feature in future audits.

**Evidence (production call site).** The hardcoded `false` at
`saved_query_cache.go:375` is gone; `MVHit` is now derived from the router's
decision (`execRes.CubeHit != nil`) and `QueryExecuteResponse` carries
`CubeHit`/`CubeMiss` from `Preview` through `QueryService.Execute`.

---

### ADR-020: Materialization Staleness Is Status-Driven, Not Clock-Driven
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Query routing

**Context.** `EvaluateMVWatermarkStaleness(mvRefreshedAt, boWatermarkTimestamp)`
compares a materialization's refresh time against a **source watermark**, not
against the current clock: a materialization refreshed ten minutes ago is fresh
unless the underlying data moved on since. `models.PreAggProperties` carries no
watermark column, and the router has no source-watermark source.

**Decision.** The router derives staleness from the **lifecycle status** the
scheduler maintains: `stale` is servable-but-flagged, `active` with a recorded
refresh is fresh, an `active` materialization that has never refreshed is stale
by definition, and every other state (`materializing`, `refreshing`, `idle`,
`failed`) is not query-visible at all.

**Consequences.** An earlier draft passed `now()` as the watermark, which marked
*every* materialization stale on sight — correct-looking code, wrong behaviour,
and a test that would have passed had it not asserted `Stale == false`. The
scheduler (ADR-013) is the component that owns staleness, because it is the
component that knows when the source moved. A future source-watermark column
can replace this without changing the router's contract.

### ADR-021: A Time Rollup Occurs When the Request Omits the Time Dimension
**Status:** accepted · **Date:** 2026-10-01 · **Scope:** Query routing

**Context.** A materialization at `country × day` answering a request for
`country` alone is a time rollup: eleven hundred daily rows collapse to one
total. The initial check only inspected *requested* dimensions for a time term,
so it concluded "no time involved" and allowed the rollup for any metric,
including `AVG` — whose rollup is arithmetically invalid.

**Decision.** A rollup is in play whenever the materialization carries the
cube's time dimension and the request does **not** select it at the same
granularity. Every requested metric must then be distributive
(`Decomposable == true`).

**Consequences.** AVG, division, and derived metrics fall through to base tables
rather than being silently mis-rolled-up. This is the conservative direction:
the router can be slow, never wrong. The test that guards it omits the time
dimension deliberately, which is what makes it a real rollup case rather than an
exact-grain match.

## Open items

- **Call-site verification for ADR-001 … ADR-010.** The imported entries assert
  no call sites because the original recorded none. Verifying each is
  outstanding; ADR-012 and ADR-013 exist because that verification already
  found two gaps.
- **ADR count discrepancy.** The review expected 17 decisions to migrate; the
  `project_history` file contains **10** (ADR-001 … ADR-010). Either further
  decisions live only in session artifacts, or 17 was counted from a different
  source. This registry imports the 10 that exist in version control rather
  than padding to an expected count.
- **Alerting host for `MarkFailed`.** The plan requires an alert when a
  materialization transitions to `failed`, since a scheduler that silently fails
  turns staleness from an edge case into the steady state. The reconciler
  health-check surface is a candidate host but is not yet chosen.
