package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// WealthManagementBundle represents the JSON structure
type WealthManagementBundle struct {
	BundleID string   `json:"bundle_id"`
	Domain   string   `json:"domain"`
	Audience string   `json:"audience"`
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
	// Read the bundle file
	bundleFile := "wealth_management_phase1_bundle.json"
	data, err := os.ReadFile(bundleFile)
	if err != nil {
		log.Fatalf("Failed to read bundle file: %v", err)
	}

	// Parse JSON
	var bundle WealthManagementBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	fmt.Printf("📦 Loading Wealth Management Bundle: %s\n", bundle.BundleID)
	fmt.Printf("📊 Domain: %s | Audience: %s | Metrics: %d\n\n", bundle.Domain, bundle.Audience, len(bundle.Metrics))

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

	// Create metrics registry table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS wealth_management.metrics_registry (
		node_id VARCHAR(255) PRIMARY KEY,
		category VARCHAR(100) NOT NULL,
		description TEXT,
		formula_type VARCHAR(50) NOT NULL,
		formula TEXT NOT NULL,
		arguments JSONB,
		governance_status VARCHAR(50) DEFAULT 'draft',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create registry table: %v", err)
	}

	// Insert metrics
	for i, metric := range bundle.Metrics {
		argsJSON, _ := json.Marshal(metric.FinancialCalc.Arguments)

		_, err = db.Exec(`
			INSERT INTO wealth_management.metrics_registry
			(node_id, category, description, formula_type, formula, arguments, governance_status)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (node_id) DO UPDATE SET
				category = EXCLUDED.category,
				description = EXCLUDED.description,
				formula_type = EXCLUDED.formula_type,
				formula = EXCLUDED.formula,
				arguments = EXCLUDED.arguments,
				governance_status = EXCLUDED.governance_status,
				updated_at = NOW()
		`, metric.NodeID, metric.Category, metric.Description,
			metric.FinancialCalc.Type, metric.FinancialCalc.Formula, argsJSON, metric.Governance.Status)

		if err != nil {
			log.Printf("Failed to insert metric %s: %v", metric.NodeID, err)
		} else {
			fmt.Printf("✅ Loaded metric %d/%d: %s (%s)\n", i+1, len(bundle.Metrics), metric.NodeID, metric.Governance.Status)
		}
	}

	// Verify loaded metrics
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM wealth_management.metrics_registry")
	if err != nil {
		log.Fatalf("Failed to count metrics: %v", err)
	}

	fmt.Printf("\n🎉 Successfully loaded %d wealth management metrics into semantic registry!\n", count)

	// Show summary by category
	rows, err := db.Query(`
		SELECT category, governance_status, COUNT(*) as count
		FROM wealth_management.metrics_registry
		GROUP BY category, governance_status
		ORDER BY category, governance_status
	`)
	if err != nil {
		log.Fatalf("Failed to query summary: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n📊 Registry Summary:")
	fmt.Println("Category | Status | Count")
	fmt.Println("---------|--------|------")
	for rows.Next() {
		var category, status string
		var count int
		rows.Scan(&category, &status, &count)
		fmt.Printf("%-8s | %-6s | %d\n", category, status, count)
	}

	fmt.Println("\n🚀 Next Steps:")
	fmt.Println("1. Review 'draft' metrics for governance approval")
	fmt.Println("2. Test calculations in your semantic layer")
	fmt.Println("3. Add preaggregation rules for high-usage metrics")
	fmt.Println("4. Integrate with frontend dashboards")
}
