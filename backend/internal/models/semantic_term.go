package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// SemanticTermType defines the nature of the semantic term
type SemanticTermType string

const (
	SemanticTypePhysical     SemanticTermType = "physical"
	SemanticTypeCalculated   SemanticTermType = "calculated"
	SemanticTypeLLM          SemanticTermType = "llm"
	SemanticTypeRelationship SemanticTermType = "relationship"
)

// SemanticDataType defines the return type of the term
type SemanticDataType string

const (
	DataTypeString   SemanticDataType = "string"
	DataTypeNumber   SemanticDataType = "number"
	DataTypeBoolean  SemanticDataType = "boolean"
	DataTypeDate     SemanticDataType = "date"
	DataTypeDateTime SemanticDataType = "datetime"
	DataTypeJSON     SemanticDataType = "json"
)

// FieldRole defines the role of a semantic term in a historical or analytical context
type FieldRole string

const (
	FieldRoleDimension     FieldRole = "DIMENSION"
	FieldRoleMeasure       FieldRole = "MEASURE"
	FieldRoleValidityStart FieldRole = "VALIDITY_START"
	FieldRoleValidityEnd   FieldRole = "VALIDITY_END"
	FieldRoleEventDate     FieldRole = "EVENT_DATE"
	FieldRolePartitionKey  FieldRole = "PARTITION_KEY"
)

// SemanticTerm represents a rich semantic definition (Workday-style)
// It maps to catalog_node with node_type_id for Semantic Term
type SemanticTerm struct {
	ID               string           `json:"id"`
	NodeName         string           `json:"node_name"`
	DisplayName      string           `json:"display_name"`
	Description      string           `json:"description"`
	Type             SemanticTermType `json:"type"`
	DataType         SemanticDataType `json:"data_type"`
	Role             FieldRole        `json:"role"`
	IsEffectiveDated bool             `json:"is_effective_dated"`

	// Specific Configurations based on Type
	PhysicalMapping *PhysicalMapping `json:"physical_mapping,omitempty"`
	Expression      string           `json:"expression,omitempty"`      // For calculated fields
	Materialization string           `json:"materialization,omitempty"` // virtual, view, materialized_table
	LLMProfile      *LLMProfile      `json:"llm_profile,omitempty"`
	Relationship    *Relationship    `json:"relationship,omitempty"`

	Tags          []string  `json:"tags"`
	Lineage       []string  `json:"lineage"` // IDs of upstream dependencies
	QualifiedPath string    `json:"qualified_path"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PhysicalMapping links a term to a physical database column
type PhysicalMapping struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

// LLMProfile defines how an LLM should derive this field
type LLMProfile struct {
	PromptTemplate string   `json:"prompt_template"`
	Model          string   `json:"model"`
	ContextTerms   []string `json:"context_terms"` // Other semantic terms to provide as context
}

// Relationship defines a join to another Business Object
type Relationship struct {
	TargetBusinessObject string `json:"target_bo"`
	JoinExpression       string `json:"join_expression"` // e.g. "this.client_id = target.id"
	Cardinality          string `json:"cardinality"`     // ONE_TO_ONE, ONE_TO_MANY
}

// Value implements the driver.Valuer interface for JSONB storage
func (s SemanticTerm) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface for JSONB storage
func (s *SemanticTerm) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &s)
}
