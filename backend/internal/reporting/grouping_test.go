package reporting_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestHierarchicalGroupingEngine_WeightedAverageRollup(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	engine := reporting.NewHierarchicalGroupingEngine()

	spec := reporting.GroupHierarchySpec{
		Levels: []reporting.GroupLevelDef{
			{
				LevelIndex:    0,
				GroupFieldKey: "asset_class",
				DisplayName:   "Asset Class",
				Rollups: []reporting.RollupFieldDef{
					{
						FieldKey:   "market_value",
						ResultKey:  "total_mv",
						Function:   reporting.AggSum,
						FormatMask: "$#,##0.00",
					},
					{
						FieldKey:       "coupon_rate",
						ResultKey:      "weighted_coupon",
						Function:       reporting.AggWeightedAvg,
						WeightFieldKey: "market_value",
						FormatMask:     "0.00%",
					},
				},
			},
		},
	}

	records := []map[string]interface{}{
		{"asset_class": "Fixed Income", "security_name": "Treasury Bond 2Y", "market_value": 1000000.0, "coupon_rate": 0.04},
		{"asset_class": "Fixed Income", "security_name": "Corporate Bond 5Y", "market_value": 3000000.0, "coupon_rate": 0.06},
		{"asset_class": "Equities", "security_name": "Tech Growth ETF", "market_value": 2000000.0, "coupon_rate": 0.01},
	}

	dataset, err := engine.BuildHierarchy(ctx, tenantID, spec, records)
	if err != nil {
		t.Fatalf("hierarchy construction failed: %v", err)
	}

	if len(dataset.RootNodes) != 2 {
		t.Fatalf("expected 2 root groups (Fixed Income, Equities), got %d", len(dataset.RootNodes))
	}

	var fiNode *reporting.GroupNode
	for _, n := range dataset.RootNodes {
		if n.GroupValue == "Fixed Income" {
			fiNode = n
			break
		}
	}

	if fiNode == nil {
		t.Fatalf("Fixed Income node not found")
	}

	if fiNode.Aggregations["total_mv"] != 4000000.0 {
		t.Errorf("expected Fixed Income total_mv = 4,000,000, got %f", fiNode.Aggregations["total_mv"])
	}

	expectedWeightedCoupon := 0.055
	if fiNode.Aggregations["weighted_coupon"] != expectedWeightedCoupon {
		t.Errorf("expected weighted coupon = %f, got %f", expectedWeightedCoupon, fiNode.Aggregations["weighted_coupon"])
	}
}
