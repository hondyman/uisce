package query

import (
	"context"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/domain"
)

// MockSchemaProvider provides a simple in-memory schema for testing
type MockSchemaProvider struct {
	schemas map[string]domain.AssetSchema
}

func NewMockSchemaProvider() *MockSchemaProvider {
	return &MockSchemaProvider{
		schemas: map[string]domain.AssetSchema{
			"orders": {
				ColumnsByScope: map[string][]string{
					"metrics":    {"avg_order_value", "total_orders"},
					"dimensions": {"region", "customer_id", "order_date"},
				},
				DefaultFilters: []string{"tenant_id = $1"},
			},
		},
	}
}

func (m *MockSchemaProvider) GetAssetSchema(assetID string) (domain.AssetSchema, error) {
	if schema, ok := m.schemas[assetID]; ok {
		return schema, nil
	}
	return domain.AssetSchema{}, nil
}

func (m *MockSchemaProvider) GetTableSchema(assetID, tableName string) (domain.TableSchema, error) {
	return domain.TableSchema{
		Name: tableName,
		Columns: []domain.ColumnSchema{
			{Name: "avg_order_value", Type: "decimal", Description: "Average order value"},
			{Name: "total_orders", Type: "integer", Description: "Total number of orders"},
			{Name: "region", Type: "varchar", Description: "Customer region"},
			{Name: "customer_id", Type: "varchar", Description: "Customer identifier"},
			{Name: "order_date", Type: "date", Description: "Order date"},
		},
		PrimaryKey:  "customer_id",
		ForeignKeys: []string{},
	}, nil
}

// MockAuditLogger provides a simple audit logger for testing
type MockAuditLogger struct {
	logs []RewriteResult
}

func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{
		logs: make([]RewriteResult, 0),
	}
}

func (m *MockAuditLogger) LogRewrite(ctx context.Context, result *RewriteResult) error {
	m.logs = append(m.logs, *result)
	return nil
}

func TestRewriteEngine_RewriteQuery(t *testing.T) {
	schemaProvider := NewMockSchemaProvider()
	auditLogger := NewMockAuditLogger()
	engine := NewRewriteEngine(schemaProvider, auditLogger)

	// Test basic query rewriting
	originalQuery := "SELECT * FROM orders_view WHERE order_date >= '2025-01-01'"
	ctx := &RewriteContext{
		UserID:   "user123",
		TenantID: "acme",
		AssetID:  "orders",
		Decision: domain.EvaluationDecision{
			Decision:      "partial",
			Reason:        "restricted to metrics scope",
			AllowedScopes: []string{"metrics"},
		},
		PruningHints: domain.PruningHints{
			Columns:    []string{"avg_order_value", "total_orders"},
			RowFilters: []string{"tenant_id = $1"},
			BindArgs:   []any{"acme"},
		},
	}

	result, err := engine.RewriteQuery(context.Background(), originalQuery, ctx)
	if err != nil {
		t.Fatalf("RewriteQuery failed: %v", err)
	}

	if result.OriginalQuery != originalQuery {
		t.Errorf("Original query mismatch: got %s, want %s", result.OriginalQuery, originalQuery)
	}

	if result.RewrittenQuery == "" {
		t.Error("Rewritten query is empty")
	}

	if len(result.AppliedRules) == 0 {
		t.Error("Expected at least one applied rule")
	}

	// Check that tenant filter was injected
	if !contains(result.RewrittenQuery, "tenant_id = 'acme'") {
		t.Error("Tenant filter not injected")
	}
}

func TestRewriteEngine_SimulateRewrite(t *testing.T) {
	schemaProvider := NewMockSchemaProvider()
	auditLogger := NewMockAuditLogger()
	engine := NewRewriteEngine(schemaProvider, auditLogger)

	originalQuery := "SELECT avg_order_value, region FROM orders_view"
	ctx := &RewriteContext{
		UserID:   "user123",
		TenantID: "acme",
		AssetID:  "orders",
		Decision: domain.EvaluationDecision{
			Decision:      "allow",
			Reason:        "full access granted",
			AllowedScopes: []string{"metrics", "dimensions"},
		},
		PruningHints: domain.PruningHints{
			RowFilters: []string{"tenant_id = $1"},
			BindArgs:   []any{"acme"},
		},
	}

	result, err := engine.SimulateRewrite(context.Background(), originalQuery, ctx)
	if err != nil {
		t.Fatalf("SimulateRewrite failed: %v", err)
	}

	if result.RewriteID == "" {
		t.Error("Simulation ID is empty")
	}

	if !contains(result.RewriteID, "simulation-") {
		t.Error("Expected simulation prefix in ID")
	}
}

