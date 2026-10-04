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

### Standing rule: registry edits rebase against current `main`

A registry edit is a **rebase-against-current-`main`** operation, never a
branch-off operation. The same-session rule binds the *base* as well as the
timing. A working copy of this file that descends from a pre-merge ancestor
silently lacks every ADR added since, and committing against that copy deletes
them: an edit here once shipped as `+425/−60` and would have reverted ADR-024
outright. Read the diffstat before pushing any change to this file, and confirm
the ADR count did not drop. `git diff --stat` is the only signal that caught it.

### Standing rule: authored identity is deliberate, not ambient

The author recorded on a commit is a decision, like any other in this file, and
not an artifact of whatever git config happened to be live when a commit was
built. Two commits on the C0 freeze PR were authored `opencode <opencode@local>`
while the merged cube work is `mavis <mavis@uisce.local>`, purely because a
plumbing commit path took the ambient identity. Provenance is load-bearing in
this repo, so a reviewer seeing an unexpected author is entitled to ask.

Automation is the obvious guard — a workflow check on protected branches that
compares commit author identity against an expected set is cheap — but the rule
stands on its own even unenforced: **set the author deliberately, and check
`git log --format="%an <%ae>"` before pushing.**

### Standing rule: a shared checkout commits by explicit path list, never `-a`

Several sessions share this repository's working tree, and the tree's uncommitted
files are not necessarily yours. `git commit -a` (and `.`) stages *every*
tracked modification, so the cheapest possible command is the one that most
easily sweeps another session's in-flight work into your commit. This happened
once already: a freeze guard being written in the shared checkout was reported
as "uncommitted changes" by the session that owned the branch, and a routine
`git commit -a` there would have carried it into a frontend PR.

**Stage by explicit path, always.** If `git status` shows a file you did not
write, stop and find out whose it is before committing anything.

### Standing rule: overrides are decisions, and decisions get recorded

An admin override on a required check is a decision to ship without a gate, and
it deserves the same treatment as any other decision here: a written reason,
recorded where the next reader will find it. The first use was **#331** ("ci:
fix a11y JWT env; disable Frontend E2E until it boots a stack", merged as
`695258beb`), where the Security Scan and the flaky `validate (pull_request)`
check were skipped by override. The justification: the change touched workflow
files only, and the scan had passed on the earlier PRs in the same run. That is
a sound reason, and it is only auditable because it was written down.

The failure mode to watch is not a bad override — it is a *habitual* one. If
overrides start appearing without recorded reasons, that is the signal to stop
using them and move the check into branch protection instead.

## Standing evidence rule

A decision marked as *implemented* or *wired* must cite a **production call
site**, not a passing test. A unit test proves a function works in isolation;
it does not prove anything calls it. If a "wired" claim cannot produce a
`file:line` citation in a non-test file, it is not wired.

This rule is bidirectional: it binds delivery reports **and** reviews. A
reviewer who asserts "X is wired" from a service being *constructed* has made
the same error as an author who asserts it from a test passing. Both must
check whether the *scheduler* or *caller* actually runs.

### Extension: the rule binds gate claims too (ADR-022)

**A test proves only what it executes.** A gate tied to a function must name
that function's **production call site** in the gate report.

This extension was not theoretical. During cubed-tables merge review, gate #10
passed while validating `ComputeQueryAndMetricsAndCubeCacheKey` — a function with
zero production callers. The live seam was `BuildCompositeCacheKey`, which had
no cube term, so a payload computed against an undeployed materialization kept
serving its rows. The gate was green and the guarantee was fictional. The same
shape as a dormant component, reached through the test suite instead of the
deployment path.

**Practice:** before claiming a gate, name the function it exercises, then
`grep` for a non-test call site of that exact function. If the grep is empty,
the gate is measuring a fiction.

### Merge record: cubed tables (2026-10-02)

`feat/cubed-tables` merged to `main` as `f6c241f1e` (merge commit, `--no-ff`),
carrying four commits: the registry itself, the `PreAggScheduler` activation,
the cube schema/model/DDL generator, and the router plus the cache-seam and
guardrails fixes.

Two findings came out of merge review rather than delivery, and both are
recorded here because the pattern recurs:

- **ADR-016** was accepted on a passing test over a function with no caller.
  Found by reading the live call path while writing the gate-#11 test, not by
  running the suite. Hence ADR-022.
- **The guardrails YAML fallback** (ADR-023) was found by investigating a
  "failing test" instead of waiving it. The test was genuinely fragile: it
  depended on `os.Remove` succeeding.

Post-merge verification was run against `main`'s tree, not the branch, because
a registry edit that merges as a conflict-resolution casualty would otherwise
be invisible: ADR-019/020/021/023 and the ADR-016 correction record are all
present, `go build ./...` is clean, and the named gates pass on `main`.

**Residue note.** Nine untracked files remain in the feature worktree (a
diagnostic probe, a now-harmless `guardrails.yaml`, an emptied red-proof test,
and editor temp files). None were staged, so they do not affect the merge or
CI, which reflects tracked content only. They are pending deletion on
workstation access; the closeout does not claim a clean workspace until then.

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
**Status:** accepted — **corrected at merge review** · **Date:** 2026-10-01 · **Scope:** Caching

**Context.** A cube deploy or edit changes which materialization serves a
query, so dependent cache entries must be invalidated.

**Decision.** A cube content hash term participates in the cache key. The
no-cube case uses a stable, non-empty marker (`NoCubeCacheTerm`), never an
omitted term.

**Correction record (2026-10-02, merge review).** The first implementation of
this decision validated `ComputeQueryAndMetricsAndCubeCacheKey` with passing
tests, and that function **had no production call site**. The live batch-execute
seam is `BuildCompositeCacheKey`, which had no cube term at all, and whose
`boSchemaVersion` argument was a hardcoded literal `"v1"` at the call site.

Two real defects followed, and both are now fixed with the production call site
cited:

- A payload computed against a materialization that was later undeployed kept
  serving its **rows**. The cache path never consults the router, so every gate
  the router applies (ABAC-below-grain, staleness, decomposability) protects
  only the *miss* path and none of them could catch a stale cache hit.
- No semantic catalog change could ever invalidate a batch-execute entry, since
  the schema-version term was a constant.

**Why it survived delivery:** the gate test (#10) passed against a function with
no callers. This is the test-suite analogue of the dormant-component pattern,
and it is the reason test #13 now extends from "wired claims" to "gate claims."

**Evidence (production call sites).** `BuildCompositeCacheKey` is called from
`backend/internal/querybuilder/saved_query_cache.go` in the batch-execute loop
and now takes the cube term; the hash is resolved by
`QueryService.CubeContentHashForCache`, which reads the router installed at
`backend/internal/api/api.go:1151`. The BO schema-version term is now the saved
query's own content hash rather than a literal.

**Consequences.** A cube deploy, edit, or undeploy invalidates exactly the
entries it should. Failure to resolve a cube hash degrades to
`NoCubeCacheTerm`, which is the safe direction: the entry misses and
re-executes rather than serving a cube-keyed payload.

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

### ADR-023: Guardrails Are DB-Only, With No Filesystem Fallback
**Status:** accepted · **Date:** 2026-10-02 · **Scope:** Bundle validation

**Context.** `loadGuardrails` read `guardrail_rules` from the database and, when
that was empty or unavailable, fell back to probing `GUARDRAILS_PATH` and then
`guardrails.yaml`, `../guardrails.yaml`, `../../guardrails.yaml`.

**Decision.** The database is the only source. There is no YAML file fallback
and no `GUARDRAILS_PATH` probing. A nil DB yields an empty configuration. A
request for the removed `"yaml"` source is an explicit error, not a silent
empty result.

**Consequences.** The fallback made configuration depend on ambient
working-directory state: a stray file on disk could silently override the
database, and the same rule set could be served from two different sources
depending on where the process was started. It also made the test suite
order-dependent — a test that wrote `guardrails.yaml` could still have it
present when a sibling asserted the file was absent, and `os.Remove` failures
were discarded with `_ =`, so the residue survived and failed unrelated runs on
any machine where deletion was restricted.

This is the same lesson as ADR-012, applied to configuration rather than
routing: **a path that can be satisfied two ways is a path whose behaviour is
ambient.** Removing the second way made the guardrail tests independent of the
filesystem and of test order.

**Evidence (production call sites).** `loadGuardrails` (`internal/bundles/handler.go`)
is the single loader, reached by `getGuardrails` and `ReloadGuardrailsHandler`
(registered at the `/guardrails/reload` route). `GUARDRAILS_PATH` and every
`guardrails.yaml` probe path are removed from the package.

**Unrelated and deliberately untouched:** the `config.yaml` DSN probe in the
same file is a database-connection helper, not a guardrail source, and still
uses the `yaml` package.

### ADR-024: The Migration Set Is the Schema Authority, Not Application Code

**Decision.** A table's definition is owned by a migration and by nothing else.
`CREATE TABLE IF NOT EXISTS` in application code is not a schema authority and
must not be relied on to bring a table into existence.

**Context.** ADR-023 made guardrails DB-only, removing the `guardrails.yaml`
fallback. CI then failed `TestReloadGuardrailsHandler_Integration` with
`500 pq: relation "guardrail_rules" does not exist (42P01)`, after a 135-second
TCP timeout. No migration in the repository created that table. It was created
by `EnsureOptimizationSchema` (`internal/bundles/optimizer.go`), which has one
non-test caller — `AnalyzeAndPropose` — and is not called at boot. The table
therefore existed only in databases where the optimizer flow had happened to
run.

**This was not caused by ADR-023, and it is not a flake.** The YAML fallback was
*masking* the gap: with a file present, reload succeeded and never touched the
database. Removing the second source removed the mask. In any database
provisioned from this repository's migrations, `POST /bundles/guardrails/reload`
returned 500 — and always had.

The local suite was green throughout. The developer's database had the table.
A green local run therefore proved only that one machine's database was
populated, which is precisely the claim the evidence rule refuses to accept.

**Consequences.** Migration `20261205_001_guardrail_rules` creates
`public.guardrail_rules` with exactly the schema the application code used, so
the two agree and cannot drift into producing two different tables. The table
stays in `public` because the unqualified `CREATE TABLE` in `optimizer.go` has
always landed there; relocating it would orphan the existing table.

The integration test now resolves its own database explicitly: it reads
`DATABASE_URL`, skips with a stated reason when absent, pins `POLICY_DB_URL` so
the DSN resolver does not wander into a config file first, and resets the
package-level connection singleton either side of the run. The CI integration
job now names `POLICY_DB_URL` as well, which is the same database
`DATABASE_URL` already pointed at — not a new dependency, only a faster and
explicit route to the one already in use.

No RLS is added. Guardrail rules are global policy configuration read wholesale
by the rule engine, not tenant-owned rows; tenant scoping here would be a
behaviour change wearing the costume of a schema fix.

**Follow-up, deliberately not taken here.** The tracked
`backend/config.yaml` carries a private-host DSN, and the resolver consults
`config.yaml` before `DATABASE_URL`. That is why the failing test spent 135
seconds on a TCP timeout before finding the right database. ADR-023 left the
`config.yaml` probe alone as connection plumbing, and this change does not
overturn that; it only stops the integration job from depending on the order in
which candidates are tried. Reordering the candidates, or removing a private
address from a tracked file, is a separate decision.

**Evidence (production call sites).** `loadGuardrails`
(`internal/bundles/handler.go:127`) is the single reader, reached by
`getGuardrails` and `ReloadGuardrailsHandler`. The writers are
`internal/bundles/handler.go:789` (insert), `:804` (list), `:855` (delete),
`:881` (update) — all against `(id, type, data)`, which is the contract the
migration encodes. The legacy creator is
`internal/bundles/optimizer.go:70`; its only caller is
`internal/bundles/optimizer.go:86`.

### ADR-025: The Metric Compiler's Expression Surface Is Frozen Until C3

**Decision.** The metric compiler may not gain expression capability while
C1-C3 unify it onto `internal/rules/vm`. New operators, functions, parsers, or
a second evaluator are refused by a guard, not by convention. The freeze is
enforced by `internal/archguard/metric_compiler_freeze_test.go`.

**Context.** Three expression systems exist: the metric compiler (A),
`internal/rules/vm` (B, the intended survivor, with dialect emission, an
allowlist and a PII gate), and `internal/calcengine` (C, Go-native NAV/VaR). The
existing `TestSingleRuleEngine` guard polices *engine library imports*; it says
nothing about a second evaluator growing inside the metric compiler, which is
exactly where the next one would appear.

**Why a guard and not a comment.** Comments do not fail builds. The freeze has
to be something CI runs, and something that fails *loudly and specifically* when
someone adds the capability anyway.

**How the freeze is expressed.** There is no operator dispatch in the compiler
to enumerate: `compileFormula` (`metric_compiler.go:127`) splits the formula on
whitespace and substitutes `@var` with positional placeholders, passing every
other token straight through. So the enforceable invariant is the
**tokenization contract**, pinned by golden cases that call the exported
`CompileMetric` and assert the exact SQL and arguments, plus a structural pin
on the file's declared func and import surface (methods receiver-qualified, so
a name cannot be shadowed onto another type).

