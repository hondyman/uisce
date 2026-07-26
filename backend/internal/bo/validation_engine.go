package bo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"go.uber.org/zap"
)

// ValidationSeverity defines the severity level of a validation violation.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "ERROR"
	SeverityWarning ValidationSeverity = "WARNING"
	SeverityInfo    ValidationSeverity = "INFO"
)

// ValidationRule is a CEL-backed rule stored in bo.validation_rule.
type ValidationRule struct {
	RuleID       string             `db:"rule_id"      json:"rule_id"`
	TenantID     string             `db:"tenant_id"    json:"tenant_id"`
	BOKey        string             `db:"bo_key"       json:"bo_key"`
	FieldKey     *string            `db:"field_key"    json:"field_key,omitempty"`
	RuleName     string             `db:"rule_name"    json:"rule_name"`
	Description  string             `db:"description"  json:"description"`
	Expression   string             `db:"expression"   json:"expression"`
	ErrorMessage string             `db:"error_message" json:"error_message"`
	Severity     ValidationSeverity `db:"severity"     json:"severity"`
	Priority     int                `db:"priority"     json:"priority"`
	IsActive     bool               `db:"is_active"    json:"is_active"`
	IsCore       bool               `db:"is_core"      json:"is_core"`
	CreatedAt    time.Time          `db:"created_at"   json:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"   json:"updated_at"`
}

// ValidationViolation is a single rule failure returned to callers.
type ValidationViolation struct {
	RuleID       string             `json:"rule_id"`
	RuleName     string             `json:"rule_name"`
	FieldKey     *string            `json:"field_key,omitempty"`
	Severity     ValidationSeverity `json:"severity"`
	ErrorMessage string             `json:"error_message"`
}

// ValidationResult is the aggregated result for a BO record validation pass.
type ValidationResult struct {
	BOKey      string                `json:"bo_key"`
	Valid       bool                  `json:"valid"`    // false if any ERROR violations exist
	Violations  []ValidationViolation `json:"violations"`
	EvaluatedAt time.Time             `json:"evaluated_at"`
}

// ValidationEngine evaluates CEL-backed validation rules against a record.
type ValidationEngine struct {
	db  *sql.DB
	log *zap.Logger
}

// NewValidationEngine constructs a ValidationEngine.
func NewValidationEngine(db *sql.DB, log *zap.Logger) *ValidationEngine {
	return &ValidationEngine{db: db, log: log}
}

