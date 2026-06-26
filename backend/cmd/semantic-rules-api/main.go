package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/cash"
	"github.com/hondyman/semlayer/backend/internal/compliance"
	"github.com/hondyman/semlayer/backend/internal/handlers"
	"github.com/hondyman/semlayer/backend/internal/observability"
	"github.com/hondyman/semlayer/backend/internal/risk"
	"github.com/hondyman/semlayer/backend/internal/scheduler"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/transaction"
	"github.com/hondyman/semlayer/backend/internal/validation"
	"github.com/hondyman/semlayer/backend/internal/wasm"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

func main() {
	// Database configuration
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")

	// Create router
	router := mux.NewRouter()

	// Initialize JWT middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Println("Warning: JWT_SECRET not set, using development secret")
		jwtSecret = "dev-secret"
	}
	jwtMw := jwtmiddleware.NewJWTMiddleware("/health", "/ready")
	router.Use(jwtMw.MiddlewareFunc)

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"semantic-rules-api"}`))
	}).Methods("GET")

	// Readiness check endpoint
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not ready","error":"database connection failed"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Initialize gold copy publisher for downstream systems (Redpanda/Kafka) FIRST - before handlers
	redpandaBrokers := os.Getenv("REDPANDA_BROKERS")
	if redpandaBrokers == "" {
		redpandaBrokers = "localhost:9092"
	}
	goldCopyPublisher, err := services.NewGoldCopyPublisher(redpandaBrokers)
	if err != nil {
		log.Printf("Warning: Failed to initialize gold copy publisher: %v", err)
	}

	// Initialize handlers with database connection
	ruleHandler := handlers.NewRuleHandlerWithDB(db, goldCopyPublisher, nil)
	templateHandler := handlers.NewTemplateHandler(db)
	bulkHandler := handlers.NewBulkOperationsHandler(db)

	// Transaction Master
	sqlxDB := sqlx.NewDb(db, "postgres")
	txRepo := transaction.NewTransactionRepository(sqlxDB, nil) // Engine nil for now
	txHandler := handlers.NewTransactionHandler(txRepo)

	// Cash Master
	cashRepo := cash.NewCashRepository(sqlxDB)
	cashHandler := handlers.NewCashHandler(cashRepo)

	// Compliance Engine
	complianceRepo := compliance.NewComplianceRepository(sqlxDB)
	complianceHandler := handlers.NewComplianceHandler(complianceRepo)

	// Risk Engine
	riskRepo := risk.NewRiskRepository(sqlxDB)
	riskHandler := handlers.NewRiskHandler(riskRepo)

	// Phase 19: Observability Telemetry Layer
	obsRepo := observability.NewSQLRepository(sqlxDB)
	obsHandler := handlers.NewWasmTelemetryHandler(obsRepo)

	// WASM Option C Orchestration
	auditLogger := &audit.StdLogAudit{}
	wasmBytes := []byte{} // Placeholder: Load from disk/catalog

	// Phase 18: Operational Intelligence Validation
	schemaDir := "./schemas" // Relative to cmd execution inside backend/
	validator, err := validation.NewValidator(schemaDir)
	if err != nil {
		log.Printf("Failed to compile internal schemas: %v", err)
	}

	wasmEngine, err := wasm.NewWazeroEngine(context.Background(), wasmBytes, auditLogger, validator)
	if err != nil {
		log.Printf("Failed to initialize WASM Engine: %v", err)
	}

	// Fake tenant for scheduler instantiation
	systemTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	complianceSvc := services.NewComplianceService(sqlxDB, wasmEngine, auditLogger, systemTenantID)
	riskSvc := services.NewRiskService(sqlxDB, wasmEngine, auditLogger, systemTenantID)

	tenantRepo := scheduler.NewSQLTenantRepository(sqlxDB)
	dailyETL := scheduler.NewDailyETLScheduler(complianceSvc, riskSvc, auditLogger, tenantRepo, sqlxDB, 5)

	// Start scheduler
	if err := dailyETL.Start(context.Background()); err != nil {
		log.Printf("Failed to start daily ETL scheduler: %v", err)
	}
	defer dailyETL.Stop(context.Background())

	// Initialize job queue and processor for async operations
	jobQueue := services.NewPostgresJobQueue(db)
	webhookNotifier := services.NewHTTPWebhookNotifier()
	operationHandler := services.NewBulkOperationHandler(db)
	jobProcessor := services.NewJobProcessor(jobQueue, db, operationHandler, webhookNotifier, 4)

	// Initialize export service (Feature 4)
	exportStoragePath := os.Getenv("EXPORT_STORAGE_PATH")
	if exportStoragePath == "" {
		exportStoragePath = "/tmp/exports"
	}
	exportURLBase := os.Getenv("EXPORT_URL_BASE")
	if exportURLBase == "" {
		exportURLBase = "http://localhost:8080"
	}
	exportService := services.NewPostgresExportService(db, exportStoragePath, exportURLBase)

	// Initialize scheduler service (Feature 4)
	schedulerService := services.NewPostgresSchedulerService(db)

	// Start job processor in background
	if err := jobProcessor.Start(context.Background()); err != nil {
		log.Printf("Warning: Failed to start job processor: %v", err)
	}

	// Start scheduler service background loop (Feature 4)
	schedulerContext := context.Background()
	if err := schedulerService.Start(schedulerContext, jobQueue); err != nil {
		log.Printf("Warning: Failed to start scheduler service: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping services...")
		_ = jobProcessor.Stop(10 * time.Second)
		schedulerService.Stop()
		if goldCopyPublisher != nil {
			_ = goldCopyPublisher.Close()
		}
		os.Exit(0)
	}()

	// Register rule routes
	ruleHandler.RegisterRoutes(api)

	// Register template routes
	templateHandler.RegisterTemplateRoutes(api)

	// Register bulk operation routes (sync)
	api.HandleFunc("/templates/bulk-create", bulkHandler.BulkCreateTemplates).Methods("POST")
	api.HandleFunc("/templates/bulk-publish", bulkHandler.BulkPublishTemplates).Methods("POST")
	api.HandleFunc("/rules/bulk-promote", bulkHandler.BulkPromoteRules).Methods("POST")

	// Transaction Master routes
	api.HandleFunc("/transactions", txHandler.ListTransactions).Methods("GET")
	api.HandleFunc("/transactions/ingest", txHandler.IngestTransactions).Methods("POST")

	// Cash Master routes
	api.HandleFunc("/cash/balances", cashHandler.ListCashBalances).Methods("GET")
	api.HandleFunc("/cash/ledger", cashHandler.ListCashLedger).Methods("GET")
	api.HandleFunc("/cash/ledger/ingest", cashHandler.IngestCashLedger).Methods("POST")
	api.HandleFunc("/cash/balances/rollforward", cashHandler.RunBalanceRollForward).Methods("POST")
	api.HandleFunc("/cash/transactions/map", cashHandler.MapTransactionsToCash).Methods("POST")

	// Compliance Engine routes
	complianceRouter := api.PathPrefix("/compliance").Subrouter()
	complianceHandler.RegisterRoutes(complianceRouter)

	// Risk Engine routes
	riskRouter := api.PathPrefix("/risk").Subrouter()
	riskHandler.RegisterRoutes(riskRouter)

	// Set up Observability routes
	obsRouter := api.PathPrefix("/telemetry").Subrouter()
	obsHandler.RegisterRoutes(obsRouter)

	// Register async job routes
	asyncJobsHandler := handlers.NewAsyncJobsHandler(jobQueue, db)
	api.HandleFunc("/templates/bulk-create/async", asyncJobsHandler.CreateAsyncBulkCreateJob).Methods("POST")
	api.HandleFunc("/templates/bulk-publish/async", asyncJobsHandler.CreateAsyncBulkPublishJob).Methods("POST")
	api.HandleFunc("/jobs/{jobId}", asyncJobsHandler.GetJobStatus).Methods("GET")
	api.HandleFunc("/jobs", asyncJobsHandler.ListJobs).Methods("GET")
	api.HandleFunc("/jobs/{jobId}/cancel", asyncJobsHandler.CancelJob).Methods("POST")
	api.HandleFunc("/jobs/stats", asyncJobsHandler.GetProcessorStats).Methods("GET")

	// Register export routes (Feature 4)
	exportHandlers := handlers.NewExportHandlers(exportService)
	api.HandleFunc("/jobs/{jobId}/exports", exportHandlers.CreateExport).Methods("POST")
	api.HandleFunc("/exports/{exportId}", exportHandlers.GetExportStatus).Methods("GET")
	api.HandleFunc("/jobs/{jobId}/exports", exportHandlers.ListExports).Methods("GET")
	api.HandleFunc("/exports/{exportId}/download", exportHandlers.DownloadExport).Methods("GET")
	api.HandleFunc("/exports/{exportId}/download-url", exportHandlers.GetDownloadURL).Methods("POST")

	// Register scheduler routes (Feature 4)
	schedulerHandlers := handlers.NewSchedulerHandlers(schedulerService)
	api.HandleFunc("/schedules", schedulerHandlers.CreateScheduledJob).Methods("POST")
	api.HandleFunc("/schedules", schedulerHandlers.ListSchedules).Methods("GET")
	api.HandleFunc("/schedules/{scheduleId}", schedulerHandlers.GetSchedule).Methods("GET")
	api.HandleFunc("/schedules/{scheduleId}/pause", schedulerHandlers.PauseSchedule).Methods("POST")
	api.HandleFunc("/schedules/{scheduleId}/resume", schedulerHandlers.ResumeSchedule).Methods("POST")
	api.HandleFunc("/schedules/{scheduleId}", schedulerHandlers.DeleteSchedule).Methods("DELETE")

	// Start server with basic CORS setup
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID, X-User-ID, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Start server
	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Semantic Rules API Server starting on %s\n", addr)
	fmt.Println("\nRegistered Endpoints:")
	fmt.Println("  Rules:")
	fmt.Println("    POST   /api/v1/rules")
	fmt.Println("    GET    /api/v1/rules")
	fmt.Println("    GET    /api/v1/rules/{ruleId}")
	fmt.Println("    PUT    /api/v1/rules/{ruleId}")
	fmt.Println("    DELETE /api/v1/rules/{ruleId}")
	fmt.Println("    POST   /api/v1/rules/{ruleId}/publish")
	fmt.Println("    POST   /api/v1/rules/{ruleId}/promote")
	fmt.Println("    POST   /api/v1/rules/{ruleId}/simulate")
	fmt.Println("    GET    /api/v1/rules/{ruleId}/versions")
	fmt.Println("    GET    /api/v1/rules/{ruleId}/diff")
	fmt.Println("    GET    /api/v1/semantic-terms")
	fmt.Println("\n  Templates:")
	fmt.Println("    POST   /api/v1/templates")
	fmt.Println("    GET    /api/v1/templates")
	fmt.Println("    GET    /api/v1/templates/{templateId}")
	fmt.Println("    PUT    /api/v1/templates/{templateId}")
	fmt.Println("    DELETE /api/v1/templates/{templateId}")
	fmt.Println("    POST   /api/v1/templates/{templateId}/create-rule")
	fmt.Println("    POST   /api/v1/templates/{templateId}/preview")
	fmt.Println("    GET    /api/v1/templates/{templateId}/instances")

	fmt.Println("\n  Bulk Operations (Sync):")
	fmt.Println("    POST   /api/v1/templates/bulk-create")
	fmt.Println("    POST   /api/v1/templates/bulk-publish")
	fmt.Println("    POST   /api/v1/rules/bulk-promote")

	fmt.Println("\n  Bulk Operations (Async):")
	fmt.Println("    POST   /api/v1/templates/bulk-create/async")
	fmt.Println("    POST   /api/v1/templates/bulk-publish/async")
	fmt.Println("    GET    /api/v1/jobs/{jobId}")
	fmt.Println("    GET    /api/v1/jobs?status=running&limit=20")
	fmt.Println("    POST   /api/v1/jobs/{jobId}/cancel")
	fmt.Println("    GET    /api/v1/jobs/stats")

	fmt.Println("\n  Exports (Feature 4):")
	fmt.Println("    POST   /api/v1/jobs/{jobId}/exports")
	fmt.Println("    GET    /api/v1/exports/{exportId}")
	fmt.Println("    GET    /api/v1/jobs/{jobId}/exports")
	fmt.Println("    GET    /api/v1/exports/{exportId}/download")
	fmt.Println("    POST   /api/v1/exports/{exportId}/download-url")

	fmt.Println("\n  Scheduling (Feature 4):")
	fmt.Println("    POST   /api/v1/schedules")
	fmt.Println("    GET    /api/v1/schedules")
	fmt.Println("    GET    /api/v1/schedules/{scheduleId}")
	fmt.Println("    POST   /api/v1/schedules/{scheduleId}/pause")
	fmt.Println("    POST   /api/v1/schedules/{scheduleId}/resume")
	fmt.Println("    DELETE /api/v1/schedules/{scheduleId}")

	fmt.Println("\n  Health:")
	fmt.Println("    GET    /health")
	fmt.Println("    GET    /ready")

	log.Fatal(http.ListenAndServe(addr, router))
}
