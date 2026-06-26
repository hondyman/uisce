package rulefabric

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler provides HTTP handlers for the Rule Fabric API
type Handler struct {
	db        *sqlx.DB
	evaluator *RuleEvaluator
}

// NewHandler creates a new Rule Fabric handler
func NewHandler(db *sqlx.DB) (*Handler, error) {
	evaluator, err := NewRuleEvaluator(db)
	if err != nil {
		return nil, err
	}
	return &Handler{
		db:        db,
		evaluator: evaluator,
	}, nil
}

// RegisterRoutes registers all Rule Fabric routes
func RegisterRoutes(router chi.Router, db *sqlx.DB) error {
	handler, err := NewHandler(db)
	if err != nil {
		return err
	}

	router.Route("/api/rule-fabric", func(r chi.Router) {
		// Rules CRUD
		r.Route("/rules", func(r chi.Router) {
			r.Get("/", handler.ListRules)
			r.Post("/", handler.CreateRule)
			r.Get("/{ruleID}", handler.GetRule)
			r.Put("/{ruleID}", handler.UpdateRule)
			r.Delete("/{ruleID}", handler.DeleteRule)

			// Rule versions
			r.Get("/{ruleID}/versions", handler.GetRuleVersions)
			r.Post("/{ruleID}/versions", handler.CreateRuleVersion)
			r.Put("/{ruleID}/versions/{version}/approve", handler.ApproveRuleVersion)
		})

		// Evaluation
		r.Route("/evaluate", func(r chi.Router) {
			r.Post("/", handler.EvaluateRules)
			r.Post("/single", handler.EvaluateSingleRule)
			r.Post("/simulate", handler.SimulateRules)
		})

		// Execution policies
		r.Route("/policies", func(r chi.Router) {
			r.Get("/", handler.ListPolicies)
			r.Post("/", handler.CreatePolicy)
			r.Get("/{policyID}", handler.GetPolicy)
			r.Put("/{policyID}", handler.UpdatePolicy)
			r.Delete("/{policyID}", handler.DeletePolicy)
		})

		// Action types (reference data)
		r.Get("/action-types", handler.ListActionTypes)
		r.Get("/action-types/{category}", handler.GetActionTypesByCategory)

		// Violations
		r.Route("/violations", func(r chi.Router) {
			r.Get("/", handler.ListViolations)
			r.Get("/{violationID}", handler.GetViolation)
			r.Put("/{violationID}/resolve", handler.ResolveViolation)
		})

		// Statistics and dashboards
		r.Get("/stats", handler.GetRuleStats)
		r.Get("/categories", handler.ListCategories)
	})

	return nil
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// CreateRuleRequest represents a request to create a rule
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

	// Initial logic
	ConditionJSON  json.RawMessage `json:"condition_json"`
	ActionsJSON    json.RawMessage `json:"actions_json"`
	ScoringFormula string          `json:"scoring_formula,omitempty"`
}

// CreateRuleVersionRequest represents a request to create a new rule version
type CreateRuleVersionRequest struct {
	ConditionJSON  json.RawMessage `json:"condition_json"`
	ActionsJSON    json.RawMessage `json:"actions_json"`
	ScoringFormula string          `json:"scoring_formula,omitempty"`
	ChangeReason   string          `json:"change_reason"`
	VersionLabel   string          `json:"version_label,omitempty"`
}

// EvaluateRequest represents a request to evaluate rules
type EvaluateRequest struct {
	Category    *RuleCategory    `json:"category,omitempty"`
	ContextType *RuleContextType `json:"context_type,omitempty"`
	Entity      string           `json:"entity,omitempty"`
	Channel     string           `json:"channel"`
	Environment string           `json:"environment"`

	// Data to evaluate
	Data        map[string]interface{} `json:"data"`
	RelatedData map[string]interface{} `json:"related_data,omitempty"`
	Extras      map[string]interface{} `json:"extras,omitempty"`
}

// EvaluateSingleRuleRequest represents a request to evaluate a single rule
type EvaluateSingleRuleRequest struct {
	RuleID uuid.UUID `json:"rule_id"`
	EvaluateRequest
}

// SimulateRequest represents a request to simulate rules
type SimulateRequest struct {
	RuleIDs     []uuid.UUID              `json:"rule_ids"`
	DataSamples []map[string]interface{} `json:"data_samples"`
	Channel     string                   `json:"channel"`
	Environment string                   `json:"environment"`
}

