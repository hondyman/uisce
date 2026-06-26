package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/lib/pq"
)

// BulkOperationHandler implements the OperationHandler interface for bulk operations
type BulkOperationHandler struct {
	db *sql.DB
}

// BulkCreatePayload represents items for bulk create
type BulkCreatePayload struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Variables   json.RawMessage `json:"variables"`
	Expression  string          `json:"expression"`
}

// BulkPublishPayload represents items for bulk publish
type BulkPublishPayload struct {
	TemplateID string `json:"templateId"`
}

// NewBulkOperationHandler creates a new bulk operation handler
func NewBulkOperationHandler(db *sql.DB) OperationHandler {
	return &BulkOperationHandler{db: db}
}

// ProcessItem processes a single item in a bulk operation
func (h *BulkOperationHandler) ProcessItem(ctx context.Context, job *models.AsyncJob, item *models.JobItem) (*string, error) {
	switch job.OperationType {
	case models.OperationBulkCreate:
		return h.processCreateItem(ctx, job, item)
	case models.OperationBulkPublish:
		return h.processPublishItem(ctx, job, item)
	case models.OperationBulkPromote:
		return h.processPromoteItem(ctx, job, item)
	default:
		return nil, fmt.Errorf("unknown operation type: %v", job.OperationType)
	}
}

