// backend/internal/rules/validation_engine_hierarchy.go

package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/lib/pq"
)

type ValidationEngineWithHierarchy struct {
	db        *sql.DB
	evaluator *ConditionEvaluator
	logger    *log.Logger
}

func NewValidationEngineWithHierarchy(
	db *sql.DB,
	logger *log.Logger,
) *ValidationEngineWithHierarchy {
	return &ValidationEngineWithHierarchy{
		db:        db,
		evaluator: NewConditionEvaluator(),
		logger:    logger,
	}
}

// ValidateHierarchical validates data with hierarchy rules.
func (ve *ValidationEngineWithHierarchy) ValidateHierarchical(
	ctx context.Context,
	entity string,
	data map[string]interface{},
	tenantID string,
	datasourceID string,
) (bool, []ValidationError, error) {
	rules, err := ve.getHierarchyRules(ctx, entity, tenantID, datasourceID)
	if err != nil {
		return false, nil, err
	}

	var errors []ValidationError
	allPassed := true

	for _, rule := range rules {
		passed, err := ve.evaluateHierarchyRule(rule, data)
		if err != nil {
			ve.logger.Printf("Error evaluating hierarchy rule %s (%s): %v", rule.Name, rule.ID, err)
			continue // Skip rules that error out
		}

		if !passed {
			allPassed = false
			errors = append(errors, ValidationError{
				RuleID:   rule.ID,
				Message:  rule.Description,
				Severity: rule.Severity,
			})
		}
	}

	return allPassed, errors, nil
}

func (ve *ValidationEngineWithHierarchy) evaluateHierarchyRule(
	rule HierarchyRule,
	data map[string]interface{},
) (bool, error) {
	var condition map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Condition), &condition); err != nil {
		return false, err
	}

	return ve.evaluator.EvaluateWithHierarchy(condition, data)
}

func (ve *ValidationEngineWithHierarchy) getHierarchyRules(
	ctx context.Context,
	entity string,
	tenantID string,
	datasourceID string,
) ([]HierarchyRule, error) {
	query := `
        SELECT id, name, entity, description, severity, condition, field_path, hierarchy_depth
        FROM validation_rules
        WHERE entity = $1 AND tenant_id = $2 AND datasource_id = $3
          AND field_path IS NOT NULL AND array_length(field_path, 1) > 0
          AND is_active = true
        ORDER BY hierarchy_depth ASC
    `

	rows, err := ve.db.QueryContext(ctx, query, entity, tenantID, datasourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []HierarchyRule
	for rows.Next() {
		var r HierarchyRule
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Entity, &r.Description, &r.Severity,
			&r.Condition, (*pq.StringArray)(&r.FieldPath), &r.HierarchyDepth,
		); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return rules, rows.Err()
}

// Structs to hold rule data and validation errors
type HierarchyRule struct {
	ID             string
	Name           string
	Entity         string
	Description    string
	Severity       string
	Condition      string
	FieldPath      []string
	HierarchyDepth int
}

type ValidationError struct {
	RuleID   string
	Message  string
	Severity string
}
