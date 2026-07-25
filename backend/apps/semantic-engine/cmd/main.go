package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	hasuraclient "github.com/hondyman/uisce/libs/hasura-client"
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

	// Initialize Hasura client (optional)
	var hasuraClient *hasuraclient.HasuraClient
	if cfg.HasuraEndpoint != "" {
		hasuraClient = hasuraclient.NewHasuraClient(&hasuraclient.HasuraConfig{
			Endpoint:    cfg.HasuraEndpoint,
			AdminSecret: cfg.HasuraAdminSecret,
		})
		log.Println("Hasura client initialized")
	} else {
		log.Println("Hasura client not configured (HASURA_ENDPOINT not set)")
	}

	// Initialize temporal client (will be nil for now if not configured)
	var temporalClient *temporalclient.Client
	// TODO: Initialize actual temporal client when ready

	// Initialize services
	semanticService := services.NewSemanticService(services.SemanticServiceConfig{
		AIEndpoint:         cfg.AIServiceEndpoint,
		GovernanceEndpoint: cfg.GovernanceServiceEndpoint,
		HasuraClient:       hasuraClient,
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
