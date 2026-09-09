# Unified Rule Engine — Handoff

Written 2026-09-09 at the end of a two-session arc that took the rule/calc
engine from "three-plus disconnected AST formats, one of them silently
broken in the browser" through a single unified engine with a *proven*
end-to-end chain: author in the AST → persist as a catalog node →
evaluate via the unified engine → agrees with ground truth on live data.
This document is the state to hand into a fresh session — what's real,
what's verified, what's still open, and exactly what the next task is.

## Where things stand, in one paragraph

`internal/rules/vm` (`RuleNode`/`RuleGroup`/`RuleCondition`/`Expression`/
`BinaryExpr`/`FuncCall`) is now the single AST behind validation rules,
MDM/rulefabric rules, calc-term SQL pushdown, and the browser WASM live
preview — one evaluator (`AdvancedEvaluator`), compiled to
`rule_engine.wasm` for the browser and run natively server-side, with a
real round-trip-tested `MarshalJSON`/`UnmarshalJSON` pair. The editor
(`AdvancedRuleBuilderPage`) is routed and click-through verified with a
real login. Validation rules are `catalog_node` rows (`config.rule_ast`),
the same convention calc terms and pre-aggregations already use, with the
`GOVERNED_BY_RULE` catalog edge (seeded, unused for years) now doing real
work. The old `catalog_validation_rules` corpus — 233 rules, confirmed
never evaluated by anything, targeting a schema that no longer exists —
is retired (archived, not deleted) and its intake closed. The whole chain
is proven with a real oracle: a rule authored against the AST, persisted
through the real service, evaluated by the real engine, agrees with a
live Postgres `CHECK` constraint on every real row *and* correctly
rejects synthetic violations. What's left is two pieces of well-scoped,
unknown-free mechanical work — editor save-wiring and a context provider
— not discovery.

## What's fully working and verified (don't redo this)

### 1. One AST, one evaluator, two build targets
- `backend/internal/rules/vm/ast.go`: `RuleNode`/`RuleGroup`/
  `RuleCondition`/`Expression`/`BinaryExpr`/`FieldRef`/`Literal`/
  `FuncCall`, with matching `MarshalJSON` on every type — **a real,
  previously-undiscovered bug**: none of these had `MarshalJSON` before
  this arc, so `json.Marshal` produced capitalized, nested JSON
  (`{"Type":"expression","Expression":{"Root":{...}}}`) that
  `UnmarshalJSON`'s flat, lowercase-keyed parser could not read back.
  Marshal always "succeeded" — the round trip silently lost data. Found
  by writing a round-trip test, not by inspecting output. Pinned in
  `ast_roundtrip_test.go` (Group/Condition/Expression/nested
  FuncCall-inside-BinaryExpr, plus the standalone-`Expression` shape
  calc terms use).
- `backend/internal/rules/vm/advanced_evaluator.go`: the tree-walking
  evaluator, moved here from `internal/rules` (it only ever imported `vm`
  + stdlib) so the browser build could use it directly. `FuncCall`
  support added: native `SUM`/`AVG`/`MIN`/`MAX`/`NPV` plus 9 field-format
  predicates (`NOT_EMPTY`, `IS_INTEGER`, `IS_NUMBER`, `IS_BOOLEAN`,
  `IS_UUID`, `IS_DATE`, `IS_DATETIME`, `IS_JSON`, `MAX_LENGTH`) — added
  for the `catalog_validation_rules` translator (below), tested in
  `predicates_test.go`.
- `backend/rule-engine/cmd/wasm/main.go`: rebuilds `rule_engine.wasm`
  directly from `internal/rules/vm` (was the old `rule-engine/runtime`
  package's own separate, smaller AST — `Group.Children` vs. the real
  `RuleGroup.Conditions` — which was silently mis-evaluating any
  Monaco-authored `Group` rule: empty children → AND vacuously true, OR
  vacuously false). Deleted the old package outright.
- **Proof artifact, permanent**:
  `backend/rule-engine/scripts/verify_wasm.js` — a functional (not
  byte-diff, Go's `GOOS=js/wasm` builds aren't reproducible) smoke test
  against the exact file the frontend serves
  (`frontend/public/rule_engine.wasm`), wired into CI
  (`.github/workflows/ci-cd.yml`, `build-frontend` job). Proven to fail
  on the pre-fix binary and pass on the fixed one before being trusted.
- **Proof artifact, permanent**: `backend/rule-engine/cmd/check-drift`
  regenerates schema/types/monaco/version *and* the wasm build, wired
  into CI (`build-backend` job).
