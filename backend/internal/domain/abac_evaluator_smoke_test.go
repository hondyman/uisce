package domain

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// TestABACEvaluator_LiveNorthwindPolicies exercises the consolidated
// evaluator against the real, already-authored abac_policies rows for the
// northwind tenant (Sales Rep / Inventory Manager / Professional Services
// baselines) — a smoke test that the tolerant parser (action vs actions vs
// action_attribute, resources vs name, flat environment keys) actually
// reproduces the intended access boundaries, not just that it compiles.
// Skipped when DATABASE_URL isn't set (e.g. CI without DB access).
func TestABACEvaluator_LiveNorthwindPolicies(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow(`select count(*) from abac_policies where name like 'Global Baseline - Sales Rep%'`).Scan(&count)
	if err != nil || count == 0 {
		t.Skipf("northwind baseline policies not present in this DB (count=%d, err=%v)", count, err)
	}
	// These are global (tenant_id IS NULL) baseline policies, so any real
	// tenant id exercises them — arbitrary UUID stands in for "some tenant".
	tenantID := "99e99e99-99e9-49e9-89e9-99e99e99e999"

	ev := NewABACEvaluator(db)
	ctx := context.Background()

	cases := []struct {
		label string
		req   EvaluationRequest
		want  bool
	}{
		{"sales_rep read order", EvaluationRequest{TenantID: tenantID, AssetID: "order", Action: "read",
			Context: map[string]interface{}{"roles": []string{"northwind_sales_rep"}}}, true},
		{"sales_rep delete product", EvaluationRequest{TenantID: tenantID, AssetID: "product", Action: "delete",
			Context: map[string]interface{}{"roles": []string{"northwind_sales_rep"}}}, false},
		{"sales_rep create order", EvaluationRequest{TenantID: tenantID, AssetID: "order", Action: "create",
			Context: map[string]interface{}{"roles": []string{"northwind_sales_rep"}}}, true},
		{"inventory_manager read order", EvaluationRequest{TenantID: tenantID, AssetID: "order", Action: "read",
			Context: map[string]interface{}{"roles": []string{"northwind_inventory_manager"}}}, false},
		{"inventory_manager create product", EvaluationRequest{TenantID: tenantID, AssetID: "product", Action: "create",
			Context: map[string]interface{}{"roles": []string{"northwind_inventory_manager"}}}, true},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			allowed, reason, _, err := ev.Evaluate(ctx, c.req)
			if err != nil {
				t.Fatalf("Evaluate error: %v", err)
			}
			if allowed != c.want {
				t.Errorf("got allowed=%v (%s), want %v", allowed, reason, c.want)
			}
		})
	}
}
