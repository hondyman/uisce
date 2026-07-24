package rulefabric

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// =============================================================================
// TYPES & ENUMS
// =============================================================================

// RuleCategory represents the type of rule
type RuleCategory string

const (
	CategoryDataQuality RuleCategory = "data_quality"
	CategoryCompliance  RuleCategory = "compliance"
	CategoryMDM         RuleCategory = "mdm"
	CategoryWashTrade   RuleCategory = "wash_trade"
	CategoryValues      RuleCategory = "values"
	CategoryRebalancing RuleCategory = "rebalancing"
	CategoryWorkflow    RuleCategory = "workflow"
	CategorySecurity    RuleCategory = "security"
	CategoryCustom      RuleCategory = "custom"
)

// RuleContextType represents what the rule evaluates
type RuleContextType string

const (
	ContextDataRecord    RuleContextType = "data_record"
	ContextTradeEvent    RuleContextType = "trade_event"
	ContextPortfolio     RuleContextType = "portfolio"
	ContextClientProfile RuleContextType = "client_profile"
	ContextMDMGroup      RuleContextType = "mdm_group"
	ContextSystemJob     RuleContextType = "system_job"
	ContextRelationship  RuleContextType = "relationship"
	ContextTimeSeries    RuleContextType = "time_series"
	ContextAggregate     RuleContextType = "aggregate"
)

// RuleSeverity represents the severity of a rule violation
type RuleSeverity string

const (
	SeverityInfo       RuleSeverity = "info"
	SeverityWarning    RuleSeverity = "warning"
	SeverityError      RuleSeverity = "error"
	SeverityHardBlock  RuleSeverity = "hard_block"
	SeveritySoftBlock  RuleSeverity = "soft_block"
	SeverityQuarantine RuleSeverity = "quarantine"
)

// RuleStatus represents the lifecycle status
type RuleStatus string

const (
	StatusDraft            RuleStatus = "draft"
	StatusAwaitingApproval RuleStatus = "awaiting_approval"
	StatusActive           RuleStatus = "active"
	StatusSuspended        RuleStatus = "suspended"
	StatusDeprecated       RuleStatus = "deprecated"
	StatusRetired          RuleStatus = "retired"
)

// EnforcementMode represents how violations are handled
type EnforcementMode string

const (
	EnforcementHardBlock EnforcementMode = "hard_block"
	EnforcementSoftBlock EnforcementMode = "soft_block"
	EnforcementLogOnly   EnforcementMode = "log_only"
	EnforcementSimulate  EnforcementMode = "simulate"
	EnforcementDisabled  EnforcementMode = "disabled"
)

// EvaluationStatus represents the result status
type EvaluationStatus string

const (
	EvalPassed        EvaluationStatus = "passed"
	EvalFailed        EvaluationStatus = "failed"
	EvalNotApplicable EvaluationStatus = "not_applicable"
	EvalError         EvaluationStatus = "error"
)

// =============================================================================
// DOMAIN MODELS
// =============================================================================