**The freeze expires on evidence, with a date backstop.** The primary trigger
is the C3 gate landing — `internal/querybuilder/metric_compiler_c3_gate_test.go`
— at which point the guard tells you to delete the freeze and let the C3 gate
police the surface. This is ADR-020 applied to the freeze itself: staleness is
status-driven, not clock-driven. The backstop date (`2026-12-31`) exists only so
the freeze cannot outlive its reason without a deliberate human decision; if it
passes without C3 landing, the guard fails and the date must be extended with a
reason rather than quietly edited. No C3 date was defined anywhere in version
control, so inventing a calendar expiry as the *primary* trigger was rejected
rather than guessed.

The backstop is a **decision checkpoint, not a deadline**. On that date — or
when C3 lands, whichever comes first — the freeze's status is reviewed and one
of two things is registered: C3 shipped, so the freeze is deleted and the gate
takes over, or C3 slipped, so a reasoned extension is written down saying why it
slipped and what the new date is. A backstop that forces a written justification
is the honest form of a deadline for a freeze whose purpose is protective.

**Two defects the freeze found on its first run, both pinned as-is.**

1. `metricFormulaUnspacedOperatorIsOneToken` — `"@a*@b"` is a *single*
   whitespace token beginning with `@`, so it is looked up as a variable named
   `a*@b`, misses, and falls through to the neutral `1.0` multiplier. The
   multiplication is silently dropped and the metric compiles to a constant.
   No error is raised.
2. `metricDerivedTwoMetricsTakeNumeratorFromAlphabeticalOrder` —
   `BaseMetricIDs` are sorted alphabetically (`:86`), and the first entry
   becomes the numerator. A declared `revenue / cost` ratio therefore compiles
   to `cost / revenue`. Numerator and denominator are decided by how the metric
   IDs are *spelled*, not by the order the author declared.

Both are pinned as current behaviour on purpose. A freeze records what the code
does; changing it must be a decision, lifted deliberately and recorded, not a
side effect of unrelated work. Fixing them is C1-C3 work.

**A comment that outruns its code.** `metric_compiler.go:70` said "Formula AST
compilation with allowlisted operators" and `:125-126` said it compiles "safe
arithmetic". Neither was true: there is no operator parsing and no allowlist for
formulas. Aggregation *is* allowlisted (`:58`); formula is not, and there was no
upstream validator. The comments now state the actual contract — operators are
authored, governed input and pass through unvalidated — because a comment
asserting a safety property nobody implemented is worse than no comment. The
behaviour is unchanged; only the false claim was removed.

### Amendment: the two defects are fixed, and a third was found

Fixed before the golden corpus (8.3) was built, on the ruling that a corpus
which snapshots inverted semantics would enshrine the bug as the reference.

1. **Derived operand order.** `MetricExpression` gained `NumeratorID` and
   `DenominatorID`. Explicit operands win and are validated against
   `BaseMetricIDs`; absent them, the author's **declared order** is the order.
   The alphabetical sort survives only for the N-operand sum, where operand
   order cannot change the value. Semantic identity no longer depends on how
   IDs are spelled.

2. **`ComputeMetricContentHash` no longer sorts `BaseMetricIDs`.** This was the
   more dangerous half and it was missed on the first pass. Operand order is now
   semantic, and the hash is the cube deploy identity *and* part of the query
   cache key (ADR-011, ADR-016) — so leaving the sort in place would have given
   `revenue/cost` and `cost/revenue` the **same content hash** and the same cache
   entry. The hash would have carried the defect into the one place that decides
   cache correctness. Expect a one-time cache invalidation and cube redeploy for
   existing derived metrics; that is the intended consequence of a changed
   content hash, not a regression.

3. **Tokenizer.** `@var` substitution now happens in place, inside any token,
   so `"@a*@b"` binds both operands and keeps the `*` verbatim. Segments are
   rejoined with `""` precisely so no whitespace is inserted — a Postgres cast
   (`@a::numeric`) and a JSON operator (`@a->>'k'`) would both break if a space
   landed at the substitution point. Bare `@` is not a variable reference.

4. **Third defect, found incidentally.** `cube_ddl.go` consumed
   `compiled.SQLExpr` and ignored `compiled.Args`, so a parameterised formula
   produced `CREATE MATERIALIZED VIEW ... SUM(t0.revenue)*$1` — an unbound
   placeholder in a statement StarRocks must execute. It failed loudly rather
   than silently, which is why it ranks below the other two, but the generator
   was knowingly emitting DDL it could not render. It now refuses a
   parameterised measure with a real reason; a literal-only formula still
   materializes with its operator intact.

   **Scope of that refusal.** Nothing shipped uses `kind: formula`, so the
   refusal refuses nothing today while closing the raw-text-to-StarRocks path
   before its first user. It is deliberately *not* a soft warning. When formulas
   become supported — post-C3, on the unified core — the gate lifts by
   **implementing the validator**, not by deleting the refusal.

`TestMetricEquivalence_Matrix/Case_3` had encoded defect (1) as its
expectation — the metric is named "Margin Ratio", declares revenue first, and
asserted cost/revenue, with a comment reading "Sorted base IDs" as if
determinism were the intent. The cube DDL suite had **no derived metric at
all**, which is how the defect reached the one production consumer of
`CompileMetric`. Both are now covered.

**Consequence to expect.** C2 adds the `vm` resolver and will break
`metricCompilerImportPins` **by design**. That is the freeze working, not a bug
in it: it forces the freeze to be lifted consciously.

**Evidence (production call sites).** `CompileMetric`
(`internal/querybuilder/metric_compiler.go:35`) is the compiler's only public
entry, called from `internal/querybuilder/cube_ddl.go:138` — the only
non-test caller in the tree. `compileFormula` (`:127`) is the formula path.
The freeze guard is `internal/archguard/metric_compiler_freeze_test.go`,
alongside the existing `TestSingleRuleEngine`.

### ADR-026: A Two-Operand Derived Metric Must Name Its Numerator And Denominator

**Decision.** A derived metric with exactly two operands is a ratio, and a ratio
must state `numeratorId` and `denominatorId` explicitly. The bare
`baseMetricIds: [a, b]` form is **rejected**, not honoured and not guessed at.
The N-operand sum is unaffected: order cannot change the value there, so it
stays sorted for determinism.

**Context.** ADR-025 fixed the mechanical defect — operands were sorted and the
first entry became the numerator, so a declared `revenue / cost` compiled to
`cost / revenue`. Honouring the author's declared order was the first fix, but
it leaves the real hazard in place: a ratio's entire meaning is its direction,
and positional order is invisible, undocumented, and decided by nothing more
substantial than array order. Reading intent from a name is how this started; a
correctness fix that reverses a *named* business metric is exactly the change
that deserves a second opinion, and a convention still needs one.

