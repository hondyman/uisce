package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/worker"

	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/api"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/temporal/activities"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/temporal/workflows"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	fmt.Println("Connected to database")

	// Create Temporal client using centralized helper (reads env and retries)
	c, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer c.Close()

	// Create Temporal worker
	w := worker.New(c, "reconciliation", worker.Options{})

	// Register workflows and activities
	w.RegisterWorkflow(workflows.AIReconciliationWorkflow)
	w.RegisterActivity(activities.FetchYesterdaysTrades)
	w.RegisterActivity(activities.FetchTradeConfirms)
	w.RegisterActivity(activities.AIReconcile)
	w.RegisterActivity(activities.SaveReconciliationResult)
	w.RegisterActivity(activities.CreateReconciliationTask)
	w.RegisterActivity(activities.NotifyDiscrepancy)
	w.RegisterActivity(activities.AutoResolveDiscrepancy)
	w.RegisterActivity(activities.LogReconciliationAudit)

	// Start worker in background
	go func() {
		if err := w.Start(); err != nil {
			log.Fatalf("Failed to start worker: %v", err)
		}
	}()
	defer w.Stop()

	// Set up Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Register API routes
	handler := api.NewHandler(db)
	api.RegisterRoutes(router, handler)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting AI Trade Reconciliation server on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
