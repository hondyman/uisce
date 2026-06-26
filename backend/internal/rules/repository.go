package rules

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

// RuleRepository defines the interface for accessing compliance rules.
type RuleRepository interface {
	CreateRule(ctx context.Context, rule *ComplianceRule) error
	GetRule(ctx context.Context, id uuid.UUID) (*ComplianceRule, error)
	ListRules(ctx context.Context, ruleType string) ([]ComplianceRule, error)
	UpdateRule(ctx context.Context, rule *ComplianceRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
}

// CoreRuleRepository defines the interface for accessing core validation rules
type CoreRuleRepository interface {
	GetCoreRuleByID(ctx context.Context, coreRuleID string) (*CoreValidationRule, error)
	ListActiveCoreRules(ctx context.Context) ([]CoreValidationRule, error)
}

// SQLRuleRepository implements RuleRepository using a SQL database.
type SQLRuleRepository struct {
	db           *sql.DB
	hasuraClient HasuraClient
}

// NewSQLRuleRepository creates a new SQLRuleRepository.
func NewSQLRuleRepository(db *sql.DB) *SQLRuleRepository {
	return &SQLRuleRepository{db: db}
}

func NewSQLRuleRepositoryWithHasura(db *sql.DB, hasuraClient HasuraClient) *SQLRuleRepository {
	return &SQLRuleRepository{
		db:           db,
		hasuraClient: hasuraClient,
	}
}

func (r *SQLRuleRepository) CreateRule(ctx context.Context, rule *ComplianceRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	return r.createRuleRecord(ctx, rule)
}

func (r *SQLRuleRepository) GetRule(ctx context.Context, id uuid.UUID) (*ComplianceRule, error) {
	return r.getRuleRecord(ctx, id)
}

func (r *SQLRuleRepository) ListRules(ctx context.Context, ruleType string) ([]ComplianceRule, error) {
	return r.listRulesRecords(ctx, ruleType)
}

func (r *SQLRuleRepository) UpdateRule(ctx context.Context, rule *ComplianceRule) error {
	return r.updateRuleRecord(ctx, rule)
}

func (r *SQLRuleRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return r.deleteRuleRecord(ctx, id)
}

// Helper methods for Hasura integration - SQL fallback for complex operations

func (r *SQLRuleRepository) createRuleRecord(ctx context.Context, rule *ComplianceRule) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for INSERT
	query := `
		INSERT INTO compliance_rules (id, name, description, rule_type, expression, severity, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query, rule.ID, rule.Name, rule.Description, rule.RuleType, rule.Expression, rule.Severity, rule.Enabled)
	return err
}

func (r *SQLRuleRepository) getRuleRecord(ctx context.Context, id uuid.UUID) (*ComplianceRule, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT by ID
	query := `
		SELECT id, name, description, rule_type, expression, severity, enabled, created_at, updated_at
		FROM compliance_rules
		WHERE id = $1
	`
	var rule ComplianceRule
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.RuleType, &rule.Expression, &rule.Severity, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *SQLRuleRepository) listRulesRecords(ctx context.Context, ruleType string) ([]ComplianceRule, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for dynamic WHERE clause
	query := `
		SELECT id, name, description, rule_type, expression, severity, enabled, created_at, updated_at
		FROM compliance_rules
	`
	var args []interface{}
	if ruleType != "" {
		query += " WHERE rule_type = $1"
		args = append(args, ruleType)
	}
	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []ComplianceRule
	for rows.Next() {
		var rule ComplianceRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.RuleType, &rule.Expression, &rule.Severity, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *SQLRuleRepository) updateRuleRecord(ctx context.Context, rule *ComplianceRule) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for multi-field UPDATE
	query := `
		UPDATE compliance_rules
		SET name = $1, description = $2, rule_type = $3, expression = $4, severity = $5, enabled = $6, updated_at = NOW()
		WHERE id = $7
	`
	result, err := r.db.ExecContext(ctx, query, rule.Name, rule.Description, rule.RuleType, rule.Expression, rule.Severity, rule.Enabled, rule.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (r *SQLRuleRepository) deleteRuleRecord(ctx context.Context, id uuid.UUID) error {
	// TODO: Use HasuraClient for DELETE when available
	// For now, use SQL fallback for DELETE
	query := `DELETE FROM compliance_rules WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

// CoreRuleRepository implementation

func (r *SQLRuleRepository) GetCoreRuleByID(ctx context.Context, coreRuleID string) (*CoreValidationRule, error) {
	query := `
		SELECT core_rule_id, rule_key, version, module_name, entrypoint, condition_src, is_active, created_at
		FROM validationrule_core
		WHERE core_rule_id = $1
	`
	var rule CoreValidationRule
	err := r.db.QueryRowContext(ctx, query, coreRuleID).Scan(
		&rule.CoreRuleID, &rule.RuleKey, &rule.Version, &rule.ModuleName, &rule.Entrypoint, &rule.ConditionSrc, &rule.IsActive, &rule.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *SQLRuleRepository) ListActiveCoreRules(ctx context.Context) ([]CoreValidationRule, error) {
	query := `
		SELECT core_rule_id, rule_key, version, module_name, entrypoint, condition_src, is_active, created_at
		FROM validationrule_core
		WHERE is_active = true
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []CoreValidationRule
	for rows.Next() {
		var rule CoreValidationRule
		if err := rows.Scan(&rule.CoreRuleID, &rule.RuleKey, &rule.Version, &rule.ModuleName, &rule.Entrypoint, &rule.ConditionSrc, &rule.IsActive, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// ScenarioRepository Implementation

func (r *SQLRuleRepository) CreateScenario(ctx context.Context, scenario *RuleScenario) error {
	query := `
		INSERT INTO rule_scenario (id, tenant_id, base_rule_id, name, description, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query, scenario.ID, scenario.TenantID, scenario.BaseRuleID, scenario.Name, scenario.Description, scenario.Status, scenario.CreatedBy, scenario.CreatedAt, scenario.UpdatedAt)
	return err
}

func (r *SQLRuleRepository) CreateScenarioVersion(ctx context.Context, version *RuleScenarioVersion) error {
	query := `
		INSERT INTO rule_scenario_version (id, scenario_id, version, rule_snapshot, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, version.ID, version.ScenarioID, version.Version, version.RuleSnapshot, version.CreatedBy, version.CreatedAt)
	return err
}

func (r *SQLRuleRepository) GetScenario(ctx context.Context, id string) (*RuleScenario, error) {
	query := `SELECT id, tenant_id, base_rule_id, name, description, status, created_by, created_at, updated_at FROM rule_scenario WHERE id = $1`
	var s RuleScenario
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.TenantID, &s.BaseRuleID, &s.Name, &s.Description, &s.Status, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SQLRuleRepository) GetScenarioVersion(ctx context.Context, id string) (*RuleScenarioVersion, error) {
	query := `SELECT id, scenario_id, version, rule_snapshot, created_by, created_at FROM rule_scenario_version WHERE id = $1`
	var v RuleScenarioVersion
	err := r.db.QueryRowContext(ctx, query, id).Scan(&v.ID, &v.ScenarioID, &v.Version, &v.RuleSnapshot, &v.CreatedBy, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *SQLRuleRepository) GetLatestScenarioVersion(ctx context.Context, scenarioID string) (*RuleScenarioVersion, error) {
	query := `SELECT id, scenario_id, version, rule_snapshot, created_by, created_at FROM rule_scenario_version WHERE scenario_id = $1 ORDER BY version DESC LIMIT 1`
	var v RuleScenarioVersion
	err := r.db.QueryRowContext(ctx, query, scenarioID).Scan(&v.ID, &v.ScenarioID, &v.Version, &v.RuleSnapshot, &v.CreatedBy, &v.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil if no versions exist
		}
		return nil, err
	}
	return &v, nil
}

func (r *SQLRuleRepository) CreateTestRun(ctx context.Context, run *RuleTestRun) error {
	query := `
		INSERT INTO rule_test_run (id, tenant_id, scenario_version_id, status, sample_size, failure_count, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query, run.ID, run.TenantID, run.ScenarioVersionID, run.Status, run.SampleSize, run.FailureCount, run.StartedAt)
	return err
}

func (r *SQLRuleRepository) UpdateTestRun(ctx context.Context, run *RuleTestRun) error {
	query := `
		UPDATE rule_test_run
		SET status = $1, failure_count = $2, completed_at = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, run.Status, run.FailureCount, run.CompletedAt, run.ID)
	return err
}

func (r *SQLRuleRepository) GetTestRun(ctx context.Context, id string) (*RuleTestRun, error) {
	query := `SELECT id, tenant_id, scenario_version_id, status, sample_size, failure_count, started_at, completed_at FROM rule_test_run WHERE id = $1`
	var tr RuleTestRun
	err := r.db.QueryRowContext(ctx, query, id).Scan(&tr.ID, &tr.TenantID, &tr.ScenarioVersionID, &tr.Status, &tr.SampleSize, &tr.FailureCount, &tr.StartedAt, &tr.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &tr, nil
}