// CreatePolicyRequest represents a request to create an execution policy
type CreatePolicyRequest struct {
	PolicyCode               string          `json:"policy_code"`
	Name                     string          `json:"name"`
	Description              string          `json:"description,omitempty"`
	Channel                  string          `json:"channel"`
	Category                 *RuleCategory   `json:"category,omitempty"`
	Enforcement              EnforcementMode `json:"enforcement"`
	MaxSeverity              *RuleSeverity   `json:"max_severity,omitempty"`
	AllowOverride            bool            `json:"allow_override"`
	OverrideRequiresApproval bool            `json:"override_requires_approval"`
	TimeoutMs                int             `json:"timeout_ms"`
	EmitEvents               bool            `json:"emit_events"`
	EventTopic               string          `json:"event_topic,omitempty"`
	Environment              string          `json:"environment"`
}

// ResolveViolationRequest represents a request to resolve a violation
type ResolveViolationRequest struct {
	Resolution string `json:"resolution"` // resolved, dismissed
	Notes      string `json:"notes"`
}

// =============================================================================
// HANDLERS - RULES
// =============================================================================

// ListRules returns all rules for a tenant
func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Query parameters
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")
	environment := r.URL.Query().Get("environment")
	if environment == "" {
		environment = "dev"
	}

	query := `
		SELECT id, tenant_id, datasource_id, rule_code, name, description,
		       category, primary_context, severity, scope_entity, scope_fields,
		       status, environment, effective_from, effective_to, owner_user_id,
		       tags, regulation_ids, control_ids, created_at, updated_at
		FROM rules
		WHERE tenant_id = $1
		  AND environment = $2
		  AND ($3::text IS NULL OR category = $3)
		  AND ($4::text IS NULL OR status = $4)
		ORDER BY category, rule_code
	`

	var rules []Rule
	if err := h.db.SelectContext(ctx, &rules, query, tenantID, environment, nullIfEmpty(category), nullIfEmpty(status)); err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch rules: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// GetRule returns a single rule
func (h *Handler) GetRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule_id", http.StatusBadRequest)
		return
	}

	// Get rule with latest approved logic
	query := `
		SELECT r.id, r.tenant_id, r.datasource_id, r.rule_code, r.name, r.description,
		       r.category, r.primary_context, r.severity, r.scope_entity, r.scope_fields,
		       r.status, r.environment, r.effective_from, r.effective_to, r.owner_user_id,
		       r.tags, r.regulation_ids, r.control_ids, r.created_at, r.updated_at,
		       rl.id as logic_id, rl.version, rl.version_label, rl.condition_json,
		       rl.actions_json, rl.scoring_formula, rl.is_approved, rl.change_reason
		FROM rules r
		LEFT JOIN rule_logic rl ON r.id = rl.rule_id
		WHERE r.tenant_id = $1 AND r.id = $2
		ORDER BY rl.version DESC
		LIMIT 1
	`

	var rule RuleWithLogic
	if err := h.db.GetContext(ctx, &rule, query, tenantID, ruleID); err != nil {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, rule)
}

// CreateRule creates a new rule
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getUserID(r)

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.RuleCode == "" || req.Name == "" {
		http.Error(w, "rule_code and name are required", http.StatusBadRequest)
		return
	}

	// Create rule
	ruleID := uuid.New()
	query := `
		INSERT INTO rules (
			id, tenant_id, rule_code, name, description, category, primary_context,
			severity, scope_entity, scope_fields, status, environment,
			effective_from, effective_to, tags, regulation_ids, control_ids, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'draft', $11, $12, $13, $14, $15, $16, $17)
	`

	_, err = h.db.ExecContext(ctx, query,
		ruleID, tenantID, req.RuleCode, req.Name, req.Description, req.Category, req.PrimaryContext,
		req.Severity, req.ScopeEntity, req.ScopeFields, req.Environment,
		req.EffectiveFrom, req.EffectiveTo, req.Tags, req.RegulationIDs, req.ControlIDs, userID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create rule: %v", err), http.StatusInternalServerError)
		return
	}

	// Create initial logic version
	logicID := uuid.New()
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

	_, err = h.db.ExecContext(ctx, logicQuery, logicID, ruleID, condJSON, actJSON, req.ScoringFormula, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create rule logic: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"rule_id":  ruleID,
		"logic_id": logicID,
		"message":  "Rule created successfully",
	})
}

