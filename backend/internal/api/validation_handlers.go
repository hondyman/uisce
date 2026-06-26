package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

func (s *Server) createValidationRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	var req struct {
		RuleName     string                 `json:"rule_name"`
		RuleType     string                 `json:"rule_type"`
		TargetEntity string                 `json:"target_entity"`
		Severity     string                 `json:"severity"`
		Description  string                 `json:"description"`
		IsActive     bool                   `json:"is_active"`
		IsCore       bool                   `json:"is_core"`
		Conditions   map[string]interface{} `json:"conditions,omitempty"`
		CreatedBy    string                 `json:"created_by,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RuleName == "" || req.RuleType == "" || req.TargetEntity == "" {
		http.Error(w, "rule_name, rule_type, and target_entity are required", http.StatusBadRequest)
		return
	}

	// Validate rule_type
	validTypes := map[string]bool{
		"business_logic":        true,
		"field_format":          true,
		"cardinality":           true,
		"uniqueness":            true,
		"referential_integrity": true,
	}
	if !validTypes[req.RuleType] {
		http.Error(w, "Invalid rule_type", http.StatusBadRequest)
		return
	}

	// Validate severity
	validSeverities := map[string]bool{"error": true, "warning": true, "info": true}
	if req.Severity == "" {
		req.Severity = "error"
	}
	if !validSeverities[req.Severity] {
		http.Error(w, "Invalid severity", http.StatusBadRequest)
		return
	}

	ruleID := uuid.New().String()
	now := time.Now()

	conditionsJSON := []byte("{}")
	if req.Conditions != nil {
		if b, err := json.Marshal(req.Conditions); err == nil {
			conditionsJSON = b
		}
	}

	query := `
		INSERT INTO public.catalog_validation_rules (
			id, tenant_id, datasource_id, rule_name, rule_type, target_entity,
			severity, description, is_active, is_core,
			condition_json, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, rule_name, rule_type, target_entity, severity, is_active, created_at
	`

	var result struct {
		ID           string    `json:"id"`
		RuleName     string    `json:"rule_name"`
		RuleType     string    `json:"rule_type"`
		TargetEntity string    `json:"target_entity"`
		Severity     string    `json:"severity"`
		IsActive     bool      `json:"is_active"`
		CreatedAt    time.Time `json:"created_at"`
	}

	err := s.DB.QueryRowContext(r.Context(), query,
		ruleID, tenantID, datasourceID, req.RuleName, req.RuleType, req.TargetEntity,
		req.Severity, req.Description, req.IsActive, req.IsCore,
		string(conditionsJSON), req.CreatedBy, now, now).Scan(
		&result.ID, &result.RuleName, &result.RuleType, &result.TargetEntity,
		&result.Severity, &result.IsActive, &result.CreatedAt)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create validation rule: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create validation rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (s *Server) getValidationRules(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	// Optional query parameters for filtering
	targetEntity := r.URL.Query().Get("target_entity")

	query := `
		SELECT id, rule_name, rule_type, target_entity, severity,
		       description, is_active, is_core, condition_json, created_at, updated_at
		FROM public.catalog_validation_rules
		WHERE tenant_id = $1 AND datasource_id = $2
	`
	args := []interface{}{tenantID, datasourceID}

	if targetEntity != "" {
		query += ` AND target_entity = $3`
		args = append(args, targetEntity)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := s.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch validation rules: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rules []map[string]interface{}
	for rows.Next() {
		var id, ruleName, ruleType, targetEntity, severity, description string
		var isActive, isCore bool
		var conditionJSON string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &ruleName, &ruleType, &targetEntity, &severity,
			&description, &isActive, &isCore, &conditionJSON, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		var conditions map[string]interface{}
		if err := json.Unmarshal([]byte(conditionJSON), &conditions); err != nil {
			conditions = map[string]interface{}{}
		}

		rule := map[string]interface{}{
			"id":             id,
			"rule_name":      ruleName,
			"rule_type":      ruleType,
			"target_entity":  targetEntity,
			"severity":       severity,
			"description":    description,
			"is_active":      isActive,
			"is_core":        isCore,
			"condition_json": conditions,
			"created_at":     createdAt,
			"updated_at":     updatedAt,
		}

		rules = append(rules, rule)
	}

	if rules == nil {
		rules = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (s *Server) getValidationRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	ruleID := chi.URLParam(r, "ruleId")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, rule_name, rule_type, target_entity, severity,
		       description, is_active, is_core, condition_json, created_at, updated_at
		FROM public.catalog_validation_rules
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`

	var id, ruleName, ruleType, targetEntity, severity, description string
	var isActive, isCore bool
	var conditionJSON string
	var createdAt, updatedAt time.Time

	err := s.DB.QueryRowContext(r.Context(), query, ruleID, tenantID, datasourceID).Scan(
		&id, &ruleName, &ruleType, &targetEntity, &severity,
		&description, &isActive, &isCore, &conditionJSON, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Validation rule not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch validation rule: %v", err), http.StatusInternalServerError)
		return
	}

	var conditions map[string]interface{}
	if err := json.Unmarshal([]byte(conditionJSON), &conditions); err != nil {
		conditions = map[string]interface{}{}
	}

	rule := map[string]interface{}{
		"id":             id,
		"rule_name":      ruleName,
		"rule_type":      ruleType,
		"target_entity":  targetEntity,
		"severity":       severity,
		"description":    description,
		"is_active":      isActive,
		"is_core":        isCore,
		"condition_json": conditions,
		"created_at":     createdAt,
		"updated_at":     updatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

func (s *Server) updateValidationRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	ruleID := chi.URLParam(r, "ruleId")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		RuleName     string                 `json:"rule_name"`
		RuleType     string                 `json:"rule_type"`
		TargetEntity string                 `json:"target_entity"`
		Severity     string                 `json:"severity"`
		Description  string                 `json:"description"`
		IsActive     bool                   `json:"is_active"`
		IsCore       bool                   `json:"is_core"`
		Conditions   map[string]interface{} `json:"conditions,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	now := time.Now()
	conditionsJSON := []byte("{}")
	if req.Conditions != nil {
		if b, err := json.Marshal(req.Conditions); err == nil {
			conditionsJSON = b
		}
	}

	query := `
		UPDATE public.catalog_validation_rules
		SET rule_name = $1, rule_type = $2, target_entity = $3,
		    severity = $4, description = $5, is_active = $6, is_core = $7,
		    condition_json = $8, updated_at = $9
		WHERE id = $10 AND tenant_id = $11 AND datasource_id = $12
		RETURNING id, rule_name, rule_type, target_entity, severity, is_active, updated_at
	`

	var result struct {
		ID           string    `json:"id"`
		RuleName     string    `json:"rule_name"`
		RuleType     string    `json:"rule_type"`
		TargetEntity string    `json:"target_entity"`
		Severity     string    `json:"severity"`
		IsActive     bool      `json:"is_active"`
		UpdatedAt    time.Time `json:"updated_at"`
	}

	err := s.DB.QueryRowContext(r.Context(), query,
		req.RuleName, req.RuleType, req.TargetEntity,
		req.Severity, req.Description, req.IsActive, req.IsCore,
		string(conditionsJSON), now, ruleID, tenantID, datasourceID).Scan(
		&result.ID, &result.RuleName, &result.RuleType, &result.TargetEntity,
		&result.Severity, &result.IsActive, &result.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Validation rule not found", http.StatusNotFound)
		return
	}
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to update validation rule: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update validation rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) deleteValidationRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	ruleID := chi.URLParam(r, "ruleId")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if ruleID == "" {
		http.Error(w, "Rule ID is required", http.StatusBadRequest)
		return
	}

	result, err := s.DB.ExecContext(r.Context(),
		`DELETE FROM public.catalog_validation_rules
		 WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3`,
		ruleID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete validation rule: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check deletion result: %v", err), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Validation rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "deleted",
		"rule_id": ruleID,
	})
}

// handleListValidationRulesForBO handles GET /api/business-objects/{id}/validations
func handleListValidationRulesForBO(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if tenantID == "" {
		http.Error(w, "missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	// For now, return empty list - this will be wired to the real service
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"validations": []interface{}{},
		"total":       0,
	})
}

// handleGetValidationRuleSchemaForBO handles GET /api/business-objects/{id}/validations/schema
func (s *Server) handleGetValidationRuleSchemaForBO(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	boID := chi.URLParam(r, "id")
	datasourceID := r.URL.Query().Get("datasource_id")
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "en"
	}

	if tenantID == "" {
		http.Error(w, "missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	if boID == "" {
		http.Error(w, "missing business object id", http.StatusBadRequest)
		return
	}

	// Parse IDs
	boUUID, err := uuid.Parse(boID)
	if err != nil {
		http.Error(w, "invalid business object id", http.StatusBadRequest)
		return
	}

	_, err = uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusBadRequest)
		return
	}

	_ = datasourceID // Keep for future use when needed for filtering

	ctx := r.Context()

	// Query bo_fields table for fields of this business object
	query := `
		SELECT id, field_name, field_type, display_label, display_order, 
		       help_text, is_required, is_system_field, is_custom_field,
		       semantic_term_id
		FROM bo_fields
		WHERE business_object_id = $1
		ORDER BY display_order ASC
	`

	logging.GetLogger().Sugar().Infof("querying fields for business_object_id=%s", boUUID.String())

	rows, err := s.DB.QueryContext(ctx, query, boUUID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to query bo_fields: %v", err)
		http.Error(w, "failed to query fields", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var fields []map[string]interface{}
	for rows.Next() {
		var (
			id             string
			fieldName      string
			fieldType      string
			displayLabel   string
			displayOrder   int
			helpText       sql.NullString
			isRequired     bool
			isSystemField  bool
			isCustomField  bool
			semanticTermID sql.NullString
		)

		err := rows.Scan(&id, &fieldName, &fieldType, &displayLabel, &displayOrder,
			&helpText, &isRequired, &isSystemField, &isCustomField, &semanticTermID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("failed to scan field row: %v", err)
			continue
		}

		// Build field object with available data
		field := map[string]interface{}{
			"id":            id,
			"name":          fieldName,
			"displayName":   displayLabel,
			"type":          fieldType,
			"displayOrder":  displayOrder,
			"isRequired":    isRequired,
			"isSystemField": isSystemField,
			"isCustomField": isCustomField,
			"path":          fieldName, // Use field_name as the path for frontend resolution
		}

		if helpText.Valid {
			field["description"] = helpText.String
		}

		if semanticTermID.Valid {
			field["semanticTermId"] = semanticTermID.String
		}

		fields = append(fields, field)
	}

	if err = rows.Err(); err != nil {
		logging.GetLogger().Sugar().Errorf("error iterating field rows: %v", err)
	}

	// Query semantic terms referenced by the fields
	var terms []map[string]interface{}
	if len(fields) > 0 {
		// Collect all semantic term IDs from fields
		var semanticTermIds []string
		for _, field := range fields {
			if termId, ok := field["semanticTermId"].(string); ok && termId != "" {
				semanticTermIds = append(semanticTermIds, termId)
			}
		}

		if len(semanticTermIds) > 0 {
			// Query catalog_node for semantic terms
			placeholders := make([]string, len(semanticTermIds))
			args := make([]interface{}, len(semanticTermIds))
			for i, id := range semanticTermIds {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
				args[i] = id
			}

			termsQuery := fmt.Sprintf(`
				SELECT id, node_name, description, properties
				FROM catalog_node
				WHERE id IN (%s)
			`, strings.Join(placeholders, ","))

			termRows, err := s.DB.QueryContext(ctx, termsQuery, args...)
			if err != nil {
				logging.GetLogger().Sugar().Errorf("failed to query semantic terms: %v", err)
			} else {
				defer termRows.Close()
				for termRows.Next() {
					var (
						id          string
						nodeName    sql.NullString
						description sql.NullString
						properties  []byte
					)

					err := termRows.Scan(&id, &nodeName, &description, &properties)
					if err != nil {
						logging.GetLogger().Sugar().Errorf("failed to scan term row: %v", err)
						continue
					}

					term := map[string]interface{}{
						"id":   id,
						"name": nodeName.String,
					}

					if description.Valid {
						term["description"] = description.String
					}

					// Parse properties JSON if present
					if len(properties) > 0 {
						var props map[string]interface{}
						if err := json.Unmarshal(properties, &props); err == nil {
							term["properties"] = props
						}
					}

					terms = append(terms, term)
				}

				if err = termRows.Err(); err != nil {
					logging.GetLogger().Sugar().Errorf("error iterating term rows: %v", err)
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"fields": fields,
		"terms":  terms,
		"locale": locale,
	})
}
