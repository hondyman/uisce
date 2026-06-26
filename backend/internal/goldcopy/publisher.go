package goldcopy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const goldCopyTopic = "semlayer.gold-copy"

// GoldCopyEvent is the Kafka message payload published when a gold copy materialises.
type GoldCopyEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	PublishedAt   time.Time              `json:"published_at"`
	PublishedBy   string                 `json:"published_by"`
	TenantID      uuid.UUID              `json:"tenant_id"`
	EntityType    string                 `json:"entity_type"`
	EntityID      uuid.UUID              `json:"entity_id"`
	EntityKey     string                 `json:"entity_key"`
	Version       int                    `json:"version"`
	SemanticLayer string                 `json:"semantic_layer"`
	Data          interface{}            `json:"data"`
	DataHash      string                 `json:"data_hash"` // sha256:<hex>
	SchemaVersion string                 `json:"schema_version"`
	ChangeType    string                 `json:"change_type"`
	ChangeReason  string                 `json:"change_reason"`
	CorrelationID string                 `json:"correlation_id"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// Publisher publishes gold copy events to the semlayer.gold-copy Kafka topic.
type Publisher struct {
	writer *kafka.Writer
}

// NewPublisher constructs a Publisher wired to the given Kafka brokers.
// brokers can be a comma-separated list or a single address such as "localhost:9092".
func NewPublisher(brokers string) *Publisher {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers),
		Topic:                  goldCopyTopic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		// Async write — caller decides whether to await
	}
	return &Publisher{writer: w}
}

// Close shuts the Kafka writer down gracefully.
func (p *Publisher) Close() error {
	return p.writer.Close()
}

// PublishPortfolioMasterGoldCopy emits a gold copy event for a portfolio master record.
func (p *Publisher) PublishPortfolioMasterGoldCopy(
	ctx context.Context,
	rec *PortfolioMasterRecord,
	changeType string, // "created" | "updated" | "closed"
	changeReason string,
	publishedBy string,
	correlationID string,
) error {
	dataHash := hashData(rec)

	evt := &GoldCopyEvent{
		EventID:       fmt.Sprintf("%s-gold.copy.portfolio.%s-%d", rec.ID, changeType, time.Now().UnixMilli()),
		EventType:     fmt.Sprintf("gold.copy.portfolio.%s", changeType),
		PublishedAt:   time.Now(),
		PublishedBy:   publishedBy,
		TenantID:      rec.TenantID,
		EntityType:    "portfolio",
		EntityID:      rec.ID,
		EntityKey:     coalesce(rec.PortfolioCode, rec.PortfolioID),
		Version:       1,
		SemanticLayer: "portfolio-master",
		Data:          rec,
		DataHash:      dataHash,
		SchemaVersion: "1.0",
		ChangeType:    changeType,
		ChangeReason:  changeReason,
		CorrelationID: correlationID,
		Metadata: map[string]interface{}{
			"confidence_score":   rec.ConfidenceScore,
			"portfolio_type":     rec.PortfolioType,
			"portfolio_category": rec.PortfolioCategory,
			"semantic_path":      []string{"Portfolio", "Mandate", "Benchmark", "Strategy"},
		},
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("goldcopy publisher: marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%s.portfolio.%s", rec.TenantID, changeType)),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "entity_type", Value: []byte("portfolio")},
			{Key: "entity_id", Value: []byte(rec.ID.String())},
			{Key: "tenant_id", Value: []byte(rec.TenantID.String())},
			{Key: "event_type", Value: []byte(evt.EventType)},
			{Key: "schema_version", Value: []byte("1.0")},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("goldcopy publisher: write message: %w", err)
	}
	return nil
}

// PublishGoldCopyRunResult emits a summary event for a completed gold copy build run.
func (p *Publisher) PublishGoldCopyRunResult(ctx context.Context, result *GoldCopyRunResult) error {
	payload, err := json.Marshal(map[string]interface{}{
		"event_type":       "gold.copy.run." + boolToStatus(result.Success),
		"run_id":           result.RunID,
		"tenant_id":        result.TenantID,
		"entity_type":      result.EntityType,
		"portfolio_id":     result.PortfolioID,
		"started_at":       result.StartedAt,
		"completed_at":     result.CompletedAt,
		"confidence_score": result.ConfidenceScore,
		"success":          result.Success,
		"dq_violations":    len(result.DQViolations),
		"error_message":    result.ErrorMessage,
	})
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(result.TenantID.String() + ".gold-copy-run"),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("gold.copy.run")},
			{Key: "run_id", Value: []byte(result.RunID.String())},
		},
	}
	return p.writer.WriteMessages(ctx, msg)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// hashData produces a SHA-256 hex digest of the JSON serialisation of v.
func hashData(v interface{}) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func boolToStatus(ok bool) string {
	if ok {
		return "completed"
	}
	return "failed"
}

// ─── Security Master Publishing ───────────────────────────────────────────────

// PublishSecurityMasterGoldCopy emits a gold copy event for a security master record.
// changeType should be "created", "updated", or "closed".
func (p *Publisher) PublishSecurityMasterGoldCopy(
	ctx context.Context,
	rec *SecurityMasterRecord,
	changeType string,
	changeReason string,
	publishedBy string,
	correlationID string,
) error {
	dataHash := hashData(rec)
	primaryKey := firstNonEmpty(rec.ISIN, rec.FIGI, rec.CUSIP, rec.Ticker, rec.SecurityID)

	evt := &GoldCopyEvent{
		EventID:       fmt.Sprintf("%s-gold.copy.security.%s-%d", rec.ID, changeType, time.Now().UnixMilli()),
		EventType:     fmt.Sprintf("gold.copy.security.%s", changeType),
		PublishedAt:   time.Now(),
		PublishedBy:   publishedBy,
		TenantID:      rec.TenantID,
		EntityType:    "security",
		EntityID:      rec.ID,
		EntityKey:     primaryKey,
		Version:       1,
		SemanticLayer: "security-master",
		Data:          rec,
		DataHash:      dataHash,
		SchemaVersion: "1.0",
		ChangeType:    changeType,
		ChangeReason:  changeReason,
		CorrelationID: correlationID,
		Metadata: map[string]interface{}{
			"confidence_score":   rec.ConfidenceScore,
			"asset_class":        rec.AssetClass,
			"sub_asset_class":    rec.SubAssetClass,
			"instrument_type":    rec.InstrumentType,
			"isin":               rec.ISIN,
			"figi":               rec.FIGI,
			"primary_identifier": primaryKey,
			"currency":           rec.Currency,
			"country_of_issue":   rec.CountryOfIssue,
			"listing_exchange":   rec.ListingExchange,
			"semantic_path":      []string{"Security", "Issuer", "AssetClass", "Instrument"},
			"source_systems":     rec.SourceSystems,
		},
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("goldcopy publisher: marshal security event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%s.security.%s", rec.TenantID, changeType)),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "entity_type", Value: []byte("security")},
			{Key: "entity_id", Value: []byte(rec.ID.String())},
			{Key: "tenant_id", Value: []byte(rec.TenantID.String())},
			{Key: "event_type", Value: []byte(evt.EventType)},
			{Key: "schema_version", Value: []byte("1.0")},
			{Key: "asset_class", Value: []byte(rec.AssetClass)},
			{Key: "primary_identifier", Value: []byte(primaryKey)},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("goldcopy publisher: write security message: %w", err)
	}
	return nil
}

// PublishSecurityGoldCopyRunResult emits a summary event for a completed security gold copy run.
func (p *Publisher) PublishSecurityGoldCopyRunResult(ctx context.Context, result *SecurityGoldCopyRunResult) error {
	payload, err := json.Marshal(map[string]interface{}{
		"event_type":       "gold.copy.security.run." + boolToStatus(result.Success),
		"run_id":           result.RunID,
		"tenant_id":        result.TenantID,
		"entity_type":      result.EntityType,
		"cluster_key":      result.ClusterKey,
		"started_at":       result.StartedAt,
		"completed_at":     result.CompletedAt,
		"confidence_score": result.ConfidenceScore,
		"success":          result.Success,
		"dq_violations":    len(result.DQViolations),
		"survivorship_log": len(result.SurvivorshipLog),
		"error_message":    result.ErrorMessage,
	})
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(result.TenantID.String() + ".security-gold-copy-run"),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("gold.copy.security.run")},
			{Key: "run_id", Value: []byte(result.RunID.String())},
			{Key: "entity_type", Value: []byte("security")},
		},
	}
	return p.writer.WriteMessages(ctx, msg)
}
