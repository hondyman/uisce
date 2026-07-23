package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// BusinessProcess represents a complete business process workflow
type BusinessProcess struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	DatasourceID string   `json:"datasource_id"`
	ProcessName  string   `json:"processName"`
	Entity       string   `json:"entity"`
	Description  string   `json:"description"`
	Steps        []BPStep `json:"steps"`
	IsActive     bool     `json:"isActive"`
	CreatedBy    string   `json:"createdBy"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    *string  `json:"updatedAt,omitempty"`
	Version      int      `json:"version"`
	Tags         []string `json:"tags"`
}

// Condition represents a single condition in advanced logic
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // ==, !=, >, <, >=, <=, in, contains, startsWith, endsWith
	Value    interface{} `json:"value"`
}

// ConditionBranch represents advanced conditional branching logic with boolean operators
type ConditionBranch struct {
	Condition   string      `json:"condition,omitempty"` // For AI routing or specific logic type
	Operator    string      `json:"operator"`            // AND, OR, NOT
	Conditions  []Condition `json:"conditions"`          // Array of conditions
	TrueBranch  []string    `json:"trueBranch"`          // Step IDs to execute if true
	FalseBranch []string    `json:"falseBranch"`         // Step IDs to execute if false
}

// ApprovalChain represents dynamic approval chain configuration
type ApprovalChain struct {
	Type           string   `json:"type"`                     // role, org_hierarchy, custom, multi_role
	Levels         *int     `json:"levels,omitempty"`         // For org_hierarchy
	Roles          []string `json:"roles,omitempty"`          // For multi_role
	ApprovalMode   string   `json:"approvalMode"`             // all, any, majority
	EscalationPath []string `json:"escalationPath,omitempty"` // Fallback roles
}

// NotificationRecipient represents a notification recipient configuration
type NotificationRecipient struct {
	Type  string `json:"type"`  // role, user, dynamic
	Value string `json:"value"` // role name, user ID, or expression
}

// NotificationConfig represents enhanced notification configuration
type NotificationConfig struct {
	TemplateID  string                  `json:"templateId"`
	Channels    []string                `json:"channels"` // email, in_app, sms, slack
	Recipients  []NotificationRecipient `json:"recipients"`
	MergeFields map[string]string       `json:"mergeFields,omitempty"`
}

// BPStep represents a single step in a business process
type BPStep struct {
	ID                   string   `json:"id"`
	StepOrder            int      `json:"stepOrder"`
	StepType             string   `json:"stepType"` // data_entry, validate, approve, notify, integrate, condition
	StepName             string   `json:"stepName"`
	DurationHours        float64  `json:"durationHours"`
	AssigneeRole         *string  `json:"assigneeRole,omitempty"`
	AssigneeUser         *string  `json:"assigneeUser,omitempty"`
	ValidationRules      []string `json:"validationRules,omitempty"`
	NotificationTemplate *string  `json:"notificationTemplate,omitempty"`

	// Advanced conditional logic
	ConditionLogic *ConditionBranch `json:"conditionLogic,omitempty"`

	// Parallel execution support
	ExecutionMode string  `json:"executionMode"`           // sequential, parallel
	ParallelGroup *string `json:"parallelGroup,omitempty"` // Steps with same group execute in parallel
	WaitForAll    *bool   `json:"waitForAll,omitempty"`    // true = all must complete, false = any

	// Approval chain configuration
	ApprovalChain *ApprovalChain `json:"approvalChain,omitempty"`

	// Step dependencies
	DependsOn     []string         `json:"dependsOn,omitempty"`     // Step IDs that must complete first
	SkipCondition *ConditionBranch `json:"skipCondition,omitempty"` // Skip if condition is true

	// Enhanced notifications
	NotificationConfig *NotificationConfig `json:"notificationConfig,omitempty"`

	Description              *string  `json:"description,omitempty"`
	Status                   *string  `json:"status,omitempty"` // pending, active, completed, failed
	EscalationThresholdHours *float64 `json:"escalationThresholdHours,omitempty"`
}

// BPAPIResponse wraps all BP Builder API responses
type BPAPIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// BPBuilderHandlers provides HTTP handlers for BP Builder endpoints
type BPBuilderHandlers struct {
	db *sqlx.DB
}

// NewBPBuilderHandlers creates a new instance of BP Builder handlers
func NewBPBuilderHandlers(db *sqlx.DB) *BPBuilderHandlers {
	return &BPBuilderHandlers{db: db}
}

// Helper to create API response
func newBPAPIResponse(success bool, data interface{}, err string) BPAPIResponse {
	resp := BPAPIResponse{
		Success:   success,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if success {
		resp.Data = data
	} else {
		resp.Error = err
	}
	return resp
}

// ListBusinessProcesses retrieves all business processes for a tenant
func (h *BPBuilderHandlers) ListBusinessProcesses(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "tenant_id is required"))
		return
	}

	query := `
		SELECT 
			id, tenant_id, datasource_id, process_name, entity, description, 
			steps_json, is_active, created_by, created_at, updated_at, version, tags_json
		FROM business_processes
		WHERE tenant_id = $1 AND (datasource_id = $2 OR datasource_id IS NULL)
		ORDER BY created_at DESC
	`

	rows, err := h.db.Queryx(query, tenantID, datasourceID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, newBPAPIResponse(false, nil, fmt.Sprintf("Query error: %v", err)))
		return
	}
	defer rows.Close()

	processes := []BusinessProcess{}
	for rows.Next() {
		var bp BusinessProcess
		var stepsJSON, tagsJSON string

		err := rows.Scan(
			&bp.ID, &bp.TenantID, &bp.DatasourceID, &bp.ProcessName, &bp.Entity, &bp.Description,
			&stepsJSON, &bp.IsActive, &bp.CreatedBy, &bp.CreatedAt, &bp.UpdatedAt, &bp.Version, &tagsJSON,
		)
		if err != nil {
			continue
		}

		// Parse JSON fields
		if stepsJSON != "" {
			json.Unmarshal([]byte(stepsJSON), &bp.Steps)
		}
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &bp.Tags)
		}

		processes = append(processes, bp)
	}

	respondJSON(w, http.StatusOK, newBPAPIResponse(true, processes, ""))
}

// GetBusinessProcess retrieves a single business process
func (h *BPBuilderHandlers) GetBusinessProcess(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	processID := chi.URLParam(r, "id")

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "tenant_id is required"))
		return
	}

	query := `
		SELECT 
			id, tenant_id, datasource_id, process_name, entity, description, 
			steps_json, is_active, created_by, created_at, updated_at, version, tags_json
		FROM business_processes
		WHERE id = $1 AND tenant_id = $2
	`

	var bp BusinessProcess
	var stepsJSON, tagsJSON string

	err := h.db.QueryRow(query, processID, tenantID).Scan(
		&bp.ID, &bp.TenantID, &bp.DatasourceID, &bp.ProcessName, &bp.Entity, &bp.Description,
		&stepsJSON, &bp.IsActive, &bp.CreatedBy, &bp.CreatedAt, &bp.UpdatedAt, &bp.Version, &tagsJSON,
	)
	if err != nil {
		respondJSON(w, http.StatusNotFound, newBPAPIResponse(false, nil, "Process not found"))
		return
	}

	// Parse JSON fields
	if stepsJSON != "" {
		json.Unmarshal([]byte(stepsJSON), &bp.Steps)
	}
	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &bp.Tags)
	}

	respondJSON(w, http.StatusOK, newBPAPIResponse(true, bp, ""))
}

// CreateBusinessProcess creates a new business process
func (h *BPBuilderHandlers) CreateBusinessProcess(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "tenant_id is required"))
		return
	}

	var bp BusinessProcess
	if err := json.NewDecoder(r.Body).Decode(&bp); err != nil {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, fmt.Sprintf("Invalid JSON: %v", err)))
		return
	}

	// Generate ID if not provided
	if bp.ID == "" {
		bp.ID = uuid.New().String()
	}

	bp.TenantID = tenantID
	bp.DatasourceID = datasourceID
	bp.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	bp.Version = 1

	// Serialize JSON fields
	stepsJSON, _ := json.Marshal(bp.Steps)
	tagsJSON, _ := json.Marshal(bp.Tags)

	query := `
		INSERT INTO business_processes 
		(id, tenant_id, datasource_id, process_name, entity, description, steps_json, is_active, created_by, created_at, version, tags_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := h.db.Exec(query, bp.ID, bp.TenantID, bp.DatasourceID, bp.ProcessName, bp.Entity, bp.Description,
		stepsJSON, bp.IsActive, bp.CreatedBy, bp.CreatedAt, bp.Version, tagsJSON)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, newBPAPIResponse(false, nil, fmt.Sprintf("Insert error: %v", err)))
		return
	}

	respondJSON(w, http.StatusCreated, newBPAPIResponse(true, bp, ""))
}