// UpdateRule updates a rule
func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule_id", http.StatusBadRequest)
		return
	}

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE rules
		SET name = $3, description = $4, severity = $5, scope_entity = $6, scope_fields = $7,
		    effective_from = $8, effective_to = $9, tags = $10, regulation_ids = $11, control_ids = $12,
		    updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := h.db.ExecContext(ctx, query,
		tenantID, ruleID, req.Name, req.Description, req.Severity, req.ScopeEntity, req.ScopeFields,
		req.EffectiveFrom, req.EffectiveTo, req.Tags, req.RegulationIDs, req.ControlIDs,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update rule: %v", err), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Rule updated successfully"})
}

// DeleteRule deletes (retires) a rule
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule_id", http.StatusBadRequest)
		return
	}

	// Soft delete - set status to retired
	query := `UPDATE rules SET status = 'retired', updated_at = NOW() WHERE tenant_id = $1 AND id = $2`
	result, err := h.db.ExecContext(ctx, query, tenantID, ruleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete rule: %v", err), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// HANDLERS - RULE VERSIONS
// =============================================================================

// GetRuleVersions returns all versions of a rule
func (h *Handler) GetRuleVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule_id", http.StatusBadRequest)
		return
	}

	query := `
		SELECT rl.id, rl.rule_id, rl.version, rl.version_label, rl.condition_json,
		       rl.actions_json, rl.scoring_formula, rl.is_approved, rl.approved_by,
		       rl.approved_at, rl.change_reason, rl.created_at
		FROM rule_logic rl
		JOIN rules r ON r.id = rl.rule_id
		WHERE r.tenant_id = $1 AND rl.rule_id = $2
		ORDER BY rl.version DESC
	`

	var versions []RuleLogic
	if err := h.db.SelectContext(ctx, &versions, query, tenantID, ruleID); err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch versions: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"versions": versions,
		"count":    len(versions),
	})
}

