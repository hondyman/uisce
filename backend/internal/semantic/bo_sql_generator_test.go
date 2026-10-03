package semantic

import (
	"testing"
)

func TestGenerateSQL(t *testing.T) {
	domain := &Domain{
		ID:   "tenant-alpha",
		Name: "Financials",
		Entities: map[string]*Entity{
			"ent-1": {
				ID:             "ent-1",
				Name:           "Customer",
				PhysicalSchema: "analytics",
				PhysicalTable:  "customers",
				Attributes: map[string]*Attribute{
					"Region": {
						ID:             "attr-1",
						EntityID:       "ent-1",
						Name:           "Region",
						Type:           Dimension,
						PhysicalColumn: "region",
					},
				},
				Edges: []*Edge{
					{
						TargetEntityID: "ent-2",
						JoinType:       "INNER",
						JoinCondition:  "Customer.id = Order.customer_id",
					},
				},
			},
			"ent-2": {
				ID:             "ent-2",
				Name:           "Order",
				PhysicalSchema: "analytics",
				PhysicalTable:  "orders",
				Attributes: map[string]*Attribute{
					"TotalRevenue": {
						ID:             "attr-2",
						EntityID:       "ent-2",
						Name:           "TotalRevenue",
						Type:           Measure,
						PhysicalColumn: "total_amount",
						AggFunction:    "SUM",
					},
					"Status": {
						ID:             "attr-3",
						EntityID:       "ent-2",
						Name:           "Status",
						Type:           Dimension,
						PhysicalColumn: "status",
					},
				},
			},
		},
	}

	ctx := GeneratorContext{
		TenantKey: "tenant-alpha",
		Dialect:   "DATAFUSION",
		Graph:     domain,
	}

	req := SemanticRequest{
		Domain:     "Financials",
		Dimensions: []string{"Customer.Region"},
		Measures:   []string{"Order.TotalRevenue"},
		Filters: []SemanticFilter{
			{
				Attribute: "Order.Status",
				Operator:  "=",
				Value:     "COMPLETED",
			},
		},
	}

	sqlStr, err := ctx.GenerateSQL(req)
	if err != nil {
		t.Fatalf("GenerateSQL failed: %v", err)
	}

	t.Logf("Generated SQL:\n%s", sqlStr)

	if !testing.Short() && (sqlStr == "") {
		t.Errorf("Expected non-empty SQL string")
	}
}

func TestResolveCalculatedMeasures(t *testing.T) {
	domain := &Domain{
		ID:   "tenant-alpha",
		Name: "Financials",
		Entities: map[string]*Entity{
			"ent-detail": {
				ID:             "ent-detail",
				Name:           "OrderDetail",
				PhysicalSchema: "analytics",
				PhysicalTable:  "order_details",
				Attributes: map[string]*Attribute{
					"GrossRevenue": {
						ID:             "attr-1",
						EntityID:       "ent-detail",
						Name:           "GrossRevenue",
						Type:           Measure,
						PhysicalColumn: "unit_price * quantity",
						AggFunction:    "SUM",
					},
					"TotalCost": {
						ID:             "attr-2",
						EntityID:       "ent-detail",
						Name:           "TotalCost",
						Type:           Measure,
						PhysicalColumn: "unit_cost * quantity",
						AggFunction:    "SUM",
					},
					"GrossProfit": {
						ID:          "attr-3",
						EntityID:    "ent-detail",
						Name:        "GrossProfit",
						Type:        CalculatedMeasure,
						Expression:  "${OrderDetail.GrossRevenue} - ${OrderDetail.TotalCost}",
						Format:      "currency",
						Description: "Gross profit",
					},
					"ProfitMargin": {
						ID:          "attr-4",
						EntityID:    "ent-detail",
						Name:        "ProfitMargin",
						Type:        CalculatedMeasure,
						Expression:  "${OrderDetail.GrossProfit} / NULLIF(${OrderDetail.GrossRevenue}, 0)",
						Format:      "percentage",
						Description: "Profit margin",
					},
				},
			},
		},
	}

	ctx := GeneratorContext{
		TenantKey: "tenant-alpha",
		Dialect:   "DATAFUSION",
		Graph:     domain,
	}

	_, exprs, err := ctx.ResolveCalculatedMeasures([]string{
		"OrderDetail.GrossRevenue",
		"OrderDetail.GrossProfit",
		"OrderDetail.ProfitMargin",
	})
	if err != nil {
		t.Fatalf("ResolveCalculatedMeasures failed: %v", err)
	}

	marginExpr, exists := exprs["OrderDetail.ProfitMargin"]
	if !exists {
		t.Fatalf("ProfitMargin compiled expression missing")
	}

	t.Logf("Compiled ProfitMargin Expression:\n%s", marginExpr)
}

// The base table and join root come from the first required entity, so the
// order must follow the request (dimensions first), not map iteration order.
func TestExtractRequiredEntitiesIsDeterministic(t *testing.T) {
	domain := &Domain{
		ID: "d", Name: "Financials",
		Entities: map[string]*Entity{
			"ent-1": {ID: "ent-1", Name: "Customer", Attributes: map[string]*Attribute{
				"Region": {ID: "a1", EntityID: "ent-1", Name: "Region", Type: Dimension, PhysicalColumn: "region"}}},
			"ent-2": {ID: "ent-2", Name: "Order", Attributes: map[string]*Attribute{
				"TotalRevenue": {ID: "a2", EntityID: "ent-2", Name: "TotalRevenue", Type: Measure, PhysicalColumn: "amt", AggFunction: "SUM"},
				"Status":       {ID: "a3", EntityID: "ent-2", Name: "Status", Type: Dimension, PhysicalColumn: "status"}}},
		},
	}
	ctx := GeneratorContext{Graph: domain}
	req := SemanticRequest{
		Dimensions: []string{"Customer.Region"},
		Measures:   []string{"Order.TotalRevenue"},
		Filters:    []SemanticFilter{{Attribute: "Order.Status", Operator: "=", Value: "X"}},
	}
	for i := 0; i < 200; i++ {
		got, err := ctx.extractRequiredEntities(req)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].ID != "ent-1" || got[1].ID != "ent-2" {
			t.Fatalf("iteration %d: want [ent-1 ent-2] (deduped, request order), got %v", i, ids(got))
		}
	}
}

func ids(es []*Entity) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.ID
	}
	return out
}
