package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// GOLD COPY EVENT TYPES
// ============================================================================

type GoldCopyEventType string

const (
	// Gold Copy Events - Published when entities become canonical/authoritative
	EventGoldCopyRuleCreated    GoldCopyEventType = "gold.copy.rule.created"
	EventGoldCopyRuleUpdated    GoldCopyEventType = "gold.copy.rule.updated"
	EventGoldCopyRuleDeprecated GoldCopyEventType = "gold.copy.rule.deprecated"
	EventGoldCopyRuleRetired    GoldCopyEventType = "gold.copy.rule.retired"

	EventGoldCopyTemplateCreated GoldCopyEventType = "gold.copy.template.created"
	EventGoldCopyTemplateUpdated GoldCopyEventType = "gold.copy.template.updated"
	EventGoldCopyTemplateRetired GoldCopyEventType = "gold.copy.template.retired"

	EventGoldCopyPreferenceCreated GoldCopyEventType = "gold.copy.preference.created"
	EventGoldCopyPreferenceUpdated GoldCopyEventType = "gold.copy.preference.updated"
	EventGoldCopyPreferenceRetired GoldCopyEventType = "gold.copy.preference.retired"

	EventGoldCopyBusinessObjectCreated GoldCopyEventType = "gold.copy.business_object.created"
	EventGoldCopyBusinessObjectUpdated GoldCopyEventType = "gold.copy.business_object.updated"
	EventGoldCopyBusinessObjectRetired GoldCopyEventType = "gold.copy.business_object.retired"

	EventGoldCopySecurityCreated GoldCopyEventType = "gold.copy.security.created"
	EventGoldCopySecurityUpdated GoldCopyEventType = "gold.copy.security.updated"
	EventGoldCopySecurityRetired GoldCopyEventType = "gold.copy.security.retired"
)

// GoldCopyEvent represents a canonical data entity published to downstream systems
type GoldCopyEvent struct {
	// Event Metadata
	EventID     string            `json:"event_id"`
	EventType   GoldCopyEventType `json:"event_type"`
	PublishedAt time.Time         `json:"published_at"`
	PublishedBy string            `json:"published_by"` // User ID who promoted to gold copy

	// Entity Identification
	TenantID      string `json:"tenant_id"`
	EntityType    string `json:"entity_type"` // rule, template, preference, business_object
	EntityID      string `json:"entity_id"`
	EntityKey     string `json:"entity_key"`     // Semantic term, BO key, etc.
	Version       int    `json:"version"`        // Entity version at time of gold copy
	SemanticLayer string `json:"semantic_layer"` // Which layer (e.g., "calendar", "pricing")

	// Canonical Data
	Data          interface{} `json:"data"`           // Full entity data (rule, template, etc.)
	DataHash      string      `json:"data_hash"`      // SHA256 for change detection
	SchemaVersion string      `json:"schema_version"` // JSON schema version for consumers

	// Lineage & Audit
	ParentEntityID string `json:"parent_entity_id,omitempty"` // If cloned/derived from
	ChangeType     string `json:"change_type"`                // "creation", "update", "deprecation"
	ChangeReason   string `json:"change_reason,omitempty"`

	// Operational Context
	CorrelationID string                 `json:"correlation_id"` // Link events across workflow
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ============================================================================
// GOLD COPY PUBLISHER
// ============================================================================

// GoldCopyPublisher publishes gold copy entities to Redpanda for downstream consumption
type GoldCopyPublisher struct {
	eventPublisher *EventPublisher
	writer         *kafka.Writer
	topic          string
	enabled        bool
}

// NewGoldCopyPublisher creates a new gold copy publisher
func NewGoldCopyPublisher(brokersOrURL string) (*GoldCopyPublisher, error) {
	// Create base event publisher
	basePublisher, err := NewEventPublisher(brokersOrURL)
	if err != nil {
		log.Printf("Warning: Failed to create base event publisher: %v", err)
		return &GoldCopyPublisher{enabled: false}, nil
	}

	if !basePublisher.enabled {
		log.Println("⚠️  Gold copy publisher not configured - gold copies will not be published")
		return &GoldCopyPublisher{enabled: false}, nil
	}

	// Create dedicated writer for gold copy topic
	if brokersOrURL != "" {
		w := &kafka.Writer{
			Addr:     kafka.TCP(brokersOrURL),
			Balancer: &kafka.LeastBytes{},
			Topic:    "semlayer.gold-copy", // Dedicated topic for gold copies
		}
		return &GoldCopyPublisher{
			eventPublisher: basePublisher,
			writer:         w,
			topic:          "semlayer.gold-copy",
			enabled:        true,
		}, nil
	}

	return &GoldCopyPublisher{enabled: false}, nil
}

// PublishGoldCopyEvent publishes a gold copy event to Redpanda
func (gcp *GoldCopyPublisher) PublishGoldCopyEvent(ctx context.Context, event *GoldCopyEvent) error {
	if !gcp.enabled {
		return nil // Silently skip if not enabled
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal gold copy event: %w", err)
	}

	// Route by entity type and event type
	routingKey := fmt.Sprintf("%s.%s.%s", event.TenantID, event.EntityType, event.EventType)
	msg := kafka.Message{
		Topic: gcp.topic,
		Key:   []byte(routingKey),
		Value: data,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{
				Key:   "entity_type",
				Value: []byte(event.EntityType),
			},
			{
				Key:   "entity_id",
				Value: []byte(event.EntityID),
			},
			{
				Key:   "tenant_id",
				Value: []byte(event.TenantID),
			},
			{
				Key:   "event_type",
				Value: []byte(string(event.EventType)),
			},
		},
	}

	if err := gcp.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("❌ Failed to publish gold copy event: %v", err)
		return err
	}

	log.Printf("✅ Published gold copy event: %s for %s %s (%s)",
		event.EventType, event.EntityType, event.EntityID, event.TenantID)
	return nil
}

