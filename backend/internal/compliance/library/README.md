# Compliance Scenario Corpus & Rule Library

This directory contains the scenario test vectors and evaluation router for the Uisce Compliance Rule Library.

## 1. Corpus Structure & Modular File Split

The scenario corpus is split into modular files to keep scenario definitions maintainable:

- **[`scenario_corpus.go`](./scenario_corpus.go)**: Core `Scenario`, `OrderContext`, `ExpectedOutcome` structs and top-level `EvaluateScenario` router.
- **[`corpus_core_pretrade.go`](./corpus_core_pretrade.go)**: 200 deterministic scenario test vectors across the 50 pre-trade rules (4 vectors per rule).
- **[`corpus_phase1.go`](./corpus_phase1.go)**: 28 deterministic scenario test vectors across the 7 post-trade rules (3 pilot + 4 Phase 1 rules).

### Invariant Equation
$$\text{Total Active Scenarios} = 228 = 200 \;(\text{Pre-Trade}) + 28 \;(\text{Post-Trade})$$
$$\text{Total Active Gold Rules} = 57 = 50 \;(\text{Pre-Trade}) + 7 \;(\text{Post-Trade})$$

Every rule in the gold-copy library is covered by exactly 4 deterministic scenario test vectors (`PASS`, `BOUNDARY`, `FAIL`, `ADVERSARIAL`), yielding 100% resolution in `TestCoreLibrary_ScenarioRuleCodeResolution`.

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

---

## 3. Working Agreement & Refactoring Discipline

- **Append-Only Go Edits**: All future scenario additions, modifications, or rule tranche additions must be authored as direct, strongly typed Go code in the respective corpus files. Ad-hoc text manipulation scripts or regex line surgery are strictly prohibited.
- **Atomic Rule Unit**: Every rule must land with:
  1. DDL Migration (`.up.sql` and `.down.sql`)
  2. RFC 8785 Canonical Go Hash snapshot agreement (`TestCoreLibrary_All57CoreRules_ContentHashAgreement`)
  3. 4 Deterministic scenario vectors (`PASS`, `BOUNDARY`, `FAIL`, `ADVERSARIAL`)
  4. Unit / E2E test verification in ephemeral DB harness
