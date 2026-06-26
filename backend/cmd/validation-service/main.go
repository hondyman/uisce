package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/hondyman/semlayer/backend/internal/handlers"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration from environment
	port := getEnv("PORT", "8082")
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")

	logger.Info("Starting Validation Service",
		zap.String("port", port),
		zap.String("database_url", maskURL(databaseURL)),
		zap.String("kafka_brokers", kafkaBrokers),
	)

	// Connect to database
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Verify database connection
	if err := db.Ping(); err != nil {
		logger.Fatal("Database connection check failed", zap.Error(err))
	}
	logger.Info("Database connection established")

	// Initialize services
	logger.Info("Initializing services")

	// Create command publisher (for async requests)
	_, err = services.NewCommandPublisher(kafkaBrokers)
	if err != nil {
		logger.Warn("Failed to create command publisher", zap.Error(err))
	}

	// Create async validator
	asyncValidator, err := services.NewAsyncValidator(db, kafkaBrokers)
	if err != nil {
		logger.Warn("Failed to create async validator", zap.Error(err))
	}

	// Initialize business object service (as InstanceProvider)
	boService := services.NewBusinessObjectService(db)

	// Create validation rule engine
	ruleEngine := services.NewValidationRuleEngine(db, boService)

	// Create Kafka writer for BP coordinator (used to publish routing/results)
	writer := &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(kafkaBrokers, ",")...),
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Create BP validation coordinator
	bpCoordinator := services.NewBPValidationCoordinator(db, ruleEngine, asyncValidator, writer)

	// Create validation handler
	validationHandler := handlers.NewValidationHandler(bpCoordinator, asyncValidator, logger)

	// Create HTTP router
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// CORS
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-ID", "X-Tenant-Datasource-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// JWT Middleware - validates JWT on all routes except /health and /metrics
	publicPaths := []string{"/health", "/metrics", "/docs"}
	jwtMiddleware := jwtmiddleware.NewJWTMiddleware(publicPaths...)
	router.Use(jwtMiddleware.Handler)

	// Health check
	router.Get("/health", healthHandler(logger))

	// Metrics endpoint
	router.Get("/metrics", metricsHandler(logger))

	// API Routes for Validation Service
	router.Route("/api/validation", func(r chi.Router) {
		// Synchronous BP step validation
		r.Post("/bp-step", validateBPStepHandler(validationHandler, logger))

		// Queue async validation
		r.Post("/queue", queueValidationHandler(validationHandler, logger))

		// Get validation result
		r.Get("/result/{validationID}", getValidationResultHandler(validationHandler, logger))

		// Subscribe to validation events (Server-Sent Events)
		r.Get("/events/subscribe", subscribeValidationEventsHandler(validationHandler, logger))

		// List recent validations
		r.Get("/recent", listRecentValidationsHandler(db, logger))

		// Get validation statistics
		r.Get("/stats", validationStatsHandler(db, logger))
	})

	// Start HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in goroutine
	go func() {
		logger.Info("Validation Service listening", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Validation Service")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
	logger.Info("Validation Service stopped")
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

// healthHandler returns service health status
func healthHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"service":   "validation-service",
			"timestamp": time.Now().UTC(),
		})
	}
}

// metricsHandler returns Prometheus-format metrics
func metricsHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		// Placeholder for actual metrics collection
		fmt.Fprintf(w, "# HELP validation_service_requests_total Total requests processed\n")
		fmt.Fprintf(w, "# TYPE validation_service_requests_total counter\n")
		fmt.Fprintf(w, "validation_service_requests_total 0\n")
	}
}

// validateBPStepHandler handles synchronous BP step validation
func validateBPStepHandler(vh *handlers.ValidationHandler, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID
		userID := claims.UserID

		var req struct {
			BPName   string                 `json:"bp_name"`
			StepName string                 `json:"step_name"`
			FormData map[string]interface{} `json:"form_data"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		result, err := vh.ValidateBPStep(r.Context(), tenantID, userID, req.BPName, req.StepName, req.FormData)
		if err != nil {
			logger.Error("BP step validation failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}

// queueValidationHandler queues an async validation
func queueValidationHandler(vh *handlers.ValidationHandler, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID
		userID := claims.UserID

		var req struct {
			BPName   string                 `json:"bp_name"`
			StepName string                 `json:"step_name"`
			FormData map[string]interface{} `json:"form_data"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		validationID, err := vh.QueueBPValidation(r.Context(), tenantID, userID, req.BPName, req.StepName, req.FormData)
		if err != nil {
			logger.Error("Failed to queue validation", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"validation_id": validationID,
			"status":        "queued",
		})
	}
}

// getValidationResultHandler retrieves a validation result
func getValidationResultHandler(vh *handlers.ValidationHandler, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		validationID := chi.URLParam(r, "validationID")

		result, err := vh.GetValidationResult(r.Context(), validationID)
		if err != nil {
			logger.Error("Failed to get validation result", zap.Error(err))
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "validation not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}

// subscribeValidationEventsHandler subscribes to validation events via Server-Sent Events
func subscribeValidationEventsHandler(vh *handlers.ValidationHandler, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		bpName := r.URL.Query().Get("bp_name")
		stepName := r.URL.Query().Get("step_name")

		if bpName == "" || stepName == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "bp_name and step_name required"})
			return
		}

		eventsChan, err := vh.SubscribeToValidationEvents(r.Context(), bpName, stepName)
		if err != nil {
			logger.Error("Failed to subscribe to events", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "streaming not supported"})
			return
		}

		// Stream events
		for {
			select {
			case event := <-eventsChan:
				if event == nil {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", toJSON(event))
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

// listRecentValidationsHandler lists recent validations
func listRecentValidationsHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID
		limitStr := r.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "10"
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}

		var results []map[string]interface{}
		query := `
			SELECT id, tenant_id, bp_name, step_name, status, passed, created_at
			FROM bp_validations
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`

		err = db.SelectContext(r.Context(), &results, query, tenantID, limit)
		if err != nil {
			logger.Error("Failed to list validations", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to list validations"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":     len(results),
			"results":   results,
			"timestamp": time.Now().UTC(),
		})
	}
}

// validationStatsHandler returns validation statistics
func validationStatsHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID

		var stats struct {
			Total    int64 `db:"total"`
			Passed   int64 `db:"passed"`
			Failed   int64 `db:"failed"`
			LastHour int64 `db:"last_hour"`
			LastDay  int64 `db:"last_day"`
		}

		query := `
			SELECT
				COUNT(*) as total,
				SUM(CASE WHEN passed THEN 1 ELSE 0 END) as passed,
				SUM(CASE WHEN NOT passed THEN 1 ELSE 0 END) as failed,
				SUM(CASE WHEN created_at > NOW() - INTERVAL '1 hour' THEN 1 ELSE 0 END) as last_hour,
				SUM(CASE WHEN created_at > NOW() - INTERVAL '1 day' THEN 1 ELSE 0 END) as last_day
			FROM bp_validations
			WHERE tenant_id = $1
		`

		err := db.GetContext(r.Context(), &stats, query, tenantID)
		if err != nil {
			logger.Error("Failed to get stats", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to get statistics"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total":     stats.Total,
			"passed":    stats.Passed,
			"failed":    stats.Failed,
			"last_hour": stats.LastHour,
			"last_day":  stats.LastDay,
			"timestamp": time.Now().UTC(),
		})
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskURL(url string) string {
	if len(url) > 30 {
		return url[:15] + "..." + url[len(url)-10:]
	}
	return url
}

func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{\"error\": \"marshal failed\"}"
	}
	return string(data)
}
