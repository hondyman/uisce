package rdl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// RuleType represents the category of business rule
type RuleType string

const (
	RuleTypeTaxLossHarvesting RuleType = "tax_loss_harvesting"
	RuleTypeTLH               RuleType = "tax_loss_harvesting" // Alias for TLH
	RuleTypeWashSale          RuleType = "wash_sale"
	RuleTypeTaxConstraint     RuleType = "tax_constraint"
	RuleTypeESGRestriction    RuleType = "esg_restriction"
	RuleTypeDriftTrigger      RuleType = "drift_trigger"
	RuleTypeCashFlow          RuleType = "cash_flow"
	RuleTypeCPPIFloor         RuleType = "cppi_floor"
	RuleTypeSectorLimit       RuleType = "sector_limit"
	RuleTypeConcentration     RuleType = "concentration_limit"
	RuleTypeCustom            RuleType = "custom"
)

// RuleDefinition represents a metadata-driven business rule
type RuleDefinition struct {
	ID                   uuid.UUID       `db:"id" json:"id"`
	TenantID             uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	RuleID               string          `db:"rule_id" json:"rule_id"`
	Type                 RuleType        `db:"type" json:"type"`
	Version              string          `db:"version" json:"version"`
	Name                 string          `db:"name" json:"name"`
	Description          string          `db:"description" json:"description"`
	Jurisdiction         string          `db:"jurisdiction" json:"jurisdiction"`
	Parameters           json.RawMessage `db:"parameters" json:"parameters"`
	Expression           string          `db:"expression" json:"expression"`
	ScoringFormula       string          `db:"scoring_formula" json:"scoring_formula,omitempty"`
	WashSaleConfig       json.RawMessage `db:"wash_sale_config" json:"wash_sale_config,omitempty"`
	SubstituteAssetRules json.RawMessage `db:"substitute_asset_rules" json:"substitute_asset_rules,omitempty"`
	Schedule             json.RawMessage `db:"schedule" json:"schedule,omitempty"`
	Notifications        json.RawMessage `db:"notifications" json:"notifications,omitempty"`
	Active               bool            `db:"active" json:"active"`
	EffectiveFrom        *time.Time      `db:"effective_from" json:"effective_from,omitempty"`
	EffectiveTo          *time.Time      `db:"effective_to" json:"effective_to,omitempty"`
	Audit                json.RawMessage `db:"audit" json:"audit,omitempty"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time       `db:"updated_at" json:"updated_at"`
}

// TLHParameters represents tax-loss harvesting specific parameters
type TLHParameters struct {
	MinLossPercentage           float64 `json:"min_loss_percentage"`
	MinLossAmountUSD            float64 `json:"min_loss_amount_usd"`
	MaxLossAmountUSD            float64 `json:"max_loss_amount_usd,omitempty"`
	HoldingPeriodDays           int     `json:"holding_period_days"`
	LongTermThresholdDays       int     `json:"long_term_threshold_days"`
	AnnualLossLimitUSD          float64 `json:"annual_loss_limit_usd"`
	CarryforwardEnabled         bool    `json:"carryforward_enabled"`
	EstimatedTaxRate            float64 `json:"estimated_tax_rate"`
	StateTaxRate                float64 `json:"state_tax_rate,omitempty"`
	TransactionCostThresholdUSD float64 `json:"transaction_cost_threshold_usd"`
}

// WashSaleConfig represents wash sale prevention configuration
type WashSaleConfig struct {
	Enabled                      bool                          `json:"enabled"`
	WindowDaysBefore             int                           `json:"window_days_before"`
	WindowDaysAfter              int                           `json:"window_days_after"`
	CheckHousehold               bool                          `json:"check_household"`
	CheckIRA                     bool                          `json:"check_ira"`
	CheckSpouse                  bool                          `json:"check_spouse"`
	SubstantiallyIdenticalConfig *SubstantiallyIdenticalConfig `json:"substantially_identical_config,omitempty"`
}

// SubstantiallyIdenticalConfig defines what securities are considered "substantially identical"
type SubstantiallyIdenticalConfig struct {
	SameTicker          bool    `json:"same_ticker"`
	SameCUSIP           bool    `json:"same_cusip"`
	OptionsOnSame       bool    `json:"options_on_same"`
	ConvertibleBonds    bool    `json:"convertible_bonds"`
	ETFOverlapThreshold float64 `json:"etf_overlap_threshold"`
	MutualFundSameIndex bool    `json:"mutual_fund_same_index"`
}

// EvaluationInput represents the input data for rule evaluation
type EvaluationInput struct {
	PortfolioID       string                 `json:"portfolio_id"`
	AccountID         string                 `json:"account_id"`
	HouseholdID       string                 `json:"household_id"`
	Ticker            string                 `json:"ticker"`
	CUSIP             string                 `json:"cusip,omitempty"`
	UnrealizedLossPct float64                `json:"unrealized_loss_pct"`
	UnrealizedLossUSD float64                `json:"unrealized_loss_usd"`
	CostBasis         float64                `json:"cost_basis"`
	CurrentValue      float64                `json:"current_value"`
	DaysHeld          int                    `json:"days_held"`
	AccountType       string                 `json:"account_type"` // TAXABLE, IRA, 401K, etc.
	IsLossSale        bool                   `json:"is_loss_sale"`
	SaleDate          *time.Time             `json:"sale_date,omitempty"`
	ClientESGPref     bool                   `json:"client_esg_preference"`
	AdditionalData    map[string]interface{} `json:"additional_data,omitempty"`
}

// EvaluationResult represents the result of rule evaluation
type EvaluationResult struct {
	RuleID           string                 `json:"rule_id"`
	RuleName         string                 `json:"rule_name"`
	Passed           bool                   `json:"passed"`
	Score            float64                `json:"score,omitempty"`
	ActionRequired   bool                   `json:"action_required"`
	Recommendation   string                 `json:"recommendation,omitempty"`
	Violations       []string               `json:"violations,omitempty"`
	Actions          []string               `json:"actions,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	EvaluatedAt      time.Time              `json:"evaluated_at"`
	EvaluationTimeMS int64                  `json:"evaluation_time_ms"`
}