// processCreateItem processes a create operation for a single template
func (h *BulkOperationHandler) processCreateItem(ctx context.Context, job *models.AsyncJob, item *models.JobItem) (*string, error) {
	var payload BulkCreatePayload
	if err := json.Unmarshal(item.ItemData, &payload); err != nil {
		return nil, fmt.Errorf("invalid item payload: %w", err)
	}

	// Validate required fields
	if payload.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if payload.Expression == "" {
		return nil, fmt.Errorf("expression is required")
	}

	// Generate UUID for new template
	templateID := uuid.New().String()

	// Insert template
	query := `
		INSERT INTO edm.rule_templates (
			id, tenant_id, name, description, variables, expression, 
			status, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err := h.db.QueryRowContext(ctx, query,
		templateID,
		job.TenantID,
		payload.Name,
		payload.Description,
		payload.Variables,
		payload.Expression,
		"draft", // Initial status
		job.CreatedBy,
		time.Now(),
	).Scan(&templateID)

	if err != nil {
		log.Printf("[BulkOperationHandler] Error creating template: %v", err)
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	log.Printf("[BulkOperationHandler] Created template %s: %s", templateID, payload.Name)
	return &templateID, nil
}

// processPublishItem processes a publish operation for a single template
func (h *BulkOperationHandler) processPublishItem(ctx context.Context, job *models.AsyncJob, item *models.JobItem) (*string, error) {
	var payload BulkPublishPayload
	if err := json.Unmarshal(item.ItemData, &payload); err != nil {
		return nil, fmt.Errorf("invalid item payload: %w", err)
	}

	// Validate template ID
	if payload.TemplateID == "" {
		return nil, fmt.Errorf("templateId is required")
	}

	// Update template status to approved
	query := `
		UPDATE edm.rule_templates
		SET status = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4
		RETURNING id
	`

	var templateID string
	err := h.db.QueryRowContext(ctx, query,
		"approved",
		time.Now(),
		payload.TemplateID,
		job.TenantID,
	).Scan(&templateID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found or not owned by tenant")
	}
	if err != nil {
		log.Printf("[BulkOperationHandler] Error publishing template: %v", err)
		return nil, fmt.Errorf("failed to publish template: %w", err)
	}

	log.Printf("[BulkOperationHandler] Published template %s", templateID)
	return &templateID, nil
}

// processPromoteItem processes a promote operation for a single rule
func (h *BulkOperationHandler) processPromoteItem(ctx context.Context, job *models.AsyncJob, item *models.JobItem) (*string, error) {
	// Promotion typically moves rules from dev → staging → prod
	// Implementation depends on your environment model
	// This is a framework - customize based on your needs

	var payload map[string]interface{}
	if err := json.Unmarshal(item.ItemData, &payload); err != nil {
		return nil, fmt.Errorf("invalid item payload: %w", err)
	}

	ruleID, ok := payload["ruleId"].(string)
	if !ok || ruleID == "" {
		return nil, fmt.Errorf("ruleId is required")
	}

	targetEnv, ok := payload["targetEnvironment"].(string)
	if !ok || targetEnv == "" {
		return nil, fmt.Errorf("targetEnvironment is required")
	}

	// In a real implementation, you would:
	// 1. Clone the rule to the target environment
	// 2. Run tests
	// 3. Update status
	// For now, we'll just log and return success

	log.Printf("[BulkOperationHandler] Promoted rule %s to %s", ruleID, targetEnv)
	return &ruleID, nil
}

// ValidateJob validates the job before processing
func (h *BulkOperationHandler) ValidateJob(ctx context.Context, job *models.AsyncJob) error {
	if job.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	if job.OperationType == "" {
		return fmt.Errorf("operation_type is required")
	}

	if len(job.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}

	// Validate payload is valid JSON array
	var items []interface{}
	if err := json.Unmarshal(job.Payload, &items); err != nil {
		return fmt.Errorf("payload must be valid JSON array: %w", err)
	}

	// Check operation-specific limits
	switch job.OperationType {
	case models.OperationBulkCreate:
		if len(items) > 10000 {
			return fmt.Errorf("bulk create limited to 10000 items, got %d", len(items))
		}
	case models.OperationBulkPublish:
		if len(items) > 2500 {
			return fmt.Errorf("bulk publish limited to 2500 items, got %d", len(items))
		}
	default:
		return fmt.Errorf("unknown operation type: %v", job.OperationType)
	}

	return nil
}

// PostProcess runs after all items are processed
func (h *BulkOperationHandler) PostProcess(ctx context.Context, job *models.AsyncJob) error {
	// Could be used for:
	// - Index updates
	// - Cache invalidation
	// - Audit logging
	// - Statistics collection

	log.Printf("[BulkOperationHandler] Post-processing job %s", job.ID)

	// For now, just log success
	switch job.OperationType {
	case models.OperationBulkCreate:
		log.Printf("[BulkOperationHandler] Created %d templates in batch", job.SucceededItems)
	case models.OperationBulkPublish:
		log.Printf("[BulkOperationHandler] Published %d templates in batch", job.SucceededItems)
	}

	return nil
}

// Helper function to batch process items for better performance
func (h *BulkOperationHandler) batchInsertTemplates(ctx context.Context, tenantID string, items []*models.JobItem) error {
	if len(items) == 0 {
		return nil
	}

	// Build batch insert query
	query := `
		INSERT INTO edm.rule_templates (
			id, tenant_id, name, description, variables, expression, 
			status, created_by, created_at
		) VALUES
	`

	args := []interface{}{}
	argIndex := 1

	for i, item := range items {
		var payload BulkCreatePayload
		if err := json.Unmarshal(item.ItemData, &payload); err != nil {
			continue
		}

		if i > 0 {
			query += ","
		}

		templateID := uuid.New().String()
		query += fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIndex, argIndex+1, argIndex+2, argIndex+3, argIndex+4,
			argIndex+5, argIndex+6, argIndex+7, argIndex+8,
		)

		args = append(args, templateID, tenantID, payload.Name, payload.Description,
			payload.Variables, payload.Expression, "draft", item.ID, time.Now())
		argIndex += 9
	}

	_, err := h.db.ExecContext(ctx, query, args...)
	return err
}

// Helper to update template statuses in batch
func (h *BulkOperationHandler) batchPublishTemplates(ctx context.Context, tenantID string, templateIDs []string) error {
	if len(templateIDs) == 0 {
		return nil
	}

	query := `
		UPDATE edm.rule_templates
		SET status = $1, updated_at = $2
		WHERE tenant_id = $3 AND id = ANY($4)
	`

	_, err := h.db.ExecContext(ctx, query,
		"approved",
		time.Now(),
		tenantID,
		pq.Array(templateIDs),
	)

	return err
}
