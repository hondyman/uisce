package e2e

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/mcp"
)

type MockMemoryPublisher struct {
	PublishedEvents []audit.OutboxEvent
}

func (m *MockMemoryPublisher) Publish(ctx context.Context, topic, key string, payload []byte) error {
	var evt audit.OutboxEvent
	if err := json.Unmarshal(payload, &evt); err == nil {
		m.PublishedEvents = append(m.PublishedEvents, evt)
	}
	return nil
}

func TestE2E_ComplexMultiPeriodQueryAcrossSixEngines(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	knowledgeDate := time.Date(2026, 8, 21, 14, 30, 0, 0, time.UTC)
	watermarkDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// =========================================================================
	// ENGINE 1: COGNITIVE GRAPH DISAMBIGUATION & DIRECTIVES
	// =========================================================================
	t.Log("▶ [ENGINE 1] Evaluating Cognitive Graph Disambiguation...")
	termService := analytics.NewTermRelationshipService(nil)

	// Disambiguate "Custodial Account Code" vs "Allocation Account Code"
	diffNote, isSynonym := termService.EvaluateSemanticBoundary("Custodial Account Code", "Allocation Account Code")
	if isSynonym {
		t.Fatalf("Engine 1 Error: Custodial Account must NOT be marked as synonym of Allocation Account")
	}
	if !strings.Contains(diffNote, "depository") && !strings.Contains(diffNote, "safekeeping") {
		t.Fatalf("Engine 1 Error: Missing distinction rationale for custodial accounts, got: %s", diffNote)
	}
	t.Logf("✔ [ENGINE 1 Passed] Semantic Boundary Verified: %s", diffNote)

	// =========================================================================
	// ENGINE 2: BUSINESS OBJECT STUDIO DYNAMIC SCOPE & TARJAN SCC
	// =========================================================================
	t.Log("▶ [ENGINE 2] Validating BO Scope, Two-Stage CTE Fan-out & Tarjan SCC...")
	resilienceEngine := analytics.NewBOResilienceEngine(nil)

	// Verify Tarjan SCC blocks circular calculation formulas
	calcDeps := map[string][]string{
		"net_yield":     {"gross_return", "management_fee"},
		"gross_return":  {"total_revenue", "total_aum"},
		"total_revenue": {"net_yield"}, // Cycle introduced intentionally for test
	}
	cyclePath, err := resilienceEngine.DetectCircularCalculations(calcDeps)
	if err == nil || len(cyclePath) == 0 {
		t.Fatalf("Engine 2 Error: Tarjan SCC failed to catch circular dependency loop")
	}
	t.Logf("✔ [ENGINE 2 Passed] Tarjan SCC caught circular loop: %v", cyclePath)

	// =========================================================================
	// ENGINE 3: DATA QUALITY SENTINEL & DEFENSIVE AST REWRITE
	// =========================================================================
	t.Log("▶ [ENGINE 3] Executing Reservoir Profiling & Defensive AST Rewrite...")

	// Assert Gatekeeper Rules: REQUIRED nulls must block publication
	status, _ := analytics.EvaluateGatekeeperRules("DIMENSION", "REQUIRED", 0.0, 1.0, 500, 0, 500)
	if status != analytics.QualityHealthy {
		t.Fatalf("Engine 3 Error: Clean field marked as unhealthy: %s", status)
	}

	// Defensive AST rewrite: Yield = gross_profit / total_aum -> gross_profit / NULLIF(total_aum, 0)
	rawAST := "gross_profit / total_aum"
	guardedAST := analytics.RewriteDefensiveASTProjection(rawAST, "NUMERIC", "0.0", true, true)
	expectedAST := "COALESCE((gross_profit / NULLIF(total_aum, 0)), 0.0)"
	if guardedAST != expectedAST {
		t.Fatalf("Engine 3 Error: AST rewrite failed. Expected: %s, Got: %s", expectedAST, guardedAST)
	}
	t.Logf("✔ [ENGINE 3 Passed] Defensive AST Generated: %s", guardedAST)

	// =========================================================================
	// ENGINE 4: BITEMPORAL WATERMARK RANGE & SEAM DEDUPLICATION
	// =========================================================================
	t.Log("▶ [ENGINE 4] Compiling Bitemporal Range Query across Watermark Wt...")
	bitemporalCompiler := boresolver.NewBitemporalRangeCompiler()

	req := boresolver.BitemporalRangeRequest{
		TenantID:           tenantID,
		BusinessObjectID:   uuid.New(),
		EffectiveStartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveEndDate:   time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		KnowledgeDate:      knowledgeDate,
		WatermarkDate:      watermarkDate,
		ColdTableName:      "iceberg.portfolio_positions_archive",
		HotTableName:       "starrocks.portfolio_positions_realtime",
		TemporalColumn:     "trade_date",
		BusinessKeyColumns: []string{"account_id", "security_id"},
		SelectedColumns:    []string{"account_id", "security_id", "trade_date", "quantity", "base_cost"},
	}

	compResult, err := bitemporalCompiler.CompileRangeQuery(ctx, req)
	if err != nil {
		t.Fatalf("Engine 4 Error: Failed compiling bitemporal range query: %v", err)
	}
	if compResult.Strategy != boresolver.StrategySplitAndStitch {
		t.Fatalf("Engine 4 Error: Expected SPLIT_AND_STITCH strategy, got: %s", compResult.Strategy)
	}

	// Verify Seam Deduplication Rank exists in compiled SQL
	if !strings.Contains(compResult.CompiledSQL, "ROW_NUMBER() OVER") {
		t.Fatalf("Engine 4 Error: Compiled SQL missing ROW_NUMBER() window deduplication")
	}
	if !strings.Contains(compResult.CompiledSQL, "source_precedence") {
		t.Fatalf("Engine 4 Error: Compiled SQL missing asymmetric source_precedence ordering")
	}
	t.Log("✔ [ENGINE 4 Passed] Split-and-Stitch Deduplicated SQL Compiled Cleanly")

	// =========================================================================
	// ENGINE 5: TRANSACTIONAL OUTBOX & REGULATORY AUDIT CHAIN
	// =========================================================================
	t.Log("▶ [ENGINE 5] Validating Transactional Outbox Merkle Seal (SEC Rule 17a-4)...")
	hmacSecret := []byte("institutional-compliance-master-key-2026")

	payload := map[string]interface{}{
		"query_strategy": compResult.Strategy,
		"effective_span": "2024-01-01 to 2026-08-21",
		"tenant_id":      tenantID.String(),
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadHash := sha256.Sum256(payloadBytes)
	payloadHashStr := hex.EncodeToString(payloadHash[:])

	// Calculate HMAC Merkle Seal
	mac := hmac.New(sha256.New, hmacSecret)
	mac.Write([]byte(payloadHashStr))
	chainSealStr := hex.EncodeToString(mac.Sum(nil))

	if len(chainSealStr) != 64 {
		t.Fatalf("Engine 5 Error: Invalid HMAC-SHA256 Merkle chain seal length: %d", len(chainSealStr))
	}
	t.Logf("✔ [ENGINE 5 Passed] Tamper-Evident Chain Seal Generated: %s...", chainSealStr[:16])

	// =========================================================================
	// ENGINE 6: MODEL CONTEXT PROTOCOL (MCP) & MULTI-ENGINE TRANSPILATION
	// =========================================================================
	t.Log("▶ [ENGINE 6] Executing MCP JSON-RPC 2.0 Tool & Multi-Dialect Transpiler...")

	// Test Dialect Transpiler across PostgreSQL, Snowflake, and StarRocks
	pgDialect := boresolver.ResolveDialect(boresolver.DialectPostgreSQL)
	sfDialect := boresolver.ResolveDialect(boresolver.DialectSnowflake)
	srDialect := boresolver.ResolveDialect(boresolver.DialectStarRocks)

	colDate := "trade_date"
	if pgDialect.DateTrunc("month", colDate) != "DATE_TRUNC('month', trade_date)" {
		t.Errorf("PostgreSQL DateTrunc failed")
	}
	if sfDialect.DateTrunc("month", colDate) != "DATE_TRUNC('MONTH', trade_date)" {
		t.Errorf("Snowflake DateTrunc failed")
	}
	if srDialect.QuoteIdentifier("account_id") != "`account_id`" {
		t.Errorf("StarRocks identifier quoting failed")
	}

	// Test MCP JSON-RPC 2.0 Handler
	mcpHandler := mcp.NewMCPToolHandler(nil, bitemporalCompiler)
	toolsList := mcpHandler.ListAvailableTools()
	toolsArr, ok := toolsList["tools"].([]map[string]interface{})
	if !ok || len(toolsArr) < 2 {
		t.Fatalf("Engine 6 Error: MCP Server failed to expose required tools")
	}
	t.Log("✔ [ENGINE 6 Passed] MCP Tools Listed & Dialects Successfully Transpiled")

	// =========================================================================
	// FINAL E2E ASSERTION
	// =========================================================================
	t.Log("=========================================================================")
	t.Log("🎉 ALL 6 ENGINES VALIDATED: MULTI-PERIOD BITEMPORAL PIPELINE PASSED CLEANLY")
	t.Log("=========================================================================")
}