// Rule represents the core rule definition
type Rule struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	TenantID       uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	DatasourceID   *uuid.UUID      `db:"datasource_id" json:"datasource_id,omitempty"`
	RuleCode       string          `db:"rule_code" json:"rule_code"`
	Name           string          `db:"name" json:"name"`
	Description    string          `db:"description" json:"description"`
	Category       RuleCategory    `db:"category" json:"category"`
	PrimaryContext RuleContextType `db:"primary_context" json:"primary_context"`
	Severity       RuleSeverity    `db:"severity" json:"severity"`
	ScopeEntity    string          `db:"scope_entity" json:"scope_entity,omitempty"`
	ScopeFields    []string        `db:"scope_fields" json:"scope_fields,omitempty"`
	Status         RuleStatus      `db:"status" json:"status"`
	Environment    string          `db:"environment" json:"environment"`
	EffectiveFrom  *time.Time      `db:"effective_from" json:"effective_from,omitempty"`
	EffectiveTo    *time.Time      `db:"effective_to" json:"effective_to,omitempty"`
	OwnerUserID    *uuid.UUID      `db:"owner_user_id" json:"owner_user_id,omitempty"`
	Tags           []string        `db:"tags" json:"tags,omitempty"`
	RegulationIDs  []string        `db:"regulation_ids" json:"regulation_ids,omitempty"`
	ControlIDs     []string        `db:"control_ids" json:"control_ids,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
}

// RuleLogic represents versioned rule logic
type RuleLogic struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	RuleID         uuid.UUID       `db:"rule_id" json:"rule_id"`
	Version        int             `db:"version" json:"version"`
	VersionLabel   string          `db:"version_label" json:"version_label,omitempty"`
	ConditionJSON  json.RawMessage `db:"condition_json" json:"condition_json"`
	ActionsJSON    json.RawMessage `db:"actions_json" json:"actions_json"`
	ScoringFormula string          `db:"scoring_formula" json:"scoring_formula,omitempty"`
	IsApproved     bool            `db:"is_approved" json:"is_approved"`
	ApprovedBy     *uuid.UUID      `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt     *time.Time      `db:"approved_at" json:"approved_at,omitempty"`
	ChangeReason   string          `db:"change_reason" json:"change_reason,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

// RuleWithLogic combines Rule and its active RuleLogic
type RuleWithLogic struct {
	Rule
	Logic       RuleLogic       `json:"logic"`
	Enforcement EnforcementMode `json:"enforcement"`
	TimeoutMs   int             `json:"timeout_ms"`
}

// RuleAction represents an action to take
type RuleAction struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params,omitempty"`
	Order  int                    `json:"order"`
}

// RuleExecutionPolicy defines channel-specific enforcement
type RuleExecutionPolicy struct {
	ID                       uuid.UUID       `db:"id" json:"id"`
	TenantID                 uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	PolicyCode               string          `db:"policy_code" json:"policy_code"`
	Name                     string          `db:"name" json:"name"`
	Channel                  string          `db:"channel" json:"channel"`
	Category                 *RuleCategory   `db:"category" json:"category,omitempty"`
	Enforcement              EnforcementMode `db:"enforcement" json:"enforcement"`
	MaxSeverity              *RuleSeverity   `db:"max_severity" json:"max_severity,omitempty"`
	AllowOverride            bool            `db:"allow_override" json:"allow_override"`
	OverrideRequiresApproval bool            `db:"override_requires_approval" json:"override_requires_approval"`
	TimeoutMs                int             `db:"timeout_ms" json:"timeout_ms"`
	EmitEvents               bool            `db:"emit_events" json:"emit_events"`
	EventTopic               string          `db:"event_topic" json:"event_topic,omitempty"`
	IsActive                 bool            `db:"is_active" json:"is_active"`
	Environment              string          `db:"environment" json:"environment"`
}

// =============================================================================
// EVALUATION TYPES
// =============================================================================

// EvaluationContext provides context for rule evaluation
type EvaluationContext struct {
	// Identity
	TenantID     uuid.UUID `json:"tenant_id"`
	DatasourceID uuid.UUID `json:"datasource_id,omitempty"`

	// Channel info
	Channel     string `json:"channel"`     // e.g., "etl_batch", "realtime_trade_api"
	Environment string `json:"environment"` // dev, test, prod

	// Data to evaluate
	Data        map[string]interface{} `json:"data"`         // Primary record/event
	RelatedData map[string]interface{} `json:"related_data"` // Related entities

	// Temporal context
	EvaluationTime time.Time  `json:"evaluation_time"`
	AsOfTime       *time.Time `json:"as_of_time,omitempty"` // For point-in-time evaluation

	// User context
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	UserRoles []string   `json:"user_roles,omitempty"`

	// Additional context
	Extras map[string]interface{} `json:"extras,omitempty"` // Market data, profiles, etc.
}

