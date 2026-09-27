package survivorship

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SourceSystem is one entry in public.mdm_source_systems.
type SourceSystem struct {
	Code        string    `db:"code" json:"code"`
	DisplayName string    `db:"display_name" json:"display_name"`
	DefaultRank int       `db:"default_rank" json:"default_rank"`
	Description string    `db:"description" json:"description"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// Rule is one public.semantic_survivorship_rules row.
type Rule struct {
	ID              uuid.UUID      `db:"id" json:"id"`
	TenantID        uuid.UUID      `db:"tenant_id" json:"tenant_id"`
	EntityType      string         `db:"entity_type" json:"entity_type"`
	SemanticTermID  uuid.UUID      `db:"semantic_term_id" json:"semantic_term_id"`
	Strategy        string         `db:"strategy" json:"strategy"`
	PriorityOrder   pq.StringArray `db:"priority_order" json:"priority_order"`
	MaxStaleSeconds int            `db:"max_stale_seconds" json:"max_stale_seconds"`
	IsActive        bool           `db:"is_active" json:"is_active"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at" json:"updated_at"`
	// Joined for UI
	SemanticTermName string `db:"semantic_term_name" json:"semantic_term_name,omitempty"`
	SemanticTermPath string `db:"semantic_term_path" json:"semantic_term_path,omitempty"`
}

// CreateRuleInput is the steward create payload.
type CreateRuleInput struct {
	EntityType      string   `json:"entity_type"`
	SemanticTermID  string   `json:"semantic_term_id"`
	Strategy        string   `json:"strategy"`
	PriorityOrder   []string `json:"priority_order"`
	MaxStaleSeconds int      `json:"max_stale_seconds"`
}

// UpdateRuleInput patches an existing rule.
type UpdateRuleInput struct {
	Strategy        *string  `json:"strategy,omitempty"`
	PriorityOrder   []string `json:"priority_order,omitempty"`
	MaxStaleSeconds *int     `json:"max_stale_seconds,omitempty"`
	IsActive        *bool    `json:"is_active,omitempty"`
}

// FieldRuleResolved maps a physical field to an engine-ready rule.
type FieldRuleResolved struct {
	FieldCd        string
	SemanticTermID uuid.UUID
	Strategy       string
	PriorityOrder  []string
	MaxStaleSecs   int
}
