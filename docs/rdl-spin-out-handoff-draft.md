# RDL Spin-Out — Project Handoff Draft

> **Status**: early draft — opening section only. Does not yet carry project scope or sign-off.
> **Created**: 2026-09-11
> **Parent**: `docs/cel-retirement-handoff.md` — RDL is Slice 5 of the CEL retirement project (separate project per §5 opt-out)

---

## Opening Finding — RDL is Data-Dead

**Probe result** (`alpha`, 2026-09-11):
```sql
SELECT count(*) FROM rule_definitions; -- = 0
```

`rule_definitions` is the storage table for RDL expressions. It has zero rows. The RDL evaluation engine (`internal/rdl/service.go`) is code-live — it has callers, a wired `cel.Env` with custom function declarations (`isInWashSaleWindow`, `hasRecentPurchase`, `daysSince`), and a test suite. But no RDL expressions have ever been written to the table.

**Contrast with the validation surface**: `catalog_validation_rules` has 233 rows in legacy `{"authored_mode":"designer"}` format. The validation surface is used. RDL is not.

---

## Opening Question

**Does RDL have users?**

The question is not "how do we port the DSL to vm" or "what is the migration path." The question is whether anyone has ever used RDL to author a rule. If the answer is no — if `rule_definitions` has always been empty since the table was first created — then this project is not a port. It is a deprecation decision: the `internal/rdl` package's evaluation engine should be assessed for what other code depends on it, and either retired or retained only if some untraced consumer (Hasura GraphQL, a frontend authoring surface, an external system) writes to `rule_definitions` outside the Go codebase.

---

## What RDL Currently Does

From `internal/rdl/service.go`:

- `EvaluateRule(ctx, ruleName, input, params)` — evaluates a named rule from `rule_definitions` against an input map and params map
- `EvaluateRuleBatch(ctx, ruleName, inputs, params)` — batch variant
- `ListRules(ctx)` — returns all enabled rules (queries `rule_definitions` by `enabled = true`)

Custom CEL functions declared in the `cel.Env`:
- `isInWashSaleWindow(string, string) bool` — wash sale window check
- `hasRecentPurchase(string, string, int) bool` — recent purchase check
- `daysSince(timestamp) int` — days since a timestamp

---

## What Needs to Be Determined Before Scope Is Set

1. **Has `rule_definitions` ever had rows?** Query migration history or seed files. If the table was always empty, the "migration" question collapses to "should we keep or delete the evaluation engine."

2. **What calls `RDLService`?** Trace `internal/rdl/service.go` callers. The CEL retirement doc (§1) notes `internal/api/rdl_routes.go:11` wires it. Are there other callers?

3. **Does Hasura or any external system write to `rule_definitions`?** If the authoring surface is GraphQL (Hasura) rather than a Go API, the zero-row finding could mean the surface exists but was never used, or the surface doesn't exist.

4. **What is the relationship between RDL and `catalog_validation_rules`?** 233 rows of designer-mode validation rules exist in `catalog_validation_rules`. Are they related to RDL? Could RDL be a reimplementation that never got adoption?

5. **If RDL is retained**: the CEL retirement project defines the shared time-function registry (§4) that RDL would consume. RDL's `isInWashSaleWindow`/`hasRecentPurchase`/`daysSince` custom functions would be replaced with `vm.Expression` calls using the shared registry.

---

## Next Step

Answer the five questions above before writing project scope. The default assumption (data-dead, deprecation) may be wrong — but the burden of proof is now on showing RDL has users, not on showing it doesn't.
