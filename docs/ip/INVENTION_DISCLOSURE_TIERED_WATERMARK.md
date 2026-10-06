# Invention Disclosure Draft: Watermark-Governed Tiered Query and Lifecycle

**Status:** DRAFT for patent counsel. Not legal advice. Prepared 2026-10-06 from code review of branch `feat/post-trade-100-rules-phase1-2`.
**Purpose:** identify mechanisms that might be patentable beyond the generic concept "one semantic layer over hot/warm/cold stores with a watermark", which is too generic and has heavy prior art.

## 1. Prior art counsel will cite first (we should address it, not ignore it)

- Apache Pinot hybrid tables: a "time boundary" splits a query between realtime and offline segments. Closest to our query-side seam.
- Apache Druid realtime/historical nodes and tiered historicals; Lambda/Kappa architectures.
- Iceberg/Delta/Hudi snapshot semantics; Flink/Beam streaming watermarks.
- Trino/Presto/Dremio/Starburst/Denodo federation and virtualization.
- Bitemporal modelling (SQL:2011, XTDB, Datomic) and `ROW_NUMBER` dedup of overlapping sources: widely known individually.

The concept-level claim fails. Any claim must rest on the specific combinations below.

## 2. Candidate mechanisms (ranked by apparent strength)

### C1. One certified watermark governing both query routing and storage lifecycle (strongest, partly unimplemented)

**What the code does today (storage side):**
- `compliance.compliance_watermark_checkpoint` holds a per-tenant, per-tier certified LSN (`WARM`, `COLD`).
- WARM loader advances the WARM watermark only after StarRocks stream-load success (`internal/compliance/jobs/warm_tier_loader.go:270`).
- Cold archiver extracts only `(coldLWM, warmLWM]`, verifies manifest contiguity, seals WORM Parquet with dual Merkle roots, then advances the COLD watermark (`cold_archival_worker.go:105-325`).
- Hot-tier pruner drops a Postgres partition only if its max LSN is at or below the global certified WARM watermark (MIN across tenants) (`hot_tier_pruner.go:41-125`). Warm pruner is gated the same way on the COLD watermark (`warm_tier_pruner.go`).

**Technical problem solved:** data is never deleted from a tier until the next tier has certified it, so a query routed by the seam never finds a gap, even during tier transitions or failures.

**Gap to close before filing:** query routing uses a separate `WatermarkResolver` (`boresolver/bo_sql_generator.go:219`), a date-valued interface. No production implementation of it was found in the repo (only the interface and its call at line 282). The date-based query seam and the LSN-based certified checkpoints are therefore **not yet unified**. The invention is strongest if the router derives its seam *from the certified checkpoint* (LSN to effective-date mapping, per tenant, per tier). Counsel should treat this as the central claim; engineering should decide whether to implement it before filing.

### C2. Bitemporal split-and-stitch across the seam with late-mutation recovery

`BitemporalRangeCompiler.CompileRangeQuery` (`boresolver/bitemporal_range_compiler.go:61`):
- Decomposes `[Te_start, Te_end]` at watermark `Wt` into `R_cold = [Te_start, min(Te_end, Wt))` and `R_hot = [max(Te_start, Wt), Te_end]`, with three strategies: pure cold, pure hot, split-and-stitch.
- In the straddling case, the hot branch also pulls rows whose effective date is *before* the watermark but whose `system_valid_from >= Wt`. These are late corrections to already-archived history. The union is deduplicated by `ROW_NUMBER() OVER (PARTITION BY business_key, effective_date ORDER BY system_valid_from DESC, source_precedence ASC)`.
- **Technical problem solved:** correct as-of results when a bitemporal correction lands in the hot tier for a period already moved to cold, without double-counting or losing the correction.

Weakness: each element is individually known; novelty is in the combination and the late-mutation recovery predicate. Note the pure-cold and pure-hot branches do not apply the late-mutation predicate, so confirm intended behaviour before describing it as an invariant.

### C3. Per-tenant tiered archive with verifiable contiguity

Contiguous LSN slices, a manifest chain checked by `cold.VerifyContiguity`, a rule-registry companion slice (rule version, resolved AST, bytecode hash) archived alongside each evaluation slice, and dual Merkle roots. **Problem solved:** an auditor can prove, years later, that cold history is complete (no LSN gaps) and was evaluated under exactly the rule versions recorded. Related: the tenant audit-copy design in `ARCHITECTURAL_DECISIONS.md` ("resume point is the destination, not a counter"; `prev_hash` continuity check). Lower patent strength (hash chains/Merkle logs are well known), better suited to trade-secret or defensive publication.

### C4. Dialect-agnostic watermark UNION (calcengine) and rule-to-SQL compilation

`calcengine/dialect.go` keeps the hot/cold UNION backend-agnostic via a placeholder/quoting dialect. The single rule engine (`vm.CompileConditionSQL`) producing equivalent filters across tiers could support a claim if there is a demonstrable equivalence property between engines. Needs evidence (tests proving identical results across Postgres, StarRocks, DataFusion/Iceberg) before it is a credible claim.

## 3. What NOT to claim

- `internal/semanticast/compiler.go` is keyword matching (`strings.Contains`) with hard-coded entities. Not a technical contribution; do not present it as the "semantic layer".
- `ResolveEffectiveDialect` (`bo_sql_generator.go:274`): "route to cold engine if as-of date is before watermark" is essentially Pinot's time boundary. The CBO telemetry override is generic.
- MV staleness vs source watermark (`querybuilder/mv_routing_policy.go`): the router currently derives staleness from lifecycle status (ADR-020), not a watermark. Nothing to claim there.

## 4. Questions for counsel

1. Is C1 patent-eligible post-Alice as "certified-checkpoint-gated tier retention with query seam derived from the same checkpoint", framed as a concrete improvement to database consistency?
2. Does Pinot's time boundary plus Lambda-style compaction anticipate C1 or C2 in combination? Request a prior-art search.
3. Provisional filing now (12-month priority) versus waiting to implement the C1 unification.
4. Disclosure audit: is this repository, any demo, any customer or investor material public? US grace period is 12 months from first public disclosure; most other jurisdictions have none. First commit dates for the relevant files: `cold_archival_worker.go` introduced in `43fb01980`; confirm dates with `git log`.
5. Trade secret versus patent for C3 and the rule registry mechanics.

## 5. Evidence to assemble

- Tests demonstrating no gap or duplicate across the seam during tier transition (currently the compiler is unit-tested for SQL shape; an end-to-end seam-consistency test would strengthen the claim).
- Invariant write-up: for every tenant and time `t`, `t` is resolvable from exactly one certified tier or from the overlap with defined precedence.
- Inventor list and contribution dates (commit history).
