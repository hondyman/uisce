package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/services"
	_ "github.com/lib/pq"
)

func main() {
	// Database connection (replace with your actual connection string)
	db, err := sql.Open("postgres", "postgres://user:password@localhost/semlayer?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to database successfully")

	// Initialize preaggregation service
	preaggService := services.NewPreaggregationService(db)

	// Initialize scheduler
	scheduler := services.NewPreaggregationScheduler(preaggService)
	jobRunner := services.NewPreaggregationJobRunner(scheduler)

	fmt.Println("🚀 Starting preaggregation demo...")

	// Run daily jobs
	fmt.Println("\n📊 Running daily preaggregation jobs...")
	startTime := time.Now()

	if err := jobRunner.RunAllDailyJobs(); err != nil {
		log.Printf("Error running daily jobs: %v", err)
	} else {
		fmt.Printf("✅ Daily jobs completed in %.2f seconds\n", time.Since(startTime).Seconds())
	}

	// Run weekly jobs
	fmt.Println("\n📈 Running weekly preaggregation jobs...")
	startTime = time.Now()

	if err := jobRunner.RunAllWeeklyJobs(); err != nil {
		log.Printf("Error running weekly jobs: %v", err)
	} else {
		fmt.Printf("✅ Weekly jobs completed in %.2f seconds\n", time.Since(startTime).Seconds())
	}

	// Query some results to verify
	fmt.Println("\n🔍 Verifying preaggregated results...")

	// Example query for Net IRR
	rows, err := db.Query(`
		SELECT node_id, COUNT(*) as record_count,
		       AVG(value) as avg_value,
		       MAX(last_refresh) as latest_refresh
		FROM semantic_layer.preaggregated_metrics
		WHERE node_id LIKE 'private_markets_%'
		GROUP BY node_id
		ORDER BY node_id
	`)
	if err != nil {
		log.Printf("Error querying results: %v", err)
	} else {
		fmt.Println("\n📋 Preaggregation Results Summary:")
		fmt.Println("Node ID | Records | Avg Value | Latest Refresh")
		fmt.Println("--------|---------|-----------|----------------")

		for rows.Next() {
			var nodeID string
			var recordCount int
			var avgValue float64
			var latestRefresh time.Time

			err := rows.Scan(&nodeID, &recordCount, &avgValue, &latestRefresh)
			if err != nil {
				log.Printf("Error scanning result row: %v", err)
				continue
			}

			fmt.Printf("%-30s | %-7d | %-9.4f | %s\n",
				nodeID, recordCount, avgValue, latestRefresh.Format("2006-01-02 15:04"))
		}
		rows.Close()
	}

	// Show scheduler status
	fmt.Println("\n⏰ Scheduler Status:")
	status := scheduler.GetJobStatus()
	for jobName, jobStatus := range status {
		statusMap := jobStatus.(map[string]interface{})
		fmt.Printf("  %s: Next run at %s\n",
			jobName, statusMap["next_run"].(time.Time).Format("2006-01-02 15:04:05"))
	}

	fmt.Println("\n🎉 Preaggregation demo completed!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Set up automated cron jobs using the scheduler")
	fmt.Println("2. Create database indexes for optimal query performance")
	fmt.Println("3. Set up monitoring dashboards for data quality")
	fmt.Println("4. Configure alerting for preaggregation failures")
}