// LoadRules loads all active rules for a given tenant + BO from the database.
// Rule 7.4: Union of Core (is_core = true) and tenant-custom rules using
// ROW_NUMBER() window to ensure Core rules are inherited, tenant rules shadow.
func (ve *ValidationEngine) LoadRules(ctx context.Context, tenantID, boKey string) ([]ValidationRule, error) {
	const q = `
	WITH combined AS (
		SELECT *,
		       ROW_NUMBER() OVER (
		           PARTITION BY COALESCE(field_key, ''), rule_name
		           ORDER BY CASE WHEN is_core = false THEN 0 ELSE 1 END
		       ) AS rn
		FROM bo.validation_rule
		WHERE bo_key = $2
		  AND (tenant_id = $1::uuid OR is_core = true)
		  AND is_active = true
	)
	SELECT rule_id, tenant_id, bo_key, field_key, rule_name, description,
	       expression, error_message, severity, priority, is_active, is_core,
	       created_at, updated_at
	FROM combined
	WHERE rn = 1
	ORDER BY priority ASC, rule_name ASC
	`
	rows, err := ve.db.QueryContext(ctx, q, tenantID, boKey)
	if err != nil {
		return nil, fmt.Errorf("validation_engine: load rules: %w", err)
	}
	defer rows.Close()

	var rules []ValidationRule
	for rows.Next() {
		var r ValidationRule
		if err := rows.Scan(
			&r.RuleID, &r.TenantID, &r.BOKey, &r.FieldKey, &r.RuleName,
			&r.Description, &r.Expression, &r.ErrorMessage, &r.Severity,
			&r.Priority, &r.IsActive, &r.IsCore, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("validation_engine: scan rule: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// Evaluate evaluates all active rules for a BO against the provided record map.
// record is a map[string]interface{} representing the BO instance under evaluation.
func (ve *ValidationEngine) Evaluate(ctx context.Context, tenantID, boKey string, record map[string]interface{}) (*ValidationResult, error) {
	rules, err := ve.LoadRules(ctx, tenantID, boKey)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{
		BOKey:       boKey,
		Valid:        true,
		EvaluatedAt: time.Now().UTC(),
	}

	for _, rule := range rules {
		// Build CEL environment with the record available as "record"
		env, err := cel.NewEnv(
			cel.Declarations(
				decls.NewVar("record", decls.NewMapType(decls.String, decls.Dyn)),
			),
		)
		if err != nil {
			ve.log.Error("cel: build env failed", zap.String("rule", rule.RuleName), zap.Error(err))
			continue
		}

		ast, issues := env.Compile(rule.Expression)
		if issues != nil && issues.Err() != nil {
			ve.log.Warn("cel: compile error, skipping rule",
				zap.String("rule", rule.RuleName),
				zap.Error(issues.Err()),
			)
			continue
		}

		prg, err := env.Program(ast)
		if err != nil {
			ve.log.Warn("cel: program error, skipping rule", zap.String("rule", rule.RuleName), zap.Error(err))
			continue
		}

		out, _, err := prg.Eval(map[string]interface{}{"record": record})
		if err != nil {
			ve.log.Warn("cel: eval error, treating as violation",
				zap.String("rule", rule.RuleName),
				zap.Error(err),
			)
			// Eval errors count as violations at error severity
			result.Violations = append(result.Violations, ValidationViolation{
				RuleID:       rule.RuleID,
				RuleName:     rule.RuleName,
				FieldKey:     rule.FieldKey,
				Severity:     SeverityError,
				ErrorMessage: fmt.Sprintf("Rule evaluation error: %v", err),
			})
			result.Valid = false
			continue
		}

		// CEL expression should return bool; false = rule violated
		passed, ok := out.Value().(bool)
		if !ok || !passed {
			violation := ValidationViolation{
				RuleID:       rule.RuleID,
				RuleName:     rule.RuleName,
				FieldKey:     rule.FieldKey,
				Severity:     rule.Severity,
				ErrorMessage: rule.ErrorMessage,
			}
			result.Violations = append(result.Violations, violation)
			if rule.Severity == SeverityError {
				result.Valid = false
			}
		}
	}

	return result, nil
}

// UpsertRule creates or updates a validation rule.
// Rule 1.3: validates UUID parameter before binding.
func (ve *ValidationEngine) UpsertRule(ctx context.Context, tenantID string, rule *ValidationRule) error {
	rule.TenantID = tenantID
	rule.UpdatedAt = time.Now().UTC()
	const q = `
	INSERT INTO bo.validation_rule
	    (rule_id, tenant_id, bo_key, field_key, rule_name, description,
	     expression, error_message, severity, priority, is_active, is_core,
	     created_by, created_at, updated_at)
	VALUES
	    (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()), $2::uuid, $3, $4, $5,
	     $6, $7, $8, $9, $10, $11, $12, $13::uuid, NOW(), NOW())
	ON CONFLICT (rule_id) DO UPDATE SET
	    rule_name    = EXCLUDED.rule_name,
	    description  = EXCLUDED.description,
	    expression   = EXCLUDED.expression,
	    error_message= EXCLUDED.error_message,
	    severity     = EXCLUDED.severity,
	    priority     = EXCLUDED.priority,
	    is_active    = EXCLUDED.is_active,
	    updated_at   = NOW()
	RETURNING rule_id
	`
	return ve.db.QueryRowContext(ctx, q,
		rule.RuleID, tenantID, rule.BOKey, rule.FieldKey, rule.RuleName,
		rule.Description, rule.Expression, rule.ErrorMessage, rule.Severity,
		rule.Priority, rule.IsActive, rule.IsCore, nil,
	).Scan(&rule.RuleID)
}

// DeleteRule soft-deletes a validation rule by marking it inactive.
// Rule 7: validates that the rule belongs to the requesting tenant.
func (ve *ValidationEngine) DeleteRule(ctx context.Context, tenantID, ruleID string) error {
	res, err := ve.db.ExecContext(ctx, `
		UPDATE bo.validation_rule SET is_active = false, updated_at = NOW()
		WHERE rule_id = $1::uuid AND tenant_id = $2::uuid
	`, ruleID, tenantID)
	if err != nil {
		return fmt.Errorf("validation_engine: delete rule: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("validation_engine: rule not found or not owned by tenant")
	}
	return nil
}

// TestExpression compiles and evaluates a single CEL expression against a sample
// record without persisting anything — used by the BO Studio live test harness.
func (ve *ValidationEngine) TestExpression(expression string, sample map[string]interface{}) (bool, string, error) {
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("record", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		return false, "", fmt.Errorf("cel: build env: %w", err)
	}
	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, "", fmt.Errorf("cel: compile: %w", issues.Err())
	}
	prg, err := env.Program(ast)
	if err != nil {
		return false, "", fmt.Errorf("cel: program: %w", err)
	}
	out, _, err := prg.Eval(map[string]interface{}{"record": sample})
	if err != nil {
		return false, fmt.Sprintf("eval error: %v", err), err
	}
	b, ok := out.Value().(bool)
	if !ok {
		encoded, _ := json.Marshal(out.Value())
		return false, string(encoded), fmt.Errorf("expression must return bool, got %T", out.Value())
	}
	return b, fmt.Sprintf("%v", b), nil
}