**Evidence, not inference.** There are **no shipped derived metrics**:
`base_metric_ids` appears in no migration, and the only `Margin Ratio` in the
tree is a test fixture. So this ruling costs nothing today and lands before the
first real ratio exists — which is the cheapest moment to require the explicit
form. The alternative, a positional convention, is precisely the thing that
silently inverts the first metric someone authors.

**One validator, three call sites.** `ValidateMetricExpression`
(`metric_definition.go`) is canonical and is called from `CompileMetric`,
`ValidateCubeMetricReferences`, and is exported for any future save path — so
the compiler and the authoring path cannot drift into disagreeing about what a
derived metric may be.

**Honest note on the requested 422.** The ruling asked for rejection at save
time with a 422. There is **no user-facing save path** for
`data_explorer.metric_definition.expression` today: `internal/metrics` is wired
to a handler but writes a *different* table (`metric_definitions`, with
`aggregation_function`/`base_query`), rows arrive by migration/seed/import, and
the reconciler only backfills `catalog_term_id`. So there is no endpoint to
return 422 from. The check is instead enforced at the earliest point that
exists — cube validation, which resolves real metric definitions — plus compile
time. When a CRUD endpoint for governed metric expressions lands, it **must**
call `ValidateMetricExpression`; the function is exported and the omission would
be the defect.

**Where the validator is actually enforced — and one path that is not.** The
obligation was to confirm the check also fires at *import*, so an ambiguous
metric cannot be stored and only rejected at first execution. Confirmed, with
one gap stated plainly:

- `CompileMetric` is the **only live entry to SQL generation from a metric**. It
  has exactly one non-test caller: `cube_ddl.go:138`. Every metric that reaches a
  database today therefore passes `ValidateMetricExpression` first, and the
  rejection cannot be bypassed by any current path.
- **There is no metric import function at all.** `ExportMetricBundle` exists
  (`metric_definition.go:436`); there is no `ImportMetricBundle` counterpart. So
  the import obligation does not attach to existing code — it attaches to
  **8.3's corpus**, which builds the first import path and must call the
  validator there. Recorded so the corpus is not written assuming compile-time
  checking was always sufficient.
- **A second SQL builder bypasses the compiler entirely — now deleted.**
  `starrocks_mv_manager.go:62` hardcoded `measureExpr := "SUM(notional)"` and
  only special-cased `Kind == "aggregation"`, so a derived or formula metric
  reaching it would materialize as `SUM(notional)` — silently wrong, and exactly
  the bug class the cube DDL guard exists to prevent. `NewStarRocksMaterializationManager`
  had **zero production callers** (its policy helpers `EvaluateABACMVCompatibility`
  and `EvaluateStaleMVAction` were used by `cube_router.go:147,169`, but emitted
  no SQL). The SQL-generating half — `StarRocksMaterializationManager`,
  `GenerateMVDDL`, `GeneratedMVDDL`, `ParseStarRocksExplainPlan`,
  `MVHitParseResult` — is removed, along with the three tests that only exercised
  it. The four live routing predicates moved to `mv_routing_policy.go` and keep
  their tests. **Any future materialization path must route through
  `CompileMetric` or call `ValidateMetricExpression` first**; that obligation is
  unchanged by the deletion.

**Evidence (production call sites).** `ValidateMetricExpression`
(`internal/querybuilder/metric_definition.go`) is called from
`CompileMetric` (`metric_compiler.go:80`) and
`ValidateCubeMetricReferences` (`internal/querybuilder/cube_definition.go`),
which is called by the cube deploy path. `MetricExpression.NumeratorID` /
`DenominatorID` are the authored data.

### ADR-027: A Cube's Content Hash Covers Its Referenced Metrics' Content

**Decision.** `ComputeCubeContentHash` folds in the content hashes of every
metric the cube names, sorted and normalized exactly as the query cache key
does. A metric definition edit is therefore a cube change: the hash moves, and
under ADR-011 the next deploy is a real redeploy rather than a no-op.

**Context and the asymmetry it closes.** The two invalidation layers disagreed.
The *query cache* already self-healed — `ComputeQueryAndMetricsAndCubeCacheKey`
folds each referenced metric's content hash into the key, so a changed metric
simply never hits its old entry. The *deploy* did not: `ComputeCubeContentHash`
hashed metric **IDs** only, never their content. So a metric edit left the cube
hash untouched, ADR-011's "unchanged hash ⇒ deploy is a no-op" rule kept the
deploy from firing, and a materialized view served the pre-edit expression
indefinitely. Cache and deploy were asymmetric, and only one of them was safe.

This is the same class as the ADR-016 defect: **a hash that cannot see the thing
it must distinguish**. That one put a cube state into a cache key; this one
removed a cube's own change from its identity.

**Consequence.** A deploy caller must pass the resolved metrics' content
hashes. An empty slice is tolerated for validation-time use but yields a hash
that does not track the cube's metrics, so the deploy path is the one place
that must not pass nil. There is no migration concern: **no cube is deployed or
seeded anywhere**, so there are no live hashes to invalidate — the rule lands
before the first real cube, the same way ADR-026 did.

**Evidence (production call sites).** `ComputeCubeContentHash`
(`internal/querybuilder/cube_definition.go:106`). The symmetric behaviour it now
mirrors is `ComputeQueryAndMetricsAndCubeCacheKey`
(`internal/querybuilder/metric_compiler.go:194`). The test is
`TestComputeCubeContentHash_TracksReferencedMetricContent`.

### ADR-028: The Golden Metric Corpus Is Versioned In-Repo And Ingested, Not Read

**Decision.** A versioned metric corpus lives at
`backend/internal/querybuilder/testdata/metric_corpus/v1/`, built on
`ExportMetricBundle`'s format with no parallel exporter, and it is **ingested
through `ImportMetricBundle`** before any metric in it is compiled. Corpus
version `uisce.metric-corpus/1` is tracked separately from the bundle schema
version, because the corpus changing shape and the wire format changing are
different events. Regenerate with `go test -run TestMetricCorpus -update-corpus`
and **read the diff** — it is a change to the semantics the corpus exists to
protect.

**The importer is the first non-compile consumer of the validator.**
`ImportMetricBundle` (`internal/querybuilder/metric_bundle_import.go`) is the
counterpart to `ExportMetricBundle`, and it is where ADR-026's obligation lands.
A bundle can arrive by migration, seed, import or corpus and never touch
`CompileMetric`; without validation at ingestion, an ambiguous metric would be
stored and rejected much later, at query or deploy time, far from whoever
authored it. It runs four passes — shape, content, references, cycles — and
rejects with the offending metric named.

**The corpus records what the code does, including what is wrong.** Two cases
are pinned deliberately rather than fixed:

- `m_calc_fed_core` — a calc-fed metric, pinned as `SUM(t0.<calcTermId>)`, a
  column that does not exist. There is no `calcTermID` field on
  `MetricExpression`; routing a metric through a calc term is the C1/C2 gap.
  Its expectation carries a `knownDefect` field saying so.
- The corpus is only trustworthy if a wrong expectation is visibly wrong. An
  unmarked defect would let the corpus quietly become the defence of a bug —
  which is precisely what would have happened to the ratio semantics had the
  corpus been built before ADR-026.

**What the corpus caught on its first run.** The first generated expectations
recorded `m_margin_core` — a ratio naming revenue as numerator — as
cost/revenue, and `m_cost_ratio_core` with the operands named the other way
round as the *same* SQL. The two were indistinguishable. The cause was not a
code regression: the working tree mixed `metric_definition.go` from merged
`main` with a pre-ADR-026 `metric_compiler.go`, so the old sorted behaviour was
being recorded. `origin/main` was correct throughout. Had the generated
expectations been committed without being read, the corpus would have enshrined
the inversion — the same trap, entered from the other direction.

**Evidence that the gate bites.** Reverting the numerator logic in
`metric_compiler.go` makes `TestMetricCorpus` fail, naming `m_margin_core` and
stating that this is a change to protected semantics. Regenerating instead of
reading is how a corpus becomes decorative.

**Span.** Core (master-tenant, `IsCore`) and custom metrics; aggregation,
formula and derived — ratio *and* n-operand sum; two tenants, so nothing can
assume a single one; a cross-tenant base reference, resolved through the
import's `existing` set. A parameterised formula is deliberately **absent**: it
is refused at cube deploy (ADR-025), so it has no stable compiled output to
protect.

**Evidence (production call sites).** `ImportMetricBundle`
(`internal/querybuilder/metric_bundle_import.go`) and `ExportMetricBundle`
(`internal/querybuilder/metric_definition.go:436`). The corpus is exercised by
`TestMetricCorpus` and `TestMetricCorpus_RejectsAmbiguousRatioAtIngestion`,
which strips each corpus ratio's operands and asserts the importer refuses it —
so the corpus is a boundary test, not a pile of fixtures.

### ADR-029: `alpha` Is the Control Plane; the Tenant Datasource Model Is the Registry

**Decision.** `alpha` holds the metadata and audit for every tenant and is the
only control plane. Where a tenant's data plane lives is recorded by the
existing datasource chain — `tenants` → `tenant_instance` → `tenant_product` →
`tenant_product_datasource` — plus a sibling `tenant_datasource_binding` table
(keyed by `tenant_product_datasource.id`) for the warm and cache tiers (StarRocks database
and role, Redis ACL user and prefix, lifecycle state) and the datasource's lake
namespace. The cold tier is bound per *tenant*, not per datasource, in
`tenant_lakehouse` (ADR-032). No second registry is created, and
`internal/ops/region_router.go` reads from this model rather than holding its
own mapping.

**Invariant: metadata never leaves `alpha`.** All metadata for all tenants, and
the audit of changes to it, lives in `alpha` and stays there. Nothing in this
group of decisions moves, tiers out, archives away or drops a metadata or audit
row from `alpha`. Other stores (StarRocks, the lake) may hold *copies* for
analytics, history or immutable retention; the copy is never the system of record
and its existence is never a reason to delete from `alpha`. Only a tenant's own
operational data (the data plane) is tiered.

