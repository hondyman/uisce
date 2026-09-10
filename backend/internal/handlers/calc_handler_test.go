package handlers

import "testing"

// validateCalcSQLExpr closes a live, mounted SQL injection at
// POST /calc/preview (CalcHandler.Preview interpolates sql_expr
// directly into "SELECT %s as result LIMIT %d" via fmt.Sprintf) -
// confirmed exploitable this session, not dead code. This is the
// allowlist proof: real attack shapes rejected, real legitimate
// expressions (what CalcFieldModal.tsx actually sends) still allowed.
func TestValidateCalcSQLExpr_BlocksInjection(t *testing.T) {
	attacks := []string{
		"1); DROP TABLE users; --",
		"(SELECT password_hash FROM users LIMIT 1)",
		"1 UNION SELECT tenant_id FROM business_objects",
		"pg_sleep(10)",
		"1; COPY (SELECT * FROM users) TO '/tmp/x'",
		"(SELECT * FROM information_schema.tables)",
	}
	for _, a := range attacks {
		if err := validateCalcSQLExpr(a); err == nil {
			t.Errorf("expected %q to be rejected, was allowed", a)
		}
	}
}

func TestValidateCalcSQLExpr_AllowsLegitimateExpressions(t *testing.T) {
	legit := []string{
		"field1 + field2",
		"SUM(amount)",
		"(price * quantity) / 100",
		"AVG(exec_price)",
		"field1 - field2 + 3.14",
	}
	for _, e := range legit {
		if err := validateCalcSQLExpr(e); err != nil {
			t.Errorf("expected %q to be allowed, got error: %v", e, err)
		}
	}
}