// EvaluationResult represents the result of evaluating a single rule
type EvaluationResult struct {
	RuleID   uuid.UUID        `json:"rule_id"`
	RuleCode string           `json:"rule_code"`
	RuleName string           `json:"rule_name"`
	Version  int              `json:"version"`
	Category RuleCategory     `json:"category"`
	Severity RuleSeverity     `json:"severity"`
	Status   EvaluationStatus `json:"status"`

	// Details
	Details EvaluationDetails `json:"details"`

	// Computed score
	Score *float64 `json:"score,omitempty"`

	// Actions
	SuggestedActions []RuleAction `json:"suggested_actions"`
	ExecutedActions  []RuleAction `json:"executed_actions,omitempty"`

	// Policy
	Enforcement EnforcementMode `json:"enforcement"`

	// Timing
	EvaluationTimeMs int64     `json:"evaluation_time_ms"`
	EvaluatedAt      time.Time `json:"evaluated_at"`
}

// EvaluationDetails provides detailed information about the evaluation
type EvaluationDetails struct {
	OperandValues       map[string]interface{} `json:"operand_values,omitempty"`
	DistanceToThreshold *float64               `json:"distance_to_threshold,omitempty"`
	MatchedPaths        []string               `json:"matched_paths,omitempty"`
	FailureReasons      []string               `json:"failure_reasons,omitempty"`
	ConditionResults    []ConditionResult      `json:"condition_results,omitempty"`
}

// ConditionResult represents the result of a single condition
type ConditionResult struct {
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator"`
	Expected interface{} `json:"expected,omitempty"`
	Actual   interface{} `json:"actual,omitempty"`
	Passed   bool        `json:"passed"`
	Message  string      `json:"message,omitempty"`
}

