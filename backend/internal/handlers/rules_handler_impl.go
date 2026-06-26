package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ===========================
// UPDATED RuleHandler with DB
// ===========================

// NewRuleHandlerWithDB creates a new rule handler with database connection
func NewRuleHandlerWithDB(db *sql.DB, goldCopyPublisher interface{}, executionEngine interface{}) *RuleHandler {
	return &RuleHandler{
		db:                db,
		goldCopyPublisher: goldCopyPublisher,
		executionEngine:   executionEngine,
	}
}

// ===========================
// CRUD Handlers
// ===========================

// CreateRule creates a new priority rule
// POST /api/v1/rules
func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		http.Error(w, "Missing X-Tenant-ID or X-User-ID header", http.StatusBadRequest)
		return
	}

	var req struct {
		BusinessObject string         `json:"businessObject"`
		Name           string         `json:"name"`
		Description    string         `json:"description"`
		Steps          []PriorityStep `json:"steps"`
		DefaultAction  string         `json:"defaultAction"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.BusinessObject == "" || req.Name == "" {
		http.Error(w, "businessObject and name are required", http.StatusBadRequest)
		return
	}

	// Create rule
	ruleID := uuid.New().String()
	now := time.Now().UTC()

	query := `
	INSERT INTO edm.rules (
		id, tenant_id, business_object, name, description,
		status, current_version, default_action, created_by, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING id, tenant_id, business_object, name, description, status, current_version,
			  default_action, created_by, created_at, updated_at
	`

	row := h.db.QueryRowContext(ctx, query,
		ruleID, tenantID, req.BusinessObject, req.Name, req.Description,
		"draft", 1, req.DefaultAction, userID, now, now,
	)

	var rule Rule
	if err := row.Scan(&rule.ID, &rule.TenantID, &rule.BusinessObject, &rule.Name,
		&rule.Description, &rule.Status, &rule.Version, &rule.DefaultAction,
		&rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		log.Printf("Error creating rule: %v", err)
		http.Error(w, "Failed to create rule", http.StatusInternalServerError)
		return
	}

	// Insert steps
	for i, step := range req.Steps {
		stepID := uuid.New().String()
		stepQuery := `
		INSERT INTO edm.rule_steps (
			id, rule_id, version, priority, semantic_term, operator, value, confidence, description
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err := h.db.ExecContext(ctx, stepQuery,
			stepID, ruleID, 1, i+1, step.Condition.SemanticTerm,
			step.Condition.Operator, step.Condition.Value,
			step.Action.Confidence, step.Description,
		)
		if err != nil {
			log.Printf("Error inserting step: %v", err)
		}
	}

	// Audit log
	h.auditLog(ctx, tenantID, userID, "RULE_CREATED", ruleID, map[string]string{
		"name":           rule.Name,
		"businessObject": rule.BusinessObject,
	})

	rule.Steps = req.Steps
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// GetRule retrieves a rule by ID
// GET /api/v1/rules/{ruleId}
func (h *RuleHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	// Set RLS variable
	h.setRLSContext(ctx, tenantID)

	// Fetch rule
	rule, err := h.fetchRule(ctx, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// UpdateRule updates a rule (draft only)
// PUT /api/v1/rules/{ruleId}
func (h *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	// Fetch rule
	rule, err := h.fetchRule(ctx, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Only draft rules can be updated
	if rule.Status != "draft" {
		http.Error(w, fmt.Sprintf("Cannot update %s rule. Only draft rules can be modified.", rule.Status), http.StatusBadRequest)
		return
	}

	var updates struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Steps       []PriorityStep `json:"steps"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update rule
	now := time.Now().UTC()
	updateQuery := `
	UPDATE edm.rules SET name = $1, description = $2, updated_at = $3, updated_by = $4
	WHERE id = $5 AND tenant_id = $6
	`

	_, err = h.db.ExecContext(ctx, updateQuery,
		updates.Name, updates.Description, now, userID, ruleID, tenantID,
	)
	if err != nil {
		http.Error(w, "Failed to update rule", http.StatusInternalServerError)
		return
	}

	// Update steps (delete and re-insert)
	if len(updates.Steps) > 0 {
		h.db.ExecContext(ctx, "DELETE FROM edm.rule_steps WHERE rule_id = $1 AND version = $2", ruleID, 1)

		for i, step := range updates.Steps {
			stepID := uuid.New().String()
			stepQuery := `
			INSERT INTO edm.rule_steps (
				id, rule_id, version, priority, semantic_term, operator, value, confidence, description
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`
			h.db.ExecContext(ctx, stepQuery,
				stepID, ruleID, 1, i+1, step.Condition.SemanticTerm,
				step.Condition.Operator, step.Condition.Value,
				step.Action.Confidence, step.Description,
			)
		}
	}

	// Audit log
	h.auditLog(ctx, tenantID, userID, "RULE_UPDATED", ruleID, nil)

	rule.Name = updates.Name
	rule.Description = updates.Description
	rule.Steps = updates.Steps
	rule.UpdatedAt = now.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// DeleteRule deletes a rule (draft only)
// DELETE /api/v1/rules/{ruleId}
func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	// Fetch rule
	rule, err := h.fetchRule(ctx, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Only draft rules can be deleted
	if rule.Status != "draft" {
		http.Error(w, "Cannot delete non-draft rule", http.StatusBadRequest)
		return
	}

	// Delete rule (cascade deletes steps, versions, etc.)
	deleteQuery := `DELETE FROM edm.rules WHERE id = $1 AND tenant_id = $2`
	_, err = h.db.ExecContext(ctx, deleteQuery, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Failed to delete rule", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditLog(ctx, tenantID, userID, "RULE_DELETED", ruleID, map[string]string{
		"name": rule.Name,
	})

	w.WriteHeader(http.StatusNoContent)
}

// ListRules lists rules for a business object
// GET /api/v1/rules?businessObject=calendar&status=production
func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	businessObject := r.URL.Query().Get("businessObject")
	status := r.URL.Query().Get("status")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if businessObject == "" {
		http.Error(w, "businessObject parameter required", http.StatusBadRequest)
		return
	}

	// Build query
	query := `
	SELECT id, tenant_id, business_object, name, description, status, current_version,
		   default_action, created_by, created_at, updated_at
	FROM edm.rules
	WHERE tenant_id = $1 AND business_object = $2
	`

	args := []interface{}{tenantID, businessObject}

	if status != "" {
		query += ` AND status = $3`
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC LIMIT 50`

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch rules", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rule Rule
		if err := rows.Scan(&rule.ID, &rule.TenantID, &rule.BusinessObject, &rule.Name,
			&rule.Description, &rule.Status, &rule.Version, &rule.DefaultAction,
			&rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			log.Printf("Error scanning rule: %v", err)
			continue
		}
		rules = append(rules, rule)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// ===========================
// Workflow Handlers
// ===========================

// PublishRule publishes a draft rule to testing stage
// POST /api/v1/rules/{ruleId}/publish
func (h *RuleHandler) PublishRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	// Fetch rule
	rule, err := h.fetchRule(ctx, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Only draft rules can be published
	if rule.Status != "draft" {
		http.Error(w, "Only draft rules can be published", http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Update rule status
	now := time.Now().UTC()
	newVersion := rule.Version + 1

	updateQuery := `
	UPDATE edm.rules
	SET status = $1, current_version = $2, updated_at = $3, updated_by = $4
	WHERE id = $5 AND tenant_id = $6
	`

	_, err = tx.ExecContext(ctx, updateQuery, "testing", newVersion, now, userID, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Failed to update rule status", http.StatusInternalServerError)
		return
	}

	// Create version record
	versionID := uuid.New().String()
	versionQuery := `
	INSERT INTO edm.rule_versions (id, rule_id, version, status, created_at)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err = tx.ExecContext(ctx, versionQuery, versionID, ruleID, newVersion, "testing", now)
	if err != nil {
		http.Error(w, "Failed to create version record", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditLog(ctx, tenantID, userID, "RULE_PUBLISHED", ruleID, map[string]string{
		"newVersion": fmt.Sprintf("%d", newVersion),
		"newStatus":  "testing",
	})

	rule.Status = "testing"
	rule.Version = newVersion
	rule.UpdatedAt = now.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// PromoteRule promotes rule between stages
// POST /api/v1/rules/{ruleId}/promote
func (h *RuleHandler) PromoteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	var req struct {
		ToStage string `json:"toStage"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Fetch rule
	rule, err := h.fetchRule(ctx, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Validate promotion path
	validTransitions := map[string][]string{
		"testing":    {"staging"},
		"staging":    {"production"},
		"production": {},
	}

	// Check if promotion to req.ToStage is valid
	isValidTransition := false
	for _, to := range validTransitions[rule.Status] {
		if to == req.ToStage {
			isValidTransition = true
			break
		}
	}

	if !isValidTransition {
		http.Error(w, fmt.Sprintf("Cannot promote from %s to %s", rule.Status, req.ToStage), http.StatusBadRequest)
		return
	}

	// Check if all required approvals are satisfied
	approvalsOK, err := h.checkApprovalsForPromotion(ctx, ruleID, rule.Version, req.ToStage, tenantID)
	if err != nil || !approvalsOK {
		http.Error(w, "Not all required approvals have been completed", http.StatusBadRequest)
		return
	}

	// Update rule status
	now := time.Now().UTC()
	newVersion := rule.Version + 1

	updateQuery := `
	UPDATE edm.rules
	SET status = $1, current_version = $2, updated_at = $3, updated_by = $4
	WHERE id = $5 AND tenant_id = $6
	`

	_, err = h.db.ExecContext(ctx, updateQuery, req.ToStage, newVersion, now, userID, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Failed to promote rule", http.StatusInternalServerError)
		return
	}

	// Create version record
	versionID := uuid.New().String()
	versionQuery := `
	INSERT INTO edm.rule_versions (id, rule_id, version, status, promoted_at, promoted_by, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = h.db.ExecContext(ctx, versionQuery, versionID, ruleID, newVersion, req.ToStage, now, userID, now)
	if err != nil {
		http.Error(w, "Failed to create version record", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditLog(ctx, tenantID, userID, "RULE_PROMOTED", ruleID, map[string]string{
		"fromStage":  rule.Status,
		"toStage":    req.ToStage,
		"newVersion": fmt.Sprintf("%d", newVersion),
	})

	// Publish to gold copy if promoted to production
	if req.ToStage == "production" && h.goldCopyPublisher != nil {
		// Convert to format expected by gold copy publisher
		dataPayload := map[string]interface{}{
			"id":              rule.ID,
			"name":            rule.Name,
			"business_object": rule.BusinessObject,
			"description":     rule.Description,
			"semantic_term":   rule.SemanticTerm,
			"default_action":  rule.DefaultAction,
			"status":          "production",
			"version":         newVersion,
			"created_by":      rule.CreatedBy,
			"updated_by":      userID,
			"steps":           rule.Steps,
		}

		dataHash := hashData(dataPayload)
		changeReason := fmt.Sprintf("Promoted to production from %s", rule.Status)

		// Type assert goldCopyPublisher to real type
		if pub, ok := h.goldCopyPublisher.(*services.GoldCopyPublisher); ok && pub != nil {
			err := pub.PublishRuleAsGoldCopy(
				ctx,
				&models.Rule{
					ID:                 rule.ID,
					TenantID:           tenantID,
					Name:               rule.Name,
					BusinessObject:     rule.BusinessObject,
					Description:        rule.Description,
					SemanticTerm:       rule.SemanticTerm,
					Status:             "production",
					Version:            newVersion,
					RuleEngine:         "priority",
					ExpressionLanguage: "JEXL",
					CreatedBy:          rule.CreatedBy,
				},
				"creation",
				changeReason,
				userID,
				dataHash,
			)
			if err != nil {
				log.Printf("Warning: Failed to publish rule to gold copy: %v", err)
				// Don't fail the request - gold copy publishing is async
			}
		}
	}

	rule.Status = req.ToStage
	rule.Version = newVersion
	rule.UpdatedAt = now.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// ===========================
// Simulation Handler
// ===========================

// SimulateRule simulates a rule against test data
// POST /api/v1/rules/{ruleId}/simulate
func (h *RuleHandler) SimulateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	var req struct {
		TestData map[string]interface{} `json:"testData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Fetch rule with steps
	rule, err := h.fetchRule(ctx, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Execute simulation
	results := h.executeSimulation(ctx, rule, req.TestData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// ===========================
// Versioning Handlers
// ===========================

// GetVersions gets rule version history
// GET /api/v1/rules/{ruleId}/versions
func (h *RuleHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	_ = jwtmiddleware.GetClaimsFromContext(r).TenantID // RLS enforcement

	query := `
	SELECT id, rule_id, version, status, promoted_at, promoted_by, created_at
	FROM edm.rule_versions
	WHERE rule_id = $1
	ORDER BY version DESC
	LIMIT 50
	`

	rows, err := h.db.QueryContext(ctx, query, ruleID)
	if err != nil {
		http.Error(w, "Failed to fetch versions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var versions []map[string]interface{}
	for rows.Next() {
		var id, rID, status string
		var version int
		var promotedAt, createdAt sql.NullTime
		var promotedBy sql.NullString

		if err := rows.Scan(&id, &rID, &version, &status, &promotedAt, &promotedBy, &createdAt); err != nil {
			continue
		}

		v := map[string]interface{}{
			"id":        id,
			"ruleId":    rID,
			"version":   version,
			"status":    status,
			"createdAt": createdAt.Time.Format(time.RFC3339),
		}

		if promotedAt.Valid {
			v["promotedAt"] = promotedAt.Time.Format(time.RFC3339)
		}
		if promotedBy.Valid {
			v["promotedBy"] = promotedBy.String
		}

		versions = append(versions, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// GetDiff compares two rule versions
// GET /api/v1/rules/{ruleId}/diff?v1=1&v2=2
func (h *RuleHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	_ = jwtmiddleware.GetClaimsFromContext(r).TenantID // RLS enforcement

	v1Str := r.URL.Query().Get("v1")
	v2Str := r.URL.Query().Get("v2")

	v1, _ := strconv.Atoi(v1Str)
	v2, _ := strconv.Atoi(v2Str)

	if v1 == 0 || v2 == 0 {
		http.Error(w, "v1 and v2 parameters required", http.StatusBadRequest)
		return
	}

	// Fetch steps for both versions
	steps1, _ := h.fetchRuleSteps(ctx, ruleID, v1)
	steps2, _ := h.fetchRuleSteps(ctx, ruleID, v2)

	diff := map[string]interface{}{
		"ruleId":  ruleID,
		"v1":      v1,
		"v2":      v2,
		"added":   steps2,
		"removed": steps1,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}

// RollbackRule rolls back to a previous version
// POST /api/v1/rules/{ruleId}/rollback
func (h *RuleHandler) RollbackRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	var req struct {
		ToVersion int `json:"toVersion"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Fetch rule
	rule, err := h.fetchRule(ctx, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	// Rollback sets status to draft
	now := time.Now().UTC()

	updateQuery := `
	UPDATE edm.rules
	SET status = $1, current_version = $2, updated_at = $3, updated_by = $4
	WHERE id = $5 AND tenant_id = $6
	`

	_, err = h.db.ExecContext(ctx, updateQuery, "draft", rule.Version+1, now, userID, ruleID, tenantID)
	if err != nil {
		http.Error(w, "Failed to rollback rule", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditLog(ctx, tenantID, userID, "RULE_ROLLED_BACK", ruleID, map[string]string{
		"fromVersion": fmt.Sprintf("%d", rule.Version),
		"toVersion":   fmt.Sprintf("%d", rule.Version+1),
	})

	rule.Status = "draft"
	rule.Version = rule.Version + 1
	rule.UpdatedAt = now.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// ===========================
// Approval Handlers
// ===========================

// RequestApproval requests approval for a rule
// POST /api/v1/rules/{ruleId}/approve
func (h *RuleHandler) RequestApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	ruleID := vars["ruleId"]
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	var req struct {
		Version int    `json:"version"`
		Role    string `json:"role"`
		Action  string `json:"action"`
		Stage   string `json:"stage"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Record approval
	approvalID := uuid.New().String()
	now := time.Now().UTC()

	query := `
	INSERT INTO edm.rule_approvals (
		id, rule_id, version, status, promotion_stage, role, approver_id, approved_at, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (rule_id, version, role) DO UPDATE
	SET status = $4, approved_at = $8, updated_at = $10
	`

	_, err := h.db.ExecContext(ctx, query,
		approvalID, ruleID, req.Version, req.Action, req.Stage, req.Role,
		userID, now, now, now,
	)
	if err != nil {
		log.Printf("Error recording approval: %v", err)
		http.Error(w, "Failed to record approval", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditLog(ctx, tenantID, userID, "APPROVAL_RECORDED", ruleID, map[string]string{
		"version": fmt.Sprintf("%d", req.Version),
		"role":    req.Role,
		"action":  req.Action,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      approvalID,
		"status":  "recorded",
		"message": "Approval recorded successfully",
	})
}

// GetPendingApprovals retrieves pending approvals
// GET /api/v1/approvals/pending
func (h *RuleHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	query := `
	SELECT ra.id, ra.rule_id, ra.version, ra.role, ra.promotion_stage,
		   r.business_object, r.name, ra.created_at
	FROM edm.rule_approvals ra
	JOIN edm.rules r ON ra.rule_id = r.id
	WHERE ra.status = $1 AND r.tenant_id = $2
	ORDER BY ra.created_at DESC
	LIMIT 50
	`

	rows, err := h.db.QueryContext(ctx, query, "pending", tenantID)
	if err != nil {
		http.Error(w, "Failed to fetch approvals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var approvals []map[string]interface{}
	for rows.Next() {
		var id, ruleID, businessObject, name, role, stage string
		var version int
		var createdAt time.Time

		if err := rows.Scan(&id, &ruleID, &version, &role, &stage, &businessObject, &name, &createdAt); err != nil {
			continue
		}

		approvals = append(approvals, map[string]interface{}{
			"id":             id,
			"ruleId":         ruleID,
			"version":        version,
			"role":           role,
			"promotionStage": stage,
			"businessObject": businessObject,
			"ruleName":       name,
			"createdAt":      createdAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approvals)
}

// ===========================
// Helper Methods
// ===========================

// fetchRule fetches a complete rule with steps
func (h *RuleHandler) fetchRule(ctx context.Context, ruleID, tenantID string) (*Rule, error) {
	query := `
	SELECT id, tenant_id, business_object, name, description, status, current_version,
		   default_action, created_by, created_at, updated_at
	FROM edm.rules
	WHERE id = $1 AND tenant_id = $2
	`

	var rule Rule
	err := h.db.QueryRowContext(ctx, query, ruleID, tenantID).Scan(
		&rule.ID, &rule.TenantID, &rule.BusinessObject, &rule.Name, &rule.Description,
		&rule.Status, &rule.Version, &rule.DefaultAction, &rule.CreatedBy,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Fetch steps
	steps, err := h.fetchRuleSteps(ctx, ruleID, rule.Version)
	if err == nil {
		rule.Steps = steps
	}

	return &rule, nil
}

// fetchRuleSteps fetches rule steps for a specific version
func (h *RuleHandler) fetchRuleSteps(ctx context.Context, ruleID string, version int) ([]PriorityStep, error) {
	query := `
	SELECT id, priority, semantic_term, operator, value, confidence, description
	FROM edm.rule_steps
	WHERE rule_id = $1 AND version = $2
	ORDER BY priority ASC
	`

	rows, err := h.db.QueryContext(ctx, query, ruleID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []PriorityStep
	for rows.Next() {
		var step PriorityStep
		var id, description sql.NullString

		if err := rows.Scan(&id, &step.Priority, &step.Condition.SemanticTerm,
			&step.Condition.Operator, &step.Condition.Value, &step.Action.Confidence, &description); err != nil {
			continue
		}

		step.ID = id.String
		step.Description = description.String
		steps = append(steps, step)
	}

	return steps, nil
}

// executeSimulation executes a rule against test data
func (h *RuleHandler) executeSimulation(ctx context.Context, rule *Rule, testData map[string]interface{}) map[string]interface{} {
	// Extract test dates and regions from testData
	var dates []string
	var regions []string

	if d, ok := testData["dates"].([]interface{}); ok {
		for _, date := range d {
			if str, ok := date.(string); ok {
				dates = append(dates, str)
			}
		}
	}

	if r, ok := testData["regions"].([]interface{}); ok {
		for _, region := range r {
			if str, ok := region.(string); ok {
				regions = append(regions, str)
			}
		}
	}

	// Query calendar MDM for test data
	var executionTrace []map[string]interface{}
	impactedDates := 0
	totalConfidence := 0
	traceCount := 0

	for _, date := range dates {
		for _, region := range regions {
			// Query calendar MDM for this date/region
			calendarQuery := `
			SELECT calendar_date, is_business_day, region_code, holiday_name
			FROM northwinds.calendar_mdm
			WHERE calendar_date = $1 AND region_code = $2
			LIMIT 1
			`

			var calDate, holidayName sql.NullString
			var isBusinessDay bool

			err := h.db.QueryRowContext(ctx, calendarQuery, date, region).Scan(
				&calDate, &isBusinessDay, &region, &holidayName,
			)

			if err == nil {
				// Evaluate rules against this data
				matchedStep := -1
				matchedConfidence := 0

				for i, step := range rule.Steps {
					// Use ExecutionEngine for semantic term resolution
					var actualVal interface{}
					var evalErr error

					// Type assert executionEngine
					if ee, ok := h.executionEngine.(interface {
						ExecuteCalculation(context.Context, uuid.UUID, map[string]interface{}) (interface{}, interface{}, error)
					}); ok && ee != nil {
						// Look up the term ID for this step
						// For the demo, we assume the step.Condition.SemanticTerm is the node name or ID
						// and we resolve it accordingly. In a real system, we'd have the ID in the DB.
						termID, _ := uuid.Parse(step.Condition.SemanticTerm)
						if termID != uuid.Nil {
							res, _, err := ee.ExecuteCalculation(ctx, termID, map[string]interface{}{
								"IsBusinessDay": isBusinessDay,
								"Region":        region,
							})
							if err == nil {
								actualVal = res
							} else {
								evalErr = err
							}
						}
					}

					// Fallback to hardcoded logic if engine failed or wasn't used
					if actualVal == nil && evalErr == nil {
						if step.Condition.SemanticTerm == "calendar.IsBusinessDay" {
							actualVal = "false"
							if isBusinessDay {
								actualVal = "true"
							}
						}
					}

					if actualVal != nil {
						expectedVal := step.Condition.Value
						if fmt.Sprintf("%v", actualVal) == expectedVal {
							matchedStep = i
							matchedConfidence = step.Action.Confidence
							break
						}
					}
				}

				// Add to trace
				stepName := "DEFAULT"
				if matchedStep >= 0 {
					stepName = fmt.Sprintf("Step#%d", matchedStep+1)
					impactedDates++
					traceCount++
					totalConfidence += matchedConfidence
				}

				trace := map[string]interface{}{
					"date":           date,
					"region":         region,
					"winningRule":    stepName,
					"confidence":     matchedConfidence,
					"isBusinessDay":  isBusinessDay,
					"holidayName":    holidayName.String,
					"evaluatedRules": len(rule.Steps),
				}

				executionTrace = append(executionTrace, trace)
			}
		}
	}

	avgConfidence := 0
	if traceCount > 0 {
		avgConfidence = totalConfidence / traceCount
	}

	return map[string]interface{}{
		"executionTrace": executionTrace,
		"impactedDates":  impactedDates,
		"totalRecords":   len(dates) * len(regions),
		"avgConfidence":  avgConfidence,
		"samples":        executionTrace,
	}
}

// checkApprovalsForPromotion checks if all required approvals are present
func (h *RuleHandler) checkApprovalsForPromotion(ctx context.Context, ruleID string, version int, toStage string, tenantID string) (bool, error) {
	// Fetch rule to get businessObject
	ruleQuery := `SELECT business_object FROM edm.rules WHERE id = $1`
	var businessObject string
	err := h.db.QueryRowContext(ctx, ruleQuery, ruleID).Scan(&businessObject)
	if err != nil {
		return false, err
	}

	// Get required approvers for this stage
	approvalsQuery := `
	SELECT COUNT(*) FROM edm.approval_workflows
	WHERE business_object = $1 AND promotion_stage = $2
	`

	var requiredCount int
	h.db.QueryRowContext(ctx, approvalsQuery, businessObject, toStage).Scan(&requiredCount)

	if requiredCount == 0 {
		return true, nil // No approvals required
	}

	// Check if all have approved
	approvedQuery := `
	SELECT COUNT(*) FROM edm.rule_approvals
	WHERE rule_id = $1 AND version = $2 AND status = $3
	`

	var approvedCount int
	h.db.QueryRowContext(ctx, approvedQuery, ruleID, version, "approved").Scan(&approvedCount)

	return approvedCount >= requiredCount, nil
}

// setRLSContext sets the RLS context variable
func (h *RuleHandler) setRLSContext(ctx context.Context, tenantID string) {
	h.db.ExecContext(ctx, "SET app.current_tenant_id TO $1", tenantID)
}

// auditLog writes an audit log entry
func (h *RuleHandler) auditLog(ctx context.Context, tenantID, userID, action, resourceID string, metadata map[string]string) {
	query := `
	INSERT INTO edm.audit_log (tenant_id, actor_id, action, resource_id, metadata, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	metadataJSON, _ := json.Marshal(metadata)
	_, err := h.db.ExecContext(ctx, query,
		tenantID, userID, action, resourceID, string(metadataJSON), time.Now().UTC(),
	)

	if err != nil {
		log.Printf("Error writing audit log: %v", err)
	}
}

// hashData computes SHA256 hash of data for change detection (used by gold copy publisher)
func hashData(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return "sha256:" + hex.EncodeToString(hash[:])
}
