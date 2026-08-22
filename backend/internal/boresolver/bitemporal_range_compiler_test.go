package boresolver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCompileRangeQuery_PureCold(t *testing.T) {
	compiler := NewBitemporalRangeCompiler()
	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")

	req := BitemporalRangeRequest{
		TenantID:           tenantID,
		EffectiveStartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveEndDate:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		WatermarkDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		KnowledgeDate:      time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
		ColdTableName:      "iceberg.t_99e99e99.portfolio_positions_archive",
		HotTableName:       "starrocks.t_99e99e99.portfolio_positions_realtime",
		TemporalColumn:     "trade_date",
		BusinessKeyColumns: []string{"account_id", "security_id"},
		SelectedColumns:    []string{"account_id", "security_id", "trade_date", "quantity", "base_cost"},
	}

	res, err := compiler.CompileRangeQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("expected successful compilation, got error: %v", err)
	}

	if res.Strategy != StrategyPureColdIceberg {
		t.Errorf("expected StrategyPureColdIceberg, got %s", res.Strategy)
	}
	if !strings.Contains(res.CompiledSQL, "FROM iceberg.t_99e99e99.portfolio_positions_archive") {
		t.Errorf("expected Iceberg cold table, got SQL:\n%s", res.CompiledSQL)
	}
	if strings.Contains(res.CompiledSQL, "starrocks") || strings.Contains(res.CompiledSQL, "UNION ALL") {
		t.Errorf("pure cold query should not contain StarRocks or UNION ALL, got:\n%s", res.CompiledSQL)
	}
	if !strings.Contains(res.CompiledSQL, "tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'") {
		t.Errorf("expected tenant_id injection, got:\n%s", res.CompiledSQL)
	}
}

func TestCompileRangeQuery_PureHot(t *testing.T) {
	compiler := NewBitemporalRangeCompiler()
	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")

	req := BitemporalRangeRequest{
		TenantID:           tenantID,
		EffectiveStartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EffectiveEndDate:   time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		WatermarkDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		KnowledgeDate:      time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
		ColdTableName:      "iceberg.t_99e99e99.portfolio_positions_archive",
		HotTableName:       "starrocks.t_99e99e99.portfolio_positions_realtime",
		TemporalColumn:     "trade_date",
		SelectedColumns:    []string{"account_id", "trade_date", "quantity"},
	}

	res, err := compiler.CompileRangeQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("expected successful compilation, got error: %v", err)
	}

	if res.Strategy != StrategyPureHotOLAP {
		t.Errorf("expected StrategyPureHotOLAP, got %s", res.Strategy)
	}
	if !strings.Contains(res.CompiledSQL, "FROM starrocks.t_99e99e99.portfolio_positions_realtime") {
		t.Errorf("expected StarRocks hot table, got SQL:\n%s", res.CompiledSQL)
	}
	if strings.Contains(res.CompiledSQL, "iceberg") || strings.Contains(res.CompiledSQL, "UNION ALL") {
		t.Errorf("pure hot query should not contain Iceberg or UNION ALL, got:\n%s", res.CompiledSQL)
	}
}

func TestCompileRangeQuery_SplitAndStitchWithDeduplication(t *testing.T) {
	compiler := NewBitemporalRangeCompiler()
	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")

	req := BitemporalRangeRequest{
		TenantID:           tenantID,
		EffectiveStartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveEndDate:   time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		WatermarkDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		KnowledgeDate:      time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
		ColdTableName:      "iceberg.t_99e99e99.portfolio_positions_archive",
		HotTableName:       "starrocks.t_99e99e99.portfolio_positions_realtime",
		TemporalColumn:     "trade_date",
		BusinessKeyColumns: []string{"account_id", "security_id"},
		SelectedColumns:    []string{"account_id", "security_id", "trade_date", "quantity"},
	}

	res, err := compiler.CompileRangeQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("expected successful compilation, got error: %v", err)
	}

	if res.Strategy != StrategySplitAndStitch {
		t.Errorf("expected StrategySplitAndStitch, got %s", res.Strategy)
	}

	// 1. Assert dual CTEs and source precedence tags
	if !strings.Contains(res.CompiledSQL, "2 AS source_precedence") {
		t.Errorf("expected cold source_precedence = 2, got:\n%s", res.CompiledSQL)
	}
	if !strings.Contains(res.CompiledSQL, "1 AS source_precedence") {
		t.Errorf("expected hot source_precedence = 1, got:\n%s", res.CompiledSQL)
	}

	// 2. Assert Late-Arriving Historical Mutations Recovery in Hot scan
	if !strings.Contains(res.CompiledSQL, "trade_date < '2026-01-01' AND system_valid_from >= '2026-01-01'") {
		t.Errorf("expected late backdated mutations clause in hot partition, got:\n%s", res.CompiledSQL)
	}

	// 3. Assert Deduplication window over entity identity (account_id, security_id) and trade_date
	if !strings.Contains(res.CompiledSQL, "PARTITION BY account_id, security_id, trade_date") {
		t.Errorf("expected deduplication partition by business key and date, got:\n%s", res.CompiledSQL)
	}
	if !strings.Contains(res.CompiledSQL, "ORDER BY system_valid_from DESC, source_precedence ASC") {
		t.Errorf("expected order by system_valid_from DESC, source_precedence ASC, got:\n%s", res.CompiledSQL)
	}
	if !strings.Contains(res.CompiledSQL, "WHERE __row_rank = 1") {
		t.Errorf("expected row_rank = 1 filter for deduplication, got:\n%s", res.CompiledSQL)
	}
}

func TestCompileRangeQuery_KnowledgeDateScoping(t *testing.T) {
	compiler := NewBitemporalRangeCompiler()
	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	knowledgeTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	req := BitemporalRangeRequest{
		TenantID:           tenantID,
		EffectiveStartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveEndDate:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		WatermarkDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		KnowledgeDate:      knowledgeTime,
		ColdTableName:      "iceberg.positions_archive",
		HotTableName:       "starrocks.positions_realtime",
	}

	res, err := compiler.CompileRangeQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedKStr := "2025-12-31T23:59:59Z"
	if !strings.Contains(res.CompiledSQL, "system_valid_from <= '"+expectedKStr+"'") {
		t.Errorf("expected system_valid_from <= '%s', got:\n%s", expectedKStr, res.CompiledSQL)
	}
	if !strings.Contains(res.CompiledSQL, "system_valid_to > '"+expectedKStr+"' OR system_valid_to IS NULL") {
		t.Errorf("expected system_valid_to condition, got:\n%s", res.CompiledSQL)
	}
}

func TestCompileRangeQuery_Rule7_TenantGuard(t *testing.T) {
	compiler := NewBitemporalRangeCompiler()

	req := BitemporalRangeRequest{
		TenantID:           uuid.Nil, // Violation of Rule 7
		EffectiveStartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveEndDate:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		WatermarkDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err := compiler.CompileRangeQuery(context.Background(), req)
	if err == nil {
		t.Fatalf("expected fatal error on Rule 7 tenant violation, got nil")
	}
	if !strings.Contains(err.Error(), "Rule 7 violation") {
		t.Errorf("expected error message mentioning Rule 7 violation, got: %v", err)
	}
}