// CreateRuleVersion creates a new version of rule logic
func (h *Handler) CreateRuleVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getUserID(r)

	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule_id", http.StatusBadRequest)
		return
	}

	var req CreateRuleVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify rule exists and belongs to tenant
	var exists bool
	err = h.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM rules WHERE tenant_id = $1 AND id = $2)", tenantID, ruleID)
	if err != nil || !exists {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	// Get next version number
	var maxVersion int
	h.db.GetContext(ctx, &maxVersion, "SELECT COALESCE(MAX(version), 0) FROM rule_logic WHERE rule_id = $1", ruleID)

	logicID := uuid.New()
	query := `
		INSERT INTO rule_logic (id, rule_id, version, version_label, condition_json, actions_json, scoring_formula, change_reason, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = h.db.ExecContext(ctx, query,
		logicID, ruleID, maxVersion+1, req.VersionLabel, req.ConditionJSON, req.ActionsJSON, req.ScoringFormula, req.ChangeReason, userID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create version: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"logic_id": logicID,
		"version":  maxVersion + 1,
		"message":  "Version created successfully",
	})
}

// ApproveRuleVersion approves a rule version
func (h *Handler) ApproveRuleVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getUserID(r)

	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule_id", http.StatusBadRequest)
		return
	}

	versionStr := chi.URLParam(r, "version")
	var version int
	fmt.Sscanf(versionStr, "%d", &version)

	// Approve version
	query := `
		UPDATE rule_logic
		SET is_approved = TRUE, approved_by = $4, approved_at = NOW()
		FROM rules r
		WHERE rule_logic.rule_id = r.id AND r.tenant_id = $1 AND rule_logic.rule_id = $2 AND rule_logic.version = $3
	`

	result, err := h.db.ExecContext(ctx, query, tenantID, ruleID, version, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to approve version: %v", err), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "version not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Version approved successfully"})
}

// =============================================================================
// HANDLERS - EVALUATION
// =============================================================================

// EvaluateRules evaluates all matching rules
func (h *Handler) EvaluateRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	datasourceID := getDatasourceID(r)

	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Environment == "" {
		req.Environment = "dev"
	}

	// Get rules for evaluation
	opts := GetRulesOptions{
		Environment: req.Environment,
		Category:    req.Category,
		ContextType: req.ContextType,
		Channel:     req.Channel,
		Entity:      req.Entity,
	}

	rules, err := h.evaluator.GetRulesForEvaluation(ctx, tenantID, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch rules: %v", err), http.StatusInternalServerError)
		return
	}

	// Build evaluation context
	evalCtx := &EvaluationContext{
		TenantID:       tenantID,
		DatasourceID:   datasourceID,
		Channel:        req.Channel,
		Environment:    req.Environment,
		Data:           req.Data,
		RelatedData:    req.RelatedData,
		Extras:         req.Extras,
		EvaluationTime: time.Now(),
	}

	// Evaluate
	result, err := h.evaluator.EvaluateBatch(ctx, rules, evalCtx)
	if err != nil {
		http.Error(w, fmt.Sprintf("evaluation failed: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// EvaluateSingleRule evaluates a specific rule
func (h *Handler) EvaluateSingleRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	datasourceID := getDatasourceID(r)

	var req EvaluateSingleRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Environment == "" {
		req.Environment = "dev"
	}

	// Get specific rule
	query := `
		SELECT r.id, r.tenant_id, r.datasource_id, r.rule_code, r.name, r.description,
		       r.category, r.primary_context, r.severity, r.scope_entity, r.scope_fields,
		       r.status, r.environment, r.effective_from, r.effective_to, r.owner_user_id,
		       r.tags, r.regulation_ids, r.control_ids, r.created_at, r.updated_at,
		       rl.id as logic_id, rl.version, rl.version_label, rl.condition_json,
		       rl.actions_json, rl.scoring_formula, rl.is_approved,
		       'hard_block' as enforcement, 5000 as timeout_ms
		FROM rules r
		JOIN rule_logic rl ON r.id = rl.rule_id
		WHERE r.tenant_id = $1 AND r.id = $2
		ORDER BY rl.version DESC
		LIMIT 1
	`

	var rule RuleWithLogic
	if err := h.db.GetContext(ctx, &rule, query, tenantID, req.RuleID); err != nil {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	// Build evaluation context
	evalCtx := &EvaluationContext{
		TenantID:       tenantID,
		DatasourceID:   datasourceID,
		Channel:        req.Channel,
		Environment:    req.Environment,
		Data:           req.Data,
		RelatedData:    req.RelatedData,
		Extras:         req.Extras,
		EvaluationTime: time.Now(),
	}

	// Evaluate
	result, err := h.evaluator.Evaluate(ctx, &rule, evalCtx)
	if err != nil {
		http.Error(w, fmt.Sprintf("evaluation failed: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// SimulateRules runs rules against sample data without persisting
func (h *Handler) SimulateRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req SimulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Environment == "" {
		req.Environment = "dev"
	}

	// Get specified rules
	rules := make([]*RuleWithLogic, 0)
	for _, ruleID := range req.RuleIDs {
		query := `
			SELECT r.id, r.tenant_id, r.datasource_id, r.rule_code, r.name, r.description,
			       r.category, r.primary_context, r.severity, r.scope_entity, r.scope_fields,
			       r.status, r.environment, r.effective_from, r.effective_to, r.owner_user_id,
			       r.tags, r.regulation_ids, r.control_ids, r.created_at, r.updated_at,
			       rl.id as logic_id, rl.version, rl.version_label, rl.condition_json,
			       rl.actions_json, rl.scoring_formula, rl.is_approved,
			       'simulate' as enforcement, 5000 as timeout_ms
			FROM rules r
			JOIN rule_logic rl ON r.id = rl.rule_id
			WHERE r.tenant_id = $1 AND r.id = $2
			ORDER BY rl.version DESC
			LIMIT 1
		`

		var rule RuleWithLogic
		if err := h.db.GetContext(ctx, &rule, query, tenantID, ruleID); err == nil {
			rules = append(rules, &rule)
		}
	}

	// Simulate against each sample
	type SampleResult struct {
		SampleIndex int                    `json:"sample_index"`
		Data        map[string]interface{} `json:"data"`
		Result      *BatchEvaluationResult `json:"result"`
	}

	var sampleResults []SampleResult
	var totalPassed, totalFailed int

	for i, sample := range req.DataSamples {
		evalCtx := &EvaluationContext{
			TenantID:       tenantID,
			Channel:        req.Channel,
			Environment:    req.Environment,
			Data:           sample,
			EvaluationTime: time.Now(),
		}

		result, _ := h.evaluator.EvaluateBatch(ctx, rules, evalCtx)
		sampleResults = append(sampleResults, SampleResult{
			SampleIndex: i,
			Data:        sample,
			Result:      result,
		})

		totalPassed += result.PassedCount
		totalFailed += result.FailedCount
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"rules_count":   len(rules),
		"samples_count": len(req.DataSamples),
		"total_passed":  totalPassed,
		"total_failed":  totalFailed,
		"results":       sampleResults,
	})
}

// =============================================================================
// HANDLERS - POLICIES
// =============================================================================

// ListPolicies returns all execution policies
func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, tenant_id, datasource_id, policy_code, name, description, channel,
		       category, enforcement, max_severity, allow_override, override_requires_approval,
		       timeout_ms, emit_events, event_topic, is_active, environment, created_at, updated_at
		FROM rule_execution_policies
		WHERE tenant_id = $1
		ORDER BY channel, category
	`

	var policies []RuleExecutionPolicy
	if err := h.db.SelectContext(ctx, &policies, query, tenantID); err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch policies: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"policies": policies,
		"count":    len(policies),
	})
}

