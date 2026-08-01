package rules

import (
	"strings"
	"testing"
)

func TestAnalyzeFallbackPatterns_BelowThreshold(t *testing.T) {
	p, err := AnalyzeFallbackPatterns(FallbackQueryPattern{
		TargetBOID:    "bo-1",
		TargetTable:   "ibor_positions",
		GroupByFields: []string{"portfolio_id"},
		MeasureFields: []string{"market_value"},
		FallbackCount: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Errorf("expected nil proposal below threshold, got %+v", p)
	}
}

func TestAnalyzeFallbackPatterns_AboveThreshold(t *testing.T) {
	p, err := AnalyzeFallbackPatterns(FallbackQueryPattern{
		TargetBOID:    "bo-portfolio",
		TargetTable:   "ibor_positions",
		GroupByFields: []string{"portfolio_id", "issuer_id"},
		MeasureFields: []string{"market_value", "position_weight_pct"},
		FallbackCount: 5000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil proposal above threshold")
	}
	if p.TargetBOID != "bo-portfolio" {
		t.Errorf("expected target_bo_id bo-portfolio, got %q", p.TargetBOID)
	}
	for _, must := range []string{
		"CREATE MATERIALIZED VIEW public.mv_auto_ibor_positions_portfolio_id_issuer_id",
		"BUILD ASYNCHRONOUS",
		"REFRESH DEFERRED MANUAL",
		"DISTRIBUTED BY HASH(portfolio_id)",
		"SUM(market_value) AS sum_market_value",
		"SUM(position_weight_pct) AS sum_position_weight_pct",
		"FROM public.ibor_positions",
		"GROUP BY tenant_id, portfolio_id, issuer_id",
	} {
		if !strings.Contains(p.SuggestedMV, must) {
			t.Errorf("expected DDL to contain %q\nfull DDL:\n%s", must, p.SuggestedMV)
		}
	}
	if !strings.Contains(p.EstimatedGain, "fallback") {
		t.Errorf("expected gain description, got %q", p.EstimatedGain)
	}
}

func TestAnalyzeFallbackPatterns_MissingTable(t *testing.T) {
	_, err := AnalyzeFallbackPatterns(FallbackQueryPattern{
		GroupByFields: []string{"x"},
		MeasureFields: []string{"y"},
		FallbackCount: 9999,
	})
	if err == nil {
		t.Error("expected error when target_table is missing")
	}
}

func TestAnalyzeFallbackPatterns_MissingGroupBy(t *testing.T) {
	_, err := AnalyzeFallbackPatterns(FallbackQueryPattern{
		TargetTable:   "t",
		MeasureFields: []string{"y"},
		FallbackCount: 9999,
	})
	if err == nil {
		t.Error("expected error when group_by_fields is empty")
	}
}

func TestAnalyzeFallbackPatterns_MissingMeasures(t *testing.T) {
	_, err := AnalyzeFallbackPatterns(FallbackQueryPattern{
		TargetTable:   "t",
		GroupByFields: []string{"x"},
		FallbackCount: 9999,
	})
	if err == nil {
		t.Error("expected error when measure_fields is empty")
	}
}

func TestAnalyzeFallbackPatterns_ExactThreshold(t *testing.T) {
	p, err := AnalyzeFallbackPatterns(FallbackQueryPattern{
		TargetTable:   "t",
		GroupByFields: []string{"g"},
		MeasureFields: []string{"m"},
		FallbackCount: minFallbackForProposal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Error("expected proposal at exact threshold")
	}
}
