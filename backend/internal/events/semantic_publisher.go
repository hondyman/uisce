package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// SemanticChangeEvent represents any mutation to the semantic layer
type SemanticChangeEvent struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	TenantID        string                 `json:"tenant_id"`
	UserID          string                 `json:"user_id"`
	ChangeType      string                 `json:"change_type"` // model_created, model_updated, measure_added, dimension_changed, join_modified, model_deleted, term_approved, term_rejected, term_created
	ModelID         string                 `json:"model_id,omitempty"`
	ModelName       string                 `json:"model_name,omitempty"`
	ElementType     string                 `json:"element_type"` // measure, dimension, join, model, semantic_term
	ElementID       string                 `json:"element_id"`
	ElementName     string                 `json:"element_name"`
	OldDefinition   json.RawMessage        `json:"old_definition,omitempty"`
	NewDefinition   json.RawMessage        `json:"new_definition,omitempty"`
	ChangeReason    string                 `json:"change_reason,omitempty"`
	ImpactedQueries int                    `json:"impacted_queries,omitempty"`
	SQLChanges      *SQLChangeDetail       `json:"sql_changes,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// SQLChangeDetail tracks SQL compilation changes
type SQLChangeDetail struct {
	OldSQL         string   `json:"old_sql"`
	NewSQL         string   `json:"new_sql"`
	DiffSummary    string   `json:"diff_summary"`
	BreakingChange bool     `json:"breaking_change"`
	AffectedTables []string `json:"affected_tables"`
}

// SemanticPublisher publishes semantic layer events to Kafka
type SemanticPublisher struct {
	writer *kafka.Writer
}

// NewSemanticPublisher creates a new publisher
func NewSemanticPublisher(brokers string) (*SemanticPublisher, error) {
	brokerList := strings.Split(brokers, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerList...),
		Balancer: &kafka.LeastBytes{},
	}

	return &SemanticPublisher{
		writer: w,
	}, nil
}

// PublishEvent publishes a semantic change event to the appropriate topic
func (p *SemanticPublisher) PublishEvent(ctx context.Context, event *SemanticChangeEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Determine topic based on event type
	topic := "semantic.changes"
	if event.ElementType == "semantic_term" {
		topic = "semantic.audit"
	}

	// Construct a key for partitioning (e.g., TenantID or ElementID)
	key := event.TenantID
	if key == "" {
		key = event.ElementID
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: body,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "change_type", Value: []byte(event.ChangeType)},
			{Key: "element_type", Value: []byte(event.ElementType)},
			{Key: "user_id", Value: []byte(event.UserID)},
		},
	}

	return p.writer.WriteMessages(ctx, msg)
}

// PublishModelChange publishes a model change event (compatibility wrapper)
func (p *SemanticPublisher) PublishModelChange(ctx context.Context, event *SemanticChangeEvent) error {
	return p.PublishEvent(ctx, event)
}

// PublishTermEvent publishes a semantic term event (audit/feedback)
func (p *SemanticPublisher) PublishTermEvent(ctx context.Context, tenantID, userID, changeType string, mappingID string, termName string, metadata map[string]interface{}) error {
	event := &SemanticChangeEvent{
		ID:          mappingID, // Use mapping/term ID as event ID
		Timestamp:   time.Now(),
		TenantID:    tenantID,
		UserID:      userID,
		ChangeType:  changeType,
		ElementType: "semantic_term",
		ElementID:   mappingID,
		ElementName: termName,
		Metadata:    metadata,
	}
	return p.PublishEvent(ctx, event)
}

// PublishDriftEvent publishes a drift detection event
func (p *SemanticPublisher) PublishDriftEvent(ctx context.Context, tenantID, modelID, severity string, issues int) error {
	event := map[string]interface{}{
		"timestamp":   time.Now(),
		"tenant_id":   tenantID,
		"model_id":    modelID,
		"severity":    severity,
		"issue_count": issues,
		"type":        "drift_detected",
	}

	body, _ := json.Marshal(event)

	msg := kafka.Message{
		Topic: "semantic.drift",
		Key:   []byte(tenantID),
		Value: body,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

// Close closes the publisher connection
func (p *SemanticPublisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// CacheManager interface for cache invalidation
type CacheManager interface {
	InvalidatePattern(ctx context.Context, pattern string) error
	Delete(ctx context.Context, key string) error
}

// CacheInvalidationSubscriber listens to semantic changes and invalidates cache
type CacheInvalidationSubscriber struct {
	reader *kafka.Reader
	cache  CacheManager
}

// NewCacheInvalidationSubscriber creates cache invalidation subscriber
func NewCacheInvalidationSubscriber(brokers string, cache CacheManager) (*CacheInvalidationSubscriber, error) {
	brokerList := strings.Split(brokers, ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokerList,
		GroupID:  "semantic-cache-invalidator",
		Topic:    "semantic.changes",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &CacheInvalidationSubscriber{
		reader: r,
		cache:  cache,
	}, nil
}

// Start starts listening for cache invalidation events
func (s *CacheInvalidationSubscriber) Start(ctx context.Context) error {
	log.Printf("✅ Cache invalidation subscriber started (Kafka)")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := s.reader.ReadMessage(ctx)
				if err != nil {
					log.Printf("❌ Failed to read Kafka message: %v", err)
					// Avoid busy loop on error
					time.Sleep(1 * time.Second)
					continue
				}

				var event SemanticChangeEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					log.Printf("❌ Failed to unmarshal event: %v", err)
					continue
				}

				// Invalidate cache based on change
				s.invalidateRelevantCaches(ctx, &event)
			}
		}
	}()

	return nil
}

// invalidateRelevantCaches invalidates caches affected by the change
func (s *CacheInvalidationSubscriber) invalidateRelevantCaches(ctx context.Context, event *SemanticChangeEvent) {
	patterns := []string{
		fmt.Sprintf("semantic:model:%s:*", event.ModelID),
		fmt.Sprintf("semantic:tenant:%s:*", event.TenantID),
		fmt.Sprintf("semantic:query_results:*:%s:*", event.ModelID),
		fmt.Sprintf("semantic:metadata:model:%s", event.ModelID),
	}

	for _, pattern := range patterns {
		if err := s.cache.InvalidatePattern(ctx, pattern); err != nil {
			log.Printf("⚠️  Failed to invalidate cache pattern %s: %v", pattern, err)
		} else {
			log.Printf("✅ Invalidated cache pattern: %s", pattern)
		}
	}
}

// Close closes the subscriber connection
func (s *CacheInvalidationSubscriber) Close() error {
	if s.reader != nil {
		return s.reader.Close()
	}
	return nil
}