// RDLService provides the Rule Definition Language evaluation engine
type RDLService struct {
	db       *sqlx.DB
	celEnv   *cel.Env
	washSale WashSaleChecker
}

// WashSaleChecker interface for wash sale lookups
type WashSaleChecker interface {
	IsInWashSaleWindow(ctx context.Context, householdID, ticker string, daysBefore, daysAfter int) (bool, error)
	HasRecentPurchase(ctx context.Context, householdID, ticker string, withinDays int) (bool, error)
}

// NewRDLService creates a new Rule Definition Language service
func NewRDLService(db *sqlx.DB, washSaleChecker WashSaleChecker) (*RDLService, error) {
	// Create CEL environment with custom declarations
	env, err := cel.NewEnv(
		cel.Declarations(
			// Input variables
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("params", decls.NewMapType(decls.String, decls.Dyn)),

			// Custom functions - these will be bound at evaluation time
			decls.NewFunction("isInWashSaleWindow",
				decls.NewOverload("isInWashSaleWindow_string_string",
					[]*exprpb.Type{decls.String, decls.String},
					decls.Bool,
				),
			),
			decls.NewFunction("hasRecentPurchase",
				decls.NewOverload("hasRecentPurchase_string_string_int",
					[]*exprpb.Type{decls.String, decls.String, decls.Int},
					decls.Bool,
				),
			),
			decls.NewFunction("daysSince",
				decls.NewOverload("daysSince_timestamp",
					[]*exprpb.Type{decls.NewObjectType("google.protobuf.Timestamp")},
					decls.Int,
				),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &RDLService{
		db:       db,
		celEnv:   env,
		washSale: washSaleChecker,
	}, nil
}

// NewService creates a new RDL service with defaults (used by HTTP handlers)
func NewService(db *sqlx.DB) *RDLService {
	// Create CEL environment with custom declarations
	env, err := cel.NewEnv(
		cel.Declarations(
			// Input variables
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("params", decls.NewMapType(decls.String, decls.Dyn)),

			// Custom functions - these will be bound at evaluation time
			decls.NewFunction("isInWashSaleWindow",
				decls.NewOverload("isInWashSaleWindow_string_string",
					[]*exprpb.Type{decls.String, decls.String},
					decls.Bool,
				),
			),
			decls.NewFunction("hasRecentPurchase",
				decls.NewOverload("hasRecentPurchase_string_string_int",
					[]*exprpb.Type{decls.String, decls.String, decls.Int},
					decls.Bool,
				),
			),
			decls.NewFunction("daysSince",
				decls.NewOverload("daysSince_timestamp",
					[]*exprpb.Type{decls.NewObjectType("google.protobuf.Timestamp")},
					decls.Int,
				),
			),
		),
	)
	if err != nil {
		// Fall back to basic environment
		env, _ = cel.NewEnv()
	}

	return &RDLService{
		db:       db,
		celEnv:   env,
		washSale: nil,
	}
}

// GetRulesByTenant retrieves all active rules for a tenant
func (s *RDLService) GetRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]RuleDefinition, error) {
	query := `
		SELECT id, tenant_id, rule_id, type, version, name, description, jurisdiction,
		       parameters, expression, scoring_formula, wash_sale_config, substitute_asset_rules,
		       schedule, notifications, active, effective_from, effective_to, audit,
		       created_at, updated_at
		FROM rule_definitions
		WHERE tenant_id = $1 AND active = true
		  AND (effective_from IS NULL OR effective_from <= CURRENT_DATE)
		  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
		ORDER BY type, rule_id
	`

	var rules []RuleDefinition
	if err := s.db.SelectContext(ctx, &rules, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %w", err)
	}

	return rules, nil
}

// GetRulesByType retrieves rules of a specific type for a tenant
func (s *RDLService) GetRulesByType(ctx context.Context, tenantID uuid.UUID, ruleType RuleType) ([]RuleDefinition, error) {
	query := `
		SELECT id, tenant_id, rule_id, type, version, name, description, jurisdiction,
		       parameters, expression, scoring_formula, wash_sale_config, substitute_asset_rules,
		       schedule, notifications, active, effective_from, effective_to, audit,
		       created_at, updated_at
		FROM rule_definitions
		WHERE tenant_id = $1 AND type = $2 AND active = true
		  AND (effective_from IS NULL OR effective_from <= CURRENT_DATE)
		  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
		ORDER BY rule_id
	`

	var rules []RuleDefinition
	if err := s.db.SelectContext(ctx, &rules, query, tenantID, ruleType); err != nil {
		return nil, fmt.Errorf("failed to fetch rules by type: %w", err)
	}

	return rules, nil
}

// GetRuleByID retrieves a specific rule
func (s *RDLService) GetRuleByID(ctx context.Context, tenantID uuid.UUID, ruleID string) (*RuleDefinition, error) {
	query := `
		SELECT id, tenant_id, rule_id, type, version, name, description, jurisdiction,
		       parameters, expression, scoring_formula, wash_sale_config, substitute_asset_rules,
		       schedule, notifications, active, effective_from, effective_to, audit,
		       created_at, updated_at
		FROM rule_definitions
		WHERE tenant_id = $1 AND rule_id = $2
		ORDER BY version DESC
		LIMIT 1
	`

	var rule RuleDefinition
	if err := s.db.GetContext(ctx, &rule, query, tenantID, ruleID); err != nil {
		return nil, fmt.Errorf("failed to fetch rule: %w", err)
	}

	return &rule, nil
}

// Evaluate evaluates a rule against input data
func (s *RDLService) Evaluate(ctx context.Context, rule *RuleDefinition, input *EvaluationInput) (*EvaluationResult, error) {
	startTime := time.Now()

	// Parse parameters from JSON
	var params map[string]interface{}
	if err := json.Unmarshal(rule.Parameters, &params); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Convert input to map
	inputMap, err := structToMap(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Compile the expression
	ast, issues := s.celEnv.Compile(rule.Expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("expression compile error: %w", issues.Err())
	}

	// Create program
	prg, err := s.celEnv.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program creation error: %w", err)
	}

	// Evaluate
	out, _, err := prg.Eval(map[string]interface{}{
		"input":  inputMap,
		"params": params,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}

	passed, ok := out.Value().(bool)
	if !ok {
		return nil, fmt.Errorf("expression did not return a boolean")
	}

	// Calculate score if scoring formula exists
	var score float64
	if rule.ScoringFormula != "" {
		scoreAst, scoreIssues := s.celEnv.Compile(rule.ScoringFormula)
		if scoreIssues == nil || scoreIssues.Err() == nil {
			scorePrg, _ := s.celEnv.Program(scoreAst)
			scoreOut, _, scoreErr := scorePrg.Eval(map[string]interface{}{
				"input":  inputMap,
				"params": params,
			})
			if scoreErr == nil {
				if s, ok := scoreOut.Value().(float64); ok {
					score = s
				}
			}
		}
	}

	evaluationTime := time.Since(startTime).Milliseconds()

	result := &EvaluationResult{
		RuleID:           rule.RuleID,
		RuleName:         rule.Name,
		Passed:           passed,
		Score:            score,
		ActionRequired:   passed, // For TLH, passing = opportunity found
		EvaluatedAt:      time.Now(),
		EvaluationTimeMS: evaluationTime,
	}

	// Generate recommendation based on rule type
	if passed {
		result.Recommendation = s.generateRecommendation(rule, input, score)
	}

	return result, nil
}

// EvaluateAll evaluates all applicable rules for given input
func (s *RDLService) EvaluateAll(ctx context.Context, tenantID uuid.UUID, ruleType RuleType, input *EvaluationInput) ([]EvaluationResult, error) {
	rules, err := s.GetRulesByType(ctx, tenantID, ruleType)
	if err != nil {
		return nil, err
	}

	results := make([]EvaluationResult, 0, len(rules))
	for _, rule := range rules {
		result, err := s.Evaluate(ctx, &rule, input)
		if err != nil {
			// Log error but continue with other rules
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// CreateRule creates a new rule definition
func (s *RDLService) CreateRule(ctx context.Context, rule *RuleDefinition) error {
	// Validate expression compiles
	_, issues := s.celEnv.Compile(rule.Expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("invalid expression: %w", issues.Err())
	}

	// Validate scoring formula if present
	if rule.ScoringFormula != "" {
		_, issues := s.celEnv.Compile(rule.ScoringFormula)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("invalid scoring formula: %w", issues.Err())
		}
	}

	query := `
		INSERT INTO rule_definitions (
			tenant_id, rule_id, type, version, name, description, jurisdiction,
			parameters, expression, scoring_formula, wash_sale_config, substitute_asset_rules,
			schedule, notifications, active, effective_from, effective_to, audit
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		RETURNING id, created_at, updated_at
	`

	row := s.db.QueryRowContext(ctx, query,
		rule.TenantID, rule.RuleID, rule.Type, rule.Version, rule.Name, rule.Description,
		rule.Jurisdiction, rule.Parameters, rule.Expression, rule.ScoringFormula,
		rule.WashSaleConfig, rule.SubstituteAssetRules, rule.Schedule, rule.Notifications,
		rule.Active, rule.EffectiveFrom, rule.EffectiveTo, rule.Audit,
	)

	return row.Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

// UpdateRule updates an existing rule (creates new version)
func (s *RDLService) UpdateRule(ctx context.Context, rule *RuleDefinition) error {
	// Validate expression
	_, issues := s.celEnv.Compile(rule.Expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("invalid expression: %w", issues.Err())
	}

	query := `
		UPDATE rule_definitions
		SET name = $3, description = $4, parameters = $5, expression = $6,
		    scoring_formula = $7, wash_sale_config = $8, substitute_asset_rules = $9,
		    schedule = $10, notifications = $11, active = $12,
		    effective_from = $13, effective_to = $14, audit = $15,
		    updated_at = NOW()
		WHERE tenant_id = $1 AND rule_id = $2 AND version = $16
	`

	_, err := s.db.ExecContext(ctx, query,
		rule.TenantID, rule.RuleID, rule.Name, rule.Description,
		rule.Parameters, rule.Expression, rule.ScoringFormula,
		rule.WashSaleConfig, rule.SubstituteAssetRules, rule.Schedule, rule.Notifications,
		rule.Active, rule.EffectiveFrom, rule.EffectiveTo, rule.Audit, rule.Version,
	)

	return err
}

// DeactivateRule deactivates a rule (soft delete)
func (s *RDLService) DeactivateRule(ctx context.Context, tenantID uuid.UUID, ruleID string) error {
	query := `
		UPDATE rule_definitions
		SET active = false, updated_at = NOW()
		WHERE tenant_id = $1 AND rule_id = $2
	`

	_, err := s.db.ExecContext(ctx, query, tenantID, ruleID)
	return err
}

// Helper: Generate recommendation based on rule type and score
func (s *RDLService) generateRecommendation(rule *RuleDefinition, input *EvaluationInput, score float64) string {
	switch rule.Type {
	case RuleTypeTaxLossHarvesting:
		benefit := input.UnrealizedLossUSD * 0.35 // Simplified tax benefit
		return fmt.Sprintf("Tax-loss harvesting opportunity: Sell %s for estimated tax benefit of $%.2f", input.Ticker, benefit)
	case RuleTypeWashSale:
		return fmt.Sprintf("Warning: Potential wash sale violation for %s. Wait 31 days before repurchasing.", input.Ticker)
	case RuleTypeDriftTrigger:
		return "Portfolio drift detected. Consider rebalancing to target allocation."
	default:
		return fmt.Sprintf("Rule %s triggered. Score: %.2f", rule.RuleID, score)
	}
}

// Helper: Convert struct to map for CEL evaluation
func structToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ValidateExpression validates a CEL expression without saving
func (s *RDLService) ValidateExpression(expression string) (bool, []string) {
	if expression == "" {
		return false, []string{"expression cannot be empty"}
	}

	ast, issues := s.celEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		var errors []string
		for _, issue := range issues.Errors() {
			errors = append(errors, issue.Message)
		}
		return false, errors
	}

	// Expression is syntactically valid, check if it can be evaluated
	if ast == nil {
		return false, []string{"failed to compile expression"}
	}

	return true, nil
}
