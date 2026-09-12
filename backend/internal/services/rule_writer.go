package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RuleCategory string
type RuleContextType string
type RuleSeverity string

type CreateRuleRequest struct {
	RuleCode       string          `json:"rule_code"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Category       RuleCategory    `json:"category"`
	PrimaryContext RuleContextType `json:"primary_context"`
	Severity       RuleSeverity    `json:"severity"`
	ScopeEntity    string          `json:"scope_entity,omitempty"`
	ScopeFields    []string        `json:"scope_fields,omitempty"`
	Environment    string          `json:"environment"`
	EffectiveFrom  *time.Time      `json:"effective_from,omitempty"`
	EffectiveTo    *time.Time      `json:"effective_to,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	RegulationIDs  []string        `json:"regulation_ids,omitempty"`
	ControlIDs     []string        `json:"control_ids,omitempty"`
	ConditionJSON  json.RawMessage `json:"condition_json"`
	ActionsJSON    json.RawMessage `json:"actions_json"`
	ScoringFormula string          `json:"scoring_formula,omitempty"`
}

func CreateRule(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, userID string, req CreateRuleRequest) (ruleID, logicID uuid.UUID, err error) {
	ruleID = uuid.New()
	query := `
		INSERT INTO rules (
			id, tenant_id, rule_code, name, description, category, primary_context,
			severity, scope_entity, scope_fields, status, environment,
			effective_from, effective_to, tags, regulation_ids, control_ids, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'draft', $11, $12, $13, $14, $15, $16, $17)
	`

	_, err = db.ExecContext(ctx, query,
		ruleID, tenantID, req.RuleCode, req.Name, req.Description, req.Category, req.PrimaryContext,
		req.Severity, req.ScopeEntity, req.ScopeFields, req.Environment,
		req.EffectiveFrom, req.EffectiveTo, req.Tags, req.RegulationIDs, req.ControlIDs, userID,
	)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("insert rule: %w", err)
	}

	logicID = uuid.New()
	logicQuery := `
		INSERT INTO rule_logic (id, rule_id, version, condition_json, actions_json, scoring_formula, changed_by)
		VALUES ($1, $2, 1, $3, $4, $5, $6)
	`

	condJSON := req.ConditionJSON
	if condJSON == nil {
		condJSON = json.RawMessage(`{"type": "group", "operator": "AND", "conditions": []}`)
	}
	actJSON := req.ActionsJSON
	if actJSON == nil {
		actJSON = json.RawMessage(`[]`)
	}

	_, err = db.ExecContext(ctx, logicQuery, logicID, ruleID, condJSON, actJSON, req.ScoringFormula, userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("insert rule_logic: %w", err)
	}

	return ruleID, logicID, nil
}
