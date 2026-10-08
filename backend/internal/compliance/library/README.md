# Compliance Scenario Corpus & Rule Library

This directory contains the scenario test vectors and evaluation router for the Uisce Compliance Rule Library.

## 1. Corpus Structure & Modular File Split

The scenario corpus is split into modular domain files to keep scenario definitions maintainable:

- **[`scenario_corpus.go`](./scenario_corpus.go)**: Core `Scenario`, `OrderContext`, `ExpectedOutcome` structs and top-level `EvaluateScenario` router.
- **[`corpus_core_pretrade.go`](./corpus_core_pretrade.go)**: 200 deterministic scenario test vectors across the 50 pre-trade rules (4 vectors per rule).
- **[`corpus_phase1.go`](./corpus_phase1.go)**: 56 deterministic scenario test vectors across the 14 post-trade rules from Phase 1.
- **[`corpus_phase2.go`](./corpus_phase2.go)**: 88 deterministic scenario test vectors across the 22 post-trade rules from Phase 2 (Taxonomy, Ratings, ESG, SFDR).
- **[`corpus_phase7.go`](./corpus_phase7.go)**: 34 deterministic scenario test vectors across the 8 Phase 7 Ownership & Disclosure rules (Rules 37–44).

### Invariant Equation & Arithmetic Convention
$$\text{Total Active Scenarios} = 378 = 200 \;(\text{Pre-Trade}) + 56 \;(\text{Phase 1}) + 88 \;(\text{Phase 2}) + 34 \;(\text{Phase 7})$$
$$\text{Total Active Gold Rules} = 94 = 50 \;(\text{Pre-Trade}) + 14 \;(\text{Phase 1}) + 22 \;(\text{Phase 2}) + 8 \;(\text{Phase 7})$$

**Vector Count Convention:**
- **Minimum Requirement:** Every active rule in the gold-copy library must have a minimum of **4 deterministic scenario test vectors** (`PASS`, `BOUNDARY`, `FAIL`, `ADVERSARIAL`).
- **Additional Specialized Vectors:** Rules with multi-step mechanics, tenant threshold override paths, or unique statutory calendar branches may include multiple adversarial vectors (e.g. `ADVERSARIAL_TENANT_OVERRIDE_4PCT`, `ADVERSARIAL_CROSSING_FRIDAY_BEFORE_HOLIDAY`).
- **Mechanical Accounting:** The exact vector count and coverage compliance are mechanically introspected and printed by `scripts/query_rules_inventory.go` (and `backend/cmd/compliance-inventory`).

---

## 2. Corpus Authoring Guidelines

### Ceilings vs. Floors (Boundary Semantics)

When authoring vectors, be aware of the comparison operator:

1. **Ceiling Rules (`LTE` / `<=`, e.g. Max Issuer Concentration 5% / 20%)**:
   - `PASS`: Below the limit (e.g. 4.0% <= 5.0%) $\rightarrow$ `PASSED`
   - `BOUNDARY`: Exactly at the limit (e.g. 5.000000% <= 5.0%) $\rightarrow$ `PASSED`
   - `FAIL`: Above the limit from the top (e.g. 5.1% > 5.0%) $\rightarrow$ `BLOCKED` / `WARNING`
   - `ADVERSARIAL`: Gross excess / hostile input (e.g. 25.0% > 5.0%) $\rightarrow$ `BLOCKED`

2. **Floor Rules (`GTE` / `>=`, e.g. Min Cash Liquidity 5%)**:
   - `PASS`: Above the floor (e.g. 8.0% >= 5.0%) $\rightarrow$ `PASSED`
   - `BOUNDARY`: Exactly at the floor (e.g. 5.000000% >= 5.0%) $\rightarrow$ `PASSED` (inclusive)
   - `FAIL`: Below the floor from the bottom (e.g. 3.2% < 5.0%) $\rightarrow$ `WARNING` / `BLOCKED`
   - `ADVERSARIAL`: Complete depletion / near-zero (e.g. 0.5% < 5.0%) $\rightarrow$ `WARNING`

3. **Dynamic Parameter Overrides (`OrderContext.ParameterOverrides`)**:
   - Rules evaluating customizable thresholds (such as Rule Families) must support dynamic parameter threshold lookup.
   - Scenario fixtures testing tenant overrides populate `OrderContext.ParameterOverrides` to verify custom tolerance boundaries without alerting.

---

## 3. Working Agreement & Refactoring Discipline

- **Append-Only Go Edits**: All scenario additions, modifications, or rule tranche additions must be authored as direct, strongly typed Go code in the respective corpus files. Ad-hoc text manipulation scripts or regex line surgery are strictly prohibited.
- **Atomic Rule Unit**: Every rule must land with:
  1. DDL Migration (`.up.sql` and `.down.sql`)
  2. RFC 8785 Canonical Go Hash snapshot agreement (`TestCoreLibrary_All94CoreRules_ContentHashAgreement` and `TestComputePhase7Hashes`)
  3. $\ge 4$ Deterministic scenario vectors (`PASS`, `BOUNDARY`, `FAIL`, `ADVERSARIAL...`)
  4. Unit / E2E test verification in ephemeral DB harness
