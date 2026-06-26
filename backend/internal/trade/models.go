package trade

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WorkflowStatus represents the status of a workflow definition
type WorkflowStatus string

const (
	WorkflowStatusDraft   WorkflowStatus = "draft"
	WorkflowStatusActive  WorkflowStatus = "active"
	WorkflowStatusRetired WorkflowStatus = "retired"
)

// WorkflowDefinition represents a trade workflow definition
type WorkflowDefinition struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Status         WorkflowStatus  `json:"status"`
	Stages         json.RawMessage `json:"stages"` // Array of stage definitions (simplified view)
	CreatedAt      time.Time       `json:"created_at"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	LastModifiedAt time.Time       `json:"last_modified_at"`
	LastModifiedBy *uuid.UUID      `json:"last_modified_by,omitempty"`
}

// WorkflowStage represents a stage within a workflow
type WorkflowStage struct {
	ID         uuid.UUID       `json:"id"`
	WorkflowID uuid.UUID       `json:"workflow_id"`
	Name       string          `json:"name"`
	OrderIndex int             `json:"order_index"`
	Config     json.RawMessage `json:"config"` // UI layout, actors, triggers
	CreatedAt  time.Time       `json:"created_at"`
}

// ComplianceRule represents a compliance rule linked to a workflow
type ComplianceRule struct {
	ID          uuid.UUID       `json:"id"`
	WorkflowID  uuid.UUID       `json:"workflow_id"`
	RuleCode    string          `json:"rule_code"`
	Description string          `json:"description,omitempty"`
	Config      json.RawMessage `json:"config"` // Thresholds, logic
	CreatedAt   time.Time       `json:"created_at"`
}

// TradeInput represents the input payload to start a trade
type TradeInput struct {
	TenantID     string          `json:"tenant_id"`
	WorkflowName string          `json:"workflow_name"`
	Data         json.RawMessage `json:"data"`
}
