package governance

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/open-policy-agent/opa/rego"
)

//go:embed policies/pipeline_validation.rego
var defaultPolicy string

//go:embed policies/trade_compliance.rego
var tradePolicy string

//go:embed policies/semantic_validation.rego
var semanticPolicy string

type ValidationResult struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

type GovernanceEngine struct {
	db             *sqlx.DB
	pipelinePolicy string
	tradePolicy    string
	semanticPolicy string
}

func NewGovernanceEngine(db *sqlx.DB) *GovernanceEngine {
	return &GovernanceEngine{
		db:             db,
		pipelinePolicy: defaultPolicy,
		tradePolicy:    tradePolicy,
		semanticPolicy: semanticPolicy,
	}
}

func (e *GovernanceEngine) ValidateSemanticTerm(ctx context.Context, term map[string]interface{}) (*ValidationResult, error) {
	options := []func(*rego.Rego){
		rego.Query("data.semlayer.governance.semantic.allow; data.semlayer.governance.semantic.deny"),
		rego.Module("semantic_validation.rego", e.semanticPolicy),
		// Mock data for restricted columns
		rego.Module("restricted_data.rego", "package restricted_columns\nlist = [\"salary\", \"ssn\"]"),
	}

	query, err := rego.New(options...).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("prepare eval: %w", err)
	}

	results, err := query.Eval(ctx, rego.EvalInput(term))
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results for input: %+v", term)
	}

	// Parse Allow/Deny similar to other methods
	allowed := false
	var reasons []string

	if len(results[0].Expressions) >= 1 {
		if b, ok := results[0].Expressions[0].Value.(bool); ok {
			allowed = b
		}
	}
	if len(results[0].Expressions) >= 2 {
		if r, ok := results[0].Expressions[1].Value.([]interface{}); ok {
			for _, v := range r {
				reasons = append(reasons, fmt.Sprintf("%v", v))
			}
		}
	}

	return &ValidationResult{Allowed: allowed, Reasons: reasons}, nil
}

// ValidatePipeline executes the OPA policy against the provided pipeline definition
func (e *GovernanceEngine) ValidatePipeline(ctx context.Context, tenantID string, pipelineDefinition interface{}) (*ValidationResult, error) {
	// Prepare input
	// pipelineDefinition should be the graph or JSON structure

	options := []func(*rego.Rego){
		rego.Query("data.semlayer.governance.pipelines.allow; data.semlayer.governance.pipelines.deny"),
		rego.Module("pipeline_validation.rego", e.pipelinePolicy),
	}

	// Dynamic Policy Loading (if DB is present)
	if e.db != nil && tenantID != "" {
		var policies []string
		start := "package tenant.rules" // Filter ensures we only load relevant scopes
		err := e.db.SelectContext(ctx, &policies, "SELECT expression FROM core_policy WHERE tenant_id = $1 AND scope = 'workflow' AND type = 'authorization'", tenantID)
		if err == nil {
			for i, p := range policies {
				// Only load if it's a valid package
				if len(p) > len(start) { // Basic check
					options = append(options, rego.Module(fmt.Sprintf("tenant_policy_%d.rego", i), p))
				}
			}
		}
	}

	query, err := rego.New(options...).PrepareForEval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare rego query: %w", err)
	}

	results, err := query.Eval(ctx, rego.EvalInput(pipelineDefinition))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results from policy evaluation")
	}

	// Extract results
	// We expect two expressions: allow (bool) and deny (array of strings)
	if len(results[0].Expressions) < 2 {
		return nil, fmt.Errorf("unexpected policy result format")
	}

	allowedVal := results[0].Expressions[0].Value
	denyVal := results[0].Expressions[1].Value

	isAllowed, ok := allowedVal.(bool)
	if !ok {
		isAllowed = false
	}

	var reasons []string
	if reasonsRaw, ok := denyVal.([]interface{}); ok {
		for _, r := range reasonsRaw {
			if s, ok := r.(string); ok {
				reasons = append(reasons, s)
			}
		}
	}

	return &ValidationResult{
		Allowed: isAllowed,
		Reasons: reasons,
	}, nil
}

// ValidateTransaction executes the Trade Compliance OPA policy
func (e *GovernanceEngine) ValidateTransaction(ctx context.Context, tenantID string, transactionPayload interface{}) (*ValidationResult, error) {
	// Base query options
	options := []func(*rego.Rego){
		rego.Query("data.semlayer.governance.trades; data.tenant.rules"), // Query both base and tenant specifics
		rego.Module("trade_compliance.rego", e.tradePolicy),
	}

	// Dynamic Policy Loading
	if e.db != nil && tenantID != "" {
		var policies []string
		// We optimistically load all 'workflow' scope policies for now, assuming they apply to trades/transactions too
		// In a real system we might distinguish scope='trade' vs 'pipeline'
		err := e.db.SelectContext(ctx, &policies, "SELECT expression FROM core_policy WHERE tenant_id = $1 AND scope = 'workflow' AND type = 'authorization'", tenantID)
		if err == nil {
			for i, p := range policies {
				options = append(options, rego.Module(fmt.Sprintf("tenant_policy_%d.rego", i), p))
			}
		}
	}

	query, err := rego.New(options...).PrepareForEval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare rego query for trade: %w", err)
	}

	results, err := query.Eval(ctx, rego.EvalInput(transactionPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate trade policy: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results from policy evaluation")
	}

	// Result processing logic
	// We gather results from strict base policy AND tenant rules
	// Base policy returns Object { allow: bool, deny: [] }
	// Tenant rules (if structured as `package tenant.rules` with `deny[msg]`) returns Object { deny: [] } usually

	isAllowed := true
	var reasons []string

	for _, expr := range results[0].Expressions {
		val := expr.Value

		// Case 1: Base Policy Map
		if valMap, ok := val.(map[string]interface{}); ok {
			// Check for standard allow/deny structure
			if allowVal, exists := valMap["allow"]; exists {
				if b, ok := allowVal.(bool); ok && !b {
					isAllowed = false
				}
			}
			if denyVal, exists := valMap["deny"]; exists {
				if denials, ok := denyVal.([]interface{}); ok {
					for _, d := range denials {
						reasons = append(reasons, fmt.Sprintf("%v", d))
					}
				}
			}
		}
	}

	if len(reasons) > 0 {
		isAllowed = false
	}

	return &ValidationResult{
		Allowed: isAllowed,
		Reasons: reasons,
	}, nil
}
