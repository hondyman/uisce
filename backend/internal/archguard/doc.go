// Package archguard holds architecture invariants enforced as tests. It has
// no runtime code.
//
// rule_engine_guard_test.go enforces the single-rule-engine decision
// (docs/rule-engine-centralization-plan.md): internal/rules/vm is the only
// rule/expression/policy engine in the backend. Any other embedded engine
// library fails CI unless it is on the shrinking allowlist.
package archguard