// BatchEvaluationResult represents results of evaluating multiple rules
type BatchEvaluationResult struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	Channel   string    `json:"channel"`
	ContextID string    `json:"context_id,omitempty"`

	Results []EvaluationResult `json:"results"`

	// Aggregates
	TotalRules    int `json:"total_rules"`
	PassedCount   int `json:"passed_count"`
	FailedCount   int `json:"failed_count"`
	NotApplicable int `json:"not_applicable_count"`
	ErrorCount    int `json:"error_count"`

	// Decision
	ShouldBlock     bool               `json:"should_block"`
	BlockingResults []EvaluationResult `json:"blocking_results,omitempty"`

	// Timing
	TotalTimeMs int64     `json:"total_time_ms"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

// =============================================================================
// CONDITION TYPES (matches AdvancedConditionBuilder schema)
// =============================================================================

// ConditionGroup represents a group of conditions with a logical operator
type ConditionGroup struct {
	Type       string        `json:"type"`       // "group"
	Operator   string        `json:"operator"`   // "AND", "OR", "NOT"
	Conditions []interface{} `json:"conditions"` // Can be Condition or ConditionGroup
}

// Condition represents a single condition
type Condition struct {
	Type     string      `json:"type"`     // "condition"
	Field    string      `json:"field"`    // Field name or path
	Operator string      `json:"operator"` // Operator name
	Value    interface{} `json:"value"`    // Expected value

	// Cross-entity support
	EntityPath *EntityPath `json:"entityPath,omitempty"`
}

// EntityPath represents a path to a related entity
type EntityPath struct {
	FromEntity   string `json:"fromEntity"`
	ToEntity     string `json:"toEntity"`
	Relationship string `json:"relationship"`
	Field        string `json:"field"`
}

// =============================================================================
// OPERATOR REGISTRY
// =============================================================================

// Operator defines an evaluation operator
type Operator struct {
	Name        string
	Description string
	Evaluate    func(actual, expected interface{}) (bool, error)
}

// OperatorRegistry holds all available operators
type OperatorRegistry struct {
	operators map[string]Operator
}

// NewOperatorRegistry creates a registry with default operators
func NewOperatorRegistry() *OperatorRegistry {
	r := &OperatorRegistry{
		operators: make(map[string]Operator),
	}
	r.registerDefaultOperators()
	return r
}

func (r *OperatorRegistry) registerDefaultOperators() {
	// Equality operators
	r.Register(Operator{
		Name:        "equals",
		Description: "Exact equality",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected), nil
		},
	})

	r.Register(Operator{
		Name:        "not_equals",
		Description: "Not equal",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected), nil
		},
	})

	// Null checks
	r.Register(Operator{
		Name:        "is_null",
		Description: "Value is null/empty",
		Evaluate: func(actual, _ interface{}) (bool, error) {
			if actual == nil {
				return true, nil
			}
			s := fmt.Sprintf("%v", actual)
			return s == "" || s == "<nil>", nil
		},
	})

	r.Register(Operator{
		Name:        "is_not_null",
		Description: "Value is not null/empty",
		Evaluate: func(actual, _ interface{}) (bool, error) {
			if actual == nil {
				return false, nil
			}
			s := fmt.Sprintf("%v", actual)
			return s != "" && s != "<nil>", nil
		},
	})

	// Comparison operators
	r.Register(Operator{
		Name:        "greater_than",
		Description: "Greater than",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := toFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := toFloat64(expected)
			if err != nil {
				return false, err
			}
			return a > e, nil
		},
	})

	r.Register(Operator{
		Name:        "greater_than_or_equals",
		Description: "Greater than or equal",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := toFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := toFloat64(expected)
			if err != nil {
				return false, err
			}
			return a >= e, nil
		},
	})

	r.Register(Operator{
		Name:        "less_than",
		Description: "Less than",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := toFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := toFloat64(expected)
			if err != nil {
				return false, err
			}
			return a < e, nil
		},
	})

	r.Register(Operator{
		Name:        "less_than_or_equals",
		Description: "Less than or equal",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := toFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := toFloat64(expected)
			if err != nil {
				return false, err
			}
			return a <= e, nil
		},
	})

	r.Register(Operator{
		Name:        "between",
		Description: "Value between two values (inclusive)",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := toFloat64(actual)
			if err != nil {
				return false, err
			}
			bounds, ok := expected.([]interface{})
			if !ok || len(bounds) != 2 {
				return false, fmt.Errorf("between requires [min, max] array")
			}
			min, err := toFloat64(bounds[0])
			if err != nil {
				return false, err
			}
			max, err := toFloat64(bounds[1])
			if err != nil {
				return false, err
			}
			return a >= min && a <= max, nil
		},
	})

	// String operators
	r.Register(Operator{
		Name:        "contains",
		Description: "String contains substring",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return strings.Contains(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "not_contains",
		Description: "String does not contain substring",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return !strings.Contains(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "starts_with",
		Description: "String starts with prefix",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return strings.HasPrefix(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "ends_with",
		Description: "String ends with suffix",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return strings.HasSuffix(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "matches_regex",
		Description: "Matches regular expression",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			pattern := fmt.Sprintf("%v", expected)
			return regexp.MatchString(pattern, a)
		},
	})

	// Set operators
	r.Register(Operator{
		Name:        "in",
		Description: "Value in list",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			list, ok := expected.([]interface{})
			if !ok {
				return false, fmt.Errorf("in operator requires array")
			}
			a := fmt.Sprintf("%v", actual)
			for _, item := range list {
				if fmt.Sprintf("%v", item) == a {
					return true, nil
				}
			}
			return false, nil
		},
	})

	r.Register(Operator{
		Name:        "not_in",
		Description: "Value not in list",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			list, ok := expected.([]interface{})
			if !ok {
				return false, fmt.Errorf("not_in operator requires array")
			}
			a := fmt.Sprintf("%v", actual)
			for _, item := range list {
				if fmt.Sprintf("%v", item) == a {
					return false, nil
				}
			}
			return true, nil
		},
	})

	// Date operators
	r.Register(Operator{
		Name:        "date_before",
		Description: "Date is before",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := parseTime(actual)
			if err != nil {
				return false, err
			}
			e, err := parseTime(expected)
			if err != nil {
				return false, err
			}
			return a.Before(e), nil
		},
	})

	r.Register(Operator{
		Name:        "date_after",
		Description: "Date is after",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := parseTime(actual)
			if err != nil {
				return false, err
			}
			e, err := parseTime(expected)
			if err != nil {
				return false, err
			}
			return a.After(e), nil
		},
	})

	r.Register(Operator{
		Name:        "days_ago_less_than",
		Description: "Date within N days ago",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := parseTime(actual)
			if err != nil {
				return false, err
			}
			days, err := toFloat64(expected)
			if err != nil {
				return false, err
			}
			threshold := time.Now().AddDate(0, 0, -int(days))
			return a.After(threshold), nil
		},
	})
}

// Register adds an operator to the registry
func (r *OperatorRegistry) Register(op Operator) {
	r.operators[op.Name] = op
}

// Get retrieves an operator by name
func (r *OperatorRegistry) Get(name string) (Operator, bool) {
	op, ok := r.operators[name]
	return op, ok
}

// =============================================================================
// RULE EVALUATOR SERVICE
// =============================================================================

// RuleEvaluator is the main evaluation service
type RuleEvaluator struct {
	db        *sqlx.DB
	operators *OperatorRegistry
	celEnv    *cel.Env
}

// NewRuleEvaluator creates a new evaluator service
func NewRuleEvaluator(db *sqlx.DB) (*RuleEvaluator, error) {
	// Create CEL environment for scoring formulas
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("data", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("related", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("context", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &RuleEvaluator{
		db:        db,
		operators: NewOperatorRegistry(),
		celEnv:    env,
	}, nil
}

// Evaluate evaluates a single rule against the context
func (e *RuleEvaluator) Evaluate(ctx context.Context, rule *RuleWithLogic, evalCtx *EvaluationContext) (*EvaluationResult, error) {
	startTime := time.Now()

	result := &EvaluationResult{
		RuleID:      rule.ID,
		RuleCode:    rule.RuleCode,
		RuleName:    rule.Name,
		Version:     rule.Logic.Version,
		Category:    rule.Category,
		Severity:    rule.Severity,
		Enforcement: rule.Enforcement,
		EvaluatedAt: startTime,
		Details:     EvaluationDetails{},
	}

	// Parse condition JSON
	var conditionGroup ConditionGroup
	if err := json.Unmarshal(rule.Logic.ConditionJSON, &conditionGroup); err != nil {
		result.Status = EvalError
		result.Details.FailureReasons = []string{fmt.Sprintf("failed to parse condition: %v", err)}
		result.EvaluationTimeMs = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Evaluate condition tree
	passed, details := e.evaluateConditionGroup(&conditionGroup, evalCtx.Data, evalCtx.RelatedData)

	result.Details = details
	if passed {
		result.Status = EvalPassed
	} else {
		result.Status = EvalFailed
	}

	// Parse and attach suggested actions if failed
	if !passed {
		var actions []RuleAction
		if err := json.Unmarshal(rule.Logic.ActionsJSON, &actions); err == nil {
			result.SuggestedActions = actions
		}
	}

	// Calculate score if formula provided
	if rule.Logic.ScoringFormula != "" {
		score, err := e.calculateScore(rule.Logic.ScoringFormula, evalCtx)
		if err == nil {
			result.Score = &score
		}
	}

	result.EvaluationTimeMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// EvaluateBatch evaluates multiple rules against the context
func (e *RuleEvaluator) EvaluateBatch(ctx context.Context, rules []*RuleWithLogic, evalCtx *EvaluationContext) (*BatchEvaluationResult, error) {
	startTime := time.Now()

	batch := &BatchEvaluationResult{
		TenantID:    evalCtx.TenantID,
		Channel:     evalCtx.Channel,
		EvaluatedAt: startTime,
		Results:     make([]EvaluationResult, 0, len(rules)),
	}

	for _, rule := range rules {
		result, err := e.Evaluate(ctx, rule, evalCtx)
		if err != nil {
			result = &EvaluationResult{
				RuleID:   rule.ID,
				RuleCode: rule.RuleCode,
				RuleName: rule.Name,
				Status:   EvalError,
				Details: EvaluationDetails{
					FailureReasons: []string{err.Error()},
				},
			}
		}

		batch.Results = append(batch.Results, *result)
		batch.TotalRules++

		switch result.Status {
		case EvalPassed:
			batch.PassedCount++
		case EvalFailed:
			batch.FailedCount++
			// Check if this is a blocking failure
			if result.Enforcement == EnforcementHardBlock || result.Enforcement == EnforcementSoftBlock {
				if result.Severity == SeverityHardBlock || result.Severity == SeverityError {
					batch.ShouldBlock = true
					batch.BlockingResults = append(batch.BlockingResults, *result)
				}
			}
		case EvalNotApplicable:
			batch.NotApplicable++
		case EvalError:
			batch.ErrorCount++
		}
	}

	batch.TotalTimeMs = time.Since(startTime).Milliseconds()
	return batch, nil
}

// GetRulesForEvaluation retrieves active rules matching the criteria
func (e *RuleEvaluator) GetRulesForEvaluation(ctx context.Context, tenantID uuid.UUID, opts GetRulesOptions) ([]*RuleWithLogic, error) {
	query := `
		SELECT DISTINCT
			r.id, r.tenant_id, r.datasource_id, r.rule_code, r.name, r.description,
			r.category, r.primary_context, r.severity, r.scope_entity, r.scope_fields,
			r.status, r.environment, r.effective_from, r.effective_to, r.owner_user_id,
			r.tags, r.regulation_ids, r.control_ids, r.created_at, r.updated_at,
			rl.id as logic_id, rl.version, rl.version_label, rl.condition_json, 
			rl.actions_json, rl.scoring_formula, rl.is_approved,
			COALESCE(ep.enforcement, 'hard_block') as enforcement,
			COALESCE(ep.timeout_ms, 5000) as timeout_ms
		FROM rules r
		JOIN rule_logic rl ON r.id = rl.rule_id
		LEFT JOIN rule_execution_policies ep ON (
			ep.tenant_id = r.tenant_id 
			AND ep.environment = r.environment
			AND ep.is_active = TRUE
			AND (ep.category IS NULL OR ep.category = r.category)
			AND ($5::text IS NULL OR ep.channel = $5)
		)
		WHERE r.tenant_id = $1
		  AND r.status = 'active'
		  AND r.environment = $2
		  AND rl.is_approved = TRUE
		  AND ($3::text IS NULL OR r.category = $3)
		  AND ($4::text IS NULL OR r.primary_context = $4)
		  AND ($6::text IS NULL OR r.scope_entity = $6)
		  AND (r.effective_from IS NULL OR r.effective_from <= NOW())
		  AND (r.effective_to IS NULL OR r.effective_to >= NOW())
		ORDER BY r.category, r.rule_code
	`

	rows, err := e.db.QueryContext(ctx, query,
		tenantID,
		opts.Environment,
		opts.Category,
		opts.ContextType,
		opts.Channel,
		opts.Entity,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []*RuleWithLogic
	for rows.Next() {
		var r RuleWithLogic
		var logicID uuid.UUID

		err := rows.Scan(
			&r.ID, &r.TenantID, &r.DatasourceID, &r.RuleCode, &r.Name, &r.Description,
			&r.Category, &r.PrimaryContext, &r.Severity, &r.ScopeEntity, &r.ScopeFields,
			&r.Status, &r.Environment, &r.EffectiveFrom, &r.EffectiveTo, &r.OwnerUserID,
			&r.Tags, &r.RegulationIDs, &r.ControlIDs, &r.CreatedAt, &r.UpdatedAt,
			&logicID, &r.Logic.Version, &r.Logic.VersionLabel, &r.Logic.ConditionJSON,
			&r.Logic.ActionsJSON, &r.Logic.ScoringFormula, &r.Logic.IsApproved,
			&r.Enforcement, &r.TimeoutMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		r.Logic.ID = logicID
		r.Logic.RuleID = r.ID
		rules = append(rules, &r)
	}

	return rules, nil
}

// GetRulesOptions specifies criteria for fetching rules
type GetRulesOptions struct {
	Environment string           `json:"environment"`
	Category    *RuleCategory    `json:"category,omitempty"`
	ContextType *RuleContextType `json:"context_type,omitempty"`
	Channel     string           `json:"channel,omitempty"`
	Entity      string           `json:"entity,omitempty"`
}

// evaluateConditionGroup evaluates a condition group recursively
func (e *RuleEvaluator) evaluateConditionGroup(group *ConditionGroup, data, related map[string]interface{}) (bool, EvaluationDetails) {
	details := EvaluationDetails{
		OperandValues:    make(map[string]interface{}),
		ConditionResults: []ConditionResult{},
	}

	if len(group.Conditions) == 0 {
		return true, details // Empty group passes
	}

	results := make([]bool, 0, len(group.Conditions))

	for _, cond := range group.Conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _ := condMap["type"].(string)

		if condType == "group" {
			// Recursive evaluation
			var subGroup ConditionGroup
			condBytes, _ := json.Marshal(condMap)
			json.Unmarshal(condBytes, &subGroup)

			passed, subDetails := e.evaluateConditionGroup(&subGroup, data, related)
			results = append(results, passed)

			// Merge details
			for k, v := range subDetails.OperandValues {
				details.OperandValues[k] = v
			}
			details.ConditionResults = append(details.ConditionResults, subDetails.ConditionResults...)
			if !passed {
				details.FailureReasons = append(details.FailureReasons, subDetails.FailureReasons...)
			}
		} else {
			// Single condition
			var condition Condition
			condBytes, _ := json.Marshal(condMap)
			json.Unmarshal(condBytes, &condition)

			passed, result := e.evaluateCondition(&condition, data, related)
			results = append(results, passed)
			details.ConditionResults = append(details.ConditionResults, result)
			details.OperandValues[condition.Field] = result.Actual

			if !passed && result.Message != "" {
				details.FailureReasons = append(details.FailureReasons, result.Message)
			}
		}
	}

	// Apply logical operator
	var finalResult bool
	switch strings.ToUpper(group.Operator) {
	case "AND":
		finalResult = true
		for _, r := range results {
			if !r {
				finalResult = false
				break
			}
		}
	case "OR":
		finalResult = false
		for _, r := range results {
			if r {
				finalResult = true
				break
			}
		}
	case "NOT":
		if len(results) > 0 {
			finalResult = !results[0]
		}
	default:
		finalResult = true
		for _, r := range results {
			if !r {
				finalResult = false
				break
			}
		}
	}

	return finalResult, details
}

// evaluateCondition evaluates a single condition
func (e *RuleEvaluator) evaluateCondition(cond *Condition, data, related map[string]interface{}) (bool, ConditionResult) {
	result := ConditionResult{
		Field:    cond.Field,
		Operator: cond.Operator,
		Expected: cond.Value,
	}

	// Get actual value
	var actual interface{}
	if cond.EntityPath != nil {
		// Cross-entity lookup
		if relatedEntity, ok := related[cond.EntityPath.ToEntity]; ok {
			if entityMap, ok := relatedEntity.(map[string]interface{}); ok {
				actual = entityMap[cond.EntityPath.Field]
			}
		}
	} else {
		// Direct field lookup
		actual = getNestedValue(data, cond.Field)
	}
	result.Actual = actual

	// Get operator
	op, ok := e.operators.Get(cond.Operator)
	if !ok {
		result.Message = fmt.Sprintf("unknown operator: %s", cond.Operator)
		return false, result
	}

	// Evaluate
	passed, err := op.Evaluate(actual, cond.Value)
	if err != nil {
		result.Message = fmt.Sprintf("evaluation error: %v", err)
		return false, result
	}

	result.Passed = passed
	if !passed {
		result.Message = fmt.Sprintf("Field '%s' %s %v (actual: %v)", cond.Field, cond.Operator, cond.Value, actual)
	}

	return passed, result
}

// calculateScore evaluates a CEL scoring formula
func (e *RuleEvaluator) calculateScore(formula string, evalCtx *EvaluationContext) (float64, error) {
	ast, issues := e.celEnv.Compile(formula)
	if issues != nil && issues.Err() != nil {
		return 0, issues.Err()
	}

	prg, err := e.celEnv.Program(ast)
	if err != nil {
		return 0, err
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"data":    evalCtx.Data,
		"related": evalCtx.RelatedData,
		"context": evalCtx.Extras,
	})
	if err != nil {
		return 0, err
	}

	switch v := out.Value().(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("score formula must return number")
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	case json.Number:
		return val.Float64()
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func parseTime(v interface{}) (time.Time, error) {
	switch val := v.(type) {
	case time.Time:
		return val, nil
	case string:
		// Try common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02",
			"01/02/2006",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse time: %s", val)
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time", v)
	}
}

func getNestedValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}