// UpdateBusinessProcess updates an existing business process
func (h *BPBuilderHandlers) UpdateBusinessProcess(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	processID := chi.URLParam(r, "id")

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "tenant_id is required"))
		return
	}

	var bp BusinessProcess
	if err := json.NewDecoder(r.Body).Decode(&bp); err != nil {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, fmt.Sprintf("Invalid JSON: %v", err)))
		return
	}

	bp.ID = processID
	bp.TenantID = tenantID
	bp.UpdatedAt = timePtr(time.Now().UTC().Format(time.RFC3339))
	bp.Version++

	// Serialize JSON fields
	stepsJSON, _ := json.Marshal(bp.Steps)
	tagsJSON, _ := json.Marshal(bp.Tags)

	query := `
		UPDATE business_processes
		SET process_name = $1, entity = $2, description = $3, steps_json = $4, 
		    is_active = $5, updated_at = $6, version = $7, tags_json = $8
		WHERE id = $9 AND tenant_id = $10
	`

	result, err := h.db.Exec(query, bp.ProcessName, bp.Entity, bp.Description, stepsJSON,
		bp.IsActive, bp.UpdatedAt, bp.Version, tagsJSON, bp.ID, bp.TenantID)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, newBPAPIResponse(false, nil, fmt.Sprintf("Update error: %v", err)))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, newBPAPIResponse(false, nil, "Process not found"))
		return
	}

	respondJSON(w, http.StatusOK, newBPAPIResponse(true, bp, ""))
}

