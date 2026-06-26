package mdmservice
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/hondyman/semlayer/mdm-service/internal/api"
	"github.com/hondyman/semlayer/mdm-service/internal/repository"
	"github.com/hondyman/semlayer/mdm-service/internal/rules"
	"github.com/hondyman/semlayer/mdm-service/internal/service"
)

func main() {
	// Load environment
	godotenv.Load()

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Database setup
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Fatal("missing DATABASE_URL environment variable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	logger.Info("database connected")

	// Initialize repositories
	calendarRepo := repository.NewCalendarRepository(pool)

	// Initialize rules engine
	rulesEngine := rules.NewRulesEngine()

	// Initialize service
	ingestionService := service.NewCalendarIngestionService(calendarRepo, rulesEngine, logger)

	// Initialize API handlers
	handler := api.NewHandler(ingestionService, logger)

	// Setup routes
	mux := http.NewServeMux()

	// REST API endpoints
	mux.HandleFunc("POST /api/v1/mdm/calendar/ingest", handler.IngestCalendarData)
	mux.HandleFunc("GET /api/v1/mdm/calendar/golden", handler.GetGoldenCalendar)
	mux.HandleFunc("GET /api/v1/mdm/calendar/is-business-day", handler.IsBusinessDay)
	mux.HandleFunc("GET /api/v1/mdm/calendar/lineage/{id}", handler.GetLineage)
	mux.HandleFunc("GET /api/v1/mdm/calendar/health", handler.GetHealthMetrics)

	// GraphQL endpoint (placeholder - would use actual GraphQL server like gqlgen)
	mux.HandleFunc("GET /api/v1/mdm/graphql/schema", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"schema": %s}`, api.GraphQLSchema())
	})

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "ok"}`)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	logger.Info("MDM Calendar Service starting", zap.String("port", port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server error", zap.Error(err))
	}
}
