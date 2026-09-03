package analytics

import (
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

func newExecTestResolver() *boresolver.Resolver {
	r := boresolver.NewResolver("cashflows", "cashflows", boresolver.PostgresDialect{})
	r.AddMapping("cashflow_amount", "cashflows", "amount")
	r.AddMapping("cashflow_date", "cashflows", "cf_date")
	return r
}

func TestBuildCalcBaseQuery_NoTerms(t *testing.T) {
	sql, err := buildCalcBaseQuery(newExecTestResolver(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sql != "SELECT 1" {
		t.Errorf("expected 'SELECT 1', got %q", sql)
	}
}

func TestBuildCalcBaseQuery_SingleTableSelectsEveryTerm(t *testing.T) {
	sql, err := buildCalcBaseQuery(newExecTestResolver(), []string{"cashflow_amount", "cashflow_date"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, `"amount" AS "cashflow_amount"`) {
		t.Errorf("missing cashflow_amount projection: %s", sql)
	}
	if !strings.Contains(sql, `"cf_date" AS "cashflow_date"`) {
		t.Errorf("missing cashflow_date projection: %s", sql)
	}
	if !strings.Contains(sql, `FROM "cashflows"`) {
		t.Errorf("missing FROM clause: %s", sql)
	}
	if !strings.Contains(sql, `WHERE "tenant_id" = $1`) {
		t.Errorf("missing tenant scoping: %s", sql)
	}
}

func TestBuildCalcBaseQuery_CrossTableFailsLoudly(t *testing.T) {
	r := boresolver.NewResolver("cashflows", "cashflows", boresolver.PostgresDialect{})
	r.AddMapping("cashflow_amount", "cashflows", "amount")
	r.AddMapping("other_field", "other_table", "value")

	_, err := buildCalcBaseQuery(r, []string{"cashflow_amount", "other_field"})
	if err == nil {
		t.Fatal("expected an error for base fields spanning multiple tables")
	}
	// Resolver.ResolveTerm itself fails loudly here (no join path
	// registered to the non-driving table) before buildCalcBaseQuery's own
	// same-table check even runs — either way, this must never silently
	// drop the second table's field.
	if !strings.Contains(err.Error(), "other_table") && !strings.Contains(err.Error(), "multiple tables") {
		t.Errorf("expected a clear cross-table/join error, got: %v", err)
	}
}

func TestBuildCalcBaseQuery_UnresolvableTermFailsLoudly(t *testing.T) {
	_, err := buildCalcBaseQuery(newExecTestResolver(), []string{"nonexistent_term"})
	if err == nil {
		t.Fatal("expected an error for an unresolvable term")
	}
}