**Context.** The datasource chain already names a tenant database
(`config.host/port/database`) and a credential reference (`secret_path`,
validated by `dscreds.CanonicalPath`). A parallel `ctl.tenant_resource` table
would have been a second source of truth for the same fact.

**Consequence.** Gold-copy metadata and the tenant overlay stay in `alpha`
(read-only inheritance, never written back). Tenant databases hold operational
data only. Connection DSNs are never stored in plain text; `tenant_connections.dsn`
moves to the `dscreds` secret-path pattern.

**Why not `public.tenant_datasources`.** That table already keys a tenant
datasource (`tenant_id`, `datasource_id`) and carries `resource_group`,
`provisioning_status` and inline `username`/`password` columns, but it is created
only in `backend/migrations/20251130_007_tenant_automation.sql`, a directory the
runner does not read (`backend/db/MIGRATION_DIRECTORY_DRIFT_AUDIT.md`), so it is
not reproducible from the migration set (ADR-024). The binding table is a new,
migration-owned table holding references only. `tenant_datasources` stays the
provisioning-state table until it is adopted or retired separately; its inline
credentials are drained by `cmd/migrate-datasource-creds`.

**Evidence.** `internal/security/datasource_resolver.go` (`Resolve`),
`internal/dscreds/dscreds.go` (`CanonicalPath`, `Hydrate`),
`db/migrations/20261206_001_tenant_datasource_binding.up.sql`.

### ADR-030: Tenant OLTP Data Lives in a Database Per Datasource, Reached Only Through `tenantdb`

**Decision.** Each OLTP datasource (e.g. ORM) gets its own Postgres database.
All access goes through `internal/tenantdb.Resolve(ctx, datasourceID)`, which
resolves the owning tenant, checks it against the caller's tenant, hydrates
credentials through `dscreds`, and returns a pooled connection. Every checkout
asserts `current_database()` equals the datasource's configured database. Any
failure — missing, ambiguous or unauthorized tenant — returns an error; there is
no fallback and no dev override.

**Consequence.** Archguard forbids opening a database handle for a tenant
datasource outside `tenantdb`. Tenant migrations are applied per target by the
migration runner, with the same sha256 drift check and fix-forward rule as
`alpha` (ADR-024).

### ADR-031: Lakekeeper Is the Single Iceberg Catalog; Polaris Is Removed

**Decision.** Lakekeeper is the only Iceberg REST catalog. The Polaris
provisioner, its onboarding call, its stub activities and its environment
variables are deleted. Namespaces and warehouses are provisioned through
`internal/iceberg/lakekeeper_provisioner.go`. Nessie, still defined in
`docker-compose.starrocks.yml` and `starrocks_init.sql`, is not removed by this
decision and remains to be retired separately.

**Context.** Polaris had already been replaced in deployment: the compose files
define Lakekeeper (with its own database, migrate job and a separate
`lakekeeper-gold` service for the gold-copy plane) and no Polaris service, and the
only Polaris deployments in history are two early commits. The code did not follow.
`OnboardTenant` still called `http://uisce-polaris:8185`, a host defined nowhere in
the repository. An earlier draft of this ADR was written before that was checked,
and a first decision to keep Polaris was reversed once it was.

**Consequence.** `OnboardTenant` no longer provisions any catalog and returns
`lakehouse_status: "unconfigured"`; a tenant's warehouse is configured and
provisioned separately (ADR-032), because its audit retention is a per-tenant
decision. The `polaris_catalog_url` response field is removed; nothing in the
repository read it.

### ADR-032: Cold Storage Isolation Is Bucket, Key and Lock Per Tenant

**Decision.** Every tenant has exactly one Iceberg warehouse, and a warehouse
belongs to exactly one tenant. The warehouse is a Lakekeeper warehouse over its
own bucket (`ivy-t-<tenant id without hyphens>`), with its own KMS key and a
tenant-scoped STS policy. The one-to-one rule is enforced by the database, not by
convention: `public.tenant_lakehouse` has `tenant_id` as its primary key, and
`CHECK` constraints force the warehouse and bucket names to be derived from that
id. A tenant's datasources do not get warehouses; each gets a *namespace* inside
the tenant's one warehouse (`tenant_datasource_binding.lake_namespace`).

`ivy-control` is the platform's own warehouse for cross-tenant audit and
metadata history and belongs to no tenant. Audit uses Object Lock in compliance
mode. Offboarding destroys the tenant's key (crypto-shred) after the retention
period. The gold-copy tenant is a tenant like any other and gets its own
warehouse; gold-copy inheritance is metadata in `alpha` (ADR-029), never shared
lake storage.

**Context.** Today every tenant shares one bucket and one default warehouse,
separated only by a namespace named after the tenant
(`LakekeeperProvisioner.CreateNamespace` posts to `/v1/namespaces` with no
warehouse selector). A namespace is a naming convention; a warehouse is a
storage, credential and authorization boundary. Per-warehouse storage credentials
are already proven by `cmd/smoke`.

**Consequence.** `tenant_lakehouse` references `tenants` with `ON DELETE
RESTRICT`: the warehouse holds WORM audit and must outlive the tenant row.
Existing tenants' tables must be migrated out of the shared warehouse into their
own; that data move is a separate, reversible step. Beyond roughly 2,000 tenants
this moves to prefixes with per-prefix IAM; that threshold is the trigger to
revisit this ADR.

### ADR-033: StarRocks Tenant Isolation Is a Database and Role, Not a Row Filter

**Decision.** Each tenant has a native StarRocks database `t_<short>` and a
role `r_<short>`, with a resource-group classifier by tier. Cold data is exposed
through read-only views over that tenant's Iceberg namespace only. Cross-tenant
analytics runs only under a platform role against `ivy_control`.

**Context.** Today's isolation is a `tenantID` argument and a `tenant_id`
column (`internal/analytics/starrocks_client.go`); one forgotten predicate leaks
a tenant. A database boundary is enforced by the server, not by the caller.

### ADR-034: Redis Keys Are Built Only by the Tenant Key Builder

**Decision.** Every key is produced by `cache.Key(ctx, ns, parts...)` as
`t:{<tenantId>}:<ns>:…`; global keys use `g:`. Each tenant has an ACL user
limited to `~t:{<id>}:*`. Pub/sub channels follow the same prefix. Archguard
bans raw key formatting outside the builder.

**Consequence.** The global `tenant:goldcopy:id` key and the ad-hoc prefixes
(`semantic_view:`, `nl:`, `sq:`, `ratelimit:`, `notifications:tenant:`) migrate
to the builder.

### ADR-035: Tenant Data Tiers Hot → Warm → Cold; `alpha` Metadata Is Never Tiered Out

**Decision.** This applies to a tenant's **operational data** (the data plane),
not to metadata. Hot is the tenant's Postgres database (90 days), warm is
StarRocks native (13 months), cold is Iceberg. Event-like tables in a tenant
database are range-partitioned and flow through CDC and Kafka to Iceberg. A tenant
database partition is detached only after a verification job proves the Iceberg
copy matches (row count and, where present, seal chain). Retention is enforced by
dropping those partitions (ADR-018); a legal hold suspends drops.

**Not covered, deliberately.** Nothing here applies to `alpha`. Metadata and the
audit of changes to it are never detached, archived away or dropped (ADR-029).

**Consequence.** ADR-009 ("audit log never purged") is satisfied by `alpha`
keeping its audit log permanently. A copy may additionally be written to the lake
under Object Lock (ADR-032) for immutable retention and time-travel queries; that
copy adds a guarantee and removes nothing from `alpha`. An earlier draft of this
ADR said Postgres would hold only a hot window of audit; that is withdrawn.

### ADR-036: The Lakehouse Audit Copy Is Written By StarRocks, Resumes From The Destination, And Never Replaces `alpha`

**Decision.** A tenant's lakehouse audit chain (`tenant_lakehouse_audit`) is copied into an
Iceberg table `audit.lakehouse_events` in that tenant's own warehouse (ADR-032). `alpha`
remains the system of record and nothing is removed from it (ADR-029); the copy adds an
immutable, queryable second record under Object Lock.

- **Engine:** StarRocks, through a per-tenant external catalog (`ivy_t_<tenant id>`) over
  Lakekeeper and `INSERT INTO`. Go has no mature Iceberg writer, StarRocks is already
  deployed (3.3), and it is the engine per-tenant isolation (ADR-033) already assumes. The
  catalog carries the tenant's OWN storage credential, never a shared one.
- **Resume point is the destination, not a counter.** Each run reads `max(id)` from the
  Iceberg table and ships the rows after it. A separate watermark can drift from the data
  (a crash between insert and update would duplicate rows on retry); the destination cannot.
  `tenant_lakehouse.audit_copied_through_id` is a status marker for operators only and never
  decides what is shipped.
- **Continuity is checked.** The first row shipped must link (`prev_hash`) to the hash of the
  last row already in the destination, so a gap or a reordered chain stops the run and is not
  copied over.
- **One writer per tenant** (a workflow id per tenant), because an Iceberg append is not
  idempotent.
- **Batched on a schedule, not per event.** Under compliance-mode Object Lock every commit's
  data, manifest and metadata files are kept for the whole retention period, so many tiny
  commits would be stored for years. Rows are shipped in batches, and nothing is committed
  when there is nothing to ship.

**Consequence.** Snapshot expiry and orphan cleanup cannot delete locked files; the
maintenance workflow must not be expected to reclaim space in a tenant warehouse until
retention lapses. The DDL, property names and Iceberg write behaviour are written against
the documented StarRocks 3.3 and Lakekeeper interfaces and the properties this repository
already uses; they have not been run against the deployed instances and need a smoke run
before anything depends on them.

### ADR-037: Inside One Tenant Bucket, Object Lock Is Compliance For Everything; History Cannot Be Governance-Mode

