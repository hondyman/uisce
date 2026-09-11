# CEL Retirement Handoff

> **Project.** Retire `github.com/google/cel-go` from the backend codebase.
> **Rescope note.** Originally framed as a mostly-deletion project to retire the policy editor's expression language. Rescoped to: *replace expression evaluation inside three live request paths and delete two dead bridges*; RDL is spun out as a separate project. The original policy-editor through-line (handoff Slices 1–2) **remains in scope**; the rescope *adds* domains, it does not replace the original surface.
>
> **Closing proof.** Under Option 2 (umbrella + RDL spin-out), this project ends with `cel-go` still pinned in `go.mod` by exactly one package — the spun-out RDL project. The dependency-removal proof ("`go mod tidy` drops the dependency") is deferred to the RDL project's completion. The "one expression language" closing claim holds when the AST equivalence gate, Slice 3, and the RDL port all land — not before.

---

## §1 — Static Audit Table

Every `cel-go` import in `backend/`. Rows are grounded to concrete file:line citations. Classification key:

- **live** — reachable from a request path; constructs `cel.NewEnv` / `Compile` / `Eval` / `Program` on data unknown at build time.
- **dead** — imports cel-go but the cel-touching symbols are themselves unreachable; delete candidates.
- **incidental** — cel types in signatures only; still pins the dependency; effort to remove is higher.

| Package | File:Line | Form | Class | Evidence |
|---|---|---|---|---|
| `pkg/policy` | `cel_eval.go:18` | `policy.CELEvaluator` (`*cel.Env` at `:16`) | **live (one caller)** | `internal/feed/rules/engine.go:18` → `EvaluateRule`:49 → `EvalBool`. `internal/genui/visibility.go` is a dead bridge — `NewVisibilityEvaluator` has zero callers; `Config["visibility"]` is never populated. |
| `internal/rulefabric` | `evaluator.go:625` | `*cel.Env` | **live** | `EvaluateCELBoolean`:1090 ← `evaluator.go:719` (CEL dispatch at `:718`) and `bo_policy_handler.go:284` (HTTP `TestCondition` handler). |
| `internal/rulefabric` | `cel_compile.go:36` | tree→CEL normalizer | **live (write-path)** | `NormalizeConditionJSONToCEL` ← `handler.go:506` (`CreateRuleVersion`). `CreateRule` (`:295-345`) writes tree JSON unchanged. |
| `internal/rdl` | `service.go:130` | `*cel.Env` | **live** | `RDLService.celEnv`:130 ← `NewRDLService`:141, wired via `internal/api/rdl_routes.go:11`. Custom functions `isInWashSaleWindow`/`hasRecentPurchase`/`daysSince` at `:150-167`. **Spun out as separate project.** |
| `internal/boresolver` | `expression_builder.go:7` | `*cel.Env`; `ToCELExpression`:227 | **dead** | Zero callers including `_test.go`. Recursive self-calls at `:242` and `:342` only. SQL counterpart `CompileFilterGroup` (`:169`) is reached from `bo_sql_generator.go:764` and `expression_bridge.go:150`. |
| `internal/genui` | `visibility.go:11` | `*policy.CELEvaluator` | **dead** | `NewVisibilityEvaluator` has zero callers. `FilterComponentsByVisibility` and `EvaluateVisibility` defined only in this file. `ComponentDef.Config["visibility"]` is never populated (verified: no producer of this key in `backend/`; only unrelated visibility enums exist). |
| `internal/rules` | `engine.go:66,85` | `*cel.Env` | **live** | `EvaluateCEL`:424, `EvaluateValue`:454, `EvaluateExpr`:482, `EvaluateExprDebug`:519, `EvaluateDurationExpr`:490. Callers: `bp/resolution_activities.go:77,123,151`; `rules/uma_rebalance_rules.go:334`. |

**`cel-go` import ledger** (five importers: `pkg/policy`, `internal/rulefabric`, `internal/rdl`, `internal/boresolver`, `internal/rules`):

| Slice | Deletion | Importer count |
|---|---|---|
| start | — | 5 |
| 1 | `internal/boresolver` — `ToCELExpression` and self-recursion | 5 → 4 |
| 2 | `pkg/policy` — `cel_eval.go`; `internal/feed` CEL strings rewritten | 4 → 3 |
| 3 | `internal/rules` — CEL env and all entry-point methods | 3 → 2 |
| 4 | `internal/rulefabric` — `evaluator.go` and `cel_compile.go` | 2 → 1 |
| RDL project | `internal/rdl/service.go` | 1 → 0 |

