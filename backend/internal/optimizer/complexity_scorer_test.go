package optimizer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/optimizer"
)

func TestComplexityScorer_LowComplexityQuery(t *testing.T) {
	ctx := context.Background()
	scorer := optimizer.NewComplexityScorer(nil)
	tenantID := uuid.New()

	ast := optimizer.QueryAST{
		DrivingEntity:    "SecurityMaster",
		SelectedFields:   []string{"security_name", "isin", "px_last"},
		JoinEntities:     []string{},
		HasDatePartition: true,
		HasEntityFilter:  true,
		CrossTierEngines: []string{"STARROCKS"},
		AggregationCount: 0,
		RawQuery:         "SELECT security_name, isin, px_last FROM security WHERE date >= '2026-08-01'",
	}

	res, err := scorer.AnalyzeQueryAST(ctx, tenantID, ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.CostBand != optimizer.CostBandLow {
		t.Errorf("expected LOW cost band, got: %s (Score: %d)", res.CostBand, res.ComplexityScore)
	}
	if !res.CanExecute {
		t.Errorf("expected query to be executable")
	}
}

func TestComplexityScorer_ForbiddenCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	scorer := optimizer.NewComplexityScorer(nil)
	tenantID := uuid.New()

	// High complexity: 4 joins, missing date partition, missing entity filter, cross-tier federation
	ast := optimizer.QueryAST{
		DrivingEntity:    "AccountMaster",
		SelectedFields:   []string{"account_id", "total_nav", "custodian_code"},
		JoinEntities:     []string{"Positions", "Transactions", "TaxLots", "LegalEntity"},
		HasDatePartition: false, // +30
		HasEntityFilter:  false, // +20
		CrossTierEngines: []string{"STARROCKS", "ICEBERG"}, // +40
		AggregationCount: 3,
		RawQuery:         "SELECT * FROM account a JOIN positions p ON ...",
	}

	res, err := scorer.AnalyzeQueryAST(ctx, tenantID, ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.CostBand != optimizer.CostBandForbidden {
		t.Errorf("expected FORBIDDEN cost band, got: %s (Score: %d)", res.CostBand, res.ComplexityScore)
	}
	if res.CanExecute {
		t.Errorf("expected circuit breaker to block query execution")
	}
	if len(res.Recommendations) < 3 {
		t.Errorf("expected multiple optimization proposals, got: %d", len(res.Recommendations))
	}
}
