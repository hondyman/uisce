package metadata

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// A rule is either unscoped (every binding) or scoped to specific bindings. The evaluator must run
// unscoped rules always, scoped rules only on their binding, and report a scoped rule whose active
// binding cannot be determined instead of skipping it.
func TestEvaluateAndEnforceRules_BindingScope(t *testing.T) {
	const tenant = "t-1"
	const boID = "00000000-0000-0000-0000-0000000000b0"
	failing := `{"type":"condition","field":"amount","fieldPath":"amount","operator":">","value":100,"valueType":"number"}`

	ruleProps := func(scope string) string {
		return `{"bo_name":"party","tenant_id":"t-1","severity":"BLOCK","timing":"pre_write"` + scope + `}`
	}
	ruleRowFor := func(name, scope, ruleTenant string, active bool) []interface{} {
		return []interface{}{"00000000-0000-0000-0000-00000000000" + name[len(name)-1:], name, "", []byte(ruleProps(scope)),
			[]byte(`{"rule_ast":` + failing + `}`), active, ruleTenant}
	}
	ruleRow := func(name, scope string) []interface{} { return ruleRowFor(name, scope, tenant, true) }
	const gold = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	expectGold := func(m sqlmock.Sqlmock) {
		m.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	}
	run := func(t *testing.T, rules [][]interface{}, expectBinding func(sqlmock.Sqlmock)) []ruleViolation {
		t.Helper()
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		rows := sqlmock.NewRows([]string{"id", "node_name", "description", "properties", "config", "is_active", "tenant_id"})
		for _, r := range rules {
			vals := make([]driver.Value, len(r))
			for i, x := range r {
				vals[i] = x
			}
			rows.AddRow(vals...)
		}
		expectGold(mock) // ListByBO
		mock.ExpectQuery(`FROM catalog_node n`).WillReturnRows(rows)
		expectBinding(mock)
		mock.ExpectQuery(`FROM business_object_fields bf`).
			WillReturnRows(sqlmock.NewRows([]string{"field_name", "node_name"}))
		s := &BusinessObjectService{db: sqlx.NewDb(db, "postgres")}
		bo := &models.BusinessObjectDefinition{ID: boID, Key: "party"}
		bo.DriverTableName = "/mdm/party"
		v, _ := s.evaluateAndEnforceRules(context.Background(), s.db, tenant, bo, map[string]interface{}{"id": "r1", "amount": 50})
		return v
	}
	names := func(vs []ruleViolation) map[string]bool {
		m := map[string]bool{}
		for _, v := range vs {
			m[v.RuleName] = v.RuleError
		}
		return m
	}
	bindingIs := func(id string) func(sqlmock.Sqlmock) {
		return func(m sqlmock.Sqlmock) {
			expectGold(m) // ResolveActiveBinding
			m.ExpectQuery(`FROM public.business_object_binding b`).
				WillReturnRows(sqlmock.NewRows([]string{"id", "driving_path"}).AddRow(id, "/mdm/party"))
		}
	}
	noBinding := func(m sqlmock.Sqlmock) {
		expectGold(m) // ResolveActiveBinding
		m.ExpectQuery(`FROM public.business_object_binding b`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "driving_path"}))
	}
	noBindingLookup := func(sqlmock.Sqlmock) {}

	t.Run("unscoped rules never look up a binding", func(t *testing.T) {
		v := run(t, [][]interface{}{ruleRow("rule-1", "")}, noBindingLookup)
		// amount=50 is not > 100, so the rule ran and failed
		if _, ok := names(v)["rule-1"]; !ok || len(v) != 1 {
			t.Fatalf("violations = %+v; want the unscoped rule to run", v)
		}
	})
	t.Run("scoped rule runs on its binding and is skipped on another", func(t *testing.T) {
		rules := [][]interface{}{
			ruleRow("rule-1", `,"binding_ids":["b-mdm"]`),
			ruleRow("rule-2", `,"binding_ids":["b-other"]`),
		}
		v := names(run(t, rules, bindingIs("b-mdm")))
		if _, ok := v["rule-1"]; !ok {
			t.Errorf("rule scoped to the active binding did not run: %v", v)
		}
		if _, ok := v["rule-2"]; ok {
			t.Errorf("rule scoped to another binding ran: %v", v)
		}
	})
	t.Run("scoped rule with no determinable binding is reported, not skipped", func(t *testing.T) {
		v := run(t, [][]interface{}{ruleRow("rule-1", `,"binding_ids":["b-mdm"]`)}, noBinding)
		if len(v) != 1 || !v[0].RuleError {
			t.Fatalf("violations = %+v; want one rule_error", v)
		}
		if b, _ := json.Marshal(v[0].Message); len(b) == 0 {
			t.Error("empty message")
		}
	})
	t.Run("an inactive rule is never enforced, core or custom", func(t *testing.T) {
		rules := [][]interface{}{
			ruleRowFor("rule-1", "", tenant, false), // custom, switched off
			ruleRowFor("rule-2", "", gold, false),   // core, retired by the gold-copy tenant
			ruleRowFor("rule-3", "", tenant, true),  // custom, on
		}
		v := names(run(t, rules, noBindingLookup))
		if _, ok := v["rule-1"]; ok {
			t.Errorf("switched-off custom rule was enforced: %v", v)
		}
		if _, ok := v["rule-2"]; ok {
			t.Errorf("retired core rule was enforced for a tenant: %v", v)
		}
		if _, ok := v["rule-3"]; !ok {
			t.Errorf("active rule did not run: %v", v)
		}
	})
	t.Run("a tenant is held to an inherited core rule as well as its own", func(t *testing.T) {
		rules := [][]interface{}{
			ruleRowFor("rule-1", "", gold, true),   // core, authored in the gold-copy tenant
			ruleRowFor("rule-2", "", tenant, true), // custom
		}
		v := names(run(t, rules, noBindingLookup))
		if _, ok := v["rule-1"]; !ok {
			t.Errorf("inherited core rule did not run for the tenant: %v", v)
		}
		if _, ok := v["rule-2"]; !ok {
			t.Errorf("tenant's own rule did not run: %v", v)
		}
	})
}
