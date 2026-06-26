package render

import (
	"testing"
)

func TestGeneratePreAggregationDDL(t *testing.T) {
	svc := NewService("", "", "", nil)

	pre := PreAggregation{
		Name:       "sales_rollup",
		Type:       "rollup",
		Measures:   []string{"sales"},
		Dimensions: []string{"store_id", "product_id"},
		Storage:    "materialized_view",
	}

	sql, err := svc.GeneratePreAggregationDDL("nested_agg_sales", pre)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "CREATE MATERIALIZED VIEW IF NOT EXISTS nested_agg_sales__sales_rollup AS SELECT store_id, product_id, SUM(sales) AS sales FROM nested_agg_sales GROUP BY store_id, product_id;"
	if sql != expected {
		t.Fatalf("generated SQL mismatch:\nexpected: %s\nactual:   %s", expected, sql)
	}
}
