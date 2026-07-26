package bo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuditEvent is one immutable audit log row from bo.audit_event.
type AuditEvent struct {
	EventID         string          `db:"event_id"         json:"event_id"`
	TenantID        string          `db:"tenant_id"        json:"tenant_id"`
	BOKey           string          `db:"bo_key"           json:"bo_key"`
	EntityID        string          `db:"entity_id"        json:"entity_id"`
	Operation       string          `db:"operation"        json:"operation"`
	ActorID         string          `db:"actor_id"         json:"actor_id"`
	ActorRole       string          `db:"actor_role"       json:"actor_role"`
	BeforeValue     json.RawMessage `db:"before_value"     json:"before_value,omitempty"`
	AfterValue      json.RawMessage `db:"after_value"      json:"after_value,omitempty"`
	PolicyTriggered string          `db:"policy_triggered" json:"policy_triggered,omitempty"`
	IPAddress       string          `db:"ip_address"       json:"ip_address,omitempty"`
	CreatedAt       time.Time       `db:"created_at"       json:"created_at"`
}

// GovernanceHandlers wires all BO governance HTTP endpoints.
// All endpoints enforce X-Tenant-ID UUID validation (Rule 1.3 & Rule 7).
type GovernanceHandlers struct {
	db         *sql.DB
	validator  *ValidationEngine
	policies   *PolicyEngine
	access     *AccessController
	fieldSec   *FieldSecurityMasker
	log        *zap.Logger
}

// NewGovernanceHandlers constructs the handler suite.
func NewGovernanceHandlers(
	db *sql.DB,
	validator *ValidationEngine,
	policies *PolicyEngine,
	access *AccessController,
	fieldSec *FieldSecurityMasker,
	log *zap.Logger,
) *GovernanceHandlers {
	return &GovernanceHandlers{
		db:       db,
		validator: validator,
		policies:  policies,
		access:    access,
		fieldSec:  fieldSec,
		log:       log,
	}
}