**Decision.** A tenant has one bucket (ADR-032), and `EnsureTenantBucket` sets its default
Object Lock retention to COMPLIANCE for `audit_retention_days`, refusing any other mode
(`ErrBucketConflict`). Every object the tenant's warehouse writes inherits that default. We do
**not** plan on a second, governance-mode bucket or on per-object retention for "history": the
writers of tenant Iceberg data are StarRocks and Lakekeeper, which cannot set a per-object
retention header, so an `audit` namespace and an `<app>` namespace in the same bucket are locked
identically. The audit/history distinction is therefore a *table* distinction (namespace and
maintenance rules), not a storage-lock one.

**Consequences.**
- Tenant history written to the lake is retained for at least the tenant's audit retention and
  cannot be dropped earlier, even by an operator. That is acceptable only because the lake holds
  copies; `alpha` and the tenant's Postgres remain where data is created and, for metadata,
  kept (ADR-029, ADR-035).
- Retention-driven drops of tenant history (ADR-035) therefore act on Postgres and StarRocks
  partitions, not on lake objects, until the lock lapses. Snapshot expiry and orphan cleanup
  skip every table in a tenant warehouse until then (same limit as ADR-036).
- Raising retention applies to new objects only (already recorded in `tenant_lakehouse`).
- If a tenant needs history that can be deleted before the audit term, that needs a second
  bucket and warehouse for that tenant, which contradicts one-warehouse-per-tenant. It is a
  decision to take per tenant, with this ADR revisited, not a default.

**Evidence.** `internal/iceberg/tenant_bucket.go` (`ensureObjectLock`, `ExtendTenantRetention`).

### ADR-038: The Binding Carries Lifecycle Windows and Legal Hold, As Additive Columns

**Decision.** `tenant_datasource_binding` gains `pg_cluster`, `hot_window_days` (default 90),
`warm_window_months` (default 13), `legal_hold` and a uniqueness guard on `redis_key_prefix`, by
a forward migration of `ADD COLUMN IF NOT EXISTS` and a new index only (ADR-024). Nothing in
`20261206_001` is edited, dropped or retyped. Cold retention is not repeated here: it is
`tenant_lakehouse.audit_retention_days`, per tenant (ADR-032). Maker-checker approval on
binding changes is not added by this decision; it needs its own, using the staging-binding
approach rather than invented columns.

**Consequence.** The tiering job (ADR-035) reads the windows and `legal_hold` from the binding;
`legal_hold` suspends partition drops and never touches Object Lock.

### ADR-039: Drift Is Reported, Not Remediated, By a Reporter Over Existing Sources

**Decision.** Any reconciler of registry versus tier state is a reporter. Its expected state
comes from the existing sources (`InspectProvisioningState`, the retention reconcile and the
registry), it reports `missing`, `orphan`, `drift` and `unknown` (a failed probe is `unknown`,
never healthy-by-absence), and it never executes. Remediation stays with the workflow that owns
the lifecycle change, so each change has one execution path and no second provisioner exists.
It must never auto-remediate a resource under compliance Object Lock.

### ADR-040: A Successful Authorization Is Cached For A Few Seconds; A Refusal Never Is

**Decision.** `tenantdb.Router` may reuse a **successful** authorization (ownership, completeness,
active binding, and the app-to-datasource lookup) for `Config.AuthTTL`; production wiring uses
`DefaultAuthTTL` (3 s). `0` disables the cache and restores a full alpha check on every `Resolve`.

- **Keyed by the verified caller tenant** as well as the datasource, so one tenant is never served
  another's cached entry.
- **Only successes are stored.** Every refusal (unbound, suspended, mismatched, incomplete, registry
  down) is evaluated afresh each time, so recovery is immediate.
- **Bounded:** `MaxAuthEntries` (default 4 x `MaxPools`); expired entries go first.
- **Coalesced:** concurrent refreshes of one key cost alpha one lookup, detached from any single
  caller's cancellation so one cancelled request cannot fail the others.
- **`Probe` never uses it:** it is the last gate before a binding goes active.
- **`InvalidateDatasource` / `InvalidateTenant`** make an in-process suspension, offboarding or
  rebind take effect at once.

