# Rule Engine Centralization — Plan

> **Status:** SIGNED 2026-09-24 (owner sign-off given in chat; D1–D4 open — resolved per slice as noted). Per the sign-then-execute protocol recorded in
> `docs/cel-retirement-handoff.md` (Amendment F failure log), no slice below is
> executed until this document is signed.
>
> **Goal (owner's words):** "the master engine is the AST/WASM and should be the
> one attached to business objects and also used for compliance … I want one and
> only one."
>
> **Relationship to the CEL retirement project.** That project retires *one
> expression language* (cel-go). This plan retires *every other rule engine*.
> CEL retirement is a subset; its open slices are folded in here as Slice 1
> rather than duplicated.

---

## §1 — The master engine (definition)

"The master engine" means exactly this, and nothing else evaluates rules:

| Part | Location |
|---|---|
| AST | `backend/internal/rules/vm/ast.go` (`RuleNode`/`RuleGroup`/`RuleCondition`/`Expression`/`FuncCall`) |
| Evaluator | `vm.AdvancedEvaluator` (tree-walk) + bytecode VM fast path (`CompileVM`, falls back to the evaluator on `Unsupported`) |
| Function library | `vm` `FunctionSpec` registry (native + StarRocks SQL pushdown) |
| Browser build | `backend/rule-engine/cmd/wasm` → `rule_engine.wasm` (CI: `verify_wasm.js`, `check-drift`) |
| Storage | `catalog_node` rows, `node_type = validation_rule`, AST in `config.rule_ast`, `domain` ∈ {validation, mdm, compliance} in `properties` |
| Binding to BOs | `GOVERNED_BY_RULE` edge from the BO's `classification_node_id` |
| Authoring | `AdvancedRuleBuilderPage` (domain selector) |
| BO write enforcement | `metadata.BusinessObjectService.writeAndEnforce` → `evaluateAndEnforceRules` → `ValidationRuleService` → `vm.AdvancedEvaluator` |

Proven end-to-end already: `cmd/verify_oracle_rule`, `cmd/verify_three_domains`
(validation + mdm + compliance on the same BO write, all blocking).

---

## §2 — Inventory: every rule evaluator on `main` (ca2d55b9a)

Verdicts are from static call-graph tracing (grep + GitNexus, index 4 commits
stale). "No callers" is a **static** claim; each deletion slice re-verifies it
immediately before executing, and DB-backed engines get a row-count probe, per
the CEL project's discipline.

| # | Engine | Tech | Verdict | Evidence |
|---|---|---|---|---|
| E1 | `internal/rules/vm` | AST + VM + WASM | **MASTER** | §1 |
| E2 | `internal/rules.RuleEngine` CEL methods (`EvaluateCEL`/`EvaluateValue`/`EvaluateExpr*`/`EvaluateDurationExpr`) | cel-go | **live, being retired** | CEL Slice 3 merged (#71); remaining cleanup is open PR #74 |
| E3 | `internal/rulefabric` | own `ConditionGroup` AST + CEL + shared bytecode | **live (mounted), zero content** | `api.go:1274` mounts `rulefabric.RegisterRoutes`; `rule_logic` = 0 rows (CEL handoff §5). Slice 4 PR #72 was closed to split it; not re-landed |
| E4 | `internal/rdl` | cel-go | **live** | `api.go:3784` `RegisterRDLRoutes`; already a separate spin-out project (CEL §5 Slice 5) |
| E5 | `internal/services.ValidationRuleEngine` + `cmd/rule-engine-service` + `cmd/validation-service` | own condition model | **deployed as separate services** | constructed by both cmds; both appear in `docker-compose.backend*.yml` / `docker-compose.local.fixed.yml` |
| E6 | `internal/services.StarlarkEngine` + `internal/starlib` | Starlark | **likely dead** | only `cmd/starlarktest` and `internal/workflows/starlark_workflows.go` (no workflow registration found) |
| E7 | `internal/validation` (`ValidationEngine`, `TriggerValidationEngine`) | hand-coded | **dead** | constructed only in `trigger_dispatch_handlers.go` / `validation_triggers_handlers.go`, whose route registration is never called from `api.go` (confirmed again; also unified handoff §6) |
| E8 | `internal/wasm` (wazero) + `internal/services.ComplianceService` | separate WASM module | **dead** | `NewWazeroEngine` has zero callers; `api.go:1294` construction is commented out; `NewDailyETLScheduler` (its consumer) has zero callers |
| E9 | `backend/services/compliance-engine` | CUE (`policy/*/trade_compliance.cue`) | **deployed separate service** | own `go.mod`; built by `Dockerfile.compliance-engine`; in `docker-compose.local*.yml`. Pre/post-trade compliance. **This is a second compliance engine.** |
| E10 | `internal/altinv.RuleEngine` | own condition model | **dead** | `NewRuleEngine(RuleEngineConfig)` has zero callers |
| E11 | `internal/feed/rules.RuleEngine` | uses `vm` | **already on master** | imports `internal/rules/vm` (CEL Slice 2) — name collision only |
| E12 | `internal/wealth` compliance | — | **not an engine** | no rule-evaluation imports; out of scope unless it proves otherwise |

**Not rule engines (out of scope):** `services/compliance-engine` at repo root is
workflow **ABAC** (access control), not data rules; `internal/mdmrules` is rule
*content* for E1; `internal/semanticast` is NL→query.

---

## §3 — Gaps that block "one engine attached to business objects"

**G1 — Two BO record APIs; only one enforces rules.**
- `/business-objects/{id}/data` → `BusinessObjectService.CreateBORecord`/`UpdateBORecord` → `writeAndEnforce` → **E1**. ✅
- `/bo/{boKey}/records` (`api/bo_crud_handler.go`) → raw `INSERT`/`UPDATE`, **no rule evaluation**; only `TriggerEngine` row events. ❌
  A BO can be written without its validation/mdm/compliance rules running.
  *(Also: the bulk-records endpoint drafted on the data-pipeline branch was
  built on this handler and inherits the gap — it must be rebuilt on G1's fix.)*

**G2 — Compliance lives in three places.** E1 (`domain = compliance`), E8 (dead
wazero), E9 (CUE service). Target: E1 only.

**G3 — Nothing stops a new engine being added.** Every past consolidation
(rule-engine/runtime, catalog_validation_rules, rulefabric) happened because a
new engine appeared silently.

---

## §4 — Slice plan

Order: close enforcement gaps first (behavioural, highest value), then delete,
then lock the door.

| Slice | Scope | DB? | Output |
|---|---|---|---|
| **0** | This document | no | Signed plan |
| **1** | Finish CEL retirement code: land PR #74; re-land Slice 4 (delete E3, extract `OperatorRegistry` to `vm`, retarget `bo_policy_handler.TestCondition` to E1) as the split code PR that #72's close comment promised | probe `rule_logic` row count again | cel-go importers 3 → 1 (RDL) |
| **2** | **G1:** route `/bo/{boKey}/records` create/update (and a new bulk endpoint) through `BusinessObjectService.writeAndEnforce`. One enforcement path for every BO write. Violations return the same 422 shape as `/business-objects/{id}/data` | no | Test: a rule that blocks on `/business-objects/{id}/data` also blocks on `/bo/{key}/records` and bulk (parity test, both directions) |
| **3** | **G2 part 1:** delete dead compliance/validation engines E7, E8 (+ `DailyETLScheduler`, `ComplianceService` in `internal/services`), E10, E6 (after confirming `starlark_workflows` unregistered) | no | Packages removed; build + tests green |
| **4** | **G2 part 2 — CUE (E9):** port the rules in `policy/2021` + `policy/2025/trade_compliance.cue` to `catalog_node` compliance-domain rules on the Order BO; decide pre-trade latency path (E1 bytecode VM is the fast path); retire the service + Dockerfile + compose entries. **Needs owner decision D2** | probe: is the service receiving traffic anywhere? | One compliance engine |
| **5** | **E5:** migrate or retire `rule-engine-service` / `validation-service` (list/facet APIs over the old condition model). **Needs owner decision D3** | probe: rule rows those services read | Services removed from compose, or reduced to thin read APIs over `catalog_node` |
| **6** | **G3 guardrail:** CI test (pattern of `mcp/no_direct_sql_test.go`) that fails if any package outside `internal/rules/vm` (+ RDL until its project closes) imports `cel-go`, `starlark`, `wazero`, `cuelang`, or declares a new rule-evaluator type; plus CODEOWNERS on `internal/rules/vm` | no | New engines cannot land silently |
| **RDL** | Existing spin-out (CEL §5 Slice 5) | yes | cel-go 1 → 0 |

Slices 2, 3, 6 are independent of each other and DB-independent → can land in
parallel after sign-off. Slice 1 is mostly already-written code.

---

## §5 — Decisions needed from the owner (block the slices named)

- **D1 (Slice 1):** Re-land Slice 4 from the closed #72 branch, or redo it fresh on current `main`?
- **D2 — RESOLVED (owner, 2026-09-24): "cue i believe is dead code" → delete.** Caveat for Slice 4 to verify first: the monolith still constructs `services.NewCueEngine()` (`api.go:1004`) and passes it to `RegisterValidationRulesRoutes` (`api.go:1879`) — that in-process CUE use must be retired too, not just `backend/services/compliance-engine`.
- ~~D2 (Slice 4):~~ Is the CUE compliance-engine used by anything real (a trading
  flow, a demo)? If yes, port its rules first; if no, delete outright.
- **D3 — RESOLVED (owner, 2026-09-24): only in old compose files → retire both services; "I can only have one engine running."**
- ~~D3 (Slice 5):~~ Are `rule-engine-service` / `validation-service` running in any
  environment you care about, or only in stale compose files?
- **D4 — RESOLVED in Slice 2:** no new mode. `/bo/{key}/records` uses the exact enforcement the other API uses, gated by the same `VALIDATION_RULES_ENFORCE` flag (shadow by default, blocking when `true`). One engine, one switch.
- ~~D4 (Slice 2):~~ On `/bo/{key}/records`, should rule violations *block* (as the
  other API does today with enforcement on) or start in shadow/log-only mode
  for one release?

### Slice 2 — landed on `feat/rule-engine-centralization` (2026-09-24)

- `metadata.BusinessObjectService.EnforceWrite` / `EnforceWriteBatch`: the single gate for BO writes outside `CreateBORecord`/`UpdateBORecord`. Same transaction, same `evaluateAndEnforceRules`, same persistence/logging (`reportViolations`, extracted from `writeAndEnforce` with no behaviour change). Batch = one transaction, one savepoint per row; dry-run rolls back and persists nothing.
- Typed `RuleRejectionError` (message unchanged) → **422 with `rules[]` on both APIs** (was 500 on `/business-objects/{id}/data`).
- `/bo/{boKey}/records` create/update and related-record create/update now write through `EnforceWrite`; related-record writes are judged by the **child** BO's rules. New `POST /bo/{boKey}/records/bulk` (create | upsert, ≤5000 rows, dry_run) through `EnforceWriteBatch`.
- Fail closed: `NewBOCRUDHandler` takes the enforcer; nil → writes refused with 503, never written around the rules.
- Proof: 6 metadata tests (real rule evaluation over sqlmock: block, shadow, unknown BO, write error, per-row savepoints, dry run) — mutation-checked: disabling evaluation fails 4 of them; 5 handler tests (422 + rollback, 503 with no INSERT, bulk attribution + tenant_id forced, upsert builder, child-BO attribution); 422 parity test on `/business-objects/{id}/data`.
- **Not proven live** against a real database in this slice (mock-level only).
- Found, not fixed (separate tickets): related-record create/update interpolate client JSON keys as column names **without** the writable-column allowlist the main handlers use; `TestGetBusinessObjectIncludesChildIntegration_Container` fails on `main` ("business object not found"), independent of this change.

### Slice 3 — dead engines removed, on `feat/rule-engine-slice3-dead-engines` (2026-09-24)

Every target re-verified dead immediately before deletion (importers of the
package, constructors of its types, route registration), per §2's rule.
Re-verification **changed the inventory** — recorded here, not silently absorbed:

| Removed | Why it was dead |
|---|---|
| `internal/starlib`, `cmd/starlarktest`, `internal/services/starlark_*` (engine, metrics, tracing, tests), `internal/workflows/starlark_workflows.go` | no constructor callers; workflow never registered |
| **Starlark inside the master package:** `internal/rules/{loader,core_compile,corecache,tenant_compile,tenantcache,bootstrap}.go` | only wiring was commented out in `cmd/worker/main.go:232-237` (also removed) — *new finding: E1's own package still carried a second engine* |
| `internal/rules/scenario_runner.go` (+ its test) | never constructed; the GraphQL `RunScenario` resolver is a "not implemented" panic |
| `internal/ai/{suggest_service,starlark_guard,upgrade_assist}.go` + `ai_test.go` | Starlark rule-authoring assistants for the dead tenant compiler; no constructor callers |
| `internal/wasm`, `internal/services/{compliance_service,risk_service}.go`, `internal/scheduler` | nothing constructs Compliance/RiskService; `internal/scheduler` has zero importers. *Correction to E8: the wazero module also backed a (dead) RiskService* |
| `internal/validation/{engine,trigger,trigger_dispatch,validator,condition_schema,test_service}.go` (+ tests); `internal/api/{validation_triggers_handlers,trigger_dispatch_handlers}.go` (+ test) | route registration never called. `internal/validation/schema.go` **kept** — `ValidateUpgradeArtifacts` is live (upgrade service) and is not a rule engine |
| `internal/api/expression_handlers.go` | never constructed; every endpoint returned 501 "Starlark removed" |
| `internal/altinv/rule_engine.go` | zero references |
| wazero runtime in `mdm.ExecutionEngine` | created, never executed a module |
| `go.starlark.net`, `github.com/tetratelabs/wazero` | removed from `go.mod`/`go.sum` by hand — `go mod tidy` is broken on `main` (`libs/db/queries` has no `go.mod`), pre-existing |

Proof: `go build ./...` clean; `go vet` on every touched package clean; tests
green for rules, rules/vm, mdm, workflows, bp, api. Failures that also fail on
unmodified `main` (not caused here): `TestSecurityManager_Checklist`,
`internal/security` vet (`ResolveTenantForRequestWarn` undefined).
`detect_changes`: CRITICAL by reach (hub functions `main`/`SetupRouter`), but
the edits in those are removal of commented-out code only.

### New inventory items found during Slice 3

- **E13 — `mdm.ExecutionEngine` is a fake calc engine on a live route.** Portfolio
  analytics (`PortfolioSecurityService` → NAV) calls `ExecuteCalculation`, whose
  `engine: "wasm"` path only sums inputs when the expression contains "sum", and
  whose default path silently returns `0.0`. **New slice (3b):** replace the
  engine switch with E1 (`vm` expression evaluation of the term's `rule_ast`,
  the calc-term convention), fail loud on unknown expressions.
- **E9 is partly live in-process.** CUE is not only the separate
  `backend/services/compliance-engine`: `services.NewCueEngine()` is constructed
  (`api.go:1004`) and `validation_rules_routes.go:1060,1235` evaluate CUE scripts
  on the mounted `/validation-rules` handlers. Slice 4 must retire both, and
  `cuelang.org/go` leaves `go.mod` then.

### Slice 4 — CUE retired, PR #128 (2026-09-24)

- In-process CUE (`services.CueEngine`, `CueSchemaGenerator`) removed; the
  mounted `/validation-rules/{schema, {id}/execute, {id}/simulate-with-instance}`
  endpoints — which could only ever evaluate the retired, all-inactive
  `catalog_validation_rules` corpus — now return 410 `endpoint_retired`.
- Standalone `backend/services/compliance-engine` + `Dockerfile.compliance-engine`
  + 3 compose entries + `docker-start.sh` entries deleted. The root
  `services/compliance-engine` (workflow ABAC, k8s port 8082) is a different
  service and is untouched.
- Dead `CubeGenerator` removed (owner: "cube generator is also dead").
- `cuelang.org/go` + CUE-only indirect deps removed from `go.mod`/`go.sum`.
- Proof: build/vet/dependency resolution clean; new test pins all three
  endpoints to 410 with zero DB queries. `detect_changes` HIGH by reach
  (`SetupRouter`).

### Items added to the plan after Slice 4

- **Slice 4b — frontend legacy rule UIs.** 15 files still author CUE/Starlark
  (`ValidationRuleCreator`, `ValidationRuleWizard`, `ValidationRuleScriptEditor`,
  `RuleJsonViewer`, `UisceRuleBuilder`, `ExpressionEditor`/`ExpressionLibrary`,
  fabric `ValidationRulesPage`, `UisceBuilderPage`, …). With no backend engine
  behind them they are dead weight; replace entry points with
  `AdvancedRuleBuilderPage` and delete.
- **E14 — `boresolver` filter-group SQL compiler** (`CompileFilterGroup`,
  reached by `/validation-rules/execute-binding` and `bo_sql_generator.go`)
  compiles the old condition format to SQL — overlaps `vm.CompileToSQL`.
  Assess in Slice 5.
- **Cube removal** (owner: "remove any cube items we are not using it ever") —
  separate cleanup branch, same verify-then-delete discipline.
- **Whole-backend dead code:** `deadcode ./...` reports ~10,300 unreachable
  functions. Out of scope here; worth its own staged program.

### Slice 5 — rule-engine-service and validation-service retired, PR #130 (2026-09-24)

- **D3 correction:** the services were not only in old compose files — they
  were also in the current `docker-compose.yml`/`.local.yml`/`.local-apps.yml`,
  `START_FULL_SYSTEM.sh`, Prometheus and Grafana. Retirement still holds
  because nothing calls them: all frontend `/api` traffic goes to the
  monolith, and no UI code calls their routes.
- Removed both binaries, `testeval`, their Dockerfiles, and everything only
  they reached: `services.ValidationRuleEngine`, `AsyncValidator`,
  `BPValidationCoordinator`, `handlers.ValidationHandler`, `rules.PathResolver`,
  plus orphaned functions of a duplicate `services.BusinessObjectService`.
- All deployment wiring removed (7 compose files, startup scripts, Prometheus,
  Grafana, empty Helm chart, dev proxy route). `deadcode`: nothing newly
  orphaned vs `main`.

### Separate: Cube.js integration removed, PR #129

Not a rule engine, but removed in the same discipline at the owner's request.
The platform's own Cube.dev-shaped semantic model (fabric cubes/views/measures,
`models.Cube`, semantic-term Cube properties) is live and was kept.

### Slice 6 — CI guardrail, PR #131 (2026-09-24)

`backend/internal/archguard/rule_engine_guard_test.go` (in CI's `go test ./...`)
fails on any import of an embedded rule/expression/scripting/policy engine
library, or such a module in `go.mod`, outside a shrinking allowlist whose every
entry names the PR/slice that removes it. Mutation-checked three ways. Also
deleted the dead `pkg/policy/rego_eval.go` and two unreferenced `.rego` files.

### Slice 3b — calc terms on E1, PR #132 (stacked on #127) (2026-09-24)

`mdm.ExecutionEngine` now evaluates calculation terms with `internal/rules/vm`
(rule_ast → config.expression → legacy properties.expression) and fails loud
instead of returning 0.0. Finding: NAV never reaches it today — term name
mismatch (`NetAssetValue` seeded vs `"Net Asset Value"` looked up) — so no NAV
figure changes.

### E15 — live OPA/Rego engine (found in Slice 6)

`pkg/governance` evaluates Rego policies for the glossary handler, pipelines
handler and worker compliance activities: `trade_compliance.rego`,
`pipeline_validation.rego`, `semantic_validation.rego`, `portal_authz.rego`.
Allowlisted in the guard as "slice 7".

**D5 — RESOLVED (owner, 2026-09-24): "all should use a single engine no
exceptions."** Every Rego policy, including `portal_authz`, moves to E1, and
RDL is not an exemption either — its CEL formulas port to E1 functions. The
guard's allowlist must end empty.

### Slice 1 — rulefabric deleted, PR #134 (stacked on #130, includes #74) (2026-09-24)

- #74 brought up to date with `main` (155 commits, clean merge) and verified;
  merge decision left to the owner per the CEL sign-off protocol.
- `internal/rulefabric` deleted: read-only probe of every database on the host
  — only `alpha` has its tables, all data tables at 0 rows.
- Every frontend consumer was dead (BO Governance Studio incl. PolicyRuleBuilder
  — unmounted, its APIs never existed; RuleFabricPage; rulefabric components)
  or reduced (`ExpressionBuilder` lost its RuleFabric autosave). CEL completion
  infra removed from the Monaco registry.
- cel-go importers: `internal/rdl` only.

### Slice 4b — legacy rule-authoring UIs retired, PR #135 (stacked on #134) (2026-09-24)

One editor (`AdvancedRuleBuilderPage` / `/api/validation-rule-nodes`) for one
engine. CUE/Starlark/legacy-corpus UIs deleted (all their backends retired or
never mounted), entry points re-pointed, flow-builder Policy Check reads E1
rules, BO AI synthesis suggests E1 expressions, dead backend `internal/builder`
(CUE generator) deleted. TypeScript: zero new errors.

### Slice 7a — OPA governance + trigger conditions on E1, PR #136 (2026-09-24)

- **Engine:** the condition operators the editor always offered but the
  evaluator never implemented (handoff item 54) — `contains`, `starts_with`,
  `ends_with`, `matches_regex`, `is_empty`, `length_*`, `in`, `not_in`,
  `contains_any`, `contains_all`, `is_true/false`, `is_positive/negative/zero`
  — with the editor's exact semantics. Additive: 0 of 160 stored rules use
  them. Browser WASM refreshed and smoke-checked (old binary fails the check).
- **E15 closed:** `pkg/governance` is E1 rules; `open-policy-agent/opa` and
  its 8 exclusive modules gone. Parity table pins decisions. Finding: on
  `main` the trade and pipeline Rego never evaluated (parse error / undefined
  query), so `POST/PUT /api/v1/pipelines` returned 500 on every call and the
  compliance activity errored on every trade.
- **E16 (new, closed):** `api.TriggerEngine` had its own hand-rolled
  condition evaluator (with a `contains` that never checked containment);
  now E1. 0 of 7 stored triggers had conditions.

### Slice 7b — dead evaluators deleted, `workflow-service` retired (2026-09-24)

Systematic scan for hand-rolled operator evaluators. Deleted (all
unreachable): BP branch evaluators (`pkg/bp`, 3 files) + never-mounted
branching handlers, `internal/uisce/filters`, the enhanced BP workflow
evaluator, the feed card-rule engine + dead `feed` root package, the
"pre-trade compliance VM", 14 dead evaluator functions in
`pkg/workflows/condition_engine.go`. `cmd/workflow-service` (own in-memory
evaluator; secondary compose files only; no callers) retired.

Left deliberately: `internal/ops` alert evaluator — unreachable, but inside
a 27-of-30-files-dead subsystem that belongs to the dead-code program;
`PolicyConditionActivity` is a pass-through stub, not an evaluator.
SQL/query compilers (`QueryBORecords` filters, `querycompiler`, `boresolver`)
translate conditions to SQL rather than evaluate them — tracked as E14.

### Slice 8 — RDL retired, cel-go gone, PR #138 (stacked on #134) (2026-09-24)

Planned as a port; the evidence made it a retirement: `rule_definitions` has
0 rows everywhere, the RDL UI was never mounted, and nothing calls `/api/rdl`
or its `/api/rules` shorthand. `internal/rdl`, its routes and UI deleted;
`cel-go` + 3 exclusive modules removed from `go.mod`. With the stack merged,
no backend package imports cel-go — the CEL retirement project's closing claim
("`go mod` drops the dependency") holds.

### Stack merged (2026-09-24)

Merged in order: #126, #127, #132, #128, #129, #74, #130, #134, #135, #138,
#136, #137, #131. Two merge resolutions, both "keep both deletions"
(`docker-start.sh`/`prometheus.yml` between #128/#129 and #130; adjacent
`go.mod` removals between #136 and #138). `rule-engine/generated` artifacts
regenerated on #136 after #130 (the retired Cube/CUE/Starlark/validation types
left `asl.*`); `check-drift` green. The guard's `engineImportAllowlist` is
empty; a planted cel-go import fails it. No `go.mod` in the repo and no
frontend `package.json` requires an engine library.

### Remaining

- E14: SQL filter compilers (`boresolver` filter groups, `QueryBORecords`
  filters, `querycompiler`) vs `vm.CompileToSQL` — translation, not
  evaluation, but a second condition vocabulary.
- Expression-parser string literals (`Literal` holds only float64).
- Dead-code program: ~10k unreachable functions, incl. `internal/ops`.
---

## §6 — Closing claim

"One rule engine" holds when: Slices 1–6 land and the RDL project closes. At that
point the §6 guardrail test is the standing proof: the only rule-evaluating
package importable in `backend/` is `internal/rules/vm`, and every BO write path
calls it.

**Data-pipeline dependency:** the pipeline's `rule_check` node binds to E1 via
the same `ValidationRuleService` path BO writes use (not `rules.RuleEngine`'s
compliance-rule batch API, which the draft adapter currently targets). The
pipeline resumes after Slice 2.

---

**Signed:** Egan PJ (in chat: "signed, start with step 2") **Date:** 2026-09-24
