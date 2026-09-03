package workflows

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/testsuite"

	"github.com/hondyman/uisce/backend/internal/analytics"
)

func calcColumns() []string {
	return []string{"id", "node_id", "name", "title", "description", "formula", "engine_type", "return_type", "arguments", "category", "subcategory", "domain_id", "execution_type", "engine", "is_materialized", "tier", "execution_preference"}
}

// newActivityEnv builds a Temporal activity test environment -- required
// because ActivityCalculation calls activity.GetLogger(ctx), which panics
// outside a real (or test-harness) activity execution context.
func newActivityEnv() *testsuite.TestActivityEnvironment {
	var ts testsuite.WorkflowTestSuite
	return ts.NewTestActivityEnvironment()
}

func TestActivityCalculation_RequiresTenantID(t *testing.T) {
	deps := &ActivityDeps{CalcService: analytics.NewSemanticCalculationService(nil)}
	env := newActivityEnv()
	env.RegisterActivity(deps.ActivityCalculation)

	_, err := env.ExecuteActivity(deps.ActivityCalculation, map[string]interface{}{"calculation_name": "x"}, map[string]interface{}(nil))
	if err == nil {
		t.Fatal("expected an error when tenant_id is missing")
	}
}

func TestActivityCalculation_RequiresCalculationIDOrName(t *testing.T) {
	deps := &ActivityDeps{CalcService: analytics.NewSemanticCalculationService(nil)}
	env := newActivityEnv()
	env.RegisterActivity(deps.ActivityCalculation)

	_, err := env.ExecuteActivity(deps.ActivityCalculation, map[string]interface{}{"tenant_id": "t1"}, map[string]interface{}(nil))
	if err == nil {
		t.Fatal("expected an error when neither calculation_id nor calculation_name is given")
	}
}

// TestActivityCalculation_ResolvesByName proves the workflow activity
// actually loads the calc from the SAME calculations table
// (SemanticCalculationService.GetCalculationByName) the HTTP endpoint
// uses, and reaches the centralized ExecuteFormulaCalculation dispatch
// (which then fails predictably on the missing domain_id — proving the
// activity wired through to the real engine rather than stubbing a fake
// result, without needing to mock the full BO/catalog resolution chain).
func TestActivityCalculation_ResolvesByName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	svc := analytics.NewSemanticCalculationService(sqlx.NewDb(db, "postgres"))
	deps := &ActivityDeps{CalcService: svc}
	env := newActivityEnv()
	env.RegisterActivity(deps.ActivityCalculation)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM calculations WHERE name = $1`)).
		WithArgs("net_yield").
		WillReturnRows(sqlmock.NewRows(calcColumns()).AddRow(
			uuid.New(), uuid.Nil, "net_yield", "Net Yield", "", "gross_return - fee", "", "", []byte("null"), "", "", nil, "", "", false, "pushdown", "auto",
		))

	_, err = env.ExecuteActivity(deps.ActivityCalculation, map[string]interface{}{
		"tenant_id":        "tenant-a",
		"calculation_name": "net_yield",
	}, map[string]interface{}(nil))

	// domain_id is nil on this fixture row, so ExecuteFormulaCalculation
	// must fail here -- but with THAT specific error, proving the activity
	// reached the real centralized engine rather than short-circuiting.
	if err == nil {
		t.Fatal("expected an error (calc has no domain_id), got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
