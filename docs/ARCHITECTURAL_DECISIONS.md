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
- **A second SQL builder bypasses the compiler entirely, and is currently dead.**
  `starrocks_mv_manager.go:62` hardcodes `measureExpr := "SUM(notional)"` and
  only special-cases `Kind == "aggregation"`, so a derived or formula metric
  reaching it would materialize as `SUM(notional)` — silently wrong, and exactly
  the bug class the cube DDL guard exists to prevent. It is dead today:
  `NewStarRocksMaterializationManager` has **zero production callers** (its
  policy helpers `EvaluateABACMVCompatibility` and `EvaluateStaleMVAction` are
  used by `cube_router.go:147,169`, but they emit no SQL). Flagged rather than
  deleted, because retiring the legacy materialization path is a separate
  decision. **If it is ever wired, it must route through `CompileMetric` or call
  `ValidateMetricExpression` first.**

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

### ADR-031: Lakekeeper Is the Single Iceberg Catalog

**Decision.** Lakekeeper is the only Iceberg REST catalog. Nessie and Polaris
are retired. Namespaces and warehouses are provisioned through
`internal/iceberg/lakekeeper_provisioner.go`.

**Context.** The repository carries three catalog stacks (Lakekeeper, Polaris,
Nessie in `docker-compose.starrocks.yml`). Three catalogs means three
authorization models for the same tables.

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

### ADR-035: Tiering Is Hot → Warm → Cold, and a Partition Is Dropped Only After the Lake Copy Verifies

**Decision.** Hot is tenant Postgres (90 days), warm is StarRocks native (13
months), cold is Iceberg. Event-like tables are range-partitioned and flow
through CDC and Kafka to Iceberg. A Postgres partition is detached only after a
verification job proves the Iceberg copy matches (row count and seal chain).
Retention is enforced by dropping partitions (ADR-018); a legal hold suspends
drops.

**Consequence.** This satisfies ADR-009 ("audit log never purged"): the audit
record is permanent in the lake, and Postgres holds only the hot window.

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