- Verified live in a real browser session (after manual login) at
  `core/validation-rules/editor`: `total > 100` against
  `{"total": 150}` → **Result: PASS**, through the actual routed page,
  actual evaluator, actual click.

### 2. No duplicate writers left in the codegen pipeline
`asl.schema.json`, `asl.monaco.json`, and `version.json` each used to
have **two** independent writers (one in `cmd/generate-types`, one in the
dedicated `cmd/generate-schema`/`cmd/generate-monaco`/`cmd/generate-version`)
producing genuinely different content — whichever ran last in
`go generate ./...` won silently, and running them concurrently (e.g.
`go test ./rule-engine/...`, one process per package) raced on the shared
file. `TestGenerateSchemaDeterministic` was actually flaking from this
before the fix. Removed the duplicates from `cmd/generate-types` entirely
— it only emits `asl.d.ts` now. Pinned with
`cmd/generate-types/writer_dedup_test.go` (source-scans `main.go` via
`go/parser` for the three forbidden method names — **not** `reflect`:
verified empirically that `reflect.Type`'s method introspection only sees
*exported* methods, so it could never observe any of these,
unexported by convention — proved both directions before trusting it).
Re-ran the full suite three times in a row after the fix to confirm the
race is gone, not just quieter.

Also pointed `cmd/generate-monaco` at `internal/rules/vm` (previously
`internal/services` only) — `RuleNode`/`TraceStep` now appear in
`asl.monaco.json`'s `nodeKinds`.

### 3. Generators are invocation-cwd-independent
Every `cmd/generate-*` command resolves its own paths via
`runtime.Caller(0)` now, not the process's working directory — fixes a
real bug where `go generate ./...` (cwd = `rule-engine/`, the directive's
own directory) and a direct `cd cmd/X && go run .` disagreed about where
things lived, which is exactly how `RuleNode`/`RuleGroup`/`Expression`/
`FuncCall` went missing from the generated schema for a long time without
anyone noticing, and how several **stray committed binaries**
(`backend/rule-engine/cmd/generated/`, `cmd/generate-version/backend/...`,
`backend/rule-engine/backend/rule-engine/generated/...`) ended up in the
repo. `.gitignore` hardened for this pattern — **twice**, because the
first pass missed `backend/rule-engine/cmd/*/` specifically (a different
nested `cmd/` tree from top-level `backend/cmd/*/`), and new stray
binaries landed there mid-session even after the "fix." Verified against
four false-positive cases (`.go` files, `Dockerfile`, `testdata/`
contents, the original `backend/cmd/*/` case) before trusting it the
second time.

`generate-schema`/`generate-types` also now **refuse to generate** (loud
error, not a silently incomplete file) when they find a cross-package
type alias whose target was never resolved by also scanning that
package — proved both directions: removing `internal/rules/vm` from the
scanned list reproduces a specific, named error; restoring it succeeds.

### 4. FuncCall in the bytecode compiler: traced, not built
`vmCompiler.compileExprNode` has no case for `*vm.FuncCall` — it falls to
`default:` → `c.fail(...)` → `CompileVM` returns
`{Unsupported: err}` with `Program` left `nil`. Every call site
(`engine.go`, `batch.go`, `orchestrator.go`) already gates on
`Unsupported != nil || len(Program.Insts) == 0` and falls back to
`AdvancedEvaluator`, which *does* support `FuncCall`. So a rule using
`FuncCall` today degrades to the slower tree-walking path with a
**correct** answer, not silent mis-evaluation — pinned in
`vm_compiler_funccall_test.go`. **Decision made explicitly**: bytecode
dispatch for `FuncCall` is demand-driven, not a prerequisite — build it
when a real workload hits the fallback (make the fallback observable
first — see open items).

### 5. Calc-term SQL pushdown (session 1)
- `backend/internal/rules/vm/sql_compiler.go`: `CompileToSQL` walks the
  same AST, resolving `FieldRef` via an injected `ColumnResolver`.
  `SUM`/`AVG`/`MIN`/`MAX` pass through; `NPV` hand-expanded (no native
  StarRocks function). Wired into `GenerateDDL`
  (`pre_aggregation_service.go`) — a calc term compiles for real if its
  `catalog_node.config` carries `rule_ast`; otherwise still an explicit
  `NULL` placeholder with a precise reason, never fabricated SQL.
- `ApplyMaterialization`/`Refresh` now execute real DDL against StarRocks
  (were stubs). Two real bugs found only by actually running it: tenant-UUID
  database names need backtick-quoting; the generated `BUILD IMMEDIATE`
  clause isn't valid StarRocks async-MV syntax (the *previous* handoff's
  "verified output" was verified as generated *text*, never executed).
