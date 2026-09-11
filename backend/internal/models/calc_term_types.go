package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CalcTermProperties is stored in catalog_node.properties, mirroring
// ValidationRuleProperties' role for validation-rule nodes. TermType is
// always "calculated" - the marker PreAggregationService.compileCalcTermToSQL
// (internal/analytics/pre_aggregation_service.go) filters on when
// resolving a pre-aggregation's Calculations list by node_name.
type CalcTermProperties struct {
	BOName   string `json:"bo_name"`
	TenantID string `json:"tenant_id"`
	TermType string `json:"term_type"` // always "calculated" for nodes this package writes
	DataType string `json:"data_type,omitempty"`
}

// CalcTermConfig is stored in catalog_node.config. RuleAST is a
// vm.Expression-shaped json.RawMessage (internal/rules/vm), same
// low-level-package rationale as ValidationRuleConfig.RuleAST: this
// package doesn't import internal/rules/vm, whoever evaluates or
// compiles the AST does. Expression keeps the original typed-text source
// alongside the parsed AST, so re-editing a saved calc term doesn't
// require decompiling the AST back to text.
type CalcTermConfig struct {
	Expression string          `json:"expression"`
	RuleAST    json.RawMessage `json:"rule_ast"`
}

// UpsertCalcTermRequest is the API request shape for creating or
// updating a calc term. Expression is typed text (e.g. "SUM(ExecQuantity
// * ExecPrice)"), parsed server-side via vm.ParseExpression - the client
// never constructs rule_ast JSON directly.
type UpsertCalcTermRequest struct {
	TenantID    string `json:"tenant_id"`
	BOName      string `json:"bo_name"`
	Name        string `json:"name"` // catalog_node.node_name
	Description string `json:"description,omitempty"`
	Expression  string `json:"expression"`
	DataType    string `json:"data_type,omitempty"`
}

// CalcTermDescriptor is the API response shape.
type CalcTermDescriptor struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    string          `json:"tenant_id"`
	BOName      string          `json:"bo_name"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Expression  string          `json:"expression"`
	RuleAST     json.RawMessage `json:"rule_ast"`
	DataType    string          `json:"data_type,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ParseCalcTermProperties unmarshals CalcTermProperties from
// catalog_node.properties.
func ParseCalcTermProperties(raw json.RawMessage) (*CalcTermProperties, error) {
	var props CalcTermProperties
	if err := json.Unmarshal(raw, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// ParseCalcTermConfig unmarshals CalcTermConfig from catalog_node.config.
func ParseCalcTermConfig(raw json.RawMessage) (*CalcTermConfig, error) {
	var cfg CalcTermConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