// RegisterRoutes mounts all governance endpoints on the given chi router.
func (gh *GovernanceHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/bo/{boKey}/governance", func(r chi.Router) {
		// Validation Rules
		r.Get("/validation-rules", gh.ListValidationRules)
		r.Post("/validation-rules", gh.CreateValidationRule)
		r.Put("/validation-rules/{ruleID}", gh.UpdateValidationRule)
		r.Delete("/validation-rules/{ruleID}", gh.DeleteValidationRule)
		r.Post("/validation-rules/test", gh.TestValidationExpression)
		r.Post("/validate", gh.ValidateRecord)

		// Policy Rules
		r.Get("/policies", gh.ListPolicies)
		r.Post("/policies", gh.CreatePolicy)
		r.Put("/policies/{policyID}", gh.UpdatePolicy)
		r.Delete("/policies/{policyID}", gh.DeletePolicy)
		r.Post("/policies/simulate", gh.SimulatePolicy)

		// Access Control Matrix
		r.Get("/access", gh.GetAccessMatrix)
		r.Post("/access", gh.UpsertAccessPolicy)

		// Field Security
		r.Get("/field-security", gh.GetFieldSecurity)
		r.Post("/field-security", gh.UpsertFieldSecurity)
		r.Post("/field-security/preview", gh.PreviewFieldMasking)

		// Audit Log
		r.Get("/audit", gh.GetAuditLog)
	})
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func (gh *GovernanceHandlers) tenantID(r *http.Request) (string, error) {
	tid := r.Header.Get("X-Tenant-ID")
	if tid == "" {
		return "", fmt.Errorf("X-Tenant-ID header is required")
	}
	if _, err := uuid.Parse(tid); err != nil {
		return "", fmt.Errorf("X-Tenant-ID must be a valid UUID")
	}
	return tid, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ─── VALIDATION RULE HANDLERS ─────────────────────────────────────────────

// ListValidationRules returns all validation rules for a BO.
func (gh *GovernanceHandlers) ListValidationRules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")
	rules, err := gh.validator.LoadRules(r.Context(), tenantID, boKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

// CreateValidationRule creates a new validation rule.
func (gh *GovernanceHandlers) CreateValidationRule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")

	var rule ValidationRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule.BOKey = boKey
	rule.RuleID = "" // Force insert

	if err := gh.validator.UpsertRule(r.Context(), tenantID, &rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	gh.emitAudit(r.Context(), tenantID, boKey, "", "CREATE_VALIDATION_RULE", "system", "", nil, mustJSON(rule))
	writeJSON(w, http.StatusCreated, rule)
}

// UpdateValidationRule updates an existing validation rule.
func (gh *GovernanceHandlers) UpdateValidationRule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")
	ruleID := chi.URLParam(r, "ruleID")

	var rule ValidationRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule.RuleID = ruleID
	rule.BOKey = boKey

	if err := gh.validator.UpsertRule(r.Context(), tenantID, &rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// DeleteValidationRule soft-deletes a validation rule.
func (gh *GovernanceHandlers) DeleteValidationRule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ruleID := chi.URLParam(r, "ruleID")
	if err := gh.validator.DeleteRule(r.Context(), tenantID, ruleID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TestValidationExpression tests a CEL expression against a sample record.
func (gh *GovernanceHandlers) TestValidationExpression(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Expression string                 `json:"expression"`
		Sample     map[string]interface{} `json:"sample"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	passed, output, err := gh.validator.TestExpression(req.Expression, req.Sample)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"passed": false,
			"output": output,
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"passed": passed,
		"output": output,
	})
}

// ValidateRecord evaluates all validation rules against a submitted record.
func (gh *GovernanceHandlers) ValidateRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")

	var record map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := gh.validator.Evaluate(r.Context(), tenantID, boKey, record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ─── POLICY HANDLERS ─────────────────────────────────────────────────────

// ListPolicies returns all policy rules for a BO.
func (gh *GovernanceHandlers) ListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")
	event := TriggerEvent(r.URL.Query().Get("event"))
	if event == "" {
		event = TriggerOnSave
	}
	policies, err := gh.policies.LoadPolicies(r.Context(), tenantID, boKey, event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

// CreatePolicy creates a new policy rule.
func (gh *GovernanceHandlers) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")

	var policy PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	policy.BOKey = boKey
	policy.PolicyID = ""

	if err := gh.policies.UpsertPolicy(r.Context(), tenantID, &policy); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, policy)
}

// UpdatePolicy updates an existing policy rule.
func (gh *GovernanceHandlers) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")
	policyID := chi.URLParam(r, "policyID")

	var policy PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	policy.PolicyID = policyID
	policy.BOKey = boKey

	if err := gh.policies.UpsertPolicy(r.Context(), tenantID, &policy); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

// DeletePolicy soft-deletes a policy rule.
func (gh *GovernanceHandlers) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	policyID := chi.URLParam(r, "policyID")
	if err := gh.policies.DeletePolicy(r.Context(), tenantID, policyID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SimulatePolicy simulates a policy's condition against a sample record.
func (gh *GovernanceHandlers) SimulatePolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConditionExpr string                 `json:"condition_expr"`
		Record        map[string]interface{} `json:"record"`
		Actor         map[string]interface{} `json:"actor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	triggered, err := gh.policies.SimulatePolicy(req.ConditionExpr, req.Record, req.Actor)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"triggered": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"triggered": triggered})
}

// ─── ACCESS CONTROL HANDLERS ─────────────────────────────────────────────

// GetAccessMatrix returns the full role × operation matrix for a BO.
func (gh *GovernanceHandlers) GetAccessMatrix(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")
	matrix, err := gh.access.GetAccessMatrix(r.Context(), tenantID, boKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, matrix)
}

// UpsertAccessPolicy creates or updates an access policy row.
func (gh *GovernanceHandlers) UpsertAccessPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")

	var policy AccessPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	policy.BOKey = boKey

	if err := gh.access.UpsertPolicy(r.Context(), tenantID, &policy); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

// ─── FIELD SECURITY HANDLERS ─────────────────────────────────────────────

// GetFieldSecurity returns field security configs for a BO.
func (gh *GovernanceHandlers) GetFieldSecurity(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")
	configs, err := gh.fieldSec.LoadFieldSecurityConfigs(r.Context(), tenantID, boKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// UpsertFieldSecurity creates or updates a field security configuration.
func (gh *GovernanceHandlers) UpsertFieldSecurity(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")

	var cfg FieldSecurityConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cfg.BOKey = boKey

	if err := gh.fieldSec.UpsertFieldSecurity(r.Context(), tenantID, &cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// PreviewFieldMasking previews how a record would look through field masking for given roles.
func (gh *GovernanceHandlers) PreviewFieldMasking(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")

	var req struct {
		Record map[string]interface{} `json:"record"`
		Roles  []string               `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sanitised, maskResults, err := gh.fieldSec.ApplyMasks(r.Context(), tenantID, boKey, req.Record, req.Roles)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sanitised":    sanitised,
		"mask_results": maskResults,
	})
}

// ─── AUDIT LOG HANDLER ───────────────────────────────────────────────────

// GetAuditLog returns the paginated audit log for a BO.
func (gh *GovernanceHandlers) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	tenantID, err := gh.tenantID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	boKey := chi.URLParam(r, "boKey")

	limit := 50
	const q = `
	SELECT event_id, tenant_id, bo_key, entity_id, operation, actor_id, actor_role,
	       before_value, after_value, policy_triggered, ip_address::text, created_at
	FROM bo.audit_event
	WHERE tenant_id = $1::uuid AND bo_key = $2
	ORDER BY created_at DESC
	LIMIT $3
	`
	rows, err := gh.db.QueryContext(r.Context(), q, tenantID, boKey, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(
			&e.EventID, &e.TenantID, &e.BOKey, &e.EntityID, &e.Operation,
			&e.ActorID, &e.ActorRole, &e.BeforeValue, &e.AfterValue,
			&e.PolicyTriggered, &e.IPAddress, &e.CreatedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		events = append(events, e)
	}
	writeJSON(w, http.StatusOK, events)
}

// emitAudit writes a single audit event.
func (gh *GovernanceHandlers) emitAudit(
	ctx context.Context,
	tenantID, boKey, entityID, operation, actorID, actorRole string,
	before, after json.RawMessage,
) {
	_, err := gh.db.ExecContext(ctx, `
		INSERT INTO bo.audit_event
		    (tenant_id, bo_key, entity_id, operation, actor_id, actor_role,
		     before_value, after_value, created_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, tenantID, boKey, entityID, operation, actorID, actorRole, before, after)
	if err != nil {
		gh.log.Error("governance: audit emit failed", zap.Error(err))
	}
}

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