- Verified live: a `rule_ast`-backed test term compiled to real SQL and
  produced a real computed value in a StarRocks materialized view.

### 6. `catalog_validation_rules` — investigated, retired, replaced
- **Phase 0 finding**: the table is write-only. `internal/api/
  validation_rules_routes.go`'s CRUD *is* mounted (`api.go:1689`), so its
  233 rows are real writes — but `internal/validation.TriggerValidationEngine`,
  the *only* code anywhere that ever reads `condition_json` and evaluates
  it, is constructed only inside two handler files whose router-registration
  functions are never called from `api.go`. Confirmed by direct grep, not
  inference: zero external callers of `Execute` anywhere.
- **Migration report** (`docs/validation_rules_migration_report.json`,
  produced by `backend/cmd/migrate_validation_rules`): 225 flat conditions
  (0 converted — `business_objects` has exactly 5 rows system-wide, none
  matching any of the 233 rules' target entities; the corpus targets a
  Customer/Employee/Product/trade_order schema that doesn't exist in the
  current catalog, for any tenant), 3 code-expressions, 2 node-graph
  payloads, 3 embedded script-DSL payloads — none hand-migrated (single-digit
  counts, per-item judgment call, not tooling-worthy).
- **Retired** (`backend/migrations/20260909_retire_validation_rule_corpus.sql`):
  all 233 rows `is_active = false`, `condition_json` untouched, one audit
  row per rule in `catalog_validation_rules_audit` naming the reason.
  Reversible.
- **Intake closed**: `POST /validation-rules` (`handleCreateValidationRule`)
  now returns `410 Gone`. `GET`/`PATCH`/`DELETE` stay open for forensics.
- **Replaced**: validation rules are now `catalog_node` rows — see next
  section.

### 7. Validation rules as catalog nodes — the proof
Third instance of the calc-term/pre-aggregation storage convention:
`catalog_node` of node type `validation_rule` (already existed, seeded
long before this session, **zero rows ever used it** — same
"infrastructure exists, first real user shows up two sessions later"
pattern as `GOVERNED_BY_RULE` below), AST in `config.rule_ast`, severity/
timing/category in `properties`.
- `backend/internal/models/validation_rule_types.go`,
  `backend/internal/analytics/validation_rule_service.go`,
  `backend/internal/handlers/validation_rule_handler.go` — mirrors
  `preaggregation_handler.go`'s shape exactly. Mounted at
  `/api/validation-rule-nodes` (deliberately not `/validation-rules`,
  which 410s).
- `ValidationRuleService.Evaluate` loads `rule_ast`, runs it through
  `vm.AdvancedEvaluator` — the real unified engine, same one the browser
  wasm build uses.
- **The `GOVERNED_BY_RULE` edge bug**: `business_objects.id` is *not* a
  `catalog_node.id` — verified live, it doesn't exist as one at all. An
  edge built from it inserts with **no error** but can never be joined
  back to anything by any real consumer. The BO's actual catalog node is
  `business_objects.classification_node_id`. Caught only by re-querying
  the edge back with the join a real consumer would use — the insert
  succeeding proved nothing.
- **Proof artifact, permanent**: `backend/cmd/verify_oracle_rule` —
  authors "Filled Quantity Within Bounds"
  (`filled_qty >= 0 AND filled_qty <= quantity`) against the Order BO,
  persists it through the real service, evaluates it via the real engine
  against every live `oms.orders` row (all pass, agreeing with the DB) —
  **and** against two synthetic violation cases (over-fill, negative
  fill — both correctly rejected). Agreement-on-live-data alone doesn't
  prove the rule can detect a violation (a rule that always returns
  `true` would "pass" that too); the negative case is what makes this
  proof rather than demonstration.
- **The oracle pivot**: the constraint this was *supposed* to verify
  against (`exec_price > 0` on Execution) does not exist — grepped every
  migration file, found no trace of it ever being defined. Found
  `oms.orders.chk_orders_filled_qty` instead by querying `pg_constraint`
  directly, and it's a *better* oracle: two conditions, one cross-field,
  exercising both the `Condition` and `BinaryExpr` evaluation paths in a
  single rule. The inverse implication is its own flag: the DDL this
  session was handed and the live database have drifted apart — worth
  keeping in mind for the OMS work generally, not just this rule.

## What's NOT done — the actual next task

