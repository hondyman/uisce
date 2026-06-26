package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/engine"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/models"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/queue"
)

// ComplianceService orchestrates validation logic
type ComplianceService struct {
	db              *sql.DB
	engine          *engine.ValidationEngine
	versionResolver *engine.VersionResolver
	kafkaClient     *queue.KafkaClient
}

// NewComplianceService creates a new compliance service
func NewComplianceService(
	db *sql.DB,
	engine *engine.ValidationEngine,
	versionResolver *engine.VersionResolver,
	kafkaClient *queue.KafkaClient,
) *ComplianceService {
	return &ComplianceService{
		db:              db,
		engine:          engine,
		versionResolver: versionResolver,
		kafkaClient:     kafkaClient,
	}
}

// PreTradeValidate performs synchronous pre-trade validation
func (s *ComplianceService) PreTradeValidate(ctx context.Context, trade models.TradeRequest) (*models.ValidationResult, error) {
	// Determine which version to use based on trade date
	version, err := s.versionResolver.GetVersionForDate(ctx, trade.TradeDate, "PRE_TRADE")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve version: %w", err)
	}

	// Run CUE validation
	err = s.engine.Validate(ctx, trade, version, "PreTrade")

	result := &models.ValidationResult{
		TraceID:     trade.ID,
		RuleVersion: version,
		ValidatedAt: time.Now(),
	}

	if err != nil {
		result.Status = "REJECTED"
		result.Errors = []string{err.Error()}

		// Log to database
		s.logEvent(ctx, trade, "PRE_TRADE", "FAIL", version, result.Errors)

		return result, nil
	}

	result.Status = "APPROVED"

	// Log success
	s.logEvent(ctx, trade, "PRE_TRADE", "PASS", version, nil)

	return result, nil
}

// PostTradeValidate performs comprehensive post-trade validation
func (s *ComplianceService) PostTradeValidate(ctx context.Context, trade models.TradeRequest) (*models.ValidationResult, error) {
	// Determine version
	version, err := s.versionResolver.GetVersionForDate(ctx, trade.TradeDate, "POST_TRADE")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve version: %w", err)
	}

	// Run CUE validation with PostTrade checks
	err = s.engine.Validate(ctx, trade, version, "PostTrade")

	result := &models.ValidationResult{
		TraceID:     trade.ID,
		RuleVersion: version,
		ValidatedAt: time.Now(),
	}

	if err != nil {
		result.Status = "REJECTED"
		result.Errors = []string{err.Error()}
		s.logEvent(ctx, trade, "POST_TRADE", "FAIL", version, result.Errors)
		return result, nil
	}

	result.Status = "APPROVED"
	s.logEvent(ctx, trade, "POST_TRADE", "PASS", version, nil)

	return result, nil
}

// SubmitTrade performs pre-trade validation and queues post-trade processing
func (s *ComplianceService) SubmitTrade(ctx context.Context, trade models.TradeRequest) (*models.ValidationResult, error) {
	// Step 1: Pre-trade validation (synchronous gate)
	result, err := s.PreTradeValidate(ctx, trade)
	if err != nil {
		return nil, err
	}

	if result.Status == "REJECTED" {
		// Don't queue for post-trade if pre-trade fails
		return result, nil
	}

	// Step 2: Queue for post-trade processing
	routingKey := fmt.Sprintf("trade.created.%s", trade.Currency)
	if err := s.kafkaClient.PublishEvent(ctx, routingKey, trade); err != nil {
		return nil, fmt.Errorf("failed to queue post-trade: %w", err)
	}

	// Step 3: Publish audit event
	s.publishAuditEvent(ctx, trade, "PRE_TRADE", "PASS", result.RuleVersion)

	return result, nil
}

// logEvent writes validation events to the database
func (s *ComplianceService) logEvent(ctx context.Context, trade models.TradeRequest, eventType string, status string, version string, errors []string) {
	tradeDataJSON, _ := json.Marshal(map[string]interface{}{
		"id":        trade.ID,
		"tradeDate": trade.TradeDate,
		"amount":    trade.Amount,
		"currency":  trade.Currency,
		"orderType": trade.OrderType,
	})

	var errorDetailsJSON []byte
	if len(errors) > 0 {
		errorDetailsJSON, _ = json.Marshal(map[string]interface{}{"errors": errors})
	}

	query := `
		INSERT INTO compliance_events (event_id, trace_id, event_type, status, rule_version, trade_data, error_details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query, uuid.New(), trade.ID, eventType, status, version, tradeDataJSON, errorDetailsJSON)
	if err != nil {
		// Log but don't fail the validation
		fmt.Printf("Failed to log event to database: %v\n", err)
	}
}

// publishAuditEvent publishes to Kafka for StarRocks ingestion
func (s *ComplianceService) publishAuditEvent(ctx context.Context, trade models.TradeRequest, eventType string, status string, version string) {
	event := models.ComplianceEvent{
		EventID:     uuid.New(),
		TraceID:     trade.ID,
		EventType:   eventType,
		Status:      status,
		RuleVersion: version,
		TradeData: map[string]interface{}{
			"id":        trade.ID,
			"amount":    trade.Amount,
			"currency":  trade.Currency,
			"orderType": trade.OrderType,
		},
		CreatedAt: time.Now(),
	}

	routingKey := fmt.Sprintf("audit.%s.%s", eventType, status)
	if err := s.kafkaClient.PublishEvent(ctx, routingKey, event); err != nil {
		fmt.Printf("Failed to publish audit event: %v\n", err)
	}
}
