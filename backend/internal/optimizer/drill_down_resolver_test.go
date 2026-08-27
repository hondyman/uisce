package optimizer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/optimizer"
)

func TestDrillDownResolver_Rule7TenantFencing(t *testing.T) {
	resolver := optimizer.NewDrillDownResolver(nil)
	ctx := context.Background()

	// 1. Assert Failure on Nil TenantID (Rule 7 Security Mandate)
	_, err := resolver.ResolveDrillDown(ctx, optimizer.DrillDownRequest{
		TenantID:        uuid.Nil,
		AggregatedField: "portfolio_xirr",
		FilterContext:   map[string]interface{}{"account_id": "ACC-101"},
	})
	if err == nil {
		t.Fatalf("expected error on nil tenant_id, got nil")
	}

	// 2. Assert Valid Mock Disaggregation Output
	tenantID := uuid.New()
	res, err := resolver.ResolveDrillDown(ctx, optimizer.DrillDownRequest{
		TenantID:        tenantID,
		AggregatedField: "portfolio_xirr",
		FilterContext:   map[string]interface{}{"account_id": "ACC-101"},
		PageSize:        10,
	})
	if err != nil {
		t.Fatalf("unexpected error resolving drilldown: %v", err)
	}

	if res.TargetBOKey != "TaxLotCashFlows" {
		t.Errorf("expected TargetBOKey 'TaxLotCashFlows', got '%s'", res.TargetBOKey)
	}
	if len(res.Columns) == 0 {
		t.Errorf("expected columns in drill response, got 0")
	}
}
