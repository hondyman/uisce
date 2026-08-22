package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/hondyman/uisce/backend/internal/finops"
	"github.com/hondyman/uisce/backend/internal/optimizer"
)

func TestQueryExecutionAuditor_ZeroLeakageAndCostCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	scorer := optimizer.NewComplexityScorer(nil)
	interceptor := audit.NewAnalyticalAuditInterceptor(nil)
	alertService := finops.NewBudgetAlertService(nil)

	auditor := api.NewQueryExecutionAuditor(nil, scorer, interceptor, alertService)

	// Test 1: Standard Executable Query (Complexity Score <= 80)
	validReq := api.ExecutionContextRequest{
		TenantID:      tenantID,
		RequestID:     "req-test-001",
		UserID:        "user-trader-1",
		ExecutionType: "SEMANTIC_QUERY",
		CompiledSQL:   "SELECT isin, px_last FROM security WHERE date >= '2026-08-01' AND account_id = 'ACC_9921'",
		AST: optimizer.QueryAST{
			DrivingEntity:    "SecurityMaster",
			SelectedFields:   []string{"isin", "px_last"},
			HasDatePartition: true,
			HasEntityFilter:  true,
			CrossTierEngines: []string{"STARROCKS"},
			RawQuery:         "SELECT isin, px_last FROM security",
		},
		EngineType: "STARROCKS",
	}

	mockRunner := func(ctx context.Context, sql string) ([]map[string]interface{}, int64, int, error) {
		return []map[string]interface{}{
			{"isin": "US0378331005", "px_last": 185.50},
		}, 1024, 5, nil
	}

	resp, err := auditor.ExecuteAndAudit(ctx, validReq, mockRunner)
	if err != nil {
		t.Fatalf("unexpected query execution failure: %v", err)
	}

	if resp.RowCount != 1 || len(resp.Rows) != 1 {
		t.Fatalf("expected 1 row returned in result envelope, got %d", resp.RowCount)
	}

	// Test 2: Circuit Breaker Trip (Score > 80)
	forbiddenReq := api.ExecutionContextRequest{
		TenantID:      tenantID,
		RequestID:     "req-test-002",
		UserID:        "user-trader-2",
		ExecutionType: "SEMANTIC_QUERY",
		CompiledSQL:   "SELECT * FROM accounts a JOIN positions p ON ...",
		AST: optimizer.QueryAST{
			DrivingEntity:    "AccountMaster",
			SelectedFields:   []string{"account_id", "total_nav"},
			JoinEntities:     []string{"Positions", "Transactions", "TaxLots", "LegalEntity"},
			HasDatePartition: false, // +30
			HasEntityFilter:  false, // +20
			CrossTierEngines: []string{"STARROCKS", "ICEBERG"}, // +40
			AggregationCount: 3,
			RawQuery:         "SELECT * FROM accounts",
		},
		EngineType: "STARROCKS",
	}

	_, forbiddenErr := auditor.ExecuteAndAudit(ctx, forbiddenReq, mockRunner)
	if forbiddenErr == nil {
		t.Fatalf("expected Cardinal Rule 8 cost circuit breaker to block query execution")
	}

	if !strings.Contains(forbiddenErr.Error(), "Rule 8 cost circuit breaker") {
		t.Errorf("error message mismatch: %v", forbiddenErr)
	}
}
