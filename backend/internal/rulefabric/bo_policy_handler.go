package rulefabric

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// =============================================================================
// BO GOVERNANCE POLICIES
// =============================================================================
//
// Bridges the WHEN/THEN policy authoring UI (frontend/src/features/bo/
// PolicyRuleBuilder.tsx) onto RuleFabric's rules/rule_logic tables and
// evaluator, so business-object governance policies are just another kind
// of Rule (category "workflow", condition stored as a raw CEL expression
// via the `{"type":"cel","expression":"..."}` shape RuleEvaluator.Evaluate
// understands) rather than a separate engine with its own storage and
// evaluation logic.
//
// A PolicyRule is scoped by (tenant, bo_key, trigger_event); bo_key maps to
// Rule.ScopeEntity and trigger_event is stored in Rule.ScopeEventTypes
// (documented on that column as "Event types for event-scoped rules" - an
// exact fit, no schema change needed). action_type/action_config map
// directly onto the existing RuleAction{Type, Params} shape already used
// for tree-based rules. is_core/priority, which RuleFabric's Rule type has
// no dedicated column for, are kept in Rule.Metadata.

// PolicyRule is the wire shape PolicyRuleBuilder.tsx sends and expects back.
type PolicyRule struct {
	PolicyID      string                 `json:"policy_id"`
	BOKey         string                 `json:"bo_key"`
	PolicyName    string                 `json:"policy_name"`
	Description   string                 `json:"description"`
	TriggerEvent  string                 `json:"trigger_event"`
	ConditionExpr string                 `json:"condition_expr"`
	ActionType    string                 `json:"action_type"`
	ActionConfig  map[string]interface{} `json:"action_config"`
	Priority      int                    `json:"priority"`
	IsActive      bool                   `json:"is_active"`
	IsCore        bool                   `json:"is_core"`
}

type policyMetadata struct {
	IsCore   bool `json:"is_core"`
	Priority int  `json:"priority"`
}

type policyRow struct {
	ID              string         `db:"id"`
	Name            string         `db:"name"`
	Description     string         `db:"description"`
	Status          string         `db:"status"`
	Metadata        []byte         `db:"metadata"`
	ScopeEventTypes pq.StringArray `db:"scope_event_types"`
	ConditionJSON   []byte         `db:"condition_json"`
	ActionsJSON     []byte         `db:"actions_json"`
}

// ListPoliciesForBO handles GET /api/rule-fabric/bo/{boKey}/policies?event=...
func (h *Handler) ListPoliciesForBO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	boKey := chi.URLParam(r, "boKey")
	event := r.URL.Query().Get("event")

	query := `
		SELECT r.id, r.name, r.description, r.status, r.metadata, r.scope_event_types,
		       rl.condition_json, rl.actions_json
		FROM rules r
		JOIN LATERAL (
			SELECT condition_json, actions_json
			FROM rule_logic
			WHERE rule_id = r.id
			ORDER BY version DESC
			LIMIT 1
		) rl ON true
		WHERE r.tenant_id = $1 AND r.scope_entity = $2 AND r.category = 'workflow'
		  AND ($3 = '' OR $3 = ANY(r.scope_event_types))
		ORDER BY r.created_at DESC
	`

	var rows []policyRow
	if err := h.db.SelectContext(ctx, &rows, query, tenantID, boKey, event); err != nil {
		http.Error(w, fmt.Sprintf("failed to list policies: %v", err), http.StatusInternalServerError)
		return
	}

	policies := make([]PolicyRule, 0, len(rows))
	for _, row := range rows {
		policies = append(policies, policyRuleFromRow(row, boKey))
	}

	respondJSON(w, http.StatusOK, policies)
}