### A. Editor save-wiring + real BO catalog data
`frontend/src/pages/AdvancedRuleBuilderPage.tsx` still uses
`MOCK_ENTITIES` (order/customer/line_item — a **third** distinct demo
domain, after Northwind and the OMS catalog) and has no Save action at
all. Two integration points, both real but small:
- Load the BO/field list from the catalog (replaces `MOCK_ENTITIES`) —
  there should already be a BO-listing endpoint elsewhere in the API
  surface to reuse; if not, `business_objects` + `business_object_fields`
  is the same query shape `validation_rule_service.go` already uses for
  BO resolution.
- Save: `POST /api/validation-rule-nodes` with
  `{tenant_id, bo_name, name, severity, timing, category, rule_ast}` —
  the `toRuleNode()` conversion already in `AdvancedRuleBuilderPage.tsx`
  (added this session for the "Backend Preview" tab) produces exactly
  the wire shape the endpoint expects; it just needs to POST instead of
  only calling `evaluateRuleWasm` locally.
- The severity/timing/category fields need a form (currently nothing in
  the editor UI collects them — `ValidationRuleProperties` requires them).

### B. Related-row context provider on the BO write path
Not started. The Tier-1 OMS validations (overfill, over-placement, the
Σ-completeness invariants) need aggregate context — "Σ executions for
this placement" — that a single-row write doesn't carry. This is
real engine wiring: something that, given a BO write, resolves and
attaches related-row aggregates to the `data map[string]interface{}`
passed to `AdvancedEvaluator.Evaluate`. No design work done yet on where
this hooks into the write path or what the aggregate-resolution query
shape looks like.

### C. Everything downstream of A and B (design already settled, not started)
1. OMS Tier-1 BLOCK set, authored natively in the routed editor (spec
   exists from earlier in this arc — overfill, over-placement, the two
   Σ-completeness invariants, limit-price presence, allocation
   completeness).
2. Shadow mode: wire evaluation to the write path in log-only mode,
   nothing blocked, before any enforcement — this is what makes
   activation safe rather than a step-function surprise.
3. Progressive enforcement, per-BO or per-rule, after triaging shadow-mode
   violations (fix the rule / fix the data / retire the rule).
4. Retirement pass, now with a fully-verified-dead inventory:
   `internal/validation.TriggerValidationEngine` (unmounted),
   `internal/services.ValidationRuleEngineImpl` (4 toy rows, dead
   evaluator), `CueEngine` path (0 live rows), `internal/calculation`,
   `bo_context_resolver.go` (`ResolveCalculation` unused), `internal/
   calcengine` (mine its watermark/tier-routing logic *first* — the one
   genuinely reusable piece — before deleting the rest).
5. Capability badges (`pushdownable`/`wasm`/`bytecode-fallback`) in the
   Monaco editor, sourced from the function registry — folds in the
   FuncCall-runs-at-tree-walking-speed perf-cliff warning as the same UI
   feature.
6. Function-name autocomplete/snippets pointed at the real function
   registry (`cmd/generate-monaco` currently derives `nodeKinds`/`enums`
   from real types but `keywords` is still a hardcoded literal list).
7. ~45 legacy calc terms: re-author or translate into `rule_ast`. Raw-SQL
   leaf-node decision still open for the 52 plain-SQL-shaped ones
   (pragmatic escape hatch vs. full translation — not yet decided).
8. IRR/XIRR: need Newton-with-bisection-fallback solvers, golden-tested
   against real Excel output, before they can be native `FuncCall`
   predicates.
9. Bytecode `FuncCall` dispatch — demand-driven per item 4 above.
   Prerequisite: make the `AdvancedEvaluator` fallback observable (a log
   line or metric at the `Unsupported != nil` gate in `engine.go`/
   `batch.go`/`orchestrator.go`) so there's a real signal for when this
   becomes worth building, instead of speculation.

## Key IDs / hosts for continuity

- Tenant ID: `99e99e99-99e9-49e9-89e9-99e99e99e999`
- Order BO: `bo_key = "order"`, `id = b611af7b-8689-407d-807a-eeb315065e7d`,
  `classification_node_id = 6b267260-a5fa-549b-a80a-7ddb1c712da0`
  (**this** is the catalog-node id to use for edges, not `business_objects.id`).
- Test validation rule: `a2154183-2d6e-45cf-b3d9-fd3950b1fbab`
  ("Filled Quantity Within Bounds", Order BO) — the oracle rule, live,
  persisted, re-runnable via `backend/cmd/verify_oracle_rule`.