// PublishRuleAsGoldCopy publishes a rule that has been promoted to gold copy status
func (gcp *GoldCopyPublisher) PublishRuleAsGoldCopy(
	ctx context.Context,
	rule *models.Rule,
	changeType string,
	changeReason string,
	publishedByUserID string,
	dataHash string,
) error {
	if !gcp.enabled {
		return nil
	}

	eventType := EventGoldCopyRuleCreated
	if changeType == "update" {
		eventType = EventGoldCopyRuleUpdated
	} else if changeType == "deprecation" {
		eventType = EventGoldCopyRuleDeprecated
	} else if changeType == "retirement" {
		eventType = EventGoldCopyRuleRetired
	}

	event := &GoldCopyEvent{
		EventID:       fmt.Sprintf("%s-%s-%d", rule.ID, eventType, time.Now().Unix()),
		EventType:     eventType,
		PublishedAt:   time.Now(),
		PublishedBy:   publishedByUserID,
		TenantID:      rule.TenantID,
		EntityType:    "rule",
		EntityID:      rule.ID,
		EntityKey:     rule.SemanticTerm,
		Version:       rule.Version,
		SemanticLayer: "semantic-rules",
		Data:          rule,
		DataHash:      dataHash,
		SchemaVersion: "1.0",
		ChangeType:    changeType,
		ChangeReason:  changeReason,
		Metadata: map[string]interface{}{
			"status":              rule.Status,
			"semantic_term":       rule.SemanticTerm,
			"rule_engine":         rule.RuleEngine,
			"expression_language": rule.ExpressionLanguage,
		},
	}

	return gcp.PublishGoldCopyEvent(ctx, event)
}

// PublishTemplateAsGoldCopy publishes a template that has been promoted to gold copy status
func (gcp *GoldCopyPublisher) PublishTemplateAsGoldCopy(
	ctx context.Context,
	template *models.Template,
	changeType string,
	changeReason string,
	publishedByUserID string,
	dataHash string,
) error {
	if !gcp.enabled {
		return nil
	}

	eventType := EventGoldCopyTemplateCreated
	if changeType == "update" {
		eventType = EventGoldCopyTemplateUpdated
	} else if changeType == "retirement" {
		eventType = EventGoldCopyTemplateRetired
	}

	event := &GoldCopyEvent{
		EventID:       fmt.Sprintf("%s-%s-%d", template.ID, eventType, time.Now().Unix()),
		EventType:     eventType,
		PublishedAt:   time.Now(),
		PublishedBy:   publishedByUserID,
		TenantID:      template.TenantID,
		EntityType:    "template",
		EntityID:      template.ID,
		EntityKey:     template.Name,
		Version:       template.Version,
		SemanticLayer: "template-library",
		Data:          template,
		DataHash:      dataHash,
		SchemaVersion: "1.0",
		ChangeType:    changeType,
		ChangeReason:  changeReason,
		Metadata: map[string]interface{}{
			"template_type": template.TemplateType,
			"category":      template.Category,
			"rule_count":    len(template.RuleIDs),
		},
	}

	return gcp.PublishGoldCopyEvent(ctx, event)
}