// CreatePolicy creates a new execution policy
func (h *Handler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.PolicyCode == "" || req.Name == "" || req.Channel == "" {
		http.Error(w, "policy_code, name, and channel are required", http.StatusBadRequest)
		return
	}

	policyID := uuid.New()
	query := `
		INSERT INTO rule_execution_policies (
			id, tenant_id, policy_code, name, description, channel, category, enforcement,
			max_severity, allow_override, override_requires_approval, timeout_ms,
			emit_events, event_topic, environment
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = h.db.ExecContext(ctx, query,
		policyID, tenantID, req.PolicyCode, req.Name, req.Description, req.Channel, req.Category, req.Enforcement,
		req.MaxSeverity, req.AllowOverride, req.OverrideRequiresApproval, req.TimeoutMs,
		req.EmitEvents, req.EventTopic, req.Environment,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create policy: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"policy_id": policyID,
		"message":   "Policy created successfully",
	})
}

// GetPolicy returns a single policy
func (h *Handler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	policyIDStr := chi.URLParam(r, "policyID")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		http.Error(w, "invalid policy_id", http.StatusBadRequest)
		return
	}

	var policy RuleExecutionPolicy
	query := `SELECT * FROM rule_execution_policies WHERE tenant_id = $1 AND id = $2`
	if err := h.db.GetContext(ctx, &policy, query, tenantID, policyID); err != nil {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, policy)
}

// UpdatePolicy updates a policy
func (h *Handler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	policyIDStr := chi.URLParam(r, "policyID")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		http.Error(w, "invalid policy_id", http.StatusBadRequest)
		return
	}

	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE rule_execution_policies
		SET name = $3, description = $4, enforcement = $5, max_severity = $6,
		    allow_override = $7, override_requires_approval = $8, timeout_ms = $9,
		    emit_events = $10, event_topic = $11, is_active = TRUE, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := h.db.ExecContext(ctx, query,
		tenantID, policyID, req.Name, req.Description, req.Enforcement, req.MaxSeverity,
		req.AllowOverride, req.OverrideRequiresApproval, req.TimeoutMs, req.EmitEvents, req.EventTopic,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update policy: %v", err), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Policy updated successfully"})
}

// DeletePolicy deletes a policy
func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	policyIDStr := chi.URLParam(r, "policyID")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		http.Error(w, "invalid policy_id", http.StatusBadRequest)
		return
	}

	query := `UPDATE rule_execution_policies SET is_active = FALSE, updated_at = NOW() WHERE tenant_id = $1 AND id = $2`
	result, err := h.db.ExecContext(ctx, query, tenantID, policyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete policy: %v", err), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// HANDLERS - ACTION TYPES & REFERENCE DATA
// =============================================================================

// ListActionTypes returns all action types
func (h *Handler) ListActionTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := `
		SELECT action_type, display_name, description, categories, contexts,
		       params_schema, handler_service, is_blocking, is_async, icon, color
		FROM rule_action_types
		WHERE is_active = TRUE
		ORDER BY action_type
	`

	type ActionType struct {
		ActionType     string            `db:"action_type" json:"action_type"`
		DisplayName    string            `db:"display_name" json:"display_name"`
		Description    string            `db:"description" json:"description"`
		Categories     []RuleCategory    `db:"categories" json:"categories"`
		Contexts       []RuleContextType `db:"contexts" json:"contexts"`
		ParamsSchema   json.RawMessage   `db:"params_schema" json:"params_schema"`
		HandlerService string            `db:"handler_service" json:"handler_service,omitempty"`
		IsBlocking     bool              `db:"is_blocking" json:"is_blocking"`
		IsAsync        bool              `db:"is_async" json:"is_async"`
		Icon           string            `db:"icon" json:"icon,omitempty"`
		Color          string            `db:"color" json:"color,omitempty"`
	}

	var actions []ActionType
	if err := h.db.SelectContext(ctx, &actions, query); err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch action types: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"action_types": actions,
		"count":        len(actions),
	})
}

// GetActionTypesByCategory returns action types for a specific category
func (h *Handler) GetActionTypesByCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	category := chi.URLParam(r, "category")

	query := `
		SELECT action_type, display_name, description, categories, contexts,
		       params_schema, handler_service, is_blocking, is_async, icon, color
		FROM rule_action_types
		WHERE is_active = TRUE AND $1 = ANY(categories)
		ORDER BY action_type
	`

	type ActionType struct {
		ActionType     string            `db:"action_type" json:"action_type"`
		DisplayName    string            `db:"display_name" json:"display_name"`
		Description    string            `db:"description" json:"description"`
		Categories     []RuleCategory    `db:"categories" json:"categories"`
		Contexts       []RuleContextType `db:"contexts" json:"contexts"`
		ParamsSchema   json.RawMessage   `db:"params_schema" json:"params_schema"`
		HandlerService string            `db:"handler_service" json:"handler_service,omitempty"`
		IsBlocking     bool              `db:"is_blocking" json:"is_blocking"`
		IsAsync        bool              `db:"is_async" json:"is_async"`
		Icon           string            `db:"icon" json:"icon,omitempty"`
		Color          string            `db:"color" json:"color,omitempty"`
	}

	var actions []ActionType
	if err := h.db.SelectContext(ctx, &actions, query, category); err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch action types: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"category":     category,
		"action_types": actions,
		"count":        len(actions),
	})
}

// ListCategories returns all rule categories with counts
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		SELECT category, COUNT(*) as count,
		       SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_count
		FROM rules
		WHERE tenant_id = $1
		GROUP BY category
		ORDER BY category
	`

	type CategoryCount struct {
		Category    RuleCategory `db:"category" json:"category"`
		Count       int          `db:"count" json:"count"`
		ActiveCount int          `db:"active_count" json:"active_count"`
	}

	var categories []CategoryCount
	if err := h.db.SelectContext(ctx, &categories, query, tenantID); err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch categories: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"categories": categories,
	})
}

