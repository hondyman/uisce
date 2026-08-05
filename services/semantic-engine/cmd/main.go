package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	temporalclient "github.com/hondyman/uisce/libs/temporal-client"
	"github.com/hondyman/uisce/services/semantic-engine/internal/api"
	"github.com/hondyman/uisce/services/semantic-engine/internal/config"
	"github.com/hondyman/uisce/services/semantic-engine/internal/services"
)

func main() {
	log.Println("Starting Semantic Engine service...")

	// Load configuration from environment
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Semantic Engine config loaded: AI=%s, Governance=%s, Port=%d",
		cfg.AIServiceEndpoint, cfg.GovernanceServiceEndpoint, cfg.ServerPort)

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// Initialize temporal client (will be nil for now if not configured)
	var temporalClient *temporalclient.Client
	// TODO: Initialize actual temporal client when ready

	// Initialize services
	semanticService := services.NewSemanticService(services.SemanticServiceConfig{
		AIEndpoint:         cfg.AIServiceEndpoint,
		GovernanceEndpoint: cfg.GovernanceServiceEndpoint,
		DB:                 db,
		TemporalClient:     temporalClient,
	})

	// Initialize API handlers
	apiHandler := api.NewHandler(api.HandlerConfig{
		SemanticService: semanticService,
	})

	// Setup Gin router
	r := gin.Default()
	api.SetupRoutes(r, apiHandler)

	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Routes configured, starting Semantic Engine service on port %d", cfg.ServerPort)
	log.Fatal(r.Run(serverAddr))
}
