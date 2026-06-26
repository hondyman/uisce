package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// WealthManagementSuperBundle represents the JSON structure
type WealthManagementSuperBundle struct {
	BundleID string   `json:"bundle_id"`
	Domain   string   `json:"domain"`
	Audience []string `json:"audience"`
	Version  string   `json:"version"`
	Owner    string   `json:"owner"`
	Tags     []string `json:"tags"`
	Metrics  []Metric `json:"metrics"`
}

type Metric struct {
	NodeID        string        `json:"node_id"`
	Category      string        `json:"category"`
	Description   string        `json:"description"`
	FinancialCalc FinancialCalc `json:"financial_calc"`
	Governance    Governance    `json:"governance"`
}

type FinancialCalc struct {
	Type      string                 `json:"type"`
	Formula   string                 `json:"formula"`
	Arguments map[string]interface{} `json:"arguments"`
}

type Governance struct {
	Status string `json:"status"`
}

func main() {
	// Read the super bundle file
	bundleFile := "wealth_management_super_bundle.json"
	data, err := os.ReadFile(bundleFile)
	if err != nil {
		log.Fatalf("Failed to read super bundle file: %v", err)
	}

	// Parse JSON
	var bundle WealthManagementSuperBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	fmt.Printf("🚀 Loading Wealth Management Super Bundle: %s\n", bundle.BundleID)
	fmt.Printf("📊 Domain: %s | Version: %s | Metrics: %d\n", bundle.Domain, bundle.Version, len(bundle.Metrics))
	fmt.Printf("👥 Audience: %v\n", bundle.Audience)
	fmt.Printf("🏷️  Tags: %v\n\n", bundle.Tags)

	// Connect to alpha database (semantic layer)
	dsn := "postgres://postgres@localhost:5432/alpha?sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Connected to semantic layer database")

	// Create wealth_management schema if it doesn't exist
	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS wealth_management")
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create metrics registry table (if not exists)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS wealth_management.metrics_registry (
		node_id VARCHAR(255) PRIMARY KEY,
		category VARCHAR(100) NOT NULL,
		description TEXT,
		formula_type VARCHAR(50) NOT NULL,
		formula TEXT NOT NULL,
		arguments JSONB,
		governance_status VARCHAR(50) DEFAULT 'draft',
		audience TEXT[],
		tags TEXT[],
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create registry table: %v", err)
	}

	// Insert/update metrics
	goldenCount := 0
	draftCount := 0
	for i, metric := range bundle.Metrics {
		argsJSON, _ := json.Marshal(metric.FinancialCalc.Arguments)

		_, err = db.Exec(`
			INSERT INTO wealth_management.metrics_registry
			(node_id, category, description, formula_type, formula, arguments, governance_status, audience, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (node_id) DO UPDATE SET
				category = EXCLUDED.category,
				description = EXCLUDED.description,
				formula_type = EXCLUDED.formula_type,
				formula = EXCLUDED.formula,
				arguments = EXCLUDED.arguments,
				governance_status = EXCLUDED.governance_status,
				audience = EXCLUDED.audience,
				tags = EXCLUDED.tags,
				updated_at = NOW()
		`, metric.NodeID, metric.Category, metric.Description,
			metric.FinancialCalc.Type, metric.FinancialCalc.Formula, argsJSON,
			metric.Governance.Status, pq.Array(bundle.Audience), pq.Array(bundle.Tags))

		if err != nil {
			log.Printf("Failed to insert metric %s: %v", metric.NodeID, err)
		} else {
			if metric.Governance.Status == "golden" {
				goldenCount++
			} else {
				draftCount++
			}
			fmt.Printf("✅ Loaded metric %d/%d: %s (%s - %s)\n",
				i+1, len(bundle.Metrics), metric.NodeID, metric.Governance.Status, metric.Category)
		}
	}

	// Verify loaded metrics
	var totalCount int
	err = db.Get(&totalCount, "SELECT COUNT(*) FROM wealth_management.metrics_registry")
	if err != nil {
		log.Fatalf("Failed to count metrics: %v", err)
	}

	fmt.Printf("\n🎉 Successfully loaded %d wealth management metrics into semantic registry!\n", totalCount)
	fmt.Printf("🏆 Golden Status: %d metrics\n", goldenCount)
	fmt.Printf("📝 Draft Status: %d metrics\n", draftCount)

	// Show breakdown by category
	rows, err := db.Query(`
		SELECT category, governance_status, COUNT(*) as count
		FROM wealth_management.metrics_registry
		GROUP BY category, governance_status
		ORDER BY category, governance_status
	`)
	if err != nil {
		log.Fatalf("Failed to query category breakdown: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n📊 Category Breakdown:")
	fmt.Println("Category | Status | Count")
	fmt.Println("---------|--------|------")
	for rows.Next() {
		var category, status string
		var count int
		rows.Scan(&category, &status, &count)
		fmt.Printf("%-15s | %-6s | %d\n", category, status, count)
	}

	// Show Phase 1 vs Phase 2
	var phase1Count, phase2Count int
	err = db.Get(&phase1Count, `
		SELECT COUNT(*) FROM wealth_management.metrics_registry
		WHERE node_id IN (
			'portfolio_allocation_pct', 'asset_class_exposure', 'portfolio_twr',
			'portfolio_mwr', 'annualized_return', 'portfolio_volatility',
			'sharpe_ratio', 'sortino_ratio', 'beta', 'alpha',
			'max_drawdown', 'dividend_yield'
		)
	`)
	if err == nil {
		phase2Count = totalCount - phase1Count
		fmt.Printf("\n📈 Phase Distribution:\n")
		fmt.Printf("Phase 1 (Core): %d metrics\n", phase1Count)
		fmt.Printf("Phase 2 (Advanced): %d metrics\n", phase2Count)
	}

	fmt.Println("\n🚀 Super Bundle Ready!")
	fmt.Println("Your wealth management semantic layer now supports:")
	fmt.Println("• Client-facing portfolio dashboards")
	fmt.Println("• Advisor performance analytics")
	fmt.Println("• Executive business intelligence")
	fmt.Println("• Risk management and compliance reporting")
}
