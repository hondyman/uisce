package analytics

import (
	"strings"
	"testing"
)

func TestRenderTrinoIcebergRollup(t *testing.T) {
	renderer, err := NewPreAggTemplateRenderer()
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	data := PreAggTemplateData{
		Tenant:     "acme",
		Datasource: "orders",
		PreAggID:   "country_day",
		GroupBy:    []string{"country", "date(created_at)"},
		Measures: []MeasureDef{
			{Expression: "COUNT(*)", Alias: "order_count"},
			{Expression: "SUM(revenue)", Alias: "total_revenue"},
		},
		Filters: nil,
	}

	sql, err := renderer.RenderTrinoIcebergRollup(data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Verify expected components
	if !strings.Contains(sql, "iceberg.acme_analytics.agg_orders__country_day") {
		t.Error("Expected table name with tenant and datasource")
	}
	if !strings.Contains(sql, "country") {
		t.Error("Expected group by column 'country'")
	}
	if !strings.Contains(sql, "COUNT(*) AS order_count") {
		t.Error("Expected measure COUNT(*) AS order_count")
	}
	if !strings.Contains(sql, "FROM iceberg.acme_analytics.fact_orders") {
		t.Error("Expected FROM clause with fact table")
	}

	t.Logf("Generated SQL:\n%s", sql)
}

func TestRenderTrinoIcebergRollupWithFilters(t *testing.T) {
	renderer, err := NewPreAggTemplateRenderer()
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	data := PreAggTemplateData{
		Tenant:     "acme",
		Datasource: "orders",
		PreAggID:   "us_orders",
		GroupBy:    []string{"state", "date(created_at)"},
		Measures: []MeasureDef{
			{Expression: "COUNT(*)", Alias: "order_count"},
		},
		Filters: []FilterDef{
			{Field: "country", Op: "=", Value: "'US'"},
			{Field: "status", Op: "=", Value: "'completed'"},
		},
	}

	sql, err := renderer.RenderTrinoIcebergRollup(data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	if !strings.Contains(sql, "WHERE") {
		t.Error("Expected WHERE clause for filters")
	}
	if !strings.Contains(sql, "country = 'US'") {
		t.Error("Expected filter country = 'US'")
	}
	if !strings.Contains(sql, "AND") {
		t.Error("Expected AND between filters")
	}

	t.Logf("Generated SQL:\n%s", sql)
}

func TestRenderStarRocksMV(t *testing.T) {
	renderer, err := NewPreAggTemplateRenderer()
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	data := PreAggTemplateData{
		Tenant:     "acme",
		Datasource: "orders",
		PreAggID:   "country_day",
		GroupBy:    []string{"country", "date(created_at)"},
		Measures: []MeasureDef{
			{Expression: "COUNT(*)", Alias: "order_count"},
			{Expression: "SUM(revenue)", Alias: "total_revenue"},
		},
	}

	sql, err := renderer.RenderStarRocksMV(data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	if !strings.Contains(sql, "CREATE MATERIALIZED VIEW mv_orders__country_day") {
		t.Error("Expected MV name with datasource and preagg ID")
	}
	if !strings.Contains(sql, "BUILD IMMEDIATE") {
		t.Error("Expected BUILD IMMEDIATE")
	}
	if !strings.Contains(sql, "REFRESH ASYNC") {
		t.Error("Expected REFRESH ASYNC")
	}
	if !strings.Contains(sql, "FROM fact_orders") {
		t.Error("Expected FROM fact_orders")
	}

	t.Logf("Generated SQL:\n%s", sql)
}

func TestRenderStarRocksRefresh(t *testing.T) {
	renderer, err := NewPreAggTemplateRenderer()
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	data := PreAggTemplateData{
		Datasource: "orders",
		PreAggID:   "country_day",
	}

	sql, err := renderer.RenderStarRocksRefresh(data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	expected := "REFRESH MATERIALIZED VIEW mv_orders__country_day"
	if sql != expected {
		t.Errorf("Expected %q, got %q", expected, sql)
	}
}

func TestRenderDropStatements(t *testing.T) {
	renderer, err := NewPreAggTemplateRenderer()
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	data := PreAggTemplateData{
		Tenant:     "acme",
		Datasource: "orders",
		PreAggID:   "country_day",
	}

	// Test StarRocks DROP
	srDrop, err := renderer.RenderStarRocksDrop(data)
	if err != nil {
		t.Fatalf("Failed to render StarRocks drop: %v", err)
	}
	if !strings.Contains(srDrop, "DROP MATERIALIZED VIEW IF EXISTS mv_orders__country_day") {
		t.Errorf("Unexpected StarRocks drop: %s", srDrop)
	}

	// Test Trino DROP
	trinoDrop, err := renderer.RenderTrinoDropRollup(data)
	if err != nil {
		t.Fatalf("Failed to render Trino drop: %v", err)
	}
	if !strings.Contains(trinoDrop, "DROP TABLE IF EXISTS iceberg.acme_analytics.agg_orders__country_day") {
		t.Errorf("Unexpected Trino drop: %s", trinoDrop)
	}
}
