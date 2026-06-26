package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/domain"
	"github.com/hondyman/semlayer/backend/internal/query"
)

// DemoSchemaProvider provides sample schema data for demonstration
type DemoSchemaProvider struct{}

func (d *DemoSchemaProvider) GetAssetSchema(assetID string) (domain.AssetSchema, error) {
	return domain.AssetSchema{
		ColumnsByScope: map[string][]string{
			"metrics":    {"avg_order_value", "total_orders", "certified_net_margin"},
			"dimensions": {"region", "customer_id", "order_date"},
		},
		DefaultFilters: []string{"tenant_id = $1"},
	}, nil
}

func (d *DemoSchemaProvider) GetTableSchema(assetID, tableName string) (domain.TableSchema, error) {
	return domain.TableSchema{
		Name: tableName,
		Columns: []domain.ColumnSchema{
			{Name: "avg_order_value", Type: "decimal", Description: "Average order value"},
			{Name: "total_orders", Type: "integer", Description: "Total number of orders"},
			{Name: "certified_net_margin", Type: "decimal", Description: "Certified net margin"},
			{Name: "region", Type: "varchar", Description: "Customer region"},
			{Name: "customer_id", Type: "varchar", Description: "Customer identifier"},
			{Name: "order_date", Type: "date", Description: "Order date"},
		},
		PrimaryKey:  "customer_id",
		ForeignKeys: []string{},
	}, nil
}

// DemoAuditLogger provides a simple console-based audit logger
type DemoAuditLogger struct{}

func (d *DemoAuditLogger) LogRewrite(ctx context.Context, result *query.RewriteResult) error {
	fmt.Printf("\n=== Query Rewrite Audit ===\n")
	fmt.Printf("ID: %s\n", result.RewriteID)
	fmt.Printf("Original: %s\n", result.OriginalQuery)
	fmt.Printf("Rewritten: %s\n", result.RewrittenQuery)
	fmt.Printf("Applied Rules: %d\n", len(result.AppliedRules))
	for i, rule := range result.AppliedRules {
		fmt.Printf("  %d. %s: %s\n", i+1, rule.RuleName, rule.Description)
	}
	fmt.Printf("Performance Tips: %d\n", len(result.PerformanceTips))
	for i, tip := range result.PerformanceTips {
		fmt.Printf("  %d. %s\n", i+1, tip)
	}
	fmt.Printf("Compliance Notes: %d\n", len(result.ComplianceNotes))
	for i, note := range result.ComplianceNotes {
		fmt.Printf("  %d. %s\n", i+1, note)
	}
	fmt.Printf("Suggestions: %d\n", len(result.Suggestions))
	for i, suggestion := range result.Suggestions {
		fmt.Printf("  %d. %s (confidence: %.1f)\n", i+1, suggestion.Description, suggestion.Confidence)
	}
	fmt.Println("========================")
	return nil
}

func main() {
	fmt.Println("🔧 Context-Aware, AI-Assisted Query Rewriting & Optimization Layer Demo")
	fmt.Println("====================================================================")

	// Initialize components
	schemaProvider := &DemoSchemaProvider{}
	auditLogger := &DemoAuditLogger{}
	engine := query.NewRewriteEngine(schemaProvider, auditLogger)

	// Example 1: Basic query with compliance restrictions
	fmt.Println("\n📝 Example 1: User restricted to metrics scope")
	originalQuery1 := "SELECT avg_order_value, net_margin, region FROM orders_view WHERE order_date >= '2025-01-01'"

	ctx1 := &query.RewriteContext{
		UserID:   "analyst123",
		TenantID: "acme_corp",
		AssetID:  "orders",
		Decision: domain.EvaluationDecision{
			Decision:      "partial",
			Reason:        "User restricted to certified metrics only",
			AllowedScopes: []string{"metrics"}, // No dimensions allowed
		},
		PruningHints: domain.PruningHints{
			Columns:    []string{"avg_order_value", "certified_net_margin"},
			RowFilters: []string{"tenant_id = $1"},
			BindArgs:   []any{"acme_corp"},
		},
		UserIntent: "Analyze order performance",
	}

	result1, err := engine.RewriteQuery(context.Background(), originalQuery1, ctx1)
	if err != nil {
		log.Printf("Rewrite failed: %v", err)
		return
	}

	fmt.Printf("✅ Rewritten Query: %s\n", result1.RewrittenQuery)

	// Example 2: Query without WHERE clause (performance optimization)
	fmt.Println("\n📝 Example 2: Query optimization suggestions")
	originalQuery2 := "SELECT * FROM orders_view ORDER BY order_date DESC"

	ctx2 := &query.RewriteContext{
		UserID:   "analyst456",
		TenantID: "tech_startup",
		AssetID:  "orders",
		Decision: domain.EvaluationDecision{
			Decision:      "allow",
			Reason:        "Full access granted",
			AllowedScopes: []string{"metrics", "dimensions"},
		},
		PruningHints: domain.PruningHints{
			RowFilters: []string{"tenant_id = $1"},
			BindArgs:   []any{"tech_startup"},
		},
		UserIntent: "Review recent orders",
	}

	result2, err := engine.RewriteQuery(context.Background(), originalQuery2, ctx2)
	if err != nil {
		log.Printf("Rewrite failed: %v", err)
		return
	}

	fmt.Printf("✅ Rewritten Query: %s\n", result2.RewrittenQuery)
	fmt.Printf("💡 Performance Tips: %d suggestions\n", len(result2.PerformanceTips))

	// Example 3: Simulation mode
	fmt.Println("\n📝 Example 3: Simulation mode (preview changes)")
	originalQuery3 := "SELECT region, SUM(total_orders) FROM orders_view GROUP BY region"

	ctx3 := &query.RewriteContext{
		UserID:   "manager789",
		TenantID: "retail_chain",
		AssetID:  "orders",
		Decision: domain.EvaluationDecision{
			Decision:      "allow",
			Reason:        "Manager has full access",
			AllowedScopes: []string{"metrics", "dimensions"},
		},
		PruningHints: domain.PruningHints{
			RowFilters: []string{"tenant_id = $1"},
			BindArgs:   []any{"retail_chain"},
		},
		UserIntent: "Regional performance analysis",
	}

	simulation, err := engine.SimulateRewrite(context.Background(), originalQuery3, ctx3)
	if err != nil {
		log.Printf("Simulation failed: %v", err)
		return
	}

	fmt.Printf("🔍 Simulated Query: %s\n", simulation.RewrittenQuery)
	fmt.Printf("🎯 Suggestions: %d AI-powered recommendations\n", len(simulation.Suggestions))

	fmt.Println("\n🎉 Demo completed! The Context-Aware Query Rewrite Engine is ready for production use.")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("• ✅ Column pruning based on access scopes")
	fmt.Println("• ✅ Row-level security filter injection")
	fmt.Println("• ✅ Performance optimization suggestions")
	fmt.Println("• ✅ Compliance auditing and logging")
	fmt.Println("• ✅ AI-powered query improvement recommendations")
	fmt.Println("• ✅ Simulation mode for previewing changes")
}
