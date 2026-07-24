package meta

import "time"

// FieldType represents the data type of a business object field
type FieldType string

const (
	FieldString  FieldType = "string"
	FieldDecimal FieldType = "decimal"
	FieldDate    FieldType = "date"
	FieldEnum    FieldType = "enum"
	FieldRef     FieldType = "ref"
	FieldJSON    FieldType = "json"
	FieldBoolean FieldType = "boolean"
	FieldInteger FieldType = "integer"
)

// BusinessObjectDefinition defines a tenant-specific business object
// Enhanced to follow Workday standards with lifecycle and versioning
type BusinessObjectDefinition struct {
	ID            string                   `json:"id"`
	TenantID      string                   `json:"tenant_id"`
	Name          string                   `json:"name"`
	DisplayName   string                   `json:"display_name"`
	Description   string                   `json:"description"`
	Icon          string                   `json:"icon"`
	Storage       string                   `json:"storage"` // row|wide_jsonb|eav
	Version       int                      `json:"version"`
	Status        string                   `json:"status"` // draft|active|deprecated|archived
	Lifecycle     *LifecycleConfig         `json:"lifecycle,omitempty"`
	Fields        []FieldDefinition        `json:"fields"`
	Relationships []RelationshipDefinition `json:"relationships"`
	Metadata      map[string]any           `json:"metadata,omitempty"`

	// Workday-style metadata
	IsCore   bool   `json:"is_core"`
	Category string `json:"category"`

	// Caching metadata
	CachedAt  time.Time `json:"cached_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FieldDefinition defines a field within a business object
type FieldDefinition struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	BusinessObjectID string    `json:"business_object_id"`
	Name             string    `json:"name"`
	Label            string    `json:"label"`
	Type             FieldType `json:"type"`
	IsRequired       bool      `json:"is_required"`
	IsUnique         bool      `json:"is_unique"`
	EnumID           *string   `json:"enum_id,omitempty"`
	RefObjectID      *string   `json:"ref_object_id,omitempty"`
	ValidationJSON   []byte    `json:"validation_json"` // zod-like JSON schema
	VisibilityJSON   []byte    `json:"visibility_json"` // UI visibility rules (CEL)
	DefaultValue     *string   `json:"default_value,omitempty"`
}

// RelationshipDefinition defines relationships between business objects
type RelationshipDefinition struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	ParentObjectID  string         `json:"parent_object_id"`
	ChildObjectID   string         `json:"child_object_id"`
	Cardinality     string         `json:"cardinality"` // 1:N, N:M
	CascadeRules    map[string]any `json:"cascade_rules,omitempty"`
	AggregationType *string        `json:"aggregation_type,omitempty"` // sum, count, avg
}

// EnumDefinition defines an enumeration
type EnumDefinition struct {
	ID       string      `json:"id"`
	TenantID string      `json:"tenant_id"`
	Name     string      `json:"name"`
	Values   []EnumValue `json:"values"`
}

// EnumValue represents a single enum value
type EnumValue struct {
	Value       string         `json:"value"`
	Label       string         `json:"label"`
	Description *string        `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// PolicyDefinition defines cross-cutting policies
type PolicyDefinition struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	Scope      string         `json:"scope"`      // object, field, workflow, ai_tool
	Expression string         `json:"expression"` // CEL or JSONLogic
	Type       string         `json:"type"`       // authorization, data_residency, retention
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// LifecycleConfig defines the lifecycle states and transitions for a business object
type LifecycleConfig struct {
	States      []LifecycleState  `json:"states"`
	Transitions []StateTransition `json:"transitions"`
}

// LifecycleState represents a single state in the lifecycle
type LifecycleState struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Description string         `json:"description,omitempty"`
	IsInitial   bool           `json:"is_initial"`
	IsFinal     bool           `json:"is_final"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// StateTransition defines allowed transitions between states
type StateTransition struct {
	FromState    string         `json:"from_state"`
	ToState      string         `json:"to_state"`
	Label        string         `json:"label"`
	Condition    string         `json:"condition,omitempty"` // CEL expression
	RequiredRole string         `json:"required_role,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}