// CreatePolicyForBO handles POST /api/rule-fabric/bo/{boKey}/policies
func (h *Handler) CreatePolicyForBO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	boKey := chi.URLParam(r, "boKey")
	userID := getUserID(r)

	var req PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.PolicyName == "" || req.TriggerEvent == "" {
		http.Error(w, "policy_name and trigger_event are required", http.StatusBadRequest)
		return
	}

	ruleID := uuid.New()
	status := "active"
	if !req.IsActive {
		status = "draft"
	}
	metadata, _ := json.Marshal(policyMetadata{IsCore: req.IsCore, Priority: req.Priority})
	ruleCode := fmt.Sprintf("POLICY_%s_%s", boKey, ruleID.String()[:8])

	tx, err := h.db.BeginTxx(ctx, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to begin transaction: %v", err), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO rules (
			id, tenant_id, rule_code, name, description, category, primary_context,
			severity, scope_entity, scope_event_types, status, environment, metadata, created_by
		) VALUES ($1, $2, $3, $4, $5, 'workflow', 'data_record', 'warning', $6, $7, $8, 'production', $9, $10)
	`, ruleID, tenantID, ruleCode, req.PolicyName, req.Description, boKey, pq.Array([]string{req.TriggerEvent}), status, metadata, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create policy: %v", err), http.StatusInternalServerError)
		return
	}

	if err := insertPolicyLogic(ctx, tx, ruleID, 1, req, userID, "initial version"); err != nil {
		http.Error(w, fmt.Sprintf("failed to create policy logic: %v", err), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("failed to commit: %v", err), http.StatusInternalServerError)
		return
	}

	req.PolicyID = ruleID.String()
	req.BOKey = boKey
	respondJSON(w, http.StatusCreated, req)
}

// UpdatePolicyForBO handles PUT /api/rule-fabric/bo/{boKey}/policies/{policyID}
func (h *Handler) UpdatePolicyForBO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	boKey := chi.URLParam(r, "boKey")
	userID := getUserID(r)
	policyID, err := uuid.Parse(chi.URLParam(r, "policyID"))
	if err != nil {
		http.Error(w, "invalid policy_id", http.StatusBadRequest)
		return
	}

	var req PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var exists bool
	if err := h.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM rules WHERE tenant_id = $1 AND id = $2 AND scope_entity = $3)", tenantID, policyID, boKey); err != nil || !exists {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	status := "active"
	if !req.IsActive {
		status = "draft"
	}
	metadata, _ := json.Marshal(policyMetadata{IsCore: req.IsCore, Priority: req.Priority})

	tx, err := h.db.BeginTxx(ctx, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to begin transaction: %v", err), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE rules SET name = $1, description = $2, scope_event_types = $3, status = $4, metadata = $5, updated_at = NOW()
		WHERE id = $6
	`, req.PolicyName, req.Description, pq.Array([]string{req.TriggerEvent}), status, metadata, policyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update policy: %v", err), http.StatusInternalServerError)
		return
	}

	var maxVersion int
	_ = tx.GetContext(ctx, &maxVersion, "SELECT COALESCE(MAX(version), 0) FROM rule_logic WHERE rule_id = $1", policyID)

	if err := insertPolicyLogic(ctx, tx, policyID, maxVersion+1, req, userID, "update"); err != nil {
		http.Error(w, fmt.Sprintf("failed to update policy logic: %v", err), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("failed to commit: %v", err), http.StatusInternalServerError)
		return
	}

	req.PolicyID = policyID.String()
	req.BOKey = boKey
	respondJSON(w, http.StatusOK, req)
}

// DeletePolicyForBO handles DELETE /api/rule-fabric/bo/{boKey}/policies/{policyID}
func (h *Handler) DeletePolicyForBO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	boKey := chi.URLParam(r, "boKey")
	policyID, err := uuid.Parse(chi.URLParam(r, "policyID"))
	if err != nil {
		http.Error(w, "invalid policy_id", http.StatusBadRequest)
		return
	}

	res, err := h.db.ExecContext(ctx, "DELETE FROM rules WHERE tenant_id = $1 AND id = $2 AND scope_entity = $3", tenantID, policyID, boKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete policy: %v", err), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SimulatePolicyExpr handles POST /api/rule-fabric/bo/{boKey}/policies/simulate
// Evaluates a CEL condition_expr ad hoc against a sample record/actor, with
// no rule persisted - used by PolicyRuleBuilder's "Simulate" button while
// authoring a policy.
func (h *Handler) SimulatePolicyExpr(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConditionExpr string      `json:"condition_expr"`
		Record        interface{} `json:"record"`
		Actor         interface{} `json:"actor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ConditionExpr == "" {
		http.Error(w, "condition_expr is required", http.StatusBadRequest)
		return
	}

	triggered, err := h.evaluator.EvaluateCELBoolean(req.ConditionExpr, req.Record, req.Actor, nil)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"triggered": false, "error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"triggered": triggered})
}

func insertPolicyLogic(ctx context.Context, tx *sqlx.Tx, ruleID uuid.UUID, version int, req PolicyRule, userID *uuid.UUID, changeReason string) error {
	conditionJSON, err := json.Marshal(map[string]interface{}{
		"type":       "cel",
		"expression": req.ConditionExpr,
	})
	if err != nil {
		return err
	}
	actionsJSON, err := json.Marshal([]RuleAction{{
		Type:   req.ActionType,
		Params: req.ActionConfig,
		Order:  1,
	}})
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO rule_logic (id, rule_id, version, condition_json, actions_json, change_reason, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New(), ruleID, version, conditionJSON, actionsJSON, changeReason, userID)
	return err
}

func policyRuleFromRow(row policyRow, boKey string) PolicyRule {
	var meta policyMetadata
	_ = json.Unmarshal(row.Metadata, &meta)

	triggerEvent := ""
	if len(row.ScopeEventTypes) > 0 {
		triggerEvent = row.ScopeEventTypes[0]
	}

	var cel struct {
		Expression string `json:"expression"`
	}
	_ = json.Unmarshal(row.ConditionJSON, &cel)

	var actions []RuleAction
	_ = json.Unmarshal(row.ActionsJSON, &actions)
	actionType, actionConfig := "", map[string]interface{}{}
	if len(actions) > 0 {
		actionType = actions[0].Type
		if actions[0].Params != nil {
			actionConfig = actions[0].Params
		}
	}

	return PolicyRule{
		PolicyID:      row.ID,
		BOKey:         boKey,
		PolicyName:    row.Name,
		Description:   row.Description,
		TriggerEvent:  triggerEvent,
		ConditionExpr: cel.Expression,
		ActionType:    actionType,
		ActionConfig:  actionConfig,
		Priority:      meta.Priority,
		IsActive:      row.Status == "active",
		IsCore:        meta.IsCore,
	}
}
