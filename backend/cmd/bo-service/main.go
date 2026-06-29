package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	metadata "github.com/hondyman/semlayer/backend/internal/metadata"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// BO Command Microservice
// This service consumes CRUD commands from the command bus and executes them against the database.
// The system uses Redpanda/Kafka for command/event transport. Legacy RabbitMQ (AMQP) is still supported
// for backwards compatibility but is deprecated.
//
// Architecture:
// - Subscribes to semlayer.commands topic (Kafka) or semlayer.commands exchange (RabbitMQ legacy)
// - Binds to bo-service-commands consumer group / queue
// - Receives commands matching pattern: command.bo.* and command.instance.*
// - Executes business logic
// - Publishes events to semlayer.events
// - Publishes responses to semlayer.replies
//
// Starting this service:
//   docker-compose up bo-service
// Or locally:
//   go run ./cmd/bo-service/main.go
//
// Environment variables:
//   KAFKA_BROKERS: redpanda:9092 (preferred)
//   RABBITMQ_URL: amqp://guest:guest@rabbitmq:5672/ (legacy fallback)
//   DATABASE_URL: postgres://user:pass@db:5432/alpha (required)
//   LOG_LEVEL: debug|info|warn|error (default: info)
//   SERVICE_NAME: bo-service (for logs)

func main() {
	// Initialize logging
	logger := logging.GetLogger()
	sugar := logger.Sugar()

	sugar.Infof("🚀 Starting BO Command Microservice...")
	sugar.Infof("📋 Service: %s | Version: %s", getEnv("SERVICE_NAME", "bo-service"), "1.0.0")

	// Load configuration from environment
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable")
	// Prefer KAFKA_BROKERS for bootstrap servers; fallback to RABBITMQ_URL for legacy setups
	brokers := getEnv("KAFKA_BROKERS", getEnv("RABBITMQ_URL", "redpanda:9092"))

	sugar.Infof("📦 Database: %s", maskURL(databaseURL))
	sugar.Infof("📨 Event/Command Brokers: %s", maskURL(brokers))

	// Initialize database connection
	db, err := sqlx.Connect("pgx", databaseURL)
	if err != nil {
		sugar.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		sugar.Fatalf("❌ Failed to ping database: %v", err)
	}

	sugar.Infof("✅ Database connected")

	// Initialize services
	boService := metadata.NewBusinessObjectService(db, nil, nil, nil)
	sugar.Infof("✅ Business Object Service initialized")

	// Initialize event publisher
	eventPublisher, err := services.NewEventPublisher(brokers)
	if err != nil {
		sugar.Warnf("⚠️  Event publisher disabled: %v", err)
	} else {
		defer eventPublisher.Close()
		sugar.Infof("✅ Event publisher initialized")
	}

	// Initialize command consumer
	sugar.Infof("🔄 Initializing command consumer...")
	consumer, err := services.NewCommandConsumer(brokers, fmt.Sprintf("bo-service-%s", hostname()))
	if err != nil {
		sugar.Fatalf("❌ Command consumer failed to initialize: %v", err)
	}
	defer consumer.Close()

	sugar.Infof("✅ Command consumer initialized")

	// Register command handlers
	sugar.Infof("📝 Registering command handlers...")

	// Business Object handlers
	boCmdHandler := services.NewBOCommandHandler(boService, eventPublisher)
	consumer.RegisterHandler(services.CommandCreateBO, boCmdHandler.HandleCreateBO)
	consumer.RegisterHandler(services.CommandUpdateBO, boCmdHandler.HandleUpdateBO)
	consumer.RegisterHandler(services.CommandDeleteBO, boCmdHandler.HandleDeleteBO)
	consumer.RegisterHandler(services.CommandCloneBO, boCmdHandler.HandleCloneBO)

	sugar.Infof("✅ Registered 4 BO handlers (Create, Update, Delete, Clone)")

	// Instance handlers
	auditLogger := security.NewPlatformAdminAuditLogger(db.DB)
	impersonationPolicy := security.ImpersonationPolicy{
		// Professional_services is unrestricted by default so it can administer the
		// tenant for the duration of the session. Populate this list to restrict
		// which BO keys professional_services may mutate via break_glass.
		ProfessionalServicesBreakGlassBOKeys: []string{},
	}
	instanceCmdHandler := services.NewInstanceCommandHandler(boService, eventPublisher, auditLogger, impersonationPolicy)
	consumer.RegisterHandler(services.CommandCreateInstance, instanceCmdHandler.HandleCreateInstance)
	consumer.RegisterHandler(services.CommandUpdateInstance, instanceCmdHandler.HandleUpdateInstance)
	consumer.RegisterHandler(services.CommandDeleteInstance, instanceCmdHandler.HandleDeleteInstance)

	sugar.Infof("✅ Registered 3 Instance handlers (Create, Update, Delete)")

	// Start consuming commands
	sugar.Infof("🎬 Starting command consumer...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to BO commands
	go func() {
		if err := consumer.Subscribe(ctx, "command.bo.*"); err != nil {
			sugar.Errorf("❌ BO command consumer error: %v", err)
		}
	}()

	// Subscribe to Instance commands
	go func() {
		if err := consumer.Subscribe(ctx, "command.instance.*"); err != nil {
			sugar.Errorf("❌ Instance command consumer error: %v", err)
		}
	}()

	sugar.Infof("✅ Command consumers started")
	sugar.Infof("📡 Listening for commands on:")
	sugar.Infof("   - command.bo.* (BO CRUD operations)")
	sugar.Infof("   - command.instance.* (Instance CRUD operations)")

	// Health check goroutine (optional - can be enabled for observability)
	// go healthCheck()

	// Graceful shutdown handling
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	<-shutdownChan
	sugar.Infof("📩 Shutdown signal received, gracefully stopping...")

	cancel()
	sugar.Infof("✅ BO Command Microservice stopped")
	logger.Sync()
}

// getEnv retrieves environment variable with fallback
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// maskURL masks password in URL for safe logging
func maskURL(url string) string {
	// Simple masking: show first 20 and last 10 chars
	if i := len(url); i > 30 {
		return url[:20] + "****..." + url[i-10:]
	}
	return "****"
}

// hostname returns the hostname for service identification
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
