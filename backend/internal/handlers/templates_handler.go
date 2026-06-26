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
	"github.com/gorilla/mux"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// TemplateStep represents a step in a template with parameter placeholders
type TemplateStep struct {
	Priority    int                    `json:"priority"`
	Condition   map[string]interface{} `json:"condition"`
	Action      map[string]interface{} `json:"action"`
	Description string                 `json:"description"`
}

// RuleTemplate is a reusable rule pattern
type RuleTemplate struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenantId"`
	BusinessObject  string                 `json:"businessObject"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	BaseRuleSteps   []TemplateStep         `json:"baseRuleSteps"`
	ParameterSchema map[string]interface{} `json:"parameterSchema"`
	Status          string                 `json:"status"` // draft, approved, deprecated
	Version         int                    `json:"version"`
	IsPublic        bool                   `json:"isPublic"`
	CreatedBy       string                 `json:"createdBy"`
	CreatedAt       string                 `json:"createdAt"`
	UpdatedBy       *string                `json:"updatedBy,omitempty"`
	UpdatedAt       *string                `json:"updatedAt,omitempty"`
	UsageCount      int                    `json:"usageCount,omitempty"`
}

// CreateTemplateRequest is the payload for creating a template
type CreateTemplateRequest struct {
	BusinessObject  string                 `json:"businessObject"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	BaseRuleSteps   []TemplateStep         `json:"baseRuleSteps"`
	ParameterSchema map[string]interface{} `json:"parameterSchema"`
	IsPublic        bool                   `json:"isPublic"`
}

// InstantiateTemplateRequest is the payload for creating a rule from a template
type InstantiateTemplateRequest struct {
	RuleName   string                 `json:"ruleName"`
	Parameters map[string]interface{} `json:"parameters"`
}

