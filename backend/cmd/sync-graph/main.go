package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting One-Time Graph Synchronization...")

	// 1. Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	ctx := context.Background()

	// 2. Initialize Lineage Repository
	lineageRepo := lineage.NewDBLineageRepository(db)
	// Note: Using relational catalog_node and catalog_edge tables instead of AGE

	// 4. Fetch all unique tenant_datasource_ids
	log.Println("Fetching datasources...")
	var datasourceIDs []string
	query := `
		SELECT DISTINCT tenant_datasource_id 
		FROM catalog_node 
		WHERE tenant_datasource_id IS NOT NULL
	`
	if err := db.SelectContext(ctx, &datasourceIDs, query); err != nil {
		log.Fatalf("Failed to fetch datasource IDs: %v", err)
	}

	log.Printf("Found %d datasources to sync", len(datasourceIDs))

	// 5. Sync each datasource
	successCount := 0
	errorCount := 0

	for _, dsID := range datasourceIDs {
		log.Printf("Syncing datasource: %s", dsID)
		startTime := time.Now()

		if err := lineageRepo.SyncDatasource(ctx, dsID); err != nil {
			log.Printf("ERROR syncing datasource %s: %v", dsID, err)
			errorCount++
		} else {
			duration := time.Since(startTime)
			log.Printf("Successfully synced datasource %s (took %v)", dsID, duration)
			successCount++
		}
	}

	// 6. Summary
	log.Printf("Graph Synchronization Complete.")
	log.Printf("Success: %d, Failures: %d", successCount, errorCount)
}
