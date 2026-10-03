// Package archguard holds architecture invariants enforced as tests. It has
// no runtime code.
//
// rule_engine_guard_test.go enforces the single-rule-engine decision
// (docs/rule-engine-centralization-plan.md): internal/rules/vm is the only
// rule/expression/policy engine in the backend. Any other embedded engine
// library fails CI unless it is on the shrinking allowlist.
//
// tenant_connection_guard_test.go and tenant_connection_inventory_test.go enforce ADR-030: a
// tenant's own database is reached only through internal/tenantdb. Every non-test file that
// opens a database connection is classified (control plane, warehouse, provisioning, tool, tenant
// datasource), a new or additional opener fails CI until it is, and the tenant-datasource entries
// are the work list for moving data access behind tenantdb.
package archguard
