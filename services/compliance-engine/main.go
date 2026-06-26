package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"github.com/jmoiron/sqlx"
	kafka "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.temporal.io/sdk/client"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ComplianceEvent represents a compliance monitoring event
type ComplianceEvent struct {
	ID           string                 `json:"id" gorm:"primaryKey"`
	EventType    string                 `json:"event_type"`
	Resource     string                 `json:"resource"`
	Action       string                 `json:"action"`
	UserID       string                 `json:"user_id"`
	TenantID     string                 `json:"tenant_id"`
	DataSourceID string                 `json:"datasource_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Details      map[string]interface{} `json:"details" gorm:"serializer:json"`
	Severity     string                 `json:"severity"`
	Status       string                 `json:"status"`
	ABACContext  map[string]interface{} `json:"abac_context" gorm:"serializer:json"`
}

// WorkflowComplianceCheck represents compliance validation for workflows
type WorkflowComplianceCheck struct {
	ID             string                 `json:"id" gorm:"primaryKey"`
	WorkflowID     string                 `json:"workflow_id"`
	CheckType      string                 `json:"check_type"`
	Status         string                 `json:"status"`
	CheckedAt      time.Time              `json:"checked_at"`
	ComplianceData map[string]interface{} `json:"compliance_data" gorm:"serializer:json"`
	Violations     []string               `json:"violations" gorm:"serializer:json"`
}

type ComplianceEngine struct {
	db             *gorm.DB
	kafkaWriter    *kafka.Writer
	temporalClient client.Client
	workflowABAC   *WorkflowABACEngine
	logger         *logrus.Logger
}

func NewComplianceEngine() (*ComplianceEngine, error) {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Database connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "100.84.126.19"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "alpha"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate schemas
	db.AutoMigrate(&ComplianceEvent{}, &WorkflowComplianceCheck{})

	// Kafka (Redpanda) writer initialization — used for publishing audit/notifications
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	brokers := strings.Split(kafkaBrokers, ",")

	kWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	// Topics used by the service (expected to exist or be auto-created by Redpanda)
	_ = []string{
		"compliance.events",
		"workflow.compliance.checks",
		"abac.audit",
		"temporal.workflow.events",
	}

	// Note: Topic provisioning is typically handled by ops; Redpanda may auto-create topics on first publish.

	// Temporal client (optional - for workflow integration)
	var temporalClient client.Client
	tc, err := temporalclient.NewClientWithRetry()
	if err != nil {
		logger.Warnf("Failed to connect to Temporal: %v", err)
	} else {
		temporalClient = tc
	}

	// Initialize Workflow ABAC Engine (pass Kafka writer)
	workflowABAC := NewWorkflowABACEngine(db, kWriter, temporalClient)

	// Initialize default policies for demo tenant
	demoTenantID := getEnv("DEMO_TENANT_ID", "00000000-0000-0000-0000-000000000000")
	demoDatasourceID := getEnv("DEMO_DATASOURCE_ID", "11111111-1111-1111-1111-111111111111")

	// Prefer sqlx for performance-sensitive, complex DB ops when requested
	if getEnv("USE_SQLX", "false") == "true" {
		if sqlxdb, err := ConnectSQLX(); err != nil {
			logger.Warnf("Failed to connect sqlx DB; falling back to GORM for default policy init: %v", err)
			if err := workflowABAC.InitializeDefaultWorkflowPolicies(demoTenantID, demoDatasourceID); err != nil {
				logger.Warnf("Failed to initialize default workflow policies with GORM fallback: %v", err)
			}
		} else {
			if err := InitializeDefaultWorkflowPoliciesSQLX(sqlxdb, demoTenantID, demoDatasourceID); err != nil {
				logger.Warnf("Failed to initialize default workflow policies (sqlx): %v", err)
			}
		}
	} else {
		if err := workflowABAC.InitializeDefaultWorkflowPolicies(demoTenantID, demoDatasourceID); err != nil {
			logger.Warnf("Failed to initialize default workflow policies: %v", err)
		}
	}

	return &ComplianceEngine{
		db:             db,
		kafkaWriter:    kWriter,
		temporalClient: temporalClient,
		workflowABAC:   workflowABAC,
		logger:         logger,
	}, nil
}

func (ce *ComplianceEngine) Start(ctx context.Context) error {
	ce.logger.Info("Starting Compliance Engine...")

	// Start event listeners
	go ce.listenComplianceEvents(ctx)
	go ce.listenWorkflowChecks(ctx)
	go ce.listenABACAudit(ctx)
	go ce.listenTemporalEvents(ctx)

	// Start HTTP server for health checks and metrics
	go ce.startHTTPServer()

	// Wait for context cancellation
	<-ctx.Done()
	return ce.Shutdown()
}

func (ce *ComplianceEngine) listenComplianceEvents(ctx context.Context) {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "compliance-events-group",
		Topic:    "compliance.events",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				ce.logger.Errorf("Error fetching message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
		}

		ce.processComplianceEvent(m.Value)

		if err := r.CommitMessages(ctx, m); err != nil {
			ce.logger.Errorf("failed to commit message: %v", err)
		}
	}
}

func (ce *ComplianceEngine) processComplianceEvent(payload []byte) {
	var event ComplianceEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		ce.logger.Errorf("Failed to unmarshal compliance event: %v", err)
		return
	}

	// Set ID if not provided
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store event in database
	if err := ce.db.Create(&event).Error; err != nil {
		ce.logger.Errorf("Failed to store compliance event: %v", err)
		return
	}

	ce.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.EventType,
		"severity":   event.Severity,
		"user_id":    event.UserID,
	}).Info("Compliance event processed")
}

func (ce *ComplianceEngine) listenWorkflowChecks(ctx context.Context) {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "compliance-workflow-checks-group",
		Topic:    "workflow.compliance.checks",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				ce.logger.Errorf("Error fetching workflow check message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
		}

		ce.processWorkflowCheck(m.Value)

		if err := r.CommitMessages(ctx, m); err != nil {
			ce.logger.Errorf("failed to commit message: %v", err)
		}
	}
}

func (ce *ComplianceEngine) processWorkflowCheck(payload []byte) {
	var check WorkflowComplianceCheck
	if err := json.Unmarshal(payload, &check); err != nil {
		ce.logger.Errorf("Failed to unmarshal workflow check: %v", err)
		return
	}

	// Set ID if not provided
	if check.ID == "" {
		check.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if check.CheckedAt.IsZero() {
		check.CheckedAt = time.Now()
	}

	// Store check in database
	if err := ce.db.Create(&check).Error; err != nil {
		ce.logger.Errorf("Failed to store workflow check: %v", err)
		return
	}

	// If violations found, create compliance event
	if len(check.Violations) > 0 {
		event := ComplianceEvent{
			ID:          uuid.New().String(),
			EventType:   "workflow_violation",
			Resource:    "workflow",
			Action:      "execute",
			TenantID:    check.WorkflowID, // Using workflow ID as tenant for now
			Timestamp:   time.Now(),
			Details:     check.ComplianceData,
			Severity:    "high",
			Status:      "violation_detected",
			ABACContext: map[string]interface{}{"workflow_check_id": check.ID},
		}

		if err := ce.db.Create(&event).Error; err != nil {
			ce.logger.Errorf("Failed to create violation event: %v", err)
		}
	}

	ce.logger.WithFields(logrus.Fields{
		"check_id":    check.ID,
		"workflow_id": check.WorkflowID,
		"check_type":  check.CheckType,
		"status":      check.Status,
		"violations":  len(check.Violations),
	}).Info("Workflow compliance check processed")
}

func (ce *ComplianceEngine) listenABACAudit(ctx context.Context) {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "compliance-abac-group",
		Topic:    "abac.audit",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				ce.logger.Errorf("Error fetching abac message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
		}

		ce.processABACAudit(m.Value)

		if err := r.CommitMessages(ctx, m); err != nil {
			ce.logger.Errorf("failed to commit message: %v", err)
		}
	}
}

func (ce *ComplianceEngine) processABACAudit(payload []byte) {
	var auditEvent map[string]interface{}
	if err := json.Unmarshal(payload, &auditEvent); err != nil {
		ce.logger.Errorf("Failed to unmarshal ABAC audit event: %v", err)
		return
	}

	event := ComplianceEvent{
		ID:           uuid.New().String(),
		EventType:    "abac_decision",
		Resource:     getStringValue(auditEvent, "resource"),
		Action:       getStringValue(auditEvent, "action"),
		UserID:       getStringValue(auditEvent, "user_id"),
		TenantID:     getStringValue(auditEvent, "tenant_id"),
		DataSourceID: getStringValue(auditEvent, "datasource_id"),
		Timestamp:    time.Now(),
		Details:      auditEvent,
		Severity:     "info",
		Status:       getStringValue(auditEvent, "decision"),
		ABACContext:  auditEvent,
	}

	if err := ce.db.Create(&event).Error; err != nil {
		ce.logger.Errorf("Failed to store ABAC audit event: %v", err)
		return
	}

	ce.logger.WithFields(logrus.Fields{
		"event_id": event.ID,
		"resource": event.Resource,
		"action":   event.Action,
		"decision": event.Status,
	}).Info("ABAC audit event processed")
}

func (ce *ComplianceEngine) listenTemporalEvents(ctx context.Context) {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "compliance-temporal-group",
		Topic:    "temporal.workflow.events",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				ce.logger.Errorf("Error fetching temporal event: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
		}

		ce.processTemporalEvent(m.Value)

		if err := r.CommitMessages(ctx, m); err != nil {
			ce.logger.Errorf("failed to commit message: %v", err)
		}
	}
}

func (ce *ComplianceEngine) processTemporalEvent(payload []byte) {
	var temporalEvent map[string]interface{}
	if err := json.Unmarshal(payload, &temporalEvent); err != nil {
		ce.logger.Errorf("Failed to unmarshal temporal event: %v", err)
		return
	}

	eventType := getStringValue(temporalEvent, "event_type")
	workflowID := getStringValue(temporalEvent, "workflow_id")

	event := ComplianceEvent{
		ID:        uuid.New().String(),
		EventType: fmt.Sprintf("temporal_%s", eventType),
		Resource:  "workflow",
		Action:    eventType,
		TenantID:  workflowID,
		Timestamp: time.Now(),
		Details:   temporalEvent,
		Severity:  "info",
		Status:    "processed",
	}

	if err := ce.db.Create(&event).Error; err != nil {
		ce.logger.Errorf("Failed to store temporal event: %v", err)
		return
	}

	ce.logger.WithFields(logrus.Fields{
		"event_id":    event.ID,
		"event_type":  eventType,
		"workflow_id": workflowID,
	}).Info("Temporal workflow event processed")
}

func (ce *ComplianceEngine) startHTTPServer() {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "compliance-engine",
		})
	})

	r.POST("/workflow-abac/evaluate", ce.workflowABAC.EvaluateWorkflowPolicyHTTP)

	r.GET("/metrics", func(c *gin.Context) {
		// Basic metrics endpoint
		var eventCount int64
		var checkCount int64

		ce.db.Model(&ComplianceEvent{}).Count(&eventCount)
		ce.db.Model(&WorkflowComplianceCheck{}).Count(&checkCount)

		c.JSON(200, gin.H{
			"compliance_events_total": eventCount,
			"workflow_checks_total":   checkCount,
			"service":                 "compliance-engine",
		})
	})

	port := getEnv("HTTP_PORT", "8082")
	ce.logger.Infof("Starting HTTP server on port %s", port)
	r.Run(fmt.Sprintf(":%s", port))
}

func (ce *ComplianceEngine) Shutdown() error {
	ce.logger.Info("Shutting down Compliance Engine...")

	if ce.kafkaWriter != nil {
		ce.kafkaWriter.Close()
	}
	if ce.temporalClient != nil {
		ce.temporalClient.Close()
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// ConnectSQLX establishes a connection to the database using sqlx.
// Returns a *sqlx.DB and error.
func ConnectSQLX() (*sqlx.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "100.84.126.19"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "alpha"),
	)
	return sqlx.Connect("postgres", dsn)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine, err := NewComplianceEngine()
	if err != nil {
		log.Fatalf("Failed to create compliance engine: %v", err)
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		// signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Compliance engine failed: %v", err)
	}
}