- `validation_rule` catalog_node_type id: `e39856ec-e9e2-4151-836a-cc93b801fe6c`
  (pre-existing, was never seeded by this arc — just discovered unused).
- `GOVERNED_BY_RULE` catalog_edge_type id: `a9772420-a01e-4e98-a6bf-c270d5b7ae44`.
- Retired corpus: 233 rows in `catalog_validation_rules`,
  `is_active = false`, audit trail in `catalog_validation_rules_audit`
  (`action = 'retired'`).
- Remote infra host (from the previous handoff, still current for
  StarRocks/CDC): `100.84.50.65`, SSH as `eganpj`, repo at
  `/mnt/github/uisce`, compose files in `/home/eganpj`.
- Test pre-aggregation: `5217fdac-ae21-461a-8223-7d65bb23d707`
  (`execution_npv_rollup`).
- Branch: `feat/unified-rule-engine`, cut at `0cdbb44c4` from
  `fix/tenants-search-path-and-bo-write-path`. Carries a complete feature
  stream now — **needs a merge plan**, not just continued commits.
  Session commits, newest last: `3b45524a0`, `18c869f94`, `7896bc63d`,
  `7113c7e90`, `0cdbb44c4` (session 1) → `e6bc30333`, `25755351f`,
  `c2f25fc0a`, `0bfcc318a` (session 2).

## Working notes worth carrying forward

- **Verify the artifact, not the inference.** Grep hits that share field
  names aren't provenance — checked field-by-field against actual
  `UnmarshalJSON` behavior, twice, before trusting a schema claim.
- **"Verified as generated text" ≠ "verified as executed."** The prior
  handoff's DDL was checked as *output*, never run — `BUILD IMMEDIATE`
  turned out to be invalid StarRocks syntax, only caught by actually
  executing it.
- **"Committed" ≠ "deployed."** `frontend/public/rule_engine.wasm` is a
  manually-synced copy with zero build-time connection to
  `backend/rule-engine/generated/rule_engine.wasm` — a fix landing in git
  said nothing about whether the browser was serving it. Caught by a hash
  diff, not by assuming.
- **"No error returned" ≠ "correct."** The `GOVERNED_BY_RULE` edge insert
  succeeded while pointing `source_node_id` into a foreign ID space
  (`business_objects.id`, not `catalog_node.id`) — `catalog_edge` has no
  FK enforcing that column actually references `catalog_node`. Only a
  re-query with the join a real consumer would use caught it. **Open
  item**: if a hand-written service call can silently create a dangling
  edge, every future edge-writer can — worth an FK constraint on
  `catalog_edge.source_node_id`/`target_node_id`, or validation at the
  edge-creation service boundary, before more edge types accumulate this
  same risk.
- **Never `git checkout --` a working tree that mixes a real fix with a
  test-only mutation.** Cost two full re-applications of otherwise-correct
  work this session, both times right after having just learned the
  lesson with a temp-file mistake. Stash, or isolate the mutation in a
  scratch file, every time — no exceptions for "it's just a quick revert."
- **Hub-function risk flags need an additive-diff check, not reflexive
  trust or dismissal.** `gitnexus detect_changes` flagged "high risk, 9
  affected processes" for a two-line addition to `SetupRouter` — correct
  to take seriously, also correct to resolve by reading the actual diff
  (purely additive) rather than either blindly trusting the flag or
  waving it off because the function is huge.
- **The demo-era stratum keeps recurring.** Northwind-flavored
  classification/key/grain nodes (session 1), the 233-rule
  Customer/Employee/Product corpus (session 2), and now
  `AdvancedRuleBuilderPage`'s own `MOCK_ENTITIES` — three separate demo
  domains found layered under the real OMS work across two sessions. The
  retirement pass should treat "demo-stratum sweep" as an explicit,
  named category, not something rediscovered one file at a time.
- **New small items, not otherwise tracked**: the RBAC nil-pointer panic
  in `internal/api/middleware/rbac_enforcement_test.go` (pre-existing,
  confirmed via `git stash` — but security-adjacent code panicking in
  its own test deserves a ticket, not a shrug); the local test-run
  failures (`API_TOKEN_ENCRYPTION_KEY`, Ignite) that may or may not also
  fail in CI — worth confirming which, since "green in CI, red locally"
  is either an env gap or a definition-of-passing split-brain; a
  background cleanup task (spawned this session, id `task_5e9af241`) is
  removing genuinely pre-existing stray committed binaries
  (`backend/bp-starter`, `backend/catalog-admin`, etc.) in its own
  worktree — check its state before assuming those are still there.
