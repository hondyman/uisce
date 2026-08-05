package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/services/ai-trade-reconciliation/backend/internal/models"
)

// RuleEngine applies low-code matching rules
type RuleEngine struct {
	db *sql.DB
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(db *sql.DB) *RuleEngine {
	return &RuleEngine{db: db}
}

// ApplyRules applies all active rules to a reconciliation result
func (re *RuleEngine) ApplyRules(ctx context.Context, result *models.ReconciliationResult) error {
	rules, err := re.GetActiveRules(ctx)
	if err != nil {
		return err
	}

	discrepancies, err := result.GetDiscrepancies()
	if err != nil {
		return err
	}

	// Apply each rule and update discrepancies
	for _, rule := range rules {
		if err := re.applyRule(ctx, rule, &discrepancies); err != nil {
			fmt.Printf("Error applying rule %s: %v\n", rule.Name, err)
			continue
		}
	}

	// Re-marshal discrepancies back to JSON
	updatedJSON, err := json.Marshal(discrepancies)
	if err != nil {
		return err
	}
	result.DiscrepancyJSON = updatedJSON

	return nil
}

// GetActiveRules retrieves all enabled rules from database
func (re *RuleEngine) GetActiveRules(ctx context.Context) ([]models.ReconciliationRule, error) {
	rows, err := re.db.QueryContext(ctx, `
		SELECT id, name, description, rule_type, enabled, rule_expr, version, created_at, updated_at
		FROM reconciliation_rules
		WHERE enabled = true
		ORDER BY rule_type, updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.ReconciliationRule
	for rows.Next() {
		var r models.ReconciliationRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.RuleType, &r.Enabled, &r.RuleExpr, &r.Version, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return rules, rows.Err()
}

// CreateOrUpdateRule creates or updates a rule
func (re *RuleEngine) CreateOrUpdateRule(ctx context.Context, rule models.ReconciliationRule) error {
	_, err := re.db.ExecContext(ctx, `
		INSERT INTO reconciliation_rules
			(id, name, description, rule_type, enabled, rule_expr, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (name) DO UPDATE SET
			description = $3,
			rule_expr = $6,
			version = version + 1,
			updated_at = NOW()
	`, rule.ID, rule.Name, rule.Description, rule.RuleType, rule.Enabled, rule.RuleExpr, rule.Version)

	return err
}

// applyRule applies a specific rule to discrepancies (placeholder for JSONata evaluation)
func (re *RuleEngine) applyRule(ctx context.Context, rule models.ReconciliationRule, discrepancies *[]models.Discrepancy) error {
	// TODO: Implement JSONata evaluation
	// For now, this is a placeholder that would evaluate rule.RuleExpr
	fmt.Printf("Applying rule: %s (type: %s)\n", rule.Name, rule.RuleType)
	return nil
}