// PublishPreferenceAsGoldCopy publishes a preference (source preference, etc.) as gold copy
func (gcp *GoldCopyPublisher) PublishPreferenceAsGoldCopy(
	ctx context.Context,
	tenantID string,
	preferenceID string,
	preferenceKey string,
	preferenceType string,
	data interface{},
	changeType string,
	changeReason string,
	publishedByUserID string,
	dataHash string,
) error {
	if !gcp.enabled {
		return nil
	}

	eventType := EventGoldCopyPreferenceCreated
	if changeType == "update" {
		eventType = EventGoldCopyPreferenceUpdated
	} else if changeType == "retirement" {
		eventType = EventGoldCopyPreferenceRetired
	}

	event := &GoldCopyEvent{
		EventID:       fmt.Sprintf("%s-%s-%d", preferenceID, eventType, time.Now().Unix()),
		EventType:     eventType,
		PublishedAt:   time.Now(),
		PublishedBy:   publishedByUserID,
		TenantID:      tenantID,
		EntityType:    "preference",
		EntityID:      preferenceID,
		EntityKey:     preferenceKey,
		Version:       1,
		SemanticLayer: "preferences",
		Data:          data,
		DataHash:      dataHash,
		SchemaVersion: "1.0",
		ChangeType:    changeType,
		ChangeReason:  changeReason,
		Metadata: map[string]interface{}{
			"preference_type": preferenceType,
		},
	}

	return gcp.PublishGoldCopyEvent(ctx, event)
}

// PublishBusinessObjectAsGoldCopy publishes a business object as gold copy
func (gcp *GoldCopyPublisher) PublishBusinessObjectAsGoldCopy(
	ctx context.Context,
	bo *models.BusinessObjectDefinition,
	changeType string,
	changeReason string,
	publishedByUserID string,
	dataHash string,
) error {
	if !gcp.enabled {
		return nil
	}

	eventType := EventGoldCopyBusinessObjectCreated
	if changeType == "update" {
		eventType = EventGoldCopyBusinessObjectUpdated
	} else if changeType == "retirement" {
		eventType = EventGoldCopyBusinessObjectRetired
	}

	event := &GoldCopyEvent{
		EventID:       fmt.Sprintf("%s-%s-%d", bo.ID, eventType, time.Now().Unix()),
		EventType:     eventType,
		PublishedAt:   time.Now(),
		PublishedBy:   publishedByUserID,
		TenantID:      bo.TenantID,
		EntityType:    "business_object",
		EntityID:      bo.ID,
		EntityKey:     bo.Key,
		Version:       1,
		SemanticLayer: "business-objects",
		Data:          bo,
		DataHash:      dataHash,
		SchemaVersion: "1.0",
		ChangeType:    changeType,
		ChangeReason:  changeReason,
		Metadata: map[string]interface{}{
			"display_name": bo.DisplayName,
			"bo_category":  bo.Category,
			"field_count":  len(bo.CoreFields) + len(bo.CustomFields),
			"is_core":      bo.IsCore,
		},
	}

	return gcp.PublishGoldCopyEvent(ctx, event)
}

// Close closes the underlying Kafka writer
func (gcp *GoldCopyPublisher) Close() error {
	if gcp.writer != nil {
		return gcp.writer.Close()
	}
	return nil
}

// PublishSecurityAsGoldCopy publishes a security master record as gold copy
func (gcp *GoldCopyPublisher) PublishSecurityAsGoldCopy(
	ctx context.Context,
	security interface{}, // Using interface{} to avoid circular dependency, map to models/goldcopy in repository
	tenantID string,
	securityID string,
	primaryIdentifier string,
	changeType string,
	changeReason string,
	publishedByUserID string,
	dataHash string,
) error {
	if !gcp.enabled {
		return nil
	}

	eventType := EventGoldCopySecurityCreated
	if changeType == "update" {
		eventType = EventGoldCopySecurityUpdated
	} else if changeType == "retirement" {
		eventType = EventGoldCopySecurityRetired
	}

	event := &GoldCopyEvent{
		EventID:       fmt.Sprintf("%s-%s-%d", securityID, eventType, time.Now().Unix()),
		EventType:     eventType,
		PublishedAt:   time.Now(),
		PublishedBy:   publishedByUserID,
		TenantID:      tenantID,
		EntityType:    "security",
		EntityID:      securityID,
		EntityKey:     primaryIdentifier,
		Version:       1,
		SemanticLayer: "security-master",
		Data:          security,
		DataHash:      dataHash,
		SchemaVersion: "1.0",
		ChangeType:    changeType,
		ChangeReason:  changeReason,
		Metadata: map[string]interface{}{
			"semantic_path": fmt.Sprintf("tenant/%s/security/%s", tenantID, securityID),
		},
	}

	return gcp.PublishGoldCopyEvent(ctx, event)
}