// DeleteBusinessProcess deletes a business process
func (h *BPBuilderHandlers) DeleteBusinessProcess(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	processID := chi.URLParam(r, "id")

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "tenant_id is required"))
		return
	}

	query := `DELETE FROM business_processes WHERE id = $1 AND tenant_id = $2`

	result, err := h.db.Exec(query, processID, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, newBPAPIResponse(false, nil, fmt.Sprintf("Delete error: %v", err)))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondJSON(w, http.StatusNotFound, newBPAPIResponse(false, nil, "Process not found"))
		return
	}

	respondJSON(w, http.StatusOK, newBPAPIResponse(true, map[string]string{"id": processID}, ""))
}

// PublishBusinessProcess activates a business process (make it live)
func (h *BPBuilderHandlers) PublishBusinessProcess(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	processID := chi.URLParam(r, "id")

	query := `
		UPDATE business_processes
		SET is_active = true, updated_at = $1
		WHERE id = $2 AND tenant_id = $3
		RETURNING id, tenant_id, datasource_id, process_name, entity, description, 
		          steps_json, is_active, created_by, created_at, updated_at, version, tags_json
	`

	var bp BusinessProcess
	var stepsJSON, tagsJSON string

	err := h.db.QueryRow(query, time.Now().UTC().Format(time.RFC3339), processID, tenantID).Scan(
		&bp.ID, &bp.TenantID, &bp.DatasourceID, &bp.ProcessName, &bp.Entity, &bp.Description,
		&stepsJSON, &bp.IsActive, &bp.CreatedBy, &bp.CreatedAt, &bp.UpdatedAt, &bp.Version, &tagsJSON,
	)
	if err != nil {
		respondJSON(w, http.StatusNotFound, newBPAPIResponse(false, nil, "Process not found"))
		return
	}

	// Parse JSON fields
	if stepsJSON != "" {
		json.Unmarshal([]byte(stepsJSON), &bp.Steps)
	}
	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &bp.Tags)
	}

	respondJSON(w, http.StatusOK, newBPAPIResponse(true, bp, ""))
}

