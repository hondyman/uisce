package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// BulkOperationsHandler handles bulk template and rule operations
type BulkOperationsHandler struct {
	db *sql.DB
}

// NewBulkOperationsHandler creates a new bulk operations handler
func NewBulkOperationsHandler(db *sql.DB) *BulkOperationsHandler {
	return &BulkOperationsHandler{db: db}
}

// ============ REQUEST/RESPONSE TYPES ============

// BulkCreateRequest is the request for bulk creating templates
type BulkCreateRequest struct {
	Templates       []CreateTemplateRequest `json:"templates"`
	ContinueOnError bool                    `json:"continueOnError"`
	Tags            []string                `json:"tags,omitempty"`
}

// BulkCreateResponse is the response from bulk template creation
type BulkCreateResponse struct {
	Status    string             `json:"status"` // success, partial, failed
	Created   int                `json:"created"`
	Failed    int                `json:"failed"`
	Results   []BulkCreateResult `json:"results"`
	BatchID   string             `json:"batchId"`
	Timestamp string             `json:"timestamp"`
}

// BulkCreateResult is a single result in bulk creation
type BulkCreateResult struct {
	TemplateName string `json:"templateName"`
	ID           string `json:"id,omitempty"`
	Status       string `json:"status"` // created, failed
	Error        string `json:"error,omitempty"`
}

// BulkPublishRequest is the request for bulk publishing templates
type BulkPublishRequest struct {
	TemplateIDs     []string `json:"templateIds"`
	TargetStatus    string   `json:"targetStatus"`
	RequireApproval bool     `json:"requireApproval,omitempty"`
	ApprovalComment string   `json:"approvalComment,omitempty"`
}

// BulkPublishResponse is the response from bulk publishing
type BulkPublishResponse struct {
	Status    string              `json:"status"`
	Published int                 `json:"published"`
	Failed    int                 `json:"failed"`
	Results   []BulkPublishResult `json:"results"`
	BatchID   string              `json:"batchId"`
	Timestamp string              `json:"timestamp"`
}

// BulkPublishResult is a single result in bulk publishing
type BulkPublishResult struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	PreviousStatus string `json:"previousStatus"`
	NewStatus      string `json:"newStatus"`
	Status         string `json:"status"` // published, failed
	Error          string `json:"error,omitempty"`
}

// ============ BULK CREATE TEMPLATES ============

