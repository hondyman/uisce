// Package governance holds the platform's built-in governance policies
// (trade compliance, semantic-term validation, pipeline validation).
//
// Every policy is a set of internal/rules/vm rules - the platform's single
// rule engine. Each rule states the compliant condition; a rule that does not
// hold contributes its deny message. The policies were originally Rego (OPA);
// their decisions are pinned by parity_test.go.
package governance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
)

type ValidationResult struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

// GovernanceEngine evaluates the built-in policies. db is kept for API
// compatibility with existing callers; no policy reads it.
type GovernanceEngine struct {
	db *sqlx.DB
}

func NewGovernanceEngine(db *sqlx.DB) *GovernanceEngine {
	return &GovernanceEngine{db: db}
}

// policyRule is one deny rule: Holds is the compliant condition; when it does
// not hold, Deny renders the reason(s) from the input.
type policyRule struct {
	Name  string
	Holds vm.RuleNode
	Deny  func(input map[string]interface{}) []string
}

// --- rule-building helpers (vm AST) ----------------------------------

func cond(field, op string, value interface{}) vm.RuleNode {
	return vm.RuleNode{Type: vm.NodeTypeCondition, Condition: &vm.RuleCondition{
		Field: field, FieldPath: field, Operator: op, Value: value,
	}}
}

func anyOf(nodes ...vm.RuleNode) vm.RuleNode {
	return vm.RuleNode{Type: vm.NodeTypeGroup, Group: &vm.RuleGroup{Operator: "OR", Conditions: nodes}}
}

func not(n vm.RuleNode) vm.RuleNode {
	return vm.RuleNode{Type: vm.NodeTypeGroup, Group: &vm.RuleGroup{Operator: "NOT", Conditions: []vm.RuleNode{n}}}
}

// absent reproduces Rego's "rule body is undefined when an input field is
// missing": the deny never fires for an absent field.
func absent(field string) vm.RuleNode { return cond(field, "is_null", nil) }

func msg(s string) func(map[string]interface{}) []string {
	return func(map[string]interface{}) []string { return []string{s} }
}

// --- policies ------------------------------------------------------------

var restrictedSymbols = []interface{}{"RSTR", "BAD", "LOCKED"}

var tradePolicy = []policyRule{
	{
		Name: "high-value trade requires pre-approval",
		Holds: anyOf(
			absent("amount"),
			cond("amount", "<", 1000000.0),
			cond("compliance_approved", "equals", true),
		),
		Deny: msg("High-value trade (>1M) requires pre-approval"),
	},
	{
		Name:  "restricted symbols",
		Holds: anyOf(absent("symbol"), cond("symbol", "not_in", restrictedSymbols)),
		Deny: func(in map[string]interface{}) []string {
			return []string{fmt.Sprintf("Symbol %v is on the restricted list", in["symbol"])}
		},
	},
	{
		Name:  "portfolio exposure limit",
		Holds: anyOf(absent("exposure"), cond("exposure", "<=", 5000000.0)),
		Deny:  msg("Portfolio exposure limit exceeded (5M)"),
	},
}

// restrictedColumns are physical columns needing PII access.
var restrictedColumns = []interface{}{"salary", "ssn"}

var semanticPolicy = []policyRule{
	{
		Name: "description length",
		Holds: anyOf(
			absent("description"),
			cond("description", "length_equals", 0),
			cond("description", "length_greater", 4),
		),
		Deny: msg("Description must be at least 5 characters long if provided"),
	},
	{
		Name:  "complexity limit",
		Holds: anyOf(absent("complexity_score"), cond("complexity_score", "<=", 5.0)),
		Deny: func(in map[string]interface{}) []string {
			return []string{fmt.Sprintf("Complexity score %v exceeds limit of 5", in["complexity_score"])}
		},
	},
	{
		Name: "restricted columns need PII access",
		Holds: anyOf(
			cond("user_has_pii_access", "equals", true),
			absent("referenced_columns"),
			not(cond("referenced_columns", "contains_any", restrictedColumns)),
		),
		Deny: func(in map[string]interface{}) []string {
			var out []string
			cols, _ := in["referenced_columns"].([]interface{})
			for _, c := range cols {
				for _, r := range restrictedColumns {
					if fmt.Sprint(c) == r {
						out = append(out, fmt.Sprintf("Access to restricted column '%v' denied", c))
					}
				}
			}
			return out
		},
	},
	{
		Name:  "node name characters",
		Holds: anyOf(absent("node_name"), cond("node_name", "matches_regex", `^[a-zA-Z0-9_ \-\(\)\.]+$`)),
		Deny:  msg("Node name contains invalid characters"),
	},
}

var pipelinePolicy = []policyRule{
	{
		Name:  "at least one node",
		Holds: anyOf(absent("nodes"), cond("nodes", "is_not_empty", nil)),
		Deny:  msg("Pipeline must have at least one node"),
	},
	{
		Name:  "start node",
		Holds: anyOf(absent("nodes"), cond("nodes.type", "contains_any", []interface{}{"start"})),
		Deny:  msg("Pipeline must have a Start node"),
	},
}

// --- evaluation ------------------------------------------------------------

// evaluate runs a policy's rules. A rule the engine cannot evaluate denies
// (never a silent pass), naming the rule.
func evaluate(policy []policyRule, raw interface{}) (*ValidationResult, error) {
	input, err := toInput(raw)
	if err != nil {
		return nil, err
	}
	ae := vm.NewAdvancedEvaluator()
	var reasons []string
	for _, r := range policy {
		ok, err := ae.Evaluate(r.Holds, input)
		if err != nil {
			reasons = append(reasons, fmt.Sprintf("policy rule %q could not be evaluated: %v", r.Name, err))
			continue
		}
		if !ok {
			reasons = append(reasons, r.Deny(input)...)
		}
	}
	sort.Strings(reasons)
	return &ValidationResult{Allowed: len(reasons) == 0, Reasons: reasons}, nil
}

// toInput normalises any caller payload to the JSON shape rules evaluate
// (numbers as float64, nested objects as maps) - the same view OPA had.
func toInput(raw interface{}) (map[string]interface{}, error) {
	if raw == nil {
		return map[string]interface{}{}, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("governance input is not JSON-serialisable: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("governance input must be a JSON object: %w", err)
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	return m, nil
}

// ValidateSemanticTerm checks a semantic term against the semantic policy.
func (e *GovernanceEngine) ValidateSemanticTerm(ctx context.Context, term map[string]interface{}) (*ValidationResult, error) {
	return evaluate(semanticPolicy, term)
}

// ValidatePipeline checks a pipeline definition against the pipeline policy.
func (e *GovernanceEngine) ValidatePipeline(ctx context.Context, tenantID string, pipelineDefinition interface{}) (*ValidationResult, error) {
	return evaluate(pipelinePolicy, pipelineDefinition)
}

// ValidateTransaction checks a trade payload against the trade compliance policy.
func (e *GovernanceEngine) ValidateTransaction(ctx context.Context, tenantID string, transactionPayload interface{}) (*ValidationResult, error) {
	return evaluate(tradePolicy, transactionPayload)
}
