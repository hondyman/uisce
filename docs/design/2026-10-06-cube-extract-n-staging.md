# Track C — Extract-N → staging for federated cube materialize

**Status:** C0–C4 done (C4 lab receipt 2026-10-06)  
**Date:** 2026-10-06  
**Depends on:** Track A (#411), Track B (#412 + ALTER syntax #413).  
**Lab anchor:** CUBE-2.5 Position×Account×Security (`b3b29340-a32b-4512-ac28-5e9a7d02fef6`).

## Problem

CUBE-2.2 compiles federation joins as a StarRocks `FROM` over **live JDBC** sources (`CUBE_SOURCE_CATALOG`, lab `pg_alpha`). CUBE-2.5 proved dual-commit that way with `REFRESH ASYNC` + forced `WITH SYNC MODE`.

That path works for the smoke grain but is the wrong long-term spine:

1. **JDBC multi-table SYNC MVs are rejected** on SR 3.3.22 — federated hot path already special-cases ASYNC.
2. **OLTP coupling** — every Deploy/Refresh hits Postgres through the JDBC catalog for N sources.
3. **Partial failure / retry** — a mid-join JDBC blip fails the whole ApplyHot; there is no attempt-scoped snapshot of each source.
4. **Program intent** — prefer datapipeline/workflow materialization over Trino; CUBE-2.2 explicitly deferred “live extract-N-to-staging Temporal activities.”

## Decision

Insert an **Extract-N** phase into `CubeMaterializeWorkflow` **after** `BeginAttempt` and **before** `ApplyHot`, only when the cube has a non-empty federation:

```
ValidateAndPlan → BeginAttempt → ExtractSources (N) → ApplyHot → ApplyCold → DualCommit
                                      ↑ new
```

For each federation source alias:

1. Resolve the qualified JDBC/driver table (reuse `FederationBindingResolver` + `QualifyCubeSourceTable`).
2. `CREATE TABLE … AS SELECT …` (or `INSERT OVERWRITE` into a pre-created table) into an **attempt-scoped StarRocks staging table** in the tenant hot DB:
   - Name: `{target_db}.cube_ext_{grain_hash8}_{attempt8}_{alias}`
   - Columns: only what the join + grain + measures need from that source (projection list from compiled term map); v1 may `SELECT *` if projection is hard — prefer projection.
3. Rewrite `CompiledFederationJoinSQL.FromSQL` so each alias reads the staging table (native OLAP), not `pg_alpha.…`.
4. `ApplyHot` builds the MV over staging (prefer **SYNC** MV when FROM is native-only).
5. On success or compensate: `DROP TABLE IF EXISTS` each staging table (attempt-scoped = safe under single-flight workflow id).

Single-BO cubes (empty federation) **skip** Extract-N entirely — today’s path unchanged.

## Why StarRocks staging (not Iceberg / not crims MDM staging)

| Option | Verdict |
|--------|---------|
| **A. StarRocks native staging in tenant hot DB (Recommended)** | Same engine as ApplyHot; no Lakekeeper token for extract; attempt-scoped drop is cheap; unlocks SYNC MV. |
| B. Iceberg staging under `lakekeeper_iceberg` | Needs Track B token on every extract; colder path; useful later for huge sources / retention, not v1. |
| C. Postgres/`staging.*` MDM tables | Wrong plane (MDM load tracking cols); couples cube materialize to crims RLS. |
| D. Keep JDBC-only forever | Contradicts deferred CUBE-2.2 item and OLTP-coupling pain. |

Honest non-claim: v1 extract is **full replace per attempt**, not incremental CDC into staging.

## Workflow / activity shape

New activity (stable name for registration):

| Activity | Timeout | Retry |
|----------|---------|-------|
| `CubeExtractSources` | long (30m) + heartbeat | max 1 (partial physical) |

Input: `CubeMaterializePlan` (must carry compiled source list + projected columns after C1 plan enrichment).  
Output: `CubeExtractResult{ StagingTables map[alias]qualifiedName, RowCounts map[alias]int64 }`.

Plan fields to add (JSON-safe for Temporal):

- `FederationSources []FederationSourcePlan` (already on compile result — persist on plan)
- `FederationFromSQL` — post-extract rewritten FROM (or ApplyHot rebuilds from staging map)
- `ExtractEnabled bool` — gate snapshot at plan time

Parallelism: Temporal `workflow.Go` fan-out per source **or** one activity that extracts sequentially. **v1 = one activity, sequential extracts** (simpler compensate). C2 may fan-out if N grows.

Compensate:

- Hot failure after extract → drop staging + existing `CubeFailAttempt`
- Cold failure → existing `CubeCompensateHot` + drop staging
- Extract failure mid-N → drop any staging created in that attempt, then FailAttempt

## Env gate

| Variable | Role |
|----------|------|
| `CUBE_FEDERATION_EXTRACT` | `1`/`true` enables Extract-N for federated cubes (default **off** so hermetic tests and JDBC smoke stay stable) |
| `CUBE_SOURCE_CATALOG` | JDBC catalog for extract **source** (unchanged; lab `pg_alpha`) |
| `CUBE_ICEBERG_*` / Track B | Unchanged; cold path only |

When federation is non-empty and gate is **off**, keep today’s JDBC FROM path (CUBE-2.5 behavior). When gate is **on**, Require extract success before ApplyHot (fail closed).

## PR plan

| PR | Scope | Gate |
|----|--------|------|
| **C0** | This design + ticket breakdown | Approval |
| **C1** | Plan enrichment: persist `FederationSources` / term projections on `CubeMaterializePlan`; staging name helper; unit tests | Unit |
| **C2** | `CubeExtractSources` activity: CTAS from JDBC → staging; rewrite FROM; register + workflow insert; workflow test with sqlmock/fakes | Unit + workflow test |
| **C3** | Staging cleanup on success/fail/compensate; fail-closed mid-extract | Unit |
| **C4** | Prefer SYNC MV when FROM is staging-only; lab smoke CUBE-2.5 with `CUBE_FEDERATION_EXTRACT=1` | Lab receipt (row counts match JDBC path) |
| **C5** | (Optional) Docs + Designer/Deploy hint when gate off vs on; datapipeline `cube_materialize` inherits gate via worker env | Doc / thin |

Out of scope for Track C:

- Track D pagestudio delete
- Iceberg-side extract / retention
- Incremental / CDC extract
- Changing semantic federation JSON (still term IDs only)
- Trino

## Risks

| Risk | Mitigation |
|------|------------|
| Staging disk blow-up on large sources | Projection pushdown in C2; attempt-scoped drop; document size class |
| Name collisions across attempts | Include `attemptID` prefix; workflow single-flight already |
| Gate off → teams think extract shipped | C0/C4 checklist; Deploy logs `extract=skipped\|applied` |
| SQL injection via alias/table | Reuse `sanitizeIdentifier` / `quoteStarRocksIdent` only |
| GitNexus stale index | Re-analyze before C1 edits; `impact` on `CubeMaterializeWorkflow` / `ApplyHot` |

## Acceptance (Track C done)

1. Federated Deploy with gate on: Temporal shows `CubeExtractSources` then dual-commit.
2. Staging tables absent after success (dropped).
3. CUBE-2.5 cold `COUNT(*)` matches pre-extract JDBC path (3 for smoke grain).
4. Gate off: CUBE-2.5 JDBC path still green (regression).
5. Single-BO Deploy never calls extract.

### C4 lab receipt (2026-10-06)

Materializer path with `CUBE_FEDERATION_EXTRACT=1` (same activities Temporal registers):

- Extract CTAS needs `PROPERTIES ("replication_num" = "1")` on single-BE SR 3.3.22.
- Staging mid-flight: pos=3, acct=3, sec=29; cold Iceberg `COUNT(*)=3`.
- Staging unknown after `DropStaging` (`C4_RECEIPT_OK`).
- Temporal Deploy path is the same activity set once worker has #414 + env gate.

## Approval ask

Approve **C0** to implement **C1→C3** in-repo next, then pause for explicit go before **C4** lab smoke with `CUBE_FEDERATION_EXTRACT=1` on the shared StarRocks/JDBC stack.