// =============================================================================
// HANDLERS - VIOLATIONS
// =============================================================================

// ListViolations returns violations with filtering
func (h *Handler) ListViolations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Query parameters
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")
	severity := r.URL.Query().Get("severity")

	query := `
		SELECT id, tenant_id, evaluation_result_id, rule_id, violation_code,
		       category, severity, channel, entity_type, entity_id,
		       title, description, status, resolution_notes, resolved_by,
		       resolved_at, regulation_ids, tags, created_at, updated_at
		FROM rule_violations
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR status = $2)
		  AND ($3::text IS NULL OR category = $3)
		  AND ($4::text IS NULL OR severity = $4)
		ORDER BY created_at DESC
		LIMIT 100
	`

	type Violation struct {
		ID                 uuid.UUID    `db:"id" json:"id"`
		TenantID           uuid.UUID    `db:"tenant_id" json:"tenant_id"`
		EvaluationResultID *uuid.UUID   `db:"evaluation_result_id" json:"evaluation_result_id,omitempty"`
		RuleID             uuid.UUID    `db:"rule_id" json:"rule_id"`
		ViolationCode      string       `db:"violation_code" json:"violation_code,omitempty"`
		Category           RuleCategory `db:"category" json:"category"`
		Severity           RuleSeverity `db:"severity" json:"severity"`
		Channel            string       `db:"channel" json:"channel,omitempty"`
		EntityType         string       `db:"entity_type" json:"entity_type,omitempty"`
		EntityID           string       `db:"entity_id" json:"entity_id,omitempty"`
		Title              string       `db:"title" json:"title,omitempty"`
		Description        string       `db:"description" json:"description,omitempty"`
		Status             string       `db:"status" json:"status"`
		ResolutionNotes    string       `db:"resolution_notes" json:"resolution_notes,omitempty"`
		ResolvedBy         *uuid.UUID   `db:"resolved_by" json:"resolved_by,omitempty"`
		ResolvedAt         *time.Time   `db:"resolved_at" json:"resolved_at,omitempty"`
		RegulationIDs      []string     `db:"regulation_ids" json:"regulation_ids,omitempty"`
		Tags               []string     `db:"tags" json:"tags,omitempty"`
		CreatedAt          time.Time    `db:"created_at" json:"created_at"`
		UpdatedAt          time.Time    `db:"updated_at" json:"updated_at"`
	}

	var violations []Violation
	if err := h.db.SelectContext(ctx, &violations, query, tenantID, nullIfEmpty(status), nullIfEmpty(category), nullIfEmpty(severity)); err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch violations: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"violations": violations,
		"count":      len(violations),
	})
}

