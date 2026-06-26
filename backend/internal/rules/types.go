package rules

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
)

// InheritMode defines how a tenant rule relates to a core rule
type InheritMode string

const (
	Inherit  InheritMode = "inherit"
	Extend   InheritMode = "extend"
	Override InheritMode = "override"
	Custom   InheritMode = "custom"
)

// RuleNodeType defines the type of a rule node (group or condition)
type RuleNodeType string

const (
	NodeTypeGroup     RuleNodeType = "group"
	NodeTypeCondition RuleNodeType = "condition"
)

// RuleNode is a wrapper that can hold either a Group or a Condition
type RuleNode struct {
	Type      RuleNodeType   `json:"type"`
	Group     *RuleGroup     `json:"group,omitempty"`
	Condition *RuleCondition `json:"condition,omitempty"`
}

// RuleGroup represents a logical grouping of rules (AND/OR/NOT)
type RuleGroup struct {
	ID         string     `json:"id"`
	Operator   string     `json:"operator"` // AND, OR, NOT
	Conditions []RuleNode `json:"conditions"`
}

// RuleCondition represents a single leaf condition
type RuleCondition struct {
	ID          string      `json:"id"`
	Field       string      `json:"field"`
	FieldPath   string      `json:"fieldPath,omitempty"` // For cross-entity: "order.customer.name"
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value"`
	ValueType   string      `json:"valueType,omitempty"`
	SecondValue interface{} `json:"secondValue,omitempty"` // For 'between' operator
}

// UnmarshalJSON implements custom unmarshalling for RuleNode to handle polymorphism
func (n *RuleNode) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type RuleNodeType `json:"type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	n.Type = temp.Type

	if n.Type == NodeTypeGroup {
		// It's a group, unmarshal into Group field, but we need to handle the flat structure
		// The JSON from frontend is likely flat: { type: "group", operator: "AND", conditions: [...] }
		// So we unmarshal the whole thing into a RuleGroup struct
		var g RuleGroup
		if err := json.Unmarshal(data, &g); err != nil {
			return err
		}
		n.Group = &g
	} else if n.Type == NodeTypeCondition {
		// It's a condition
		var c RuleCondition
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		n.Condition = &c
	} else {
		return fmt.Errorf("unknown rule node type: %s", n.Type)
	}

	return nil
}

// RuleRecord represents a raw validation rule from the database
type RuleRecord struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	TargetEntityID    uuid.UUID
	Name              string
	Description       sql.NullString
	RuleType          string
	CompiledSQL       sql.NullString
	CompiledWASM      []byte
	CompiledCUE       sql.NullString
	ExecuteServerSide bool
	ExecuteClientSide bool
	RunOnSubmit       bool
	Severity          string
	RemediationHint   sql.NullString
	EvaluationOrder   int
	IsActive          bool
	CoreRuleID        *uuid.UUID // Null if this is a core rule, points to core rule if this is a tenant override
	DatasourceID      *uuid.UUID // Tenant-specific datasource override
}

// ResolvedRule represents a finalized validation rule ready for evaluation
type ResolvedRule struct {
	ID                uuid.UUID
	Name              string
	Description       string
	RuleType          string
	CompiledSQL       *string
	CompiledWASM      []byte
	CompiledCUE       string
	ExecuteServerSide bool
	ExecuteClientSide bool
	RunOnSubmit       bool
	Severity          string
	RemediationHint   *string
	EvaluationOrder   int
	IsActive          bool
	SemanticTerms     []string // Term IDs
	Fields            []models.FieldDefinition
	ImpactNodes       []ImpactNode
}

// SemanticTerm is a type alias for the models package version
type SemanticTerm = models.SemanticTerm

// FieldDefinition is a type alias for the models package version
type FieldDefinition = models.FieldDefinition

// ImpactNode represents a node in the rule impact analysis graph
type ImpactNode struct {
	ID         uuid.UUID              `json:"id"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
}

// RuleSchema contains all available fields and terms for building validation rules
type RuleSchema struct {
	Fields []models.FieldDefinition `json:"fields"`
	Terms  []models.SemanticTerm    `json:"terms"`
	Locale string                   `json:"locale"`
}