**The trade, stated plainly.** After a suspension, offboarding or cutover in alpha, a process keeps
resolving the old answer for at most `AuthTTL`. That is a fail-closed *latency* decision, not a
performance tweak. In exchange a warm `Resolve` no longer costs alpha 18 statements: at 200 tenants
it went from p50 0.74 ms / 5,986 resolves/s (32 goroutines, bounded by alpha's pool) to microseconds
and over a million resolves/s, with zero alpha statements.

**Consequence for cutovers (Phase 4b).** A cutover is a registry flip plus a binding version bump. With
the cache on, each process follows it within `AuthTTL`, so the cutover procedure is: pause writes, flip,
wait at least `AuthTTL` (or invalidate in-process), verify, resume. Writes must not resume before every
process has converged.

### ADR-041: The Hermetic Backend Suite Is Sharded, Not Shortened

**Decision.** `go test -short -race -coverprofile=... ./...` leaves `build-backend` and becomes a
4-way `backend-tests` matrix, partitioned by `go list ./... | awk '(NR-1)%n+1 == i'`.

**Why this and not something else.** The suite was measured at **7.6m of a 13.4m job** — the single
largest item on the pipeline's critical path, and the only part still growing linearly with the
package count (498 today, `backend/` only). Everything else in the job was build-shaped work that
does not parallelise: `Build & Vet` 2.4m, `Build Binaries` 1.4m, `Checkout` 1.1m, the ASL drift gate
0.1m. Sharding takes the suite to ~1.9m per shard and the critical path to roughly the build tier,
~6m. Caching was already on (`setup-go` `cache: true`, `setup-node` `cache-dependency-path`), so it
was not a lever.

**No test is removed or skipped.** The union of the four shards is exactly `go test ./...`; the only
change is that they run concurrently. `-race` and `-short` are kept on every shard — dropping
`-race` to non-default branches was considered and rejected here, because it trades a real defect
class (the `WithOptimizer` race this engagement already found) for wall-clock, and the sharding gets
most of that wall-clock back without the trade.

**Two properties were verified, not assumed, before the workflow was touched.**

1. The shards *partition* the package set — checked as a bijection: 498 in, 498 assigned, 498
   unique, none in two shards. A package that fell through every shard would be a silent coverage
   hole, which is the failure mode that makes a green gate meaningless.
2. Coverage profiles from disjoint package sets *concatenate* into one valid profile — checked by
   generating two real profiles and running `go tool cover -func` on the concatenation. Each shard
   uploads under the same `backend` flag and codecov merges server-side, which keeps a merge job off
   the critical path.

**Not gated on `build-backend`.** This job compiles too, so a build failure fails both; adding
`needs: build-backend` would serialise ~6m of build ahead of ~4m of test and surrender most of the
gain.

**The first draft of this shard selection was wrong, and the way it was wrong is the point.** The
matrix was `[1,2,3,4]` and the expression was `NR % n == i`. Because `NR % 4` only ever yields
0–3, **shard 4 selected zero packages, reported success, and left 124 of 498 packages untested.**
The local verification had passed because the throwaway script used 0-based shard ids while the
workflow used 1-based — two different expressions, one of which was never the thing that shipped.
It was caught by running *the workflow's exact expression* against the real package list before
pushing, not by review. The step now carries a count guard (`< 100` packages fails loudly) so a
broken partition cannot report green again.
### ADR-042: `orm` Reaches A Tenant Database As A Migration, Without RLS, And Without Its Three `oms` Foreign Keys

**Decision.** The ORM (crims) schema is carried into a tenant's own database by a
tenant migration under `backend/db/tenant_migrations/orm/`. It arrives **without row-level
security** and **without its three foreign keys into `oms`**, and the shared-reference read
model is served by a per-tenant copy rather than by a policy.

**How the table set was established, because getting this wrong is expensive.** Two answers
were available and they disagreed, so the migration set was treated as authoritative
(ADR-024) and the snapshot used only as a cross-check:

- `backend/db/snapshots/schema-snapshot.sql` shows **0 of 34** `orm` tables carrying
  `tenant_id`. Taken alone it says no ORM table is tenant-scoped, and the move looks
  impossible.
- The migration set says otherwise. `20261017_001_orm_tenant_order_chain.up.sql` gives
  `tenant_id uuid NOT NULL` to **7** tables (`account`, `broker`, `"order"`,
  `order_allocation`, `placement`, `execution`, `execution_allocation`), and
  `20261026_015_orm_rls_new_tables.up.sql` puts **27 more** under RLS policies keyed to
  `app.current_tenant`.

The two reconcile exactly: 27 listed − `quote_default` (a partition of `quote`, not a table)
= 26, plus 7 = **33 real tables**, and the snapshot's 34 lines are those 33 plus the
partition. **Every `orm` table is tenant-scoped; none is global.** The snapshot is stale
(generated Oct 2 22:28, before `20261017_001`) and is regenerated with its migration log
as one pair before either is trusted again.

**What RLS was doing, and why it stops.** The policy pair is not isolation, it is a
*shared-data* mechanism: read allows the tenant's own rows **or** the shared reference
tenant's, defaulting to the sentinel `00000000-0000-0000-0000-000000000001`; write is the
tenant's own rows only. In a shared database that sentinel is how 200 tenants read one copy
of reference data. In a **per-tenant database the database is the boundary** (ADR-030's
principle that isolation is structural, not filter-based), and a policy that admits rows
owned by a different tenant id is the row-filter model this architecture moved away from.
The tenant migration therefore creates the columns and no policies, and the reference rows
are copied into each tenant database as part of the move (ADR-043) — read-only by
convention, enforced by the fact that nothing writes them there.

**The three `oms` foreign keys are dropped, deliberately and explicitly.**

| Constraint | From | To |
|---|---|---|
| `fk_ama_account` | `orm.account_model_assignment` | `oms.account` |
| `fk_bi_security` | `orm.basket_item` | `oms.security` |
| `fk_pl_position` | `orm.position_lot` | `oms."position"` |

`oms` stays in `alpha` (ADR-029: metadata never leaves `alpha`), and a foreign key cannot
cross databases. The alternative — carrying a local copy of those three `oms` tables — was
rejected because it would fork the position and account of record into a second store that
`alpha` does not know about, which is a worse failure than a missing constraint. The
columns stay; the constraint does not. **This is written down because "the constraint
silently vanished" is the failure mode that gets rediscovered**, and because the three
tables still carry the columns, so nothing about the schema announces that the guarantee is
gone.

**Consequence.** `internal/migrations/ormmove` owns the ordered, dependency-checked copy.
Nothing edits an applied migration (ADR-024); the reference-row handling and any future
constraint work are new numbered files.

### ADR-043: The ORM Move Is A Fleet Of Per-Tenant Copies, Verified Before The Cutover, And The Cutover Is A Registry Flip

**Decision.** Moving ORM data is a `migrations.MoveFleet` rollout of one idempotent,
self-verifying copy per tenant, followed by a registry flip. It is not a dump-and-restore
and not a one-off script.

**Per-tenant, in dependency order.** The copy order is derived from the `orm → orm`
foreign-key graph (16 constraints) rather than hand-listed, and
`internal/migrations/ormmove` asserts that order against the declared edges so it cannot
rot silently. Roots are `order`, `basket`, `model_portfolio`; `position_lot` has no
internal parent at all, its only reference being the dropped `oms."position"` one.

**Idempotent and resumable.** Each table is copied with a tenant predicate and an
`ON CONFLICT DO NOTHING` on the primary key, so a second run is a no-op rather than a
duplicate-key error, and a run interrupted mid-fleet resumes without redoing completed
tenants. `MoveFleet` reuses the wave, concurrency and stop-after-a-bad-wave semantics that
`migrations.Fleet` already has for migrations, and keeps the same property that matters
most: **a bad tenant does not take the fleet down with it**.

**A tenant whose counts do not match is reported and not cut over.** Every table is counted
on both sides after the copy and the tenant is marked `Done` only when they agree. This is
the whole safety property of the move: a partial copy that reports success is worse than no
copy, because the cutover is a one-way door for writes.

**The cutover is a registry flip, not a deployment.** Per ADR-040 a cutover is the binding
version bump plus, when every process cannot be restarted, a wait of at least `AuthTTL`
(3s in production) or an in-process `InvalidateDatasource`. Writes pause, the flip happens,
convergence is awaited, the result is verified, and only then do writes resume — **not
before every process has converged**, because a process still inside its `AuthTTL` is
still writing to the old database. The shared database stays readable through the
verification window; dropping it is a separate, reversible step.

**The rollback is the status quo.** Until the shared database is dropped, reverting is a
binding flip back. That is why this decision does not delete anything.

### ADR-044: The Iceberg Audit Copy Is Verified Against `alpha` By A Read-Only Workflow; A Finding Is A Result, Not An Error

**Decision.** `TenantLakehouseAuditVerifyWorkflow` proves a tenant's Iceberg audit copy (ADR-036)
matches `alpha`, and is the gate ADR-035 requires before anything depends on the copy (a hot
partition detach, a retention drop). It reports and never repairs, copies, or marks anything.

- **`alpha` first.** Its own chain is recomputed once (`tenant_lakehouse_audit_verify`). A copy that
  faithfully matches a record that no longer recomputes proves nothing, so that is reported as
  `alpha_chain_broken` before any comparison starts.
- **Lag is not a finding; a hole is.** The copy runs every 15 minutes, so entries `alpha` holds beyond the
  copy's highest id are *pending*. The report says how far the proof reaches (`ThroughID`). An entry `alpha`
  has *below* that id and the copy lacks is `missing_in_copy`, because the copy only appends in order.
- **Every field is compared**, not just the hash: timestamp at the microsecond the database keeps, actor,
  role, action, before/after as JSON data (key order is not meaning), and `prev_hash`/`hash`. The copy's
  own `prev_hash` linkage is checked separately, so two records that agree and are both mislinked are
  still caught (`chain_break`). Unreadable JSON on either side is a difference, never a pass.
- **A finding is returned, not raised.** Retrying cannot change it, so it is a result with a nil error;
  only an outage is retried. A finding names the entry and the field and never a value, because audit
  payloads can hold personal data and the report is logged.
- **Bounded history.** One run is 50 pages of 1000; a longer chain continues as a new run from its cursor
  (`AfterID`, the previous hash, rows so far), so the link is carried across the seam and `alpha` is not
  re-checked from the start.

- **The outcome is recorded, and is not monotonic.** Each run replaces `tenant_lakehouse`'s
  `audit_verified_through_id`, `audit_verified_at` and finding kind/entry in one statement
  (`20261215_001`), so a later bad run takes back an earlier pass. This is the opposite of
  `audit_copied_through_id`, which only rises. Only the final run of a continued chain records, since
  only it knows the outcome. A run that cannot record fails, so silence is never read as a pass; a run
  that cannot finish records nothing, and the earlier outcome stands with its age showing. The finding
  kind is a `CHECK`-constrained enumeration, so a value from a payload cannot be stored there.
- **Scheduled nightly** (`TenantLakehouseAuditVerifyAllWorkflow`), a few tenants at a time; a finding or
  failure for one tenant is in the result and never fails the run. It only reads, so it adds nothing
  under Object Lock.

**For the tiering job.** A hot partition may be detached or dropped only when the tenant's recorded
finding is empty, `audit_verified_through_id` covers the partition's last entry, and
`audit_verified_at` is recent enough for the job's own tolerance. The gate itself is not implemented
here. The StarRocks read (`AuditRange`) is written against the documented 3.3 interface like the rest of
ADR-036 and has not been run against a live instance.

### ADR-045: The Platform Warehouse `ivy-control` Is Built By The Same Code As A Tenant's

**Decision.** `ivy-control` (ADR-032) is created by `EnsureControlBucket` and `EnsureControlWarehouse`,
which call the same `ensureBucket` and `ensureWarehouse` the tenant versions call. So the platform's
storage cannot be weaker than a tenant's: Object Lock at creation, COMPLIANCE for the whole retention,
default SSE-KMS under its own key, and a warehouse over its own bucket with a credential scoped to that
bucket. The KMS key and the retention have no default, and an existing bucket is never re-locked or
re-keyed (`ErrBucketConflict`). Its name cannot fall in the tenant namespace (`ivy-t-`), and a test
holds that.

**The credential has its own path, not a reserved tenant id.** Tenant credentials live at
`/lakehouse/<tenant uuid>` under a MinIO account `ivy-lh-<12 hex>`. The platform's lives at
`/lakehouse/platform/ivy-control` under `ivy-lh-platform-control`, so it can never equal a tenant's, and the
platform is never given a tenant id to stand in for it. It is issued by the same `ensureCredential` (stored before
created, policy limited to the one bucket, healed on re-run), and the same rule applies: whether one may be
minted is decided by the registry, not the secrets store, which reports an outage as "not found". Once issued, a
missing credential is `ErrCredentialLost` and needs a person.

**The registry row** is `public.platform_lakehouse` (`20261216_001`): exactly one row (the name is the primary key
and the only legal value), no tenant and no row-level security since it holds no tenant data. Retention can only
rise, the warehouse id and the credential-issued time are set once, and a provisioned row cannot be deleted. It
has no audit chain of its own; a run's result and Temporal history are the record.

**Provisioning is started by an operator, never at boot**, because the retention cannot be shortened afterwards and
must be an explicit number:

    temporal workflow start --task-queue bp_queue --type PlatformLakehouseProvisioningWorkflow \
      --workflow-id platform-lakehouse-provision --input '{"RetentionDays": 2555, "ActorID": "<you>"}'

Re-running with the same or a lower number does nothing. A higher number against an already-provisioned bucket is
refused and writes nothing: raising an existing bucket's default retention is a separate operation, and a record
must not claim retention the bucket does not enforce. As for a tenant, there is no compensation: a failed run
never deletes the bucket, and the next run resumes where it stopped. The activities are not registered as BP
Designer client-safe.

The Lakekeeper and MinIO calls are written against the same interfaces the tenant path already uses and have not
been run against a live instance for the platform warehouse.

## Open items

- **C1 (9.1) ported metric primitives into the rule VM.**
  `MetricVariable`, `MetricFormatConfig`, `MetricExpression`,
  `DeriveDecomposable`, `NormalizeFormulaForHash` and
  `ComputeMetricContentHash` now live in `internal/rules/vm`
  (`metric_expression.go`), and the `querybuilder` implementations
  **delegate** rather than reimplement. One implementation, so the compiler's
  canonicalization and the content hash cannot drift. Recorded because the port
  is a *move*, and the next person will otherwise look for the originals.

  Two properties are pinned by this decision and must survive it:

  - **Operand order is semantic.** `BaseMetricIDs` is deliberately unsorted, and
    `NumeratorID`/`DenominatorID` (ADR-026) are hashed as distinct fields.
    Sorting, or dropping the named operands, makes `revenue/cost` and
    `cost/revenue` share a content hash — and that hash is the cube deploy
    identity and part of the query cache key, so the two opposite metrics would
    share a cache entry. This is not hypothetical: the first port attempt
    dropped the two named-operand fields, and the 8.3 golden corpus caught it.
  - **The hash input surface is explicit.** `MetricContentInput` is a flat
    struct, not the full `MetricDefinition`, so adding a definition field does
    not silently change deploy identity or cache keys.

  The port is behavioural, and the golden corpus
  (`internal/querybuilder/testdata/metric_corpus`) is the evidence: it was
  regenerated *not at all*, and the failures it reported were real defects in
  the new mapper. `NormalizeFormulaForHash` is byte-identical to the function it
  replaced, including its paren-stripping quirk (`SUM(a)` → `suma`) — fixing
  that would have changed already-deployed hashes, so it is preserved and
  documented instead.

  Go will not convert between distinct named struct types, so the
  querybuilder→VM mappers are written field by field. That is intentional: a
  field added to one side without the other becomes a compile error rather than
  a silently dropped field, which is exactly how the named-operand regression
  would otherwise have shipped.

  The port also had a side effect that had nothing to do with metric semantics.
  `generate-types`, `generate-schema` and `generate-monaco` enumerate
  `internal/rules/vm`, so any exported struct there lands in the published ASL
  schema — and, carrying a discriminator-shaped field, as an insertable Monaco
  node. The four metric types were in `internal/querybuilder`, never enumerated,
  so `main` has no `Metric*` entry in any generated artifact; the port alone
  added four, including `MetricExpression`, whose `kind` field looks exactly
  like the discriminator that earns a struct a node kind. The WASM evaluator has
  no node kind for any of them, so the port would have let an author insert a
  node the browser could not evaluate — the precise failure
  `cmd/check-drift`'s own header says that pipeline exists to prevent.

  Resolution: an `// asl:ignore` doc-comment marker, honored by all three
  generators (`generate-schema` and `generate-types` read it from the AST,
  `generate-monaco` recovers it from the syntax trees it already loads, since
  `go/types` discards comments). The four types carry it, and the generated
  artifacts and their goldens are byte-identical to `main` again. The marker
  keeps a type's new home from silently changing the published browser
  contract; it is the escape hatch C2/C3 lineage and calc-term types will need.
  Removing one marker reintroduces the type into all three artifacts and fails
  both golden tests, so the guard is enforced rather than decorative.

  Note for the next person: `frontend/public/asl.monaco.json` is a
  **manually-synced** copy of the backend artifact, and no CI step guards it
  (only `rule_engine.wasm` has a verify step). Nothing in C1 needed to sync it,
  because the contract no longer changes — but a future change to the Monaco
  surface will desync it silently unless the copy is updated in the same commit.

- **C2 (9.2) gates the metric path for PII before a term becomes a column.**
  The BO path has always cleared a term for reading before handing back a
  physical column — `boresolver`'s `resolveCol` closure refuses a calc term
  whose column carries a sensitivity tag above the caller's masking tier
  (`DetermineMaskingTier` with the request's role and clearance). The metric
  compiler had no equivalent. Its aggregation case took the authored term
  straight to a column name:

      colRef := fmt.Sprintf("t0.%s", sanitizeIdentifier(termID))

  so `MetricExpression{TermNodeID: "customer_ssn"}` compiled to
  `SUM(t0.customer_ssn)` with no error. Measured and pinned by
  `TestUngatedMetricCompilerIsUnsafe`. `MetricCompiler` held only a `dialect`,
  so it could not have run the check: it had neither the `BODefinition` carrying
  `SensitivityTag` nor the request's role and clearance.

  Three decisions, each with a reason:

  - **The gate sits in the compiler, but the requirement sits in the DDL
    generator.** `MetricTermGate` is consulted for every term a metric reads, and
    `NewSensitivityTermGate` is the standard implementation — it reuses
    `resolveTermToField` and the same `boresolver.DetermineMaskingTier` the BO
    path uses, so a metric and a calc term over the same column are permitted or
    refused identically. `GenerateCubeMaterializationDDL` **refuses to run
    ungated**. That asymmetry is deliberate: compilation is also used by the 8.3
    golden corpus and the equivalence suite, which have no BO and no request
    context, and making the gate mandatory there would mean changing the corpus
    — the artifact that has caught a real regression on each of its last three
    uses — to accommodate a control those tests are not about. The DDL path is
    where a cleared term becomes a *stored* artifact, and only there is the
    asymmetry worth paying.
  - **A refused term must not reach the emitted SQL at all.** The gate runs
    before `sanitizeIdentifier`, not after, so a refusal cannot leave a partial
    artifact behind.
  - **`ErrMetricTermNotPermitted` is distinct from `ErrInvalidMetricFormula`.**
    The formula is well-formed; it is the data it reaches that is not permitted.
    The two have different remediation, so they are not the same error.

  **The gate must recurse into calc terms — and the first version did not.**
  The BO predicate fires *inside* a calc term's expression, on the sensitivity tag
  of the column that expression references. A metric-path gate that inspected
  only the term the metric names would therefore miss the exact case the BO rule
  exists for: a calc term carries no `PhysicalColumn` and usually no
  `SensitivityTag` of its own, so it passes, and the PII is read one level down.

  That was a real hole in the first implementation of this gate, found before
  merge rather than after: a metric over calc term `net_per_order`, whose
  expression reads `customers.ssn`, compiled to `SUM(t0.net_per_order)` with no
  error. `NewSensitivityTermGate` now takes the calc-term expressions (the same
  shape as `boresolver.GenerationContext.CalcTermConfigs`) and walks them, with
  the same two structural guards the BO resolver's `resolveCol` closure carries:
  a cycle guard and `maxMetricCalcTermDepth`, matched to the BO path's value of
  `1` so both refuse the same chains. A calc term with **no** preloaded
  expression is refused rather than permitted — an uninspectable chain is not a
  clean chain, and permitting it would make "clear the term" mean "clear the term
  if someone remembered to hand us its expression". The walk
  (`collectFieldRefs`) panics on an unhandled AST node type rather than skipping
  it, so a new node cannot become a silently unchecked branch.

  **Equivalence is measured against the reference, not restated.** The BO path's
  behaviour is already pinned by `boresolver`'s own
  `TestCalcTerm_MaskingBlocked`, which drives the real generator and asserts the
  refusal names `REDACT_FULL`. This gate does not re-implement that test: a copy
  would only assert a reading of the predicate back to itself, and would pass
  equally happily if the reading were wrong. The metric-side equivalence cases
  are what pin the two paths together — and they must be built from a **tagged**
  term at passthrough tier, not an untagged one. An untagged field returns before
  the tier check is ever consulted, so an equivalence case built from one passes
  even against a gate that treats "tagged" as "forbidden". Both mutations were
  run to confirm the tests bite: disabling the calc-term recursion fails four
  tests, and ignoring the tier fails the passthrough equivalence case.

  **Reachability, stated plainly:** the vulnerable path has **no production
  caller today**. `CompileMetric` is called only from
  `CubeDDLGenerator.GenerateCubeMaterializationDDL` (`cube_ddl.go:138`), and
  `CubeDDLGenerator` is referenced only by its own test. So this is not an
  active exposure — and that is the argument for landing the gate *now*, before
  the cube stream wires DDL generation into a deploy path. The gate should be in
  place before the first materialized view can be built, not after.

  **C0 freeze lifted deliberately, still enforced.** Adding the three funcs
  broke `TestMetricCompilerDeclaredSurfaceIsFrozen`, whose own message says "C2
  adds the vm resolver, and that is expected to break this pin — lift the freeze
  deliberately when it does". The pin was **widened by three named entries** with
  the rationale recorded in the guard, not deleted: deleting it would have turned
  the guard off, whereas widening it keeps the next added func failing the test.
  None of the three adds expression capability — the freeze protects against a
  second expression system, and these decide only whether a term may be read.

  Still outstanding for C2: routing metric term resolution through the `vm`
  resolver proper (cycle detection and the calc-term depth cap, which the
  metric compiler's own cycle detection does not cover), and calc-term lineage
  edges plus reconciler backfill.

- **The metric-to-term `USES_TERM` lineage edge was never written. C2 fixes it, and
  the unit test was part of the bug.** `SyncMetricToCatalogGraph` guarded the
  insert with `uuid.Parse(m.Expression.TermNodeID)`, so the edge ran only when a
  metric named its term by UUID. Real metrics name terms by semantic key — the
  8.3 corpus carries `revenue`, `cost`, `units_sold`, `returns`,
  `calc_term_net_interest_income` — and 0 of 5 parse. Every metric's lineage to
  its underlying term was absent, and the reconciler (`metric_reconciler.go:51`)
  called the function and appeared to succeed.

  Two pieces of evidence, not one. The corpus gave the data shape. The unit test
  gave the confirmation: `TestMetricCatalogSync_TransactionalEmission` set
  `TermNodeID` to a **UUID**, because that was the only shape the implementation
  accepted. A test written around the implementation's constraint rather than
  production's data passes while production takes the other branch — the same
  family as a vacuous fixture, one level up.

  Resolution now matches `DERIVED_FROM`, twelve lines below, which has always
  resolved its target by `node_key`. Both steps resolve a name the same way, so
  a metric whose base metrics are linked is no longer one whose term is not.

  The three `_, _ = tx.ExecContext(...)` in that function are gone, and the two
  failure classes are distinguished rather than merged: a **database error is
  returned** and rolls the transaction back, because a metric node in the graph
  with no lineage reads as "it never had any"; a **missing target is not fatal**
  (a metric can legitimately be imported before its term exists) but is
  **recorded and logged**, so an absent edge is always explicable. An absent edge
  with no log line is a bug, not a state. This is the `G104` argument from the
  security-scan discussion made concrete in the product rather than in the
  tooling.

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
- **Seal-chain amendments (not adopted).** Two ideas were considered for the audit chain and
  parked: (1) a per-partition boundary seal in `ivy-control`, and (2) a key version in each seal.
  (1) exists to prove continuity after a Postgres partition is detached, but `alpha` audit is
  never detached (ADR-029), so it is only needed once a *tenant database* event table carries a
  seal chain and ADR-035's detach-verification job is built. (2) does not apply: the shipped
  chain is an unkeyed hash, so there is no key to version. Decide (1) before that job is written.
- **The `Backend Datasource Rename CI` workflow is deleted — it was pure duplicate
  compute, not a second opinion.** It ran `go test -short ./backend/... -v`, while
  `Build Backend` in `ci-cd.yml` runs `go test -short -v -race
  -coverprofile=coverage.out ./...` from `backend/`. Those are the same packages,
  and the same `-short` mode; the only differences were that `Build Backend` adds
  the race detector and coverage. Its two other steps were also duplicates:
  `go vet ./backend/...` restates `go vet ./...`, and
  `go test ./backend/internal/region_test -v` looks unique because it omits
  `-short` — but `internal/region_test` is in `./...` and contains **zero**
  `testing.Short()` guards, so `-short` excludes nothing there and all three of
  its tests already run in `Build Backend`. Verified rather than inferred:
  `go list ./...` contains the package, and `go test -short -list '.*'` on it
  shows all three tests.

  It was not even a faithful duplicate. That workflow pinned
  `go-version: '1.25'`, a floating minor, while `Build Backend` pins
  `go-version-file: backend/go.mod` (1.25.5) — so the second run could execute a
  different toolchain patch than the one whose result it was supposedly
  corroborating.

  The workflow's *name* promised a datasource-rename tripwire it never
  implemented: it ran the whole backend suite, not the rename paths. No test in it
  protected a contract the main suite does not already run, so per the decision
  rule (narrow to the unique paths if any exist, otherwise consume the existing
  result) the correct action was to remove it rather than narrow it. Deleting it
  cannot strand a required check: `main` has `enforce_admins: true` but no
  required status checks.

  **Race detection is deliberately retained.** `-race` is why `Run Tests` is the
  dominant cost on the critical path (7.6m of a 13.4m job), and it is the only
  thing that catches the shared-global-cache class of bug — the guardrails cache
  race this engagement already fixed being a live example. The time is recovered
  by removing the duplicate, not by weakening the suite.
- **Decision: security scanning is NOT tiered. One full scan on every PR.**
  Considered and declined: a three-tier design (PR diff-scan, nightly full scan
  with baseline, release gate on unexplained HIGH/CRITICAL). The tiering was
  designed on the premise that *"PR merges are Gosec-bound"*, and that premise
  turned out to be measured **false** once `security-scan` stopped declaring
  `needs: [build-backend, build-frontend]`.

  That job checks the repository out fresh and scans the source tree itself —
  Trivy filesystem, Snyk, Gosec over `./backend/...` — so it consumes no
  artifact from either build job. The `needs:` was gratuitous and serialized a
  full 8–11 minute scan behind 10–13 minutes of builds, making it additive to
  every PR. With it removed the scan overlaps the builds and costs approximately
  zero wall-clock, so a nightly tier would buy runner compute in exchange for
  per-PR detection latency, a baseline file to maintain forever, and permanent
  baseline drift. **Full-scan-every-PR is simultaneously the more secure and the
  cheaper option**, which is rare enough to be worth stating plainly rather than
  discovering again.

  The measurement correction behind this is recorded because it is the kind of
  error that is invisible until measured: the scan looked like the critical path
  on one run in three when judged by longest-single-job, but the pipeline's
  critical path is the longest **dependency chain**. Longest-job was off by
  9.8–22.7m against actual run wall-clock; the chain figure tracked reality to
  0.8–12.2m.

  **If runner compute ever becomes a real constraint — measured, not assumed —
  the tiered design is preserved here for that contingency**, together with the
  two defects that must be fixed before it is built: a baseline keyed on
  `rule_id` alone suppresses every *future* finding of that rule repo-wide (it
  needs file+line+rule, re-baselined only by reviewed commit), and a PR gate
  that applies no baseline fails on pre-existing debt in any touched file. Any
  wholesale `G104` exclusion is also out: unhandled errors are the class behind
  the three discarded `tx.ExecContext` results in `SyncMetricToCatalogGraph`, so
  silencing the rule would blind the scanner to a defect class this codebase
  keeps producing. Per-site `#nosec` with a reason is the reviewed-exclusion
  pattern.

## Metric catalog lineage: the writer targets a schema that does not exist

Recorded 2026-10-03, found while starting C2's calc-term lineage work. This
qualifies the #359 entry above, which treated `SyncMetricToCatalogGraph` as a
live code path needing a smaller fix.

**The finding.** `SyncMetricToCatalogGraph` has no production caller, and its
SQL does not match the authoritative schema. Two independent defects, either
of which alone would stop it executing.

*Reachability.* `SyncMetricToCatalogGraph` is called from exactly one place,
`metric_reconciler.go:51`. `ReconcileAll` and `NewMetricCatalogReconciler` have
**no production callers at all** — their only callers are
`starrocks_mv_and_ridealongs_test.go:90` and `:98`. Verified by searching all
Go source plus `cmd/`, `k8s/`, `deploy/`, `scripts/`, `.github/` and `docs/`,
not just the one package: this is *looked-and-found-nothing*, not
*couldn't-look*. The #359 fix is a correct fix to a real bug, but it repairs a
path that never runs. That was a miss — the PII gate work established that
reachability is checked before building, and the check was not applied to the
lineage branch before merging.

*Schema.* `backend/db/snapshots/schema-snapshot.sql` (the authoritative dump,
regenerated 2026-10-02) defines:

- `catalog_node` (line 33593): `id`, `node_type_id uuid NOT NULL`, `node_name`,
  `qualified_path NOT NULL`, `properties`, `config`, `tenant_id`, … There is
  **no `node_id` and no `node_key` column.**
- `catalog_edge` (line 33391): `id`, `source_node_id uuid NOT NULL`,
  `target_node_id uuid NOT NULL`, `edge_type_id uuid NOT NULL`, `tenant_id`,
  `properties`, … **No `edge_type` text column and no `source_id`/`target_id`**,
  and the table is `PARTITION BY RANGE (created_at)` with only `2026q1` and
  `2026q2` partitions present.

Against that, `metric_definition.go:573` inserts
`(node_id, tenant_id, node_type, node_key, node_name, qualified_path,
properties)` — two non-existent columns plus an omitted `node_type_id NOT
NULL`. `metric_definition.go:650` inserts into `catalog_edge` using a text
`edge_type` that does not exist, omitting both `id` and `edge_type_id`, neither
of which has a default. The `SELECT node_id … WHERE node_key = $2` lookups
fail the same way.

No migration in any of the repository's 15 migration roots adds `node_id` or
`node_key` to `catalog_node`; the eight `node_key` hits under
`backend/db/migrations/` all belong to the unrelated `navigation_menu_nodes`
table. The one clause that *is* valid is `ON CONFLICT (tenant_id,
qualified_path)`, backed by `catalog_node_tenant_path_uniq` (line 71916).

**Blast radius beyond C2.** The same missing columns appear in other non-test
production files: `semanticmatch/resolver.go:75` (also `ON CONFLICT (tenant_id,
node_key)` and `RETURNING node_id`), `catalog/sti_column_scanner.go:56` and
`:70`, `catalog/subtype_bo_builder.go:36`, and `bo/layout_service.go:108–115`
(which joins on `st.node_id`, and on `e_bt.from_node_id` / `e_bt.edge_type IN
(…)` — `from_node_id` is not a column of the real `catalog_edge` either).
Same class as #346, the deleted unwired StarRocks MV path: a generation of code
left behind when the glossary model moved `catalog_node` to `node_type_id` and
`catalog_edge` to `edge_type_id`.

**The reader is sound, which is why this is a missing link and not dead code.**
`lineage/sql_repo.go:113` walks the graph with a recursive CTE that is
edge-type-agnostic and traverses `ce.source_node_id` / `ce.target_node_id` —
columns that *do* exist. Its consumers are production
(`catalog-worker/main.go:122`, `worker/main.go:422`, `cmd/sync-graph`). The
graph walk runs today and cannot traverse metric → term → BO, because nothing
ever emits those edges.

**Decision: wire it, but not yet.** The ruling is to give `ReconcileAll` a
production entry point with the complete writer — calc-term lineage included —
landing in the same PR, so the writer is whole before it can execute against
real data. The port is not a mechanical rename, and it is blocked on a fact the
repository cannot supply:

1. Whether the snapshot is faithful to the live alpha database, or alpha
   carries columns no migration creates. Everything above is verified *against
   the repository*, not against a running system.
2. Whether `METRIC_OF`, `USES_TERM` and `DERIVED_FROM` exist in
   `catalog_edge_types`. The snapshot contains **no row data**, so their
   absence from it proves nothing — a genuine *couldn't-look*, and load-bearing,
   because a correct port needs their `edge_type_id` UUIDs.
3. Whether `catalog_edge` partitions beyond `2026q2` exist operationally. No
   migration creates them and there is no `pg_partman` configuration for this
   table, so on the repository's evidence an insert dated 2026-10-03 (2026q4)
   has no partition to land in.

Item 2 decides whether the port is a rename or a vocabulary-creation task.
Verification queries are in `docs/FAILURES_LEDGER.md`; they need alpha access,
which is already blocking other work.

**Ruling 2026-10-03: hold.** No port work until the schema answers land —
building a corrected writer against answers that do not exist yet would repeat
the original error one level up. The alpha session runs Q5 first, because it
can dissolve the finding entirely; then Q1/Q2 together to settle the mismatch
against live reality rather than the dump; then Q3/Q4, which are input to the
fix rather than to the diagnosis. The same session carries the four-file sweep
(`semanticmatch/resolver.go`, `catalog/sti_column_scanner.go`,
`catalog/subtype_bo_builder.go`, `bo/layout_service.go` — same root cause, so
one investigation) and the cube stream's staging DDL validation, which shares
the access and faces the same risk class. Dispositions for the sweep are a 2×2
of schema-match against reachability; two of the four cells end in deletion.
The C2 calc-term and reconciler-backfill work stays parked, and may be
reshaped or invalidated by what the answers say.

**To fix while wiring.** `metric_reconciler.go:22` claims the reconciler is
"Safe for concurrent runs across multiple instance replicas using advisory
locking / upsert semantics." There is no advisory lock and no upsert — the edge
write is `INSERT … SELECT … WHERE NOT EXISTS`, which races across replicas.
`TestMetricCatalogReconciler_IdempotentRun` passes only because a
single-threaded run cannot expose the race. The comment asserts a property the
code lacks, and the test's name claims coverage it does not provide.
