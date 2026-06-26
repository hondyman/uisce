//go:build ignore

package main

import (
	"database/sql"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/dynamic"
	"github.com/hondyman/semlayer/backend/internal/handlers"
	"github.com/hondyman/semlayer/backend/internal/query"
	"github.com/hondyman/semlayer/backend/internal/services"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Initialize database connection
	db, err := sql.Open("pgx", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "pgx")

	// Initialize core components
	cubeEngine := &cube.Cube{}
	templateMgr := query.NewQueryTemplateManager()
	dynamicEngine := dynamic.NewDynamicQueryEngine(cubeEngine, templateMgr)

	// Initialize handlers
	basicPoPHandler := handlers.NewPoPHandler(db)
	enhancedPoPHandler := handlers.NewEnhancedPoPHandler(db, dynamicEngine, templateMgr)
	dynamicHandler := handlers.NewDynamicQueryHandler(dynamicEngine, templateMgr)

	// Initialize lineage services and handlers
	lineageService := services.NewLineageService(sqlxDB)
	lineageHandler := handlers.NewLineageHandler(lineageService)
	lineageVisualizationHandler := handlers.NewLineageVisualizationHandler(db)

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API versioning
	v1 := r.Group("/api/v1")
	{
		// Original PoP endpoints (backward compatibility)
		pop := v1.Group("/pop")
		{
			pop.GET("/manifest", basicPoPHandler.GetPoPManifest)
			pop.GET("/metrics/:id", basicPoPHandler.GetPoPMetric)
			pop.POST("/metrics/:id/analyze", basicPoPHandler.AnalyzePoPMetric)
			pop.POST("/metrics/:id/promote", basicPoPHandler.PromotePoPMetric)
			pop.POST("/metrics/:id/flag", basicPoPHandler.FlagPoPAnomaly)
			pop.POST("/metrics/:id/comment", basicPoPHandler.AddPoPComment)
		}

		// Enhanced dynamic endpoints
		dynamic := v1.Group("/dynamic")
		{
			// Core dynamic query endpoint
			dynamic.POST("/query", dynamicHandler.HandleDynamicQuery)

			// Specialized dynamic endpoints for PoP system
			dynamic.POST("/pop/analysis", enhancedPoPHandler.HandleDynamicPoPAnalysis)
			dynamic.POST("/pop/anomalies", enhancedPoPHandler.HandleDynamicAnomalyDetection)
			dynamic.POST("/pop/steward", enhancedPoPHandler.HandleDynamicStewardReview)
			dynamic.POST("/pop/dashboards", enhancedPoPHandler.HandleDynamicDashboardAnalysis)
			dynamic.POST("/pop/comparison", enhancedPoPHandler.HandleDynamicMetricComparison)

			// Utility endpoints
			dynamic.POST("/suggest-measures", dynamicHandler.HandleDynamicMeasureSuggestion)
			dynamic.POST("/validate-params", dynamicHandler.HandleParameterValidation)
			dynamic.POST("/cube-config", dynamicHandler.HandleCubeConfigGeneration)
		}

		// Lineage Endpoints
		// The panic `conflicts with existing wildcard ':asset_id' in existing prefix '/api/lineage/:asset_id'`
		// indicates that two routes like `/api/lineage/:asset_id` and `/api/lineage/:node_id` were registered.
		// To fix this, we must use a consistent parameter name for the same path segment.
		// Here, we standardize on `:id`.
		lineage := v1.Group("/lineage")
		{
			lineage.POST("", lineageHandler.HandleLineage)
			lineage.POST("/technical", lineageHandler.HandleTechnicalLineage)
			lineage.POST("/semantic", lineageHandler.HandleSemanticLineage)
			lineage.GET("/:id", lineageVisualizationHandler.GetLineage)
			lineage.GET("/:id/graph", lineageVisualizationHandler.GetLineageGraph)
		}
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"services": []string{
				"database",
				"dynamic_engine",
				"cube_engine",
				"template_manager",
			},
		})
	})

	// Start server
	log.Println("🚀 Dynamic PoP API Server starting on :8080")
	log.Println("📊 Available endpoints:")
	log.Println("   GET  /health")
	log.Println("   GET  /api/v1/pop/manifest")
	log.Println("   POST /api/v1/dynamic/pop/analysis")
	log.Println("   POST /api/v1/dynamic/pop/anomalies")
	log.Println("   POST /api/v1/dynamic/pop/steward")
	log.Println("   POST /api/v1/dynamic/pop/dashboards")
	log.Println("   POST /api/v1/dynamic/pop/comparison")
	log.Println("   POST /api/v1/dynamic/query")
	log.Println("   POST /api/v1/dynamic/suggest-measures")
	log.Println("   POST /api/v1/dynamic/validate-params")
	log.Println("   POST /api/v1/dynamic/cube-config")

	r.Run(":8080")
}
