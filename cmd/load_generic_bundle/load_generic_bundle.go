package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hondyman/semlayer/pkg/types"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <bundle_file.json>", os.Args[0])
	}

	bundleFile := os.Args[1]

	// Read the bundle file
	data, err := os.ReadFile(bundleFile)
	if err != nil {
		log.Fatalf("Failed to read bundle file: %v", err)
	}

	// Parse JSON
	var bundle types.GenericBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	fmt.Printf("🚀 Loading Bundle: %s\n", bundle.BundleID)
	fmt.Printf("📊 Domain: %s | Version: %s | Metrics: %d\n", bundle.Domain, bundle.Version, len(bundle.Metrics))
	fmt.Printf("👥 Audience: %v\n", bundle.Audience)
	fmt.Printf("🏷️  Tags: %v\n", bundle.Tags)

	if len(bundle.Functions) > 0 {
		fmt.Printf("🔧 DAX Functions: %d\n", len(bundle.Functions))
		for _, fn := range bundle.Functions {
			fmt.Printf("   %s %s - %s\n", fn.Badge, fn.Name, fn.Description)
		}
	}

	// Connect to alpha database (semantic layer)
	dsn := "postgres://postgres@localhost:5432/alpha?sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Connected to semantic layer database")

	// Create schema if it doesn't exist (use domain as schema name)
	schemaName := strings.ReplaceAll(bundle.Domain, "-", "_")
	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create metrics registry table (if not exists)
	createTableSQL := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s.metrics_registry (
		node_id VARCHAR(255) PRIMARY KEY,
		category VARCHAR(100) NOT NULL,
		description TEXT,
		formula_type VARCHAR(50) NOT NULL,
		formula TEXT NOT NULL,
		arguments JSONB,
		badge VARCHAR(10),
		function_class VARCHAR(50),
		functions_used TEXT[],
		governance_status VARCHAR(50) DEFAULT 'draft',
		audience TEXT[],
		tags TEXT[],
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`, schemaName)
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create registry table: %v", err)
	}

	// Create DAX functions table if functions are present
	if len(bundle.Functions) > 0 {
		createFunctionsTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.dax_functions (
			name VARCHAR(100) PRIMARY KEY,
			class VARCHAR(50) NOT NULL,
			badge VARCHAR(10),
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`, schemaName)
		_, err = db.Exec(createFunctionsTableSQL)
		if err != nil {
			log.Fatalf("Failed to create functions table: %v", err)
		}

		// Insert DAX functions
		for _, fn := range bundle.Functions {
			_, err = db.Exec(fmt.Sprintf(`
				INSERT INTO %s.dax_functions (name, class, badge, description)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (name) DO UPDATE SET
					class = EXCLUDED.class,
					badge = EXCLUDED.badge,
					description = EXCLUDED.description
			`, schemaName), fn.Name, fn.Class, fn.Badge, fn.Description)
			if err != nil {
				log.Printf("Failed to insert function %s: %v", fn.Name, err)
			} else {
				fmt.Printf("✅ Loaded DAX function: %s %s\n", fn.Badge, fn.Name)
			}
		}
	}

	// Insert/update metrics
	goldenCount := 0
	draftCount := 0
	for i, metric := range bundle.Metrics {
		argsJSON, _ := json.Marshal(metric.FinancialCalc.Arguments)

		_, err = db.Exec(fmt.Sprintf(`
			INSERT INTO %s.metrics_registry
			(node_id, category, description, formula_type, formula, arguments, badge, function_class, functions_used, governance_status, audience, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (node_id) DO UPDATE SET
				category = EXCLUDED.category,
				description = EXCLUDED.description,
				formula_type = EXCLUDED.formula_type,
				formula = EXCLUDED.formula,
				arguments = EXCLUDED.arguments,
				badge = EXCLUDED.badge,
				function_class = EXCLUDED.function_class,
				functions_used = EXCLUDED.functions_used,
				governance_status = EXCLUDED.governance_status,
				audience = EXCLUDED.audience,
				tags = EXCLUDED.tags,
				updated_at = NOW()
		`, schemaName), metric.NodeID, metric.Category, metric.Description,
			metric.FinancialCalc.Type, metric.FinancialCalc.Formula, argsJSON,
			metric.Badge, metric.FunctionClass, pq.Array(metric.FunctionsUsed),
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
	err = db.Get(&totalCount, fmt.Sprintf("SELECT COUNT(*) FROM %s.metrics_registry", schemaName))
	if err != nil {
		log.Fatalf("Failed to count metrics: %v", err)
	}

	fmt.Printf("\n🎉 Successfully loaded %d %s metrics into semantic registry!\n", totalCount, bundle.Domain)
	fmt.Printf("🏆 Golden Status: %d metrics\n", goldenCount)
	fmt.Printf("📝 Draft Status: %d metrics\n", draftCount)

	// Show breakdown by category
	rows, err := db.Query(fmt.Sprintf(`
		SELECT category, governance_status, COUNT(*) as count
		FROM %s.metrics_registry
		GROUP BY category, governance_status
		ORDER BY category, governance_status
	`, schemaName))
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
		fmt.Printf("%-25s | %-6s | %d\n", category, status, count)
	}

	// Use x/text cases.Title for Unicode-safe title casing
	tc := cases.Title(language.Und)
	fmt.Printf("\n🚀 %s Bundle Ready!\n", tc.String(bundle.Domain))
	fmt.Printf("Your semantic layer now supports %s analytics with:\n", bundle.Domain)
	fmt.Printf("• %d DAX-powered metrics\n", totalCount)
	if len(bundle.Functions) > 0 {
		fmt.Printf("• %d DAX functions\n", len(bundle.Functions))
	}
	fmt.Printf("• Governance and audience controls\n")
	fmt.Printf("• Registry-driven metric discovery\n")
}
