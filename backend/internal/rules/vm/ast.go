package vm

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// AST types are defined in this package to break an import cycle:
// the rules package imports vm, but vm's compiler needs the AST types.
// Re-exported as type aliases in rules/types.go for backward compat.

// RuleNodeType identifies whether a rule node is a group or a leaf condition.
type RuleNodeType string

const (
	NodeTypeGroup     RuleNodeType = "group"
	NodeTypeCondition RuleNodeType = "condition"
)

// RuleNode is a wrapper that holds either a Group or a Condition.
type RuleNode struct {
	Type      RuleNodeType
	Group     *RuleGroup
	Condition *RuleCondition
}

// RuleGroup represents a logical grouping of rules (AND/OR/NOT).
type RuleGroup struct {
	ID         string
	Operator   string // AND, OR, NOT
	Conditions []RuleNode
}

// RuleCondition represents a single leaf condition.
type RuleCondition struct {
	ID          string
	Field       string
	FieldPath   string
	Operator    string
	Value       interface{}
	ValueType   string
	SecondValue interface{}
}

// UnmarshalJSON implements custom unmarshalling for polymorphic AST nodes.
func (n *RuleNode) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type RuleNodeType `json:"type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	n.Type = temp.Type

	switch n.Type {
	case NodeTypeGroup:
		var g RuleGroup
		if err := json.Unmarshal(data, &g); err != nil {
			return err
		}
		n.Group = &g
	case NodeTypeCondition:
		var c RuleCondition
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		n.Condition = &c
	default:
		return fmt.Errorf("unknown rule node type: %s", n.Type)
	}
	return nil
}

// RuleRecord represents a persisted rule from the database. Used by
// callers that load rules before evaluating them.
type RuleRecord struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	TargetEntityID    uuid.UUID
	Name              string
	Description       string
	RuleType          string
	CompiledSQL       string
	CompiledWASM      []byte
	CompiledCUE       string
	ExecuteServerSide bool
	ExecuteClientSide bool
	RunOnSubmit       bool
	Severity          string
	RemediationHint   string
	EvaluationOrder   int
	IsActive          bool
	CoreRuleID        *uuid.UUID
	DatasourceID      *uuid.UUID
}

// RuleID returns a stable string identifier for cache keying.
// For rules without an ID (e.g., ad-hoc test rules), this is empty.
func (n *RuleNode) ID() string {
	if n == nil {
		return ""
	}
	if n.Condition != nil && n.Condition.ID != "" {
		return "cond:" + n.Condition.ID
	}
	if n.Group != nil && n.Group.ID != "" {
		return "grp:" + n.Group.ID
	}
	return ""
}