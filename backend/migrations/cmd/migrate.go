package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hondyman/semlayer/backend/migrations"
	_ "github.com/lib/pq"
)

func main() {
	var (
		action       = flag.String("action", "up", "Migration action: up, down, status")
		databaseURL  = flag.String("database-url", "", "Database connection URL")
		migrationDir = flag.String("migration-dir", "./migrations", "Directory containing migration files")
	)
	flag.Parse()

	if *databaseURL == "" {
		*databaseURL = os.Getenv("DATABASE_URL")
		if *databaseURL == "" {
			log.Fatal("Database URL must be provided via -database-url flag or DATABASE_URL environment variable")
		}
	}

	// Connect to database
	db, err := sql.Open("postgres", *databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Get absolute path for migration directory
	absMigrationDir, err := filepath.Abs(*migrationDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for migration directory: %v", err)
	}

	// Create migration runner
	runner := migrations.NewMigrationRunner(db, absMigrationDir)

	// Execute action
	switch *action {
	case "up":
		if err := runner.Up(); err != nil {
			log.Fatalf("Failed to run migrations up: %v", err)
		}
		fmt.Println("Migrations completed successfully")
	case "down":
		if err := runner.Down(); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Println("Migration rollback completed successfully")
	case "status":
		if err := runner.Status(); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
	default:
		log.Fatalf("Unknown action: %s. Use: up, down, or status", *action)
	}
}
