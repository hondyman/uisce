package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/models"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// RuleEngine applies low-code matching rules
type RuleEngine struct {
	db     *sql.DB
	hasura HasuraClient
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(db *sql.DB) *RuleEngine {
	return &RuleEngine{db: db}
}

// NewRuleEngineWithHasura creates a new rule engine with Hasura support
func NewRuleEngineWithHasura(db *sql.DB, hasura HasuraClient) *RuleEngine {
	return &RuleEngine{db: db, hasura: hasura}
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
	if re.hasura != nil {
		rules, err := re.getActiveRulesWithHasura(ctx)
		if err == nil {
			return rules, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	// TODO: This SQL fallback can be removed once Hasura is fully deployed
	//   Hasura implementation already exists in getActiveRulesWithHasura() above
	//   Query: reconciliation_rules(where: {enabled: {_eq: true}}, order_by: [{rule_type: asc}, {updated_at: desc}])
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

func (re *RuleEngine) getActiveRulesWithHasura(ctx context.Context) ([]models.ReconciliationRule, error) {
	query := `
		query GetActiveRules {
			reconciliation_rules(where: {enabled: {_eq: true}}, order_by: [{rule_type: asc}, {updated_at: desc}]) {
				id
				name
				description
				rule_type
				enabled
				rule_expr
				version
				created_at
				updated_at
			}
		}
	`

	resp, err := re.hasura.Query(query, nil)
	if err != nil {
		return nil, err
	}

	rulesData, ok := resp["reconciliation_rules"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	var rules []models.ReconciliationRule
	for _, item := range rulesData {
		ruleMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		r := models.ReconciliationRule{}
		if id, ok := ruleMap["id"].(string); ok {
			r.ID, _ = uuid.Parse(id)
		}
		if name, ok := ruleMap["name"].(string); ok {
			r.Name = name
		}
		if description, ok := ruleMap["description"].(string); ok {
			r.Description = description
		}
		if ruleType, ok := ruleMap["rule_type"].(string); ok {
			r.RuleType = ruleType
		}
		if enabled, ok := ruleMap["enabled"].(bool); ok {
			r.Enabled = enabled
		}
		if ruleExpr, ok := ruleMap["rule_expr"].(string); ok {
			r.RuleExpr = ruleExpr
		}
		if version, ok := ruleMap["version"].(float64); ok {
			r.Version = int(version)
		}
		if createdAt, ok := ruleMap["created_at"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, createdAt); err == nil {
				r.CreatedAt = parsedTime
			}
		}
		if updatedAt, ok := ruleMap["updated_at"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				r.UpdatedAt = parsedTime
			}
		}

		rules = append(rules, r)
	}

	return rules, nil
}

// CreateOrUpdateRule creates or updates a rule
func (re *RuleEngine) CreateOrUpdateRule(ctx context.Context, rule models.ReconciliationRule) error {
	if re.hasura != nil {
		err := re.createOrUpdateRuleWithHasura(ctx, rule)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	// TODO: This SQL fallback can be removed once Hasura is fully deployed
	//   Hasura implementation already exists in createOrUpdateRuleWithHasura() above
	//   Mutation: insert_reconciliation_rules_one with on_conflict upsert (constraint: reconciliation_rules_name_key)
	//   Note: version = version + 1 increments on conflict, updated_at uses now()
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

func (re *RuleEngine) createOrUpdateRuleWithHasura(ctx context.Context, rule models.ReconciliationRule) error {
	mutation := `
		mutation UpsertRule($rule: reconciliation_rules_insert_input!) {
			insert_reconciliation_rules_one(
				object: $rule,
				on_conflict: {
					constraint: reconciliation_rules_name_key,
					update_columns: [description, rule_expr, version, updated_at]
				}
			) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"rule": map[string]interface{}{
			"id":          rule.ID.String(),
			"name":        rule.Name,
			"description": rule.Description,
			"rule_type":   rule.RuleType,
			"enabled":     rule.Enabled,
			"rule_expr":   rule.RuleExpr,
			"version":     rule.Version,
		},
	}

	_, err := re.hasura.Mutate(mutation, variables)
	return err
}

// applyRule applies a specific rule to discrepancies (placeholder for JSONata evaluation)
func (re *RuleEngine) applyRule(ctx context.Context, rule models.ReconciliationRule, discrepancies *[]models.Discrepancy) error {
	// TODO: Implement JSONata evaluation
	// For now, this is a placeholder that would evaluate rule.RuleExpr
	fmt.Printf("Applying rule: %s (type: %s)\n", rule.Name, rule.RuleType)
	return nil
}