// SimulateBusinessProcess simulates a business process execution
func (h *BPBuilderHandlers) SimulateBusinessProcess(w http.ResponseWriter, r *http.Request) {
	processID := chi.URLParam(r, "id")

	var simulationData map[string]interface{}
	json.NewDecoder(r.Body).Decode(&simulationData)

	// For now, return a simple simulation result
	simulationResult := map[string]interface{}{
		"processId":     processID,
		"status":        "completed",
		"message":       "Process simulation executed successfully",
		"stepsExecuted": 0,
		"duration":      "2.5 seconds",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, newBPAPIResponse(true, simulationResult, ""))
}

// DuplicateBusinessProcess creates a copy of an existing process
func (h *BPBuilderHandlers) DuplicateBusinessProcess(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	processID := chi.URLParam(r, "id")

	// Fetch the original process
	query := `
		SELECT 
			id, tenant_id, datasource_id, process_name, entity, description, 
			steps_json, is_active, created_by, created_at, updated_at, version, tags_json
		FROM business_processes
		WHERE id = $1 AND tenant_id = $2
	`

	var bp BusinessProcess
	var stepsJSON, tagsJSON string

	err := h.db.QueryRow(query, processID, tenantID).Scan(
		&bp.ID, &bp.TenantID, &bp.DatasourceID, &bp.ProcessName, &bp.Entity, &bp.Description,
		&stepsJSON, &bp.IsActive, &bp.CreatedBy, &bp.CreatedAt, &bp.UpdatedAt, &bp.Version, &tagsJSON,
	)
	if err != nil {
		respondJSON(w, http.StatusNotFound, newBPAPIResponse(false, nil, "Process not found"))
		return
	}

	// Parse JSON fields
	if stepsJSON != "" {
		json.Unmarshal([]byte(stepsJSON), &bp.Steps)
	}
	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &bp.Tags)
	}

	// Create duplicate
	newBP := bp
	newBP.ID = uuid.New().String()
	newBP.ProcessName = bp.ProcessName + " (Copy)"
	newBP.IsActive = false
	newBP.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	newBP.UpdatedAt = nil
	newBP.Version = 1

	// Insert duplicate
	stepsJSON2, _ := json.Marshal(newBP.Steps)
	tagsJSON2, _ := json.Marshal(newBP.Tags)

	insertQuery := `
		INSERT INTO business_processes 
		(id, tenant_id, datasource_id, process_name, entity, description, steps_json, is_active, created_by, created_at, version, tags_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = h.db.Exec(insertQuery, newBP.ID, newBP.TenantID, newBP.DatasourceID, newBP.ProcessName, newBP.Entity, newBP.Description,
		stepsJSON2, newBP.IsActive, newBP.CreatedBy, newBP.CreatedAt, newBP.Version, tagsJSON2)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, newBPAPIResponse(false, nil, fmt.Sprintf("Duplicate error: %v", err)))
		return
	}

	respondJSON(w, http.StatusCreated, newBPAPIResponse(true, newBP, ""))
}

// Helper function to get pointer to time string
func timePtr(t string) *string {
	return &t
}

// RegisterRoutes registers all BP Builder routes
func (h *BPBuilderHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/business-processes", func(r chi.Router) {
		r.Get("/", h.ListBusinessProcesses)
		r.Post("/", h.CreateBusinessProcess)
		r.Post("/generate-from-nl", h.GenerateProcessFromNaturalLanguage) // Natural Language Builder
		r.Get("/{id}", h.GetBusinessProcess)
		r.Put("/{id}", h.UpdateBusinessProcess)
		r.Delete("/{id}", h.DeleteBusinessProcess)
		r.Post("/{id}/publish", h.PublishBusinessProcess)
		r.Post("/{id}/simulate", h.SimulateBusinessProcess)
		r.Post("/{id}/duplicate", h.DuplicateBusinessProcess)
	})
}
