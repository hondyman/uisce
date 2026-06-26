package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/audit"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/engine"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/handlers"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/queue"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/service"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/workers"
)

func main() {
	log.Println("🚀 Starting Compliance Engine...")

	// Load configuration from environment
	cfg := loadConfig()

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Database connected")

	// Initialize Kafka client
	kafkaClient, err := queue.NewKafkaClient(cfg.KafkaBrokers)
	if err != nil {
		log.Fatalf("Failed to connect to Kafka: %v", err)
	}
	defer kafkaClient.Close()
	log.Println("✅ Kafka connected")

	// Initialize validation engine
	validationEngine := engine.NewValidationEngine(cfg.PolicyPath)
	versionResolver := engine.NewVersionResolver(db)

	// Initialize compliance service
	complianceService := service.NewComplianceService(
		db,
		validationEngine,
		versionResolver,
		kafkaClient,
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start post-trade worker
	postTradeWorker := workers.NewPostTradeWorker(kafkaClient, complianceService, validationEngine)
	if err := postTradeWorker.Start(ctx); err != nil {
		log.Fatalf("Failed to start post-trade worker: %v", err)
	}

	// Start StarRocks audit sink
	starRocksSink := audit.NewStarRocksSink(
		kafkaClient,
		cfg.StarRocksHTTP,
		cfg.StarRocksUser,
		cfg.StarRocksPass,
		cfg.StarRocksDB,
		cfg.StarRocksTable,
	)
	if err := starRocksSink.Start(ctx); err != nil {
		log.Fatalf("Failed to start StarRocks sink: %v", err)
	}

	// Initialize HTTP router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// JWT middleware enforces valid token, automatically attaches claims
	publicPaths := []string{"/health"}
	jwtMw := jwtmiddleware.NewJWTMiddleware(publicPaths...)
	r.Use(jwtMw.Handler)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Register trade handler routes
	tradeHandler := handlers.NewTradeHandler(complianceService)
	tradeHandler.RegisterRoutes(r)

	// Start HTTP server
	server := &http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")
		cancel() // Cancel context to stop workers

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("✅ Compliance Engine listening on %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// Config holds application configuration
type Config struct {
	DatabaseURL    string
	KafkaBrokers   string
	PolicyPath     string
	Port           string
	StarRocksHTTP  string
	StarRocksUser  string
	StarRocksPass  string
	StarRocksDB    string
	StarRocksTable string
}

func loadConfig() Config {
	return Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"),
		KafkaBrokers:   getEnv("KAFKA_BROKERS", "redpanda:9092"),
		PolicyPath:     getEnv("POLICY_PATH", "/app/policy"),
		Port:           formatPort(getEnv("PORT", "8080")),
		StarRocksHTTP:  getEnv("STARROCKS_HTTP", "http://starrocks-fe:8030"),
		StarRocksUser:  getEnv("STARROCKS_USER", "root"),
		StarRocksPass:  getEnv("STARROCKS_PASSWORD", ""),
		StarRocksDB:    getEnv("STARROCKS_DB", "alpha"),
		StarRocksTable: getEnv("STARROCKS_TABLE", "compliance_audit"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func formatPort(p string) string {
	if !strings.HasPrefix(p, ":") {
		return ":" + p
	}
	return p
}