// GetViolation returns a single violation
func (h *Handler) GetViolation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	violationIDStr := chi.URLParam(r, "violationID")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		http.Error(w, "invalid violation_id", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM rule_violations WHERE tenant_id = $1 AND id = $2`

	var violation map[string]interface{}
	if err := h.db.GetContext(ctx, &violation, query, tenantID, violationID); err != nil {
		http.Error(w, "violation not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, violation)
}

// ResolveViolation resolves a violation
func (h *Handler) ResolveViolation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := getUserID(r)

	violationIDStr := chi.URLParam(r, "violationID")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		http.Error(w, "invalid violation_id", http.StatusBadRequest)
		return
	}

	var req ResolveViolationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE rule_violations
		SET status = $3, resolution_notes = $4, resolved_by = $5, resolved_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := h.db.ExecContext(ctx, query, tenantID, violationID, req.Resolution, req.Notes, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve violation: %v", err), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "violation not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Violation resolved successfully"})
}

// GetRuleStats returns rule statistics
func (h *Handler) GetRuleStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stats := make(map[string]interface{})

	// Rule counts by status
	var statusCounts []struct {
		Status RuleStatus `db:"status"`
		Count  int        `db:"count"`
	}
	h.db.SelectContext(ctx, &statusCounts, "SELECT status, COUNT(*) as count FROM rules WHERE tenant_id = $1 GROUP BY status", tenantID)
	stats["by_status"] = statusCounts

	// Rule counts by category
	var categoryCounts []struct {
		Category RuleCategory `db:"category"`
		Count    int          `db:"count"`
	}
	h.db.SelectContext(ctx, &categoryCounts, "SELECT category, COUNT(*) as count FROM rules WHERE tenant_id = $1 GROUP BY category", tenantID)
	stats["by_category"] = categoryCounts

	// Violation counts
	var violationStats struct {
		Total    int `db:"total"`
		Open     int `db:"open"`
		Resolved int `db:"resolved"`
	}
	h.db.GetContext(ctx, &violationStats, `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END) as open,
			SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END) as resolved
		FROM rule_violations WHERE tenant_id = $1
	`, tenantID)
	stats["violations"] = violationStats

	respondJSON(w, http.StatusOK, stats)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	if tenantIDStr == "" {
		tenantIDStr = r.URL.Query().Get("tenant_id")
	}
	if tenantIDStr == "" {
		return uuid.Nil, fmt.Errorf("tenant_id is required")
	}
	return uuid.Parse(tenantIDStr)
}

func getDatasourceID(r *http.Request) uuid.UUID {
	dsIDStr := r.Header.Get("X-Tenant-Datasource-ID")
	if dsIDStr == "" {
		dsIDStr = r.URL.Query().Get("datasource_id")
	}
	id, _ := uuid.Parse(dsIDStr)
	return id
}

func getUserID(r *http.Request) *uuid.UUID {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return nil
	}
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil
	}
	return &id
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