---

## §2 — Live-Path Entry-Point Inventory

Per live package, the concrete caller chain for each request-path entry point.

### `pkg/policy` / `internal/feed`
```
feed/rules/engine.go:18    NewRuleEngine → policy.NewCELEvaluator
feed/rules/engine.go:49    EvaluateRule → celEvaluator.EvalBool
```
No other callers of `policy.CELEvaluator` in request paths.

### `internal/rulefabric`
```
evaluator.go:719   Evaluate → EvaluateCELBoolean  (internal dispatch)
bo_policy_handler.go:284   TestCondition → EvaluateCELBoolean  (HTTP handler)
```
### `internal/rulefabric` (write-path)
```
handler.go:506   CreateRuleVersion → NormalizeConditionJSONToCEL
```
### `internal/rdl` (spun out)
```
rdl_routes.go:11  → rdl.NewService(db)
rdl/service.go:289  Evaluate
rdl/service.go:369  EvaluateAll
rdl/service.go:389  CreateRule
rdl/service.go:426  UpdateRule
```
### `internal/rules`
```
bp/resolution_activities.go:77    ResolveApprovalChain → Engine.EvaluateExpr
bp/resolution_activities.go:123   ResolveBranchActivity → Engine.EvaluateExpr
bp/resolution_activities.go:151   EvaluateDurationActivity → Engine.EvaluateDurationExpr
rules/uma_rebalance_rules.go:334  EvaluateRebalancePlan → engine.EvaluateCEL
```

---

## §2.5 — Authoring-Surface Inventory

One row per live CEL-storage site, with its editor.

### `rule_logic.condition_json` (rulefabric)
- **Editor**: `PolicyRuleBuilder.tsx` via `AdvancedRuleBuilderPage` → `POST /api/rulefabric/rules` (CreateRuleVersion).
- **Note**: `CreateRule` (separate handler at `handler.go:295-345`) also writes to this table without going through the normalizer; default shape is `{"type":"group",...}` (tree).
- **Representation**: both `{"type":"group",...}` (tree) and `{"type":"cel",...}` (CEL) may coexist.

### `compliance_rules.expression` (internal/rules)
- **Authoring surface**: Hasura (GraphQL mutations write `compliance_rules`; no Go API handler found that calls `ComplianceRuleRepository.CreateRule`/`UpdateRule` from the `rules` package — the `guardrail_handler.go` uses a separate `GuardrailRule` type with `Conditions json.RawMessage`, not the `ComplianceRule` that has an `Expression` field).
- **Note**: `compliance_rules` with an `expression` column **never existed in any migration** — the only compliance_rules-adjacent table is `financial_compliance_rules` (rule_expression column, 20260731_financial_superpowers.up.sql). The Hasura authoring surface, SQL fallback write path, and read path via `ListRules` all reference a table that was never created. Every invocation errors at query time. Dead-on-arrival — not a content migration, a latent bug fixed incidentally by removing the calling code.

### `bp/model.go` `entryCondition` / `delayExpr`
- **Authoring surface**: `bp/designer.go` serializes `ApprovalChain` JSON containing `entryCondition` (bare string) and `DelayExpr` (bare string) to workflow/step persistence.
- **Frontend**: `BusinessProcessBuilderEnhanced.tsx` uses `conditionLogic?.condition` (a different field) for condition steps. The frontend does not directly set `entryCondition` — it is set through the BP Designer backend flow or through workflow JSON authoring.
- **Representation**: bare CEL text (no envelope).

### `compliance_rules.expression` — note on UMA read path
UMA rebalance reads from `compliance_rules.expression` via `repo.ListRules("")` at `uma_rebalance_rules.go:317`. This is a *reader* — but it errors at query time because the table/column never existed. The "data-live" premise was wrong. The path was dead-on-arrival. Removal of the calling code (Slice 3 scope) fixes the latent bug incidentally.

---

## §3 — Content Migration Inventory

DB introspection complete. All four probes run against `alpha` on 2026-09-11.

### §3.1 `rule_logic`
```sql
SELECT count(*) AS total,
       count(*) FILTER (WHERE condition_json::jsonb->>'type' = 'cel')   AS cel_rows,
       count(*) FILTER (WHERE condition_json::jsonb->>'type' = 'group') AS tree_rows,
       count(*) - (cel_rows + tree_rows) AS residue
FROM rule_logic;
```
**Result**: total=0, cel_rows=0, tree_rows=0, residue=0.

