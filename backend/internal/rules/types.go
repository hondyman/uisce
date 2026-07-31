package rules

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// InheritMode defines how a tenant rule relates to a core rule
type InheritMode string

const (
	Inherit  InheritMode = "inherit"
	Extend   InheritMode = "extend"
	Override InheritMode = "override"
	Custom   InheritMode = "custom"
)

// AST types are defined in vm package to break an import cycle.
// Re-exported here for backward compatibility — existing callers of
// rules.RuleNode / rules.RuleCondition / rules.RuleGroup / etc. continue
// to work unchanged.
type (
	RuleNodeType = vm.RuleNodeType
	RuleNode     = vm.RuleNode
	RuleGroup    = vm.RuleGroup
	RuleCondition = vm.RuleCondition
)

const (
	NodeTypeGroup     = vm.NodeTypeGroup
	NodeTypeCondition = vm.NodeTypeCondition
)

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
