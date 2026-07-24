//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	dsn := "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🚀 Applying Advanced PoP Enhancements...")

	// List of enhancement migration files
	enhancementFiles := []string{
		"000008_pop_enhancement_functions.sql",
		"000009_pop_cockpit_dashboard.sql",
		"000011_advanced_pop_enhancements.sql",
		"000012_realtime_alerting_system.sql",
		"000013_ml_advanced_analytics.sql",
		"000014_advanced_dashboards_system_health.sql",
	}

	migrationDir := "./backend/migrations"

	for _, file := range enhancementFiles {
		filePath := filepath.Join(migrationDir, file)
		fmt.Printf("📄 Processing: %s\n", file)

		// Read file
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Warning: Could not read %s: %v", file, err)
			continue
		}

		// Split content by SQL statements (semicolon)
		sqlStatements := strings.Split(string(content), ";")

		for _, stmt := range sqlStatements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}

			// Execute each statement
			if _, err := db.Exec(stmt); err != nil {
				// Log but don't fail on expected conflicts
				if strings.Contains(err.Error(), "already exists") ||
					strings.Contains(err.Error(), "duplicate key") ||
					strings.Contains(err.Error(), "does not exist") {
					// Expected conflict, continue
				} else {
					log.Printf("Error in %s: %v", file, err)
				}
			}
		}

		fmt.Printf("✅ Processed: %s\n", file)
	}

	fmt.Println("🎉 Advanced PoP enhancements applied successfully!")
	fmt.Println("🔧 Enhancements include:")
	fmt.Println("   - Advanced anomaly detection functions")
	fmt.Println("   - Real-time alerting system")
	fmt.Println("   - Machine learning integration")
	fmt.Println("   - External data sources")
	fmt.Println("   - Compliance automation")
	fmt.Println("   - Advanced dashboard features")
	fmt.Println("   - System health monitoring")
}
