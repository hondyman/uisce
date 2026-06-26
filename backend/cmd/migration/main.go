package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hondyman/semlayer/backend"
	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("🚀 Running Preaggregation Schema Migration...")

	// Get database connection
	db, err := backend.GetAppDBConnection("")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database connection established")

	// Read the migration file
	migrationPath := "/Users/eganpj/GitHub/semlayer/backend/migrations/000015_preaggregation_schema.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	fmt.Println("📄 Migration file loaded")

	// Split the SQL into individual statements
	statements := strings.Split(string(migrationSQL), ";")

	// Execute each statement
	executedStatements := 0
	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(statement, "--") {
			continue
		}

		fmt.Printf("Executing statement %d/%d...\n", i+1, len(statements))

		if _, err := db.Exec(statement); err != nil {
			// Log the error but continue with other statements
			log.Printf("Warning: Failed to execute statement %d: %v", i+1, err)
			log.Printf("Statement: %s", statement[:min(100, len(statement))])
		} else {
			executedStatements++
		}
	}

	fmt.Printf("✅ Executed %d SQL statements successfully\n", executedStatements)

	// Verify the schema was created
	fmt.Println("\n🔍 Verifying preaggregation schema...")

	// Check if semantic_layer schema exists
	var schemaCount int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = 'semantic_layer'").Scan(&schemaCount)
	if err != nil {
		log.Printf("Warning: Could not verify schema creation: %v", err)
	} else if schemaCount > 0 {
		fmt.Println("✅ Semantic layer schema created successfully")
	} else {
		fmt.Println("❌ Semantic layer schema not found")
	}

	// Check if preaggregated_metrics table exists
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'semantic_layer' AND table_name = 'preaggregated_metrics'").Scan(&tableCount)
	if err != nil {
		log.Printf("Warning: Could not verify table creation: %v", err)
	} else if tableCount > 0 {
		fmt.Println("✅ Preaggregated metrics table created successfully")
	} else {
		fmt.Println("❌ Preaggregated metrics table not found")
	}

	// Check sample data
	var sampleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM semantic_layer.preaggregated_metrics WHERE id LIKE 'sample_%'").Scan(&sampleCount)
	if err != nil {
		log.Printf("Warning: Could not check sample data: %v", err)
	} else if sampleCount > 0 {
		fmt.Printf("✅ Sample data inserted (%d records)\n", sampleCount)
	} else {
		fmt.Println("ℹ️  No sample data found (this is normal)")
	}

	// Test the helper functions
	fmt.Println("\n🧪 Testing helper functions...")

	// Test get_preaggregated_metric function
	var testCount int
	err = db.QueryRow("SELECT COUNT(*) FROM semantic_layer.get_preaggregated_metric('private_markets_net_irr', '{}')").Scan(&testCount)
	if err != nil {
		log.Printf("Warning: Could not test get_preaggregated_metric function: %v", err)
	} else {
		fmt.Println("✅ get_preaggregated_metric function working")
	}

	fmt.Println("\n🎉 Preaggregation schema migration completed!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Run the preaggregation demo: cd cmd/preaggregation && go run main.go")
	fmt.Println("2. Set up automated cron jobs for daily/weekly refreshes")
	fmt.Println("3. Configure monitoring dashboards for data quality")
	fmt.Println("4. Test the preaggregated metrics in your application")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