// BulkCreateTemplates handles POST /api/v1/templates/bulk-create
func (h *BulkOperationsHandler) BulkCreateTemplates(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	if tenantID == "" || userID == "" {
		http.Error(w, `{"error":"Missing required headers"}`, http.StatusBadRequest)
		return
	}

	var req BulkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate batch size
	if len(req.Templates) == 0 {
		http.Error(w, `{"error":"At least 1 template required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Templates) > 1000 {
		http.Error(w, `{"error":"Maximum 1000 templates per request"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batchID := uuid.New().String()
	response := BulkCreateResponse{
		BatchID:   batchID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Results:   []BulkCreateResult{},
	}

	// Start transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Error starting transaction for bulk create: %v", err)
		http.Error(w, `{"error":"Failed to start transaction"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Set RLS context
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		log.Printf("Error setting RLS context: %v", err)
		http.Error(w, `{"error":"Failed to set tenant context"}`, http.StatusForbidden)
		return
	}

	// Process each template
	for i, template := range req.Templates {
		result := BulkCreateResult{
			TemplateName: template.Name,
			Status:       "created",
		}

		// Validate template
		if template.Name == "" {
			result.Status = "failed"
			result.Error = "Template name cannot be empty"
			response.Results = append(response.Results, result)
			response.Failed++

			if !req.ContinueOnError {
				http.Error(w, `{"error":"Validation failed","template":"`+template.Name+`"}`, http.StatusBadRequest)
				return
			}
			continue
		}

		// Generate UUID for template
		templateID := uuid.New().String()

		// Prepare JSON for steps and schema
		stepsJSON, _ := json.Marshal(template.BaseRuleSteps)
		schemaJSON, _ := json.Marshal(template.ParameterSchema)

		// Insert template
		insertQuery := `
			INSERT INTO edm.rule_templates 
			(id, tenant_id, business_object, name, description, category, 
			 base_rule_steps, parameter_schema, status, version, is_public, 
			 created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		`

		_, err := tx.ExecContext(ctx,
			insertQuery,
			templateID,
			tenantID,
			template.BusinessObject,
			template.Name,
			template.Description,
			template.Category,
			stepsJSON,
			schemaJSON,
			"draft",
			1,
			template.IsPublic,
			userID,
		)

		if err != nil {
			result.Status = "failed"
			if strings.Contains(err.Error(), "duplicate") {
				result.Error = "Template name already exists in this tenant"
			} else {
				result.Error = fmt.Sprintf("Database error: %v", err)
			}
			response.Failed++

			if !req.ContinueOnError {
				http.Error(w, `{"error":"Creation failed","index":`+fmt.Sprintf("%d", i)+`,"reason":"`+result.Error+`"}`, http.StatusInternalServerError)
				return
			}
			response.Results = append(response.Results, result)
			continue
		}

		result.ID = templateID
		response.Created++
		response.Results = append(response.Results, result)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, `{"error":"Failed to commit bulk operation"}`, http.StatusInternalServerError)
		return
	}

	// Determine response status
	if response.Failed == 0 {
		response.Status = "success"
		w.WriteHeader(http.StatusCreated)
	} else if response.Created == 0 {
		response.Status = "failed"
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response.Status = "partial"
		w.WriteHeader(http.StatusMultiStatus)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("[BULK_OP] BatchID=%s Operation=bulk-create Status=%s Created=%d Failed=%d",
		batchID, response.Status, response.Created, response.Failed)
}

// ============ BULK PUBLISH TEMPLATES ============

// BulkPublishTemplates handles POST /api/v1/templates/bulk-publish
func (h *BulkOperationsHandler) BulkPublishTemplates(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, `{"error":"Missing tenant ID"}`, http.StatusBadRequest)
		return
	}

	var req BulkPublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate input
	if len(req.TemplateIDs) == 0 {
		http.Error(w, `{"error":"At least 1 template ID required"}`, http.StatusBadRequest)
		return
	}

	if len(req.TemplateIDs) > 500 {
		http.Error(w, `{"error":"Maximum 500 templates per batch"}`, http.StatusBadRequest)
		return
	}

	if req.TargetStatus == "" {
		http.Error(w, `{"error":"targetStatus is required"}`, http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := map[string]bool{"approved": true, "archived": true, "deprecated": true}
	if !validStatuses[req.TargetStatus] {
		http.Error(w, `{"error":"Invalid target status"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batchID := uuid.New().String()
	response := BulkPublishResponse{
		BatchID:   batchID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Results:   []BulkPublishResult{},
	}

	// Start transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, `{"error":"Failed to start transaction"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Set RLS context
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		http.Error(w, `{"error":"Failed to set tenant context"}`, http.StatusForbidden)
		return
	}

	// Process each template
	for _, templateID := range req.TemplateIDs {
		result := BulkPublishResult{
			ID:     templateID,
			Status: "published",
		}

		// Get current template
		getQuery := `SELECT name, status FROM edm.rule_templates WHERE id = $1`
		var currentName, currentStatus string
		err := tx.QueryRowContext(ctx, getQuery, templateID).Scan(&currentName, &currentStatus)
		if err != nil {
			result.Status = "failed"
			result.Error = "Template not found"
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		result.Name = currentName
		result.PreviousStatus = currentStatus

		// Check if already in target status
		if currentStatus == req.TargetStatus {
			result.Status = "failed"
			result.Error = fmt.Sprintf("Already in status: %s", currentStatus)
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		// Update status
		updateQuery := `
			UPDATE edm.rule_templates
			SET status = $1, updated_at = NOW()
			WHERE id = $2
		`

		_, err = tx.ExecContext(ctx, updateQuery, req.TargetStatus, templateID)
		if err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("Update failed: %v", err)
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		result.NewStatus = req.TargetStatus
		response.Published++
		response.Results = append(response.Results, result)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing bulk publish transaction: %v", err)
		http.Error(w, `{"error":"Failed to commit changes"}`, http.StatusInternalServerError)
		return
	}

	// Determine response status
	if response.Failed == 0 {
		response.Status = "success"
		w.WriteHeader(http.StatusOK)
	} else if response.Published == 0 {
		response.Status = "failed"
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response.Status = "partial"
		w.WriteHeader(http.StatusMultiStatus)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("[BULK_OP] BatchID=%s Operation=bulk-publish Status=%s Published=%d Failed=%d",
		batchID, response.Status, response.Published, response.Failed)
}

// ============ BULK PROMOTE RULES ============

// BulkPromoteRulesRequest is the request for bulk promoting rules
type BulkPromoteRulesRequest struct {
	RuleIDs               []string `json:"ruleIds"`
	FromEnvironment       string   `json:"fromEnvironment"`
	ToEnvironment         string   `json:"toEnvironment"`
	IncludeVersionHistory bool     `json:"includeVersionHistory,omitempty"`
	ExecuteTests          bool     `json:"executeTests,omitempty"`
	NotifyOnComplete      []string `json:"notifyOnComplete,omitempty"`
}

// BulkPromoteRulesResponse is the response from bulk promoting
type BulkPromoteRulesResponse struct {
	Status      string                   `json:"status"`
	Promoted    int                      `json:"promoted"`
	Failed      int                      `json:"failed"`
	PromotionID string                   `json:"promotionId"`
	Results     []BulkPromoteRulesResult `json:"results"`
	Timestamp   string                   `json:"timestamp"`
}

// BulkPromoteRulesResult is a single result in bulk promotion
type BulkPromoteRulesResult struct {
	RuleID          string `json:"ruleId"`
	RuleName        string `json:"ruleName"`
	FromEnvironment string `json:"fromEnvironment"`
	ToEnvironment   string `json:"toEnvironment"`
	NewVersion      int    `json:"newVersion"`
	Status          string `json:"status"` // promoted, failed
	Error           string `json:"error,omitempty"`
}

// BulkPromoteRules handles POST /api/v1/rules/bulk-promote
func (h *BulkOperationsHandler) BulkPromoteRules(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, `{"error":"Missing tenant ID"}`, http.StatusBadRequest)
		return
	}

	var req BulkPromoteRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate input
	if len(req.RuleIDs) == 0 {
		http.Error(w, `{"error":"At least 1 rule ID required"}`, http.StatusBadRequest)
		return
	}

	if len(req.RuleIDs) > 100 {
		http.Error(w, `{"error":"Maximum 100 rules per promotion batch"}`, http.StatusBadRequest)
		return
	}

	if req.FromEnvironment == "" || req.ToEnvironment == "" {
		http.Error(w, `{"error":"fromEnvironment and toEnvironment required"}`, http.StatusBadRequest)
		return
	}

	if req.FromEnvironment == req.ToEnvironment {
		http.Error(w, `{"error":"Source and destination environments cannot be the same"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	promotionID := uuid.New().String()
	response := BulkPromoteRulesResponse{
		PromotionID: promotionID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Results:     []BulkPromoteRulesResult{},
	}

	// Start transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, `{"error":"Failed to start transaction"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Set RLS context
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		http.Error(w, `{"error":"Failed to set tenant context"}`, http.StatusForbidden)
		return
	}

	// Process each rule
	for _, ruleID := range req.RuleIDs {
		result := BulkPromoteRulesResult{
			RuleID:          ruleID,
			FromEnvironment: req.FromEnvironment,
			ToEnvironment:   req.ToEnvironment,
			Status:          "promoted",
		}

		// Get current rule (placeholder - would need environment tracking in real implementation)
		getQuery := `SELECT name, current_version FROM edm.rules WHERE id = $1`
		var ruleName string
		var currentVersion int
		err := tx.QueryRowContext(ctx, getQuery, ruleID).Scan(&ruleName, &currentVersion)
		if err != nil {
			result.Status = "failed"
			result.Error = "Rule not found"
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		result.RuleName = ruleName
		result.NewVersion = currentVersion + 1

		// In a real implementation, this would:
		// 1. Copy rule to destination environment
		// 2. Increment version
		// 3. Mark as promoted
		// For now, we'll just log the promotion

		log.Printf("Promoted rule %s from %s to %s (v%d → v%d)",
			ruleID, req.FromEnvironment, req.ToEnvironment, currentVersion, result.NewVersion)

		response.Promoted++
		response.Results = append(response.Results, result)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"Failed to commit promotion"}`, http.StatusInternalServerError)
		return
	}

	// Determine response status
	if response.Failed == 0 {
		response.Status = "success"
		w.WriteHeader(http.StatusOK)
	} else if response.Promoted == 0 {
		response.Status = "failed"
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response.Status = "partial"
		w.WriteHeader(http.StatusMultiStatus)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("[BULK_OP] PromotionID=%s Operation=bulk-promote Status=%s Promoted=%d Failed=%d",
		promotionID, response.Status, response.Promoted, response.Failed)
}

// RegisterBulkRoutes registers bulk operation routes
func (h *BulkOperationsHandler) RegisterBulkRoutes(router interface{}) {
	// Routes will be registered in main.go
}