// TemplateHandler handles all template-related endpoints
type TemplateHandler struct {
	db *sql.DB
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(db *sql.DB) *TemplateHandler {
	return &TemplateHandler{db: db}
}

// CreateTemplate creates a new rule template
// POST /api/v1/templates
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		http.Error(w, `{"error":"Missing tenant or user ID"}`, http.StatusUnauthorized)
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.BusinessObject == "" || req.Category == "" {
		http.Error(w, `{"error":"name, businessObject, and category are required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set RLS context using set_config function
	if _, err := h.db.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		log.Printf("Error setting RLS context: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"Failed to set tenant context: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Marshal steps and schema to JSON
	stepsJSON, _ := json.Marshal(req.BaseRuleSteps)
	schemaJSON, _ := json.Marshal(req.ParameterSchema)

	// Insert template
	templateID := generateUUID()
	query := `
		INSERT INTO edm.rule_templates (
			id, tenant_id, business_object, name, description, category,
			base_rule_steps, parameter_schema, status, is_public, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'draft', $9, $10)
		RETURNING id, created_at
	`

	var createdAt string
	err := h.db.QueryRowContext(ctx, query,
		templateID, tenantID, req.BusinessObject, req.Name,
		req.Description, req.Category, stepsJSON, schemaJSON,
		req.IsPublic, userID,
	).Scan(&templateID, &createdAt)

	if err != nil {
		fmt.Printf("Error creating template: %v\n", err)
		http.Error(w, `{"error":"Failed to create template"}`, http.StatusInternalServerError)
		return
	}

	template := RuleTemplate{
		ID:              templateID,
		TenantID:        tenantID,
		BusinessObject:  req.BusinessObject,
		Name:            req.Name,
		Description:     req.Description,
		Category:        req.Category,
		BaseRuleSteps:   req.BaseRuleSteps,
		ParameterSchema: req.ParameterSchema,
		Status:          "draft",
		Version:         1,
		IsPublic:        req.IsPublic,
		CreatedBy:       userID,
		CreatedAt:       createdAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// ListTemplates lists templates for a business object
// GET /api/v1/templates?businessObject=calendar&category=weekend&status=approved
func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, `{"error":"Missing tenant ID"}`, http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set RLS context using set_config function
	if _, err := h.db.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		log.Printf("Error setting RLS context in ListTemplates: %v", err)
	}

	// Build query
	businessObject := r.URL.Query().Get("businessObject")
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")

	query := `
		SELECT 
			id, tenant_id, business_object, name, description, category,
			base_rule_steps, parameter_schema, status, version, is_public,
			created_by, created_at, updated_by, updated_at
		FROM edm.rule_templates
		WHERE (tenant_id = $1 OR is_public = TRUE)
	`
	args := []interface{}{tenantID}
	argNum := 2

	if businessObject != "" {
		query += fmt.Sprintf(" AND business_object = $%d", argNum)
		args = append(args, businessObject)
		argNum++
	}

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argNum)
		args = append(args, category)
		argNum++
	}

	if status == "" {
		status = "approved"
	}
	query += fmt.Sprintf(" AND status = $%d", argNum)
	args = append(args, status)

	query += " ORDER BY name ASC"

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Query failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	templates := []RuleTemplate{}
	for rows.Next() {
		var t RuleTemplate
		var stepsJSON, schemaJSON []byte

		err := rows.Scan(
			&t.ID, &t.TenantID, &t.BusinessObject, &t.Name, &t.Description,
			&t.Category, &stepsJSON, &schemaJSON, &t.Status, &t.Version,
			&t.IsPublic, &t.CreatedBy, &t.CreatedAt, &t.UpdatedBy, &t.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("Error scanning template: %v\n", err)
			continue
		}

		json.Unmarshal(stepsJSON, &t.BaseRuleSteps)
		json.Unmarshal(schemaJSON, &t.ParameterSchema)

		templates = append(templates, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// GetTemplate fetches a single template by ID
// GET /api/v1/templates/{templateId}
func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, `{"error":"Missing tenant ID"}`, http.StatusUnauthorized)
		return
	}

	templateID := mux.Vars(r)["templateId"]
	if templateID == "" {
		http.Error(w, `{"error":"Missing template ID"}`, http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, tenant_id, business_object, name, description, category,
		       base_rule_steps, parameter_schema, status, version, is_public,
		       created_by, created_at, updated_by, updated_at
		FROM edm.rule_templates
		WHERE id = $1 AND (tenant_id = $2 OR is_public = true)
	`

	var stepsJSON, schemaJSON sql.NullString
	template := RuleTemplate{}

	err := h.db.QueryRowContext(r.Context(), query, templateID, tenantID).Scan(
		&template.ID, &template.TenantID, &template.BusinessObject,
		&template.Name, &template.Description, &template.Category,
		&stepsJSON, &schemaJSON, &template.Status, &template.Version,
		&template.IsPublic, &template.CreatedBy, &template.CreatedAt,
		&template.UpdatedBy, &template.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"Template not found"}`, http.StatusNotFound)
		} else {
			log.Printf("Error fetching template: %v", err)
			http.Error(w, `{"error":"Failed to fetch template"}`, http.StatusInternalServerError)
		}
		return
	}

	if stepsJSON.Valid {
		json.Unmarshal([]byte(stepsJSON.String), &template.BaseRuleSteps)
	}
	if schemaJSON.Valid {
		json.Unmarshal([]byte(schemaJSON.String), &template.ParameterSchema)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// UpdateTemplate updates a template (must be in draft status)
// PUT /api/v1/templates/{templateId}
func (h *TemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
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

	templateID := mux.Vars(r)["templateId"]
	if templateID == "" {
		http.Error(w, `{"error":"Missing template ID"}`, http.StatusBadRequest)
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start transaction to ensure RLS context persists through all queries
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, `{"error":"Failed to start transaction"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Set RLS context within transaction
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		log.Printf("Error setting RLS context in UpdateTemplate: %v", err)
		http.Error(w, `{"error":"Failed to set tenant context"}`, http.StatusForbidden)
		return
	}

	// Verify template exists and owner can update
	checkQuery := `SELECT status, tenant_id FROM edm.rule_templates WHERE id = $1`
	var status, checkTenant string
	err = tx.QueryRowContext(ctx, checkQuery, templateID).Scan(&status, &checkTenant)
	if err != nil {
		http.Error(w, `{"error":"Template not found"}`, http.StatusNotFound)
		return
	}

	// Case-insensitive UUID comparison (database returns lowercase)
	if strings.ToLower(checkTenant) != strings.ToLower(tenantID) {
		http.Error(w, `{"error":"Forbidden"}`, http.StatusForbidden)
		return
	}

	if status != "draft" {
		http.Error(w, `{"error":"Can only update draft templates"}`, http.StatusConflict)
		return
	}

	stepsJSON, _ := json.Marshal(req.BaseRuleSteps)
	schemaJSON, _ := json.Marshal(req.ParameterSchema)

	updateQuery := `
		UPDATE edm.rule_templates
		SET name = $1, description = $2, category = $3,
		    base_rule_steps = $4, parameter_schema = $5,
		    is_public = $6, updated_by = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING id, created_at
	`

	var createdAt string
	err = tx.QueryRowContext(ctx, updateQuery,
		req.Name, req.Description, req.Category,
		stepsJSON, schemaJSON, req.IsPublic, userID, templateID,
	).Scan(&templateID, &createdAt)

	if err != nil {
		log.Printf("Error updating template: %v", err)
		http.Error(w, `{"error":"Failed to update template"}`, http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, `{"error":"Failed to update template"}`, http.StatusInternalServerError)
		return
	}

	template := RuleTemplate{
		ID:              templateID,
		TenantID:        tenantID,
		BusinessObject:  req.BusinessObject,
		Name:            req.Name,
		Description:     req.Description,
		Category:        req.Category,
		BaseRuleSteps:   req.BaseRuleSteps,
		ParameterSchema: req.ParameterSchema,
		Status:          "draft",
		IsPublic:        req.IsPublic,
		CreatedBy:       userID,
		CreatedAt:       createdAt,
		UpdatedBy:       &userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// DeleteTemplate marks a template as deprecated
// DELETE /api/v1/templates/{templateId}
func (h *TemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
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

	templateID := mux.Vars(r)["templateId"]
	if templateID == "" {
		http.Error(w, `{"error":"Missing template ID"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start transaction to ensure RLS context persists through all queries
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, `{"error":"Failed to start transaction"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Set RLS context within transaction
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		log.Printf("Error setting RLS context in DeleteTemplate: %v", err)
		http.Error(w, `{"error":"Failed to set tenant context"}`, http.StatusForbidden)
		return
	}

	// Verify template exists and belongs to tenant
	checkQuery := `SELECT tenant_id FROM edm.rule_templates WHERE id = $1`
	var checkTenant string
	err = tx.QueryRowContext(ctx, checkQuery, templateID).Scan(&checkTenant)
	if err != nil {
		http.Error(w, `{"error":"Template not found"}`, http.StatusNotFound)
		return
	}

	// Case-insensitive UUID comparison (database returns lowercase)
	if strings.ToLower(checkTenant) != strings.ToLower(tenantID) {
		http.Error(w, `{"error":"Forbidden"}`, http.StatusForbidden)
		return
	}

	// Mark as deprecated
	deleteQuery := `
		UPDATE edm.rule_templates
		SET status = 'deprecated', updated_at = NOW()
		WHERE id = $1
	`

	_, err = tx.ExecContext(ctx, deleteQuery, templateID)
	if err != nil {
		log.Printf("Error deleting template: %v", err)
		http.Error(w, `{"error":"Failed to delete template"}`, http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, `{"error":"Failed to delete template"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"Template deleted"}`)
}

// GetTemplatePreview returns a template with sample parameters resolved
// GET /api/v1/templates/{id}/preview
func (h *TemplateHandler) GetTemplatePreview(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, `{"error":"Missing tenant ID"}`, http.StatusUnauthorized)
		return
	}

	templateID := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	templateID = strings.TrimSuffix(templateID, "/preview")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.db.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		log.Printf("Error setting RLS context in PreviewTemplate: %v", err)
	}

	// Fetch template
	query := `
		SELECT id, name, description, category, base_rule_steps, parameter_schema
		FROM edm.rule_templates
		WHERE id = $1 AND (tenant_id = $2 OR is_public = TRUE)
	`

	var t RuleTemplate
	var stepsJSON, schemaJSON []byte

	err := h.db.QueryRowContext(ctx, query, templateID, tenantID).Scan(
		&t.ID, &t.Name, &t.Description, &t.Category, &stepsJSON, &schemaJSON,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Template not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, `{"error":"Failed to fetch template"}`, http.StatusInternalServerError)
		return
	}

	json.Unmarshal(stepsJSON, &t.BaseRuleSteps)
	json.Unmarshal(schemaJSON, &t.ParameterSchema)

	// Generate sample parameters from schema
	sampleParams := generateSampleParameters(t.ParameterSchema)

	// Resolve template with samples
	resolvedSteps := resolveTemplateParameters(t.BaseRuleSteps, sampleParams)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"template":         t,
		"sampleParameters": sampleParams,
		"previewSteps":     resolvedSteps,
	})
}

// InstantiateTemplate creates a rule from a template with parameters
// POST /api/v1/templates/{id}/create-rule
func (h *TemplateHandler) InstantiateTemplate(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		http.Error(w, `{"error":"Missing tenant or user ID"}`, http.StatusUnauthorized)
		return
	}

	templateID := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	templateID = strings.TrimSuffix(templateID, "/create-rule")

	var req InstantiateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.RuleName == "" {
		http.Error(w, `{"error":"ruleName is required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := h.db.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		log.Printf("Error setting RLS context in InstantiateTemplate: %v", err)
	}

	// Fetch template
	templateQuery := `
		SELECT base_rule_steps, business_object, parameter_schema
		FROM edm.rule_templates
		WHERE id = $1 AND (tenant_id = $2 OR is_public = TRUE)
	`

	var stepsJSON, schemaJSON []byte
	var businessObject string

	err := h.db.QueryRowContext(ctx, templateQuery, templateID, tenantID).Scan(
		&stepsJSON, &businessObject, &schemaJSON,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Template not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, `{"error":"Failed to fetch template"}`, http.StatusInternalServerError)
		return
	}

	// Unmarshal template steps
	var templateSteps []TemplateStep
	json.Unmarshal(stepsJSON, &templateSteps)

	// Validate parameters against schema
	var paramSchema map[string]interface{}
	json.Unmarshal(schemaJSON, &paramSchema)

	if !validateParameters(req.Parameters, paramSchema) {
		http.Error(w, `{"error":"Invalid parameters"}`, http.StatusBadRequest)
		return
	}

	// Create rule from template
	ruleID := uuid.New().String()
	insertQuery := `
		INSERT INTO edm.rules (
			id, tenant_id, business_object, name, description, 
			status, current_version, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'draft', 1, $6, NOW(), NOW())
		RETURNING id
	`

	err = h.db.QueryRowContext(ctx, insertQuery,
		ruleID, tenantID, businessObject, req.RuleName,
		fmt.Sprintf("Created from template: %s", templateID), userID,
	).Scan(&ruleID)

	if err != nil {
		log.Printf("Error creating rule from template: %v", err)
		http.Error(w, `{"error":"Failed to create rule"}`, http.StatusInternalServerError)
		return
	}

	// Record template usage
	usageSQL := `
		INSERT INTO edm.template_usage (template_id, created_rule_id, parameters_used, created_by)
		VALUES ($1, $2, $3, $4)
	`
	paramsJSON, _ := json.Marshal(req.Parameters)
	h.db.ExecContext(ctx, usageSQL, templateID, ruleID, paramsJSON, userID)

	rule := Rule{
		ID:             ruleID,
		BusinessObject: businessObject,
		Name:           req.RuleName,
		Version:        1,
		Status:         "draft",
		CreatedBy:      userID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// ========== HELPER FUNCTIONS ==========

// generateUUID generates a UUID string
func generateUUID() string {
	return uuid.New().String()
}

// resolveTemplateParameters replaces {{param}} placeholders in template steps
func resolveTemplateParameters(steps []TemplateStep, params map[string]interface{}) []TemplateStep {
	resolved := make([]TemplateStep, len(steps))

	for i, step := range steps {
		resolved[i] = step

		// Resolve condition
		if _, ok := step.Condition["value"].(string); ok {
			resolved[i].Condition = make(map[string]interface{})
			for k, v := range step.Condition {
				if strVal, ok := v.(string); ok {
					resolved[i].Condition[k] = resolveString(strVal, params)
				} else {
					resolved[i].Condition[k] = v
				}
			}
		}

		// Resolve action
		if _, ok := step.Action["confidence"].(string); ok {
			resolved[i].Action = make(map[string]interface{})
			for k, v := range step.Action {
				if strVal, ok := v.(string); ok {
					resolved[i].Action[k] = resolveString(strVal, params)
				} else {
					resolved[i].Action[k] = v
				}
			}
		}
	}

	return resolved
}

// resolveString replaces {{param}} with actual value from params map
func resolveString(s string, params map[string]interface{}) interface{} {
	if !strings.Contains(s, "{{") {
		return s
	}

	// Extract parameter name: {{param}} -> param
	start := strings.Index(s, "{{")
	end := strings.Index(s, "}}")

	if start >= 0 && end > start {
		paramName := s[start+2 : end]
		if val, ok := params[paramName]; ok {
			return val
		}
	}

	return s
}

// generateSampleParameters creates sample values based on parameter schema
func generateSampleParameters(schema map[string]interface{}) map[string]interface{} {
	samples := make(map[string]interface{})

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for paramName, paramDef := range properties {
			if def, ok := paramDef.(map[string]interface{}); ok {
				// Use default if available
				if defaultVal, hasDefault := def["default"]; hasDefault {
					samples[paramName] = defaultVal
				} else if example, hasExample := def["example"]; hasExample {
					samples[paramName] = example
				} else {
					// Generate type-based sample
					if typeVal, ok := def["type"].(string); ok {
						switch typeVal {
						case "string":
							samples[paramName] = "sample"
						case "number":
							samples[paramName] = 75 // Default confidence
						case "boolean":
							samples[paramName] = false
						}
					}
				}
			}
		}
	}

	return samples
}

// validateParameters checks parameters against schema
func validateParameters(params map[string]interface{}, schema map[string]interface{}) bool {
	// Simplified validation - in production, use JSON Schema validator
	if required, ok := schema["required"].([]interface{}); ok {
		for _, reqField := range required {
			if name, ok := reqField.(string); ok {
				if _, exists := params[name]; !exists {
					return false
				}
			}
		}
	}

	return true
}

// convertemplatesToPrioritySteps converts template steps to rule priority steps
func convertemplatesToPrioritySteps(steps []TemplateStep) []interface{} {
	result := make([]interface{}, len(steps))
	for i, step := range steps {
		result[i] = map[string]interface{}{
			"priority":    step.Priority,
			"condition":   step.Condition,
			"action":      step.Action,
			"description": step.Description,
		}
	}
	return result
}

// GetInstances lists rules created from a template
// GET /api/v1/templates/{templateId}/instances
func (h *TemplateHandler) GetInstances(w http.ResponseWriter, r *http.Request) {
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

	templateID := mux.Vars(r)["templateId"]
	if templateID == "" {
		http.Error(w, `{"error":"Missing template ID"}`, http.StatusBadRequest)
		return
	}

	// First verify template exists
	templateQuery := `SELECT tenant_id FROM edm.rule_templates WHERE id = $1`
	var templateTenant string
	err := h.db.QueryRowContext(r.Context(), templateQuery, templateID).Scan(&templateTenant)
	if err != nil {
		http.Error(w, `{"error":"Template not found"}`, http.StatusNotFound)
		return
	}

	// Case-insensitive UUID comparison (database returns lowercase)
	if strings.ToLower(templateTenant) != strings.ToLower(tenantID) {
		http.Error(w, `{"error":"Forbidden"}`, http.StatusForbidden)
		return
	}

	// Get all rules created from this template
	query := `
		SELECT r.id, r.tenant_id, r.business_object, r.name, r.status, r.current_version, r.created_at, tu.created_at
		FROM edm.template_usage tu
		JOIN edm.rules r ON tu.created_rule_id = r.id
		WHERE tu.template_id = $1 AND r.tenant_id = $2
		ORDER BY tu.created_at DESC
		LIMIT 100
	`

	rows, err := h.db.QueryContext(r.Context(), query, templateID, tenantID)
	if err != nil {
		log.Printf("Error fetching template instances: %v", err)
		http.Error(w, `{"error":"Failed to fetch instances"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Instance struct {
		RuleID        string `json:"ruleId"`
		Name          string `json:"name"`
		Status        string `json:"status"`
		Version       int    `json:"version"`
		CreatedAt     string `json:"createdAt"`
		CreatedFromAt string `json:"createdFromAt"`
	}

	var instances []Instance
	for rows.Next() {
		var id, tenantID, businessObject, name, status, createdAt, createdFromAt string
		var version int
		err := rows.Scan(&id, &tenantID, &businessObject, &name, &status, &version, &createdAt, &createdFromAt)
		if err != nil {
			continue
		}

		instances = append(instances, Instance{
			RuleID:        id,
			Name:          name,
			Status:        status,
			Version:       version,
			CreatedAt:     createdAt,
			CreatedFromAt: createdFromAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if len(instances) == 0 {
		w.Write([]byte(`[]`))
	} else {
		json.NewEncoder(w).Encode(instances)
	}
}

// RegisterTemplateRoutes registers template routes with the provided router
// This method follows the same pattern as RegisterRoutes for consistency
func (h *TemplateHandler) RegisterTemplateRoutes(router *mux.Router) {
	router.HandleFunc("/templates", h.CreateTemplate).Methods("POST")
	router.HandleFunc("/templates/{templateId}", h.GetTemplate).Methods("GET")
	router.HandleFunc("/templates/{templateId}", h.UpdateTemplate).Methods("PUT")
	router.HandleFunc("/templates/{templateId}", h.DeleteTemplate).Methods("DELETE")
	router.HandleFunc("/templates", h.ListTemplates).Methods("GET")
	router.HandleFunc("/templates/{templateId}/create-rule", h.InstantiateTemplate).Methods("POST")
	router.HandleFunc("/templates/{templateId}/instances", h.GetInstances).Methods("GET")
	router.HandleFunc("/templates/{templateId}/preview", h.GetTemplatePreview).Methods("POST")
}