func TestRewriteEngine_RemoveDisallowedColumns(t *testing.T) {
	schemaProvider := NewMockSchemaProvider()
	auditLogger := NewMockAuditLogger()
	engine := NewRewriteEngine(schemaProvider, auditLogger)

	ctx := &RewriteContext{
		RewrittenQuery: "SELECT avg_order_value, net_margin, region FROM orders_view",
		Decision: domain.EvaluationDecision{
			Decision:      "partial",
			AllowedScopes: []string{"metrics"},
		},
		AssetID: "orders",
	}

	err := engine.removeDisallowedColumns(ctx)
	if err != nil {
		t.Fatalf("removeDisallowedColumns failed: %v", err)
	}

	// Should only contain allowed columns (avg_order_value from metrics scope)
	if !contains(ctx.RewrittenQuery, "avg_order_value") {
		t.Error("Allowed column not preserved")
	}

	if contains(ctx.RewrittenQuery, "net_margin") {
		t.Error("Disallowed column not removed")
	}

	if contains(ctx.RewrittenQuery, "region") {
		t.Error("Disallowed column not removed")
	}
}

func TestRewriteEngine_InjectTenantFilter(t *testing.T) {
	schemaProvider := NewMockSchemaProvider()
	auditLogger := NewMockAuditLogger()
	engine := NewRewriteEngine(schemaProvider, auditLogger)

	ctx := &RewriteContext{
		RewrittenQuery: "SELECT * FROM orders_view",
		TenantID:       "test-tenant",
	}

	err := engine.injectTenantFilter(ctx)
	if err != nil {
		t.Fatalf("injectTenantFilter failed: %v", err)
	}

	expected := "SELECT * FROM orders_view WHERE tenant_id = 'test-tenant'"
	if ctx.RewrittenQuery != expected {
		t.Errorf("Expected %s, got %s", expected, ctx.RewrittenQuery)
	}
}

func TestRewriteEngine_InjectTenantFilter_WithExistingWhere(t *testing.T) {
	schemaProvider := NewMockSchemaProvider()
	auditLogger := NewMockAuditLogger()
	engine := NewRewriteEngine(schemaProvider, auditLogger)

	ctx := &RewriteContext{
		RewrittenQuery: "SELECT * FROM orders_view WHERE order_date >= '2025-01-01'",
		TenantID:       "test-tenant",
	}

	err := engine.injectTenantFilter(ctx)
	if err != nil {
		t.Fatalf("injectTenantFilter failed: %v", err)
	}

	expected := "SELECT * FROM orders_view WHERE tenant_id = 'test-tenant' AND order_date >= '2025-01-01'"
	if ctx.RewrittenQuery != expected {
		t.Errorf("Expected %s, got %s", expected, ctx.RewrittenQuery)
	}
}

func TestRewriteEngine_GeneratePerformanceTips(t *testing.T) {
	schemaProvider := NewMockSchemaProvider()
	auditLogger := NewMockAuditLogger()
	engine := NewRewriteEngine(schemaProvider, auditLogger)

	ctx := &RewriteContext{
		RewrittenQuery: "SELECT * FROM orders_view", // No WHERE clause
		PruningHints: domain.PruningHints{
			Columns: []string{"col1", "col2", "col3", "col4", "col5", "col6", "col7", "col8", "col9", "col10", "col11"}, // Many columns
		},
	}

	tips := engine.generatePerformanceTips(ctx)

	if len(tips) == 0 {
		t.Error("Expected performance tips for query without WHERE and many columns")
	}

	foundWhereTip := false
	foundSelectTip := false
	for _, tip := range tips {
		if contains(tip, "WHERE") {
			foundWhereTip = true
		}
		if contains(tip, "SELECT *") {
			foundSelectTip = true
		}
	}

	if !foundWhereTip {
		t.Error("Expected tip about adding WHERE clause")
	}

	if !foundSelectTip {
		t.Error("Expected tip about SELECT *")
	}
}

func TestRewriteEngine_GenerateComplianceNotes(t *testing.T) {
	schemaProvider := NewMockSchemaProvider()
	auditLogger := NewMockAuditLogger()
	engine := NewRewriteEngine(schemaProvider, auditLogger)

	ctx := &RewriteContext{
		Decision: domain.EvaluationDecision{
			Decision:      "partial",
			AllowedScopes: []string{"metrics", "dimensions"},
		},
		PruningHints: domain.PruningHints{
			RowFilters: []string{"tenant_id = $1"},
		},
		TenantID: "test-tenant",
	}

	notes := engine.generateComplianceNotes(ctx)

	if len(notes) == 0 {
		t.Error("Expected compliance notes")
	}

	foundPartial := false
	foundTenant := false
	foundRowFilters := false

	for _, note := range notes {
		if contains(note, "partial") || contains(note, "restricted") {
			foundPartial = true
		}
		if contains(note, "tenant") {
			foundTenant = true
		}
		if contains(note, "row-level") || contains(note, "filter") {
			foundRowFilters = true
		}
	}

	if !foundPartial {
		t.Error("Expected note about partial access")
	}

	if !foundTenant {
		t.Error("Expected note about tenant isolation")
	}

	if !foundRowFilters {
		t.Error("Expected note about row filters")
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains check
	return len(s) >= len(substr) &&
		(s[:len(substr)] == substr ||
			containsIgnoreCase(s[1:], substr))
}