**Interpretation**: The entire rulefabric storage is data-dead. No CEL rows, no tree rows, nothing. The policy editor has zero production footprint — no rule has ever persisted through PolicyRuleBuilder → CreateRule → rule_logic end-to-end. The migration is of an unused surface. The editor swap remains in scope (user-visible convergence); the backend migration is deletion-only.

**Slice 4 implication**: collapses from migration to deletion. `NormalizeConditionJSONToCEL`, `EvaluateCELBoolean`, and the CEL dispatch in `evaluator.go:718` are all code-live but data-dead. Confirm before committing: trace the AST equivalence gate (§4 condition 1). If rulefabric's ConditionGroup and vm.RuleNode are incompatible shapes, the tree path also has no migration target — Slice 4 becomes pure deletion of the CEL branch, with the tree evaluator retained.

### §3.2 `compliance_rules.expression`
**Probe**: `financial_compliance_rules` (the table with `rule_expression` column; `compliance_rules` has no `expression` column — schema mismatch with the Go code's queries).

**Result**: total=0, non_empty=0. `compliance_rules` (the Go code's table): empty. `financial_compliance_rules`: empty.

**Schema mismatch finding**: The Go code at `rules/repository.go:94` queries `SELECT ... expression ... FROM compliance_rules` — but `compliance_rules` has no `expression` column. Every invocation of `SQLRuleRepository.listRulesRecords` errors loudly at query time (missing column). This is not a content migration question — the path has never executed. **UMA rebalance's "live" classification is under reclassification** pending the caller trace below.

### §3.3 `rule_definitions` (RDL — separate project)
**Result**: count=0.

**Interpretation**: RDL is code-live, data-dead. The validation surface (`catalog_validation_rules`, 233 rows) is used; RDL never was. The RDL spin-out's opening question is not "how do we port the DSL" but "does RDL have users at all" — the project may shrink from a port to a deprecation decision. **Opening question for the RDL handoff draft.**

### §3.4 BP config (`entryCondition`, `delayExpr`)
**Probe**: All candidate tables (`bp_steps`, `workflow_edges`, `bp_approval_delegations`, `bp_triggers`, `visibility_rules`, `catalog_validation_rules`, `validation_rules`) returned 0 non-empty CEL strings.

**Result**: Zero CEL content in any BP config table.

**Adjacent finding — `catalog_validation_rules`**: 233 rows exist but in a different format (`{"payload":...,"authored_mode":"designer","schema_version":"1"}`) — legacy validation rule authoring, not CEL. Not CEL-coupled. Verified separately: `trigger_engine.go:391` references `evaluateComplianceRules` for a different evaluation path. This material belongs to the validation-unification / DQ track, not this project. **Record as adjacent finding; do not lose.**

### §3.5 `visibility_rules` (empty table — genui artifact)
**Result**: 0 rows. Second artifact of the never-wired genui feature (dead code + empty table). The empty `visibility_rules` table should be dropped alongside `genui/visibility.go` in Slice 1 — documented per Rule 3. Decision: drop table in Slice 1 migration, or defer with explicit note. Slice 1 execution receipt should state which was chosen.

**UMA caller trace** (complete): `UMARebalanceRulesEngine` is live in `internal/workflows/uma_activities.go:35` — a Temporal activity called from production rebalance workflows. `EvaluateRebalancePlan` (uma_rebalance_rules.go:317) calls `e.repo.ListRules(ctx, "")` which queries `compliance_rules.expression` — **every call errors at query time** (missing column). The path is reachable and fails silently (logged, not surfaced). This is a live latent bug, not a content question: the schema never matched. The migration that removes `compliance_rules`-querying code from the codebase ( Slice 3 scope) also fixes this bug — the bug fix and the migration are the same deletion.

**UMA rebalance reclassification**: `UMARebalanceRulesEngine` moves from "live — under CEL" to "live — latent schema bug, fix on deletion". Not a stop-and-report. Flag for Slice 3 scope note.

### §3.6 `compliance_rules` DDL migrations grep (complete)
Only `20260731_financial_superpowers.up.sql` matches. It creates `public.financial_compliance_rules` with `rule_expression` column — NOT `compliance_rules` with `expression`. **The Go code's `compliance_rules.expression` table never existed in any migration.** Written against a schema that never shipped. Confirms dead-on-arrival classification.

---

## §4 — Closing Claim Gate

Two conditions gate the "one expression language" closing claim. Both must be resolved before the claim can be made.

### Condition 1: `rulefabric.ConditionGroup` ↔ `vm.RuleNode` AST equivalence

Static read completed. Field-by-field:

| field | rulefabric `ConditionGroup`/`Condition` | vm `RuleNode`/`RuleCondition` |
|---|---|---|
| type tag | `Type string` ("group"/"condition") | `Type RuleNodeType` + pointer to `Group`/`Condition`/`Expression` |
| operator | `Operator string` | `Operator string` (matches) |
| conditions | `Conditions []interface{}` (heterogeneous) | `Conditions []RuleNode` (homogeneous) |
| cross-entity | `EntityPath *EntityPath` | absent — `RuleCondition.FieldPath string` instead |
| operator value | `Value interface{}` | `Value any` + `ValueType string` + `SecondValue any` |
| expression AST | absent | `Expression *Expression` (full AST) |

**Conclusion**: Different ASTs. Convergence requires either migrating `rulefabric.ConditionGroup` → `vm.RuleNode` (bounded work), or accepting a second AST (closing claim fails for the rulefabric domain).

### Condition 2: `internal/rules` time/duration arithmetic

`EvaluateDurationExpr` (engine.go:490) parses any expression and casts to `int` (seconds). The vm path requires a parser-compiler that produces a vm.Program from a string. `vm.ParseExpression` exists (WASM export); whether `vm.Library` covers time/duration arithmetic is a capability check that must land **before Slice 3 is committed**, not during it.

### Structural coupling: shared time-function registry

BP's `delayExpr` duration arithmetic and RDL's `daysSince` will want the same time-function specs in the FunctionSpec registry. **Who defines the shared time-function registry?** Recommended: this project defines the registry; the RDL spin-out consumes it. State this in the sign-off.

---

## §5 — Slice Plan

Option 2: umbrella project with RDL spun out as a separate project.

| Slice | Package(s) | DB-dependent? | Inputs | Output |
|---|---|---|---|---|
| 0 | — | no | — | This doc lands; unified doc gets pointer correction at line 2475 |
| 1 | `internal/boresolver` | **no** | Dead-bridge finding | Delete `ToCELExpression` and its self-recursion. cel-go import count: 5 → 4 |
| 2 | `pkg/policy` + `internal/feed` | **no** | 6 CEL strings enumerated | Rewrite to vm.Expression; `getHardcodedRulesWithCEL` deleted; feed hardcoded rules in vm.Expression form. `pkg/policy/cel_eval.go` deleted. cel-go import count: 4 → 3 |
| 3 | `internal/rules` | yes (`compliance_rules` probe, BP config probe, duration capability) | Q1/Q2 results, §4 condition 2 check, BP representation read | `EvaluateCEL`/`EvaluateValue`/`EvaluateExpr`/`EvaluateExprDebug`/`EvaluateDurationExpr` retired; vm-backed equivalents live; BP config migration executed; `compliance_rules` content migration executed. cel-go import count: 3 → 2 |
| 4 | `internal/rulefabric` | yes (Q1 confirms zero-CEL-row + §4 condition 1 resolves) | Q1 + §4 condition 1 | **Conditional on §4 condition 1:** migrate path → `CreateRuleVersion` writes vm shape, `NormalizeConditionJSONToCEL` retired, `evaluator.go:718-734` CEL dispatch deleted, `EvaluateCELBoolean` deleted, `bo_policy_handler.go:284` retargeted. Second-AST path → `CreateRuleVersion` writes tree shape (matches `CreateRule`), same dispatch deletion. cel-go import count: 2 → 1 (RDL project is sole remaining importer at project end) |
| 5 | `internal/rdl` | yes (`rule_definitions` row count) | (separately scoped project) | RDL eval engine replaced with vm-backed expression + FunctionSpec registry entries for `isInWashSaleWindow`/`hasRecentPurchase`/`daysSince`. Consumes shared time-function registry from §4. **cel-go import count: 1 → 0 (RDL project completes removal)** |

**Slices 1 and 2 are DB-independent** — land first, in either order.

**Slice 3 is the critical path** — gates on BP representation, `compliance_rules` content, and duration capability check.

**Slice 4 is conditional on §4 condition 1** — the AST equivalence decision must resolve before the slice text is finalized. If deferred, record the deferral explicitly in the doc before proceeding.

**Slice 5 (RDL) is a separate project** — begins after this project's Slice 3 lands, consumes shared time-function registry defined by this project.

---

## §6 — Closing Claim

> "One expression language" closing claim is preserved if and only if: (a) Slice 4 §4 condition 1 resolves to migrate `rulefabric.ConditionGroup` → `vm.RuleNode`; (b) Slice 3 lands; (c) Slice 5 (separate project) ports to the same FunctionSpec registry. Otherwise, the closing claim holds for the policy-editor domain only, not for rulefabric.
>
> **Dependency-removal closing claim** (Option 2): this project ends with `cel-go` pinned by exactly one package (RDL). The full removal proof ("`go mod tidy` drops cel-go") is deferred to the RDL spin-out project and must be recorded in that project's sign-off.

---

## §7 — Sign-Off Paragraph

> **CEL retirement project rescope, with explicit sign-off.** Three live request-path packages (`internal/rules`, `internal/rulefabric`, `internal/feed` via `pkg/policy`). Two dead-bridge packages (`internal/boresolver`, `internal/genui`). One spin-out package (`internal/rdl`). Original policy-editor through-line (handoff Slices 1–2) preserved. Closing claim and dependency removal both gated on cross-project boundary with RDL spin-out.
>
> **Static facts** (no DB dependency): code-liveness per §1; both dead-bridge findings are independent; feed enumeration is exact (6 CEL strings in `getHardcodedRulesWithCEL` — 3 cards × 2 CEL fields); UMA rebalance expressions are database-backed via `compliance_rules.expression`; genui dead-bridge confirmed by unpopulated `Config["visibility"]` (no producer found).
>
> **Authoring surfaces** (§2.5): `rule_logic.condition_json` authored through `PolicyRuleBuilder.tsx` and `CreateRuleVersion` handler; `compliance_rules.expression` authored through Hasura (GraphQL, not Go API); BP `entryCondition`/`delayExpr` authored through `bp/designer.go` backend or workflow JSON.
>
> **DB facts** (gating content migrations and Slice 3/4 sequencing): Q1 across `rule_logic` (jsonb probe for `cel`/`group`/`residue`), `compliance_rules` (DISTINCT sampling by `rule_type`), `rule_definitions` (count for RDL), BP config tables (count + DISTINCT non-empty).
>
> **Cross-project boundary**: shared time-function registry is defined by this project (§4) and consumed by the RDL spin-out. This boundary is explicit in Slice 5's scope and must be recorded in the RDL project's sign-off.
>
> **Gating the closing claim**: §4 condition 1 (AST equivalence decision — sequenced before Slice 4 text is finalized), shared time-function registry ownership (RDL spin-out handoff), duration capability check (lands before Slice 3 commit), RDL project completion (dependency removal).
>
> **Signed**: Egan PJ  **Date**: 2026-09-11

---

## §7.1 — Amendment to §7 (2026-09-11, post-probe)

> The following corrections to §7 were established by probe execution and caller trace, and are recorded here as amendments rather than modifications to the signed text. Signed text is frozen; corrections are addenda.

### Amendment A — Static facts: UMA rebalance

The §7 static facts stated: *"UMA rebalance expressions are database-backed via `compliance_rules.expression`."*

**Correction**: `UMARebalanceRulesEngine` (`internal/workflows/uma_activities.go:35`) calls `ListRules` which queries `compliance_rules.expression` — but `compliance_rules` with an `expression` column never existed in any migration (confirmed by grep of all migrations; only `financial_compliance_rules.rule_expression` exists). Every invocation of `ListRules` errors at query time. The path was dead-on-arrival, not database-backed. The §7 characterization was wrong. Reclassification: "live — latent schema bug, fix on deletion" (not a CEL content migration question; see §3.5).

### Amendment B — Authoring surfaces: `compliance_rules.expression`

The §7 authoring surfaces entry stated: *"`compliance_rules.expression` authored through Hasura (GraphQL, not Go API)."*

**Correction**: `compliance_rules` with an `expression` column was never created in any migration. The Hasura authoring surface, the `SQLRuleRepository` fallback write path, and the `ListRules` read path all reference a table that never existed. Not a live authoring surface — dead-on-arrival. The table/column pair that does exist is `financial_compliance_rules.rule_expression` (20260731_financial_superpowers.up.sql), which has a different schema and zero rows.

### Amendment C — Feed rank score expressions (Slice 2 functional regression)

Slice 2 execution dropped two numeric `CELRankScore` expressions rather than porting them to `Conditions`-compatible Go code:
- `tax_loss_harvest`: `abs(client.Portfolio.UnrealizedLossPct) * 100.0` — dropped
- `portfolio_drift`: `client.Portfolio.DriftPct * 100.0` — dropped

Current `evaluateHardcoded` returns `RankScore: 1.0` for all eligible cards. The rank-based card ordering is a constant tie — the original dynamic ranking by loss magnitude and drift percentage is lost. **This is a functional regression introduced by Slice 2 execution.** Fix required before Slice 2 is considered complete.
