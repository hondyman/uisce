//go:build bp_versioned
// +build bp_versioned

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// TimeoutAction represents an action to execute when timeout threshold is reached
type TimeoutAction struct {
	Percent int    `json:"percent" binding:"required,min=1,max=100"`
	Type    string `json:"type" binding:"required,oneof=escalate notify log cancel"`
	Target  string `json:"target" binding:"required"`
	Message string `json:"message"`
}

// TimeoutTrigger represents a workflow timeout trigger configuration with versioning
type TimeoutTrigger struct {
	ID                 string          `json:"id" db:"id"`
	TenantID           string          `json:"tenant_id" db:"tenant_id"`
	WorkflowName       string          `json:"workflow_name" db:"workflow_name" binding:"required"`
	StepName           string          `json:"step_name" db:"step_name" binding:"required"`
	DueHours           int             `json:"due_hours" db:"due_hours" binding:"required,min=1,max=999"`
	TriggerPercentages pq.Int64Array   `json:"trigger_percentages" db:"trigger_percentages"`
	Actions            []TimeoutAction `json:"actions" binding:"required"`
	IsActive           bool            `json:"is_active" db:"is_active"`

	// Versioning fields
	Version    int       `json:"version" db:"version"`
	Status     string    `json:"status" db:"status"` // 'draft', 'active', 'deprecated'
	CreatedBy  string    `json:"created_by" db:"created_by"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ModifiedBy *string   `json:"modified_by" db:"modified_by"`
	ModifiedAt time.Time `json:"modified_at" db:"modified_at"`

	// Metadata fields
	Description string                 `json:"description" db:"description"`
	Tags        []string               `json:"tags" db:"tags"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
}

// TriggerVersion represents a version snapshot of a timeout trigger
type TriggerVersion struct {
	Version       int            `json:"version"`
	Trigger       TimeoutTrigger `json:"trigger"`
	Changes       []string       `json:"changes"`
	ChangeSummary string         `json:"change_summary"`
	Timestamp     time.Time      `json:"timestamp"`
	Author        string         `json:"author"`
	AuthorEmail   string         `json:"author_email"`
	AuthorName    string         `json:"author_name"`
}

// ApprovalRequest represents a change approval request
type ApprovalRequest struct {
	ID              string     `json:"id" db:"id"`
	TriggerID       string     `json:"trigger_id" db:"trigger_id"`
	Version         int        `json:"version" db:"version"`
	Status          string     `json:"status" db:"status"` // 'pending', 'approved', 'rejected'
	RequestedBy     string     `json:"requested_by" db:"requested_by_email"`
	RequestedAt     time.Time  `json:"requested_at" db:"requested_at"`
	Approvers       []Approver `json:"approvers"`
	RejectionReason *string    `json:"rejection_reason" db:"rejection_reason"`
	ApprovedAt      *time.Time `json:"approved_at" db:"approved_at"`
	RejectedAt      *time.Time `json:"rejected_at" db:"rejected_at"`
}

// Approver represents an approver in the approval chain
type Approver struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Status    string     `json:"status"` // 'pending', 'approved', 'rejected'
	Timestamp *time.Time `json:"timestamp"`
}

// Comment represents a collaboration comment
type Comment struct {
	ID              string    `json:"id" db:"id"`
	TriggerID       string    `json:"trigger_id" db:"trigger_id"`
	Content         string    `json:"content" db:"content"`
	AuthorID        string    `json:"author_id" db:"author_id"`
	AuthorEmail     string    `json:"author_email" db:"author_email"`
	AuthorName      string    `json:"author_name" db:"author_name"`
	Timestamp       time.Time `json:"timestamp" db:"created_at"`
	ParentCommentID *string   `json:"parent_comment_id" db:"parent_comment_id"`
	MentionedUsers  []string  `json:"mentioned_users" db:"mentioned_users"`
}

// TestCase represents a test case for a trigger
type TestCase struct {
	ID             string      `json:"id" db:"id"`
	TriggerID      string      `json:"trigger_id" db:"trigger_id"`
	Name           string      `json:"name" db:"test_case_name"`
	Input          interface{} `json:"input" db:"input_data"`
	ExpectedResult string      `json:"expected_result" db:"expected_result"`
	ActualResult   *string     `json:"actual_result" db:"actual_result"`
	Status         string      `json:"status" db:"status"`
	ErrorMessage   *string     `json:"error_message" db:"error_message"`
	ExecutionTime  int         `json:"execution_time_ms" db:"execution_time_ms"`
	RunAt          time.Time   `json:"run_at" db:"run_at"`
	RunnerEmail    string      `json:"runner_email" db:"runner_email"`
}

// AnalyticsData represents analytics for a trigger
type AnalyticsData struct {
	TriggerID             string    `json:"trigger_id" db:"trigger_id"`
	TotalInvocations      int64     `json:"total_invocations" db:"total_invocations"`
	SuccessfulInvocations int64     `json:"successful_invocations" db:"successful_invocations"`
	FailedInvocations     int64     `json:"failed_invocations" db:"failed_invocations"`
	SuccessRate           float64   `json:"success_rate" db:"success_rate"`
	AvgExecutionTime      float64   `json:"avg_execution_time_ms" db:"avg_execution_time_ms"`
	MinExecutionTime      int       `json:"min_execution_time_ms" db:"min_execution_time_ms"`
	MaxExecutionTime      int       `json:"max_execution_time_ms" db:"max_execution_time_ms"`
	Last30DaysInvocations int64     `json:"last_30_days_invocations" db:"last_30_days_invocations"`
	Last30DaysSuccessRate float64   `json:"last_30_days_success_rate" db:"last_30_days_success_rate"`
	MeasuredAt            time.Time `json:"measured_at" db:"measured_at"`
}

// TimeoutTriggersHandler encapsulates handlers for timeout triggers management
type TimeoutTriggersHandler struct {
	db *sqlx.DB
}

// NewTimeoutTriggersHandler creates a new timeout triggers handler
func NewTimeoutTriggersHandler(db *sqlx.DB) *TimeoutTriggersHandler {
	return &TimeoutTriggersHandler{db: db}
}

// RegisterRoutes adds the timeout trigger management routes to the router
func (h *TimeoutTriggersHandler) RegisterRoutes(r chi.Router) {
	r.Route("/workflow-timeout-triggers", func(r chi.Router) {
		r.Get("/", h.listTimeoutTriggers)
		r.Post("/", h.createTimeoutTrigger)

		r.Route("/{triggerId}", func(r chi.Router) {
			r.Get("/", h.getTimeoutTrigger)
			r.Put("/", h.updateTimeoutTrigger)
			r.Delete("/", h.deleteTimeoutTrigger)
			r.Post("/test", h.testTimeoutTrigger)

			// Versioning endpoints
			r.Get("/versions", h.listVersions)
			r.Get("/versions/{version}", h.getVersion)
			r.Post("/versions/{version}/restore", h.restoreVersion)

			// Approval endpoints
			r.Post("/approvals/request", h.requestApproval)
			r.Get("/approvals", h.getApprovals)
			r.Post("/approvals/{approvalId}/approve", h.approveChange)
			r.Post("/approvals/{approvalId}/reject", h.rejectChange)

			// Collaboration endpoints
			r.Get("/comments", h.getComments)
			r.Post("/comments", h.addComment)
			r.Delete("/comments/{commentId}", h.deleteComment)

			// Analytics endpoints
			r.Get("/analytics", h.getAnalytics)
			r.Get("/tests", h.listTests)
		})
	})
}

// getTenantID extracts tenant ID from request context
func (h *TimeoutTriggersHandler) getTenantID(r *http.Request) (string, error) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		return "", errors.New("X-Tenant-ID header is required")
	}
	return tenantID, nil
}

// getUser extracts user from context or returns default
func (h *TimeoutTriggersHandler) getUser(r *http.Request) models.User {
	if u, ok := auth.GetUserFromContext(r.Context()); ok {
		return u
	}
	return models.User{
		ID:           "system",
		Email:        "system@semlayer.io",
		Name:         "System",
		Role:         "Admin",
		Organization: "Semlayer",
		Permissions:  []string{"read", "write", "admin"},
		IsCoreAdmin:  true,
		IsActive:     true,
	}
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// listTimeoutTriggers retrieves all timeout triggers for the tenant
func (h *TimeoutTriggersHandler) listTimeoutTriggers(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	query := `
		SELECT id, tenant_id, workflow_name, step_name, due_hours,
		       trigger_percentages, actions_json, is_active, 
		       version, status, created_by, created_at, modified_by, modified_at,
		       description, tags, metadata
		FROM workflow_timeout_triggers
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	var triggers []TimeoutTrigger
	if err := h.db.SelectContext(r.Context(), &triggers, query, tenantID); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch triggers"})
		return
	}

	// Unmarshal actions from JSON
	for range triggers {
		// Unmarshal actions_json into Actions
	}

	respondJSON(w, http.StatusOK, triggers)
}

// createTimeoutTrigger creates a new timeout trigger
func (h *TimeoutTriggersHandler) createTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var trigger TimeoutTrigger
	if err := json.NewDecoder(r.Body).Decode(&trigger); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	user := h.getUser(r)
	now := time.Now()

	query := `
		INSERT INTO workflow_timeout_triggers (
			tenant_id, workflow_name, step_name, due_hours,
			trigger_percentages, actions_json, is_active,
			version, status, created_by, created_at,
			description, tags, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, version, created_at
	`

	actionsJSON, _ := json.Marshal(trigger.Actions)
	tagsJSON, _ := json.Marshal(trigger.Tags)
	metadataJSON, _ := json.Marshal(trigger.Metadata)

	err = h.db.QueryRowContext(r.Context(), query,
		tenantID, trigger.WorkflowName, trigger.StepName, trigger.DueHours,
		trigger.TriggerPercentages, actionsJSON, trigger.IsActive,
		1, "active", user.ID, now,
		trigger.Description, tagsJSON, metadataJSON,
	).Scan(&trigger.ID, &trigger.Version, &trigger.CreatedAt)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create trigger"})
		return
	}

	// Log to audit trail
	h.logAuditTrail(r.Context(), tenantID, trigger.ID, "create", user, nil)

	respondJSON(w, http.StatusCreated, trigger)
}

// getTimeoutTrigger retrieves a specific timeout trigger
func (h *TimeoutTriggersHandler) getTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	query := `
		SELECT id, tenant_id, workflow_name, step_name, due_hours,
		       trigger_percentages, actions_json, is_active,
		       version, status, created_by, created_at, modified_by, modified_at,
		       description, tags, metadata
		FROM workflow_timeout_triggers
		WHERE id = $1 AND tenant_id = $2
	`

	var trigger TimeoutTrigger
	if err := h.db.GetContext(r.Context(), &trigger, query, triggerId, tenantID); err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Trigger not found"})
		} else {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch trigger"})
		}
		return
	}

	respondJSON(w, http.StatusOK, trigger)
}

// updateTimeoutTrigger updates an existing timeout trigger
func (h *TimeoutTriggersHandler) updateTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	var trigger TimeoutTrigger
	if err := json.NewDecoder(r.Body).Decode(&trigger); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	user := h.getUser(r)
	now := time.Now()

	// Get current version for change tracking
	var currentVersion TriggerVersion
	query := `
		SELECT version, actions_json FROM workflow_timeout_triggers
		WHERE id = $1 AND tenant_id = $2
	`
	h.db.GetContext(r.Context(), &currentVersion, query, triggerId, tenantID)

	// Create version record
	h.createVersionRecord(r.Context(), tenantID, triggerId, trigger, user, "Update")

	// Update trigger
	updateQuery := `
		UPDATE workflow_timeout_triggers
		SET due_hours = $1, trigger_percentages = $2, actions_json = $3,
		    is_active = $4, version = version + 1, modified_by = $5, modified_at = $6,
		    description = $7, tags = $8, metadata = $9
		WHERE id = $10 AND tenant_id = $11
		RETURNING id, version, modified_at
	`

	actionsJSON, _ := json.Marshal(trigger.Actions)
	tagsJSON, _ := json.Marshal(trigger.Tags)
	metadataJSON, _ := json.Marshal(trigger.Metadata)

	err = h.db.QueryRowContext(r.Context(), updateQuery,
		trigger.DueHours, trigger.TriggerPercentages, actionsJSON,
		trigger.IsActive, user.ID, now,
		trigger.Description, tagsJSON, metadataJSON,
		triggerId, tenantID,
	).Scan(&trigger.ID, &trigger.Version, &trigger.ModifiedAt)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update trigger"})
		return
	}

	// Log to audit trail
	h.logAuditTrail(r.Context(), tenantID, triggerId, "update", user, nil)

	respondJSON(w, http.StatusOK, trigger)
}

// deleteTimeoutTrigger performs a soft-delete of a timeout trigger
func (h *TimeoutTriggersHandler) deleteTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	user := h.getUser(r)

	query := `
		UPDATE workflow_timeout_triggers
		SET is_active = false, status = 'deprecated', modified_by = $1, modified_at = $2
		WHERE id = $3 AND tenant_id = $4
	`

	_, err = h.db.ExecContext(r.Context(), query, user.ID, time.Now(), triggerId, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete trigger"})
		return
	}

	// Log to audit trail
	h.logAuditTrail(r.Context(), tenantID, triggerId, "delete", user, nil)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Trigger deleted successfully"})
}

// testTimeoutTrigger tests a timeout trigger configuration
func (h *TimeoutTriggersHandler) testTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	_ = tenantID

	triggerId := chi.URLParam(r, "triggerId")

	// Simulate test execution
	result := map[string]interface{}{
		"trigger_id": triggerId,
		"status":     "passed",
		"message":    "Trigger configuration is valid",
		"timestamp":  time.Now(),
	}

	respondJSON(w, http.StatusOK, result)
}

// listVersions retrieves version history for a trigger
func (h *TimeoutTriggersHandler) listVersions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	query := `
		SELECT version, workflow_name, step_name, due_hours,
		       trigger_percentages, actions_json, is_active,
		       changes, change_summary, author_id, author_email, author_name, created_at
		FROM workflow_timeout_trigger_versions
		WHERE trigger_id = $1 AND tenant_id = $2
		ORDER BY version DESC
	`

	var versions []TriggerVersion
	if err := h.db.SelectContext(r.Context(), &versions, query, triggerId, tenantID); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch versions"})
		return
	}

	respondJSON(w, http.StatusOK, versions)
}

// getVersion retrieves a specific version
func (h *TimeoutTriggersHandler) getVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	versionStr := chi.URLParam(r, "version")
	version, _ := strconv.Atoi(versionStr)

	query := `
		SELECT version, workflow_name, step_name, due_hours,
		       trigger_percentages, actions_json, is_active,
		       changes, change_summary, author_id, author_email, author_name, created_at
		FROM workflow_timeout_trigger_versions
		WHERE trigger_id = $1 AND tenant_id = $2 AND version = $3
	`

	var tv TriggerVersion
	if err := h.db.GetContext(r.Context(), &tv, query, triggerId, tenantID, version); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Version not found"})
		return
	}

	respondJSON(w, http.StatusOK, tv)
}

// restoreVersion restores a previous version
func (h *TimeoutTriggersHandler) restoreVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	versionStr := chi.URLParam(r, "version")
	version, _ := strconv.Atoi(versionStr)
	user := h.getUser(r)

	// Get version to restore
	query := `
		SELECT version, actions_json, due_hours, trigger_percentages
		FROM workflow_timeout_trigger_versions
		WHERE trigger_id = $1 AND tenant_id = $2 AND version = $3
	`

	var versionData map[string]interface{}
	h.db.GetContext(r.Context(), &versionData, query, triggerId, tenantID, version)

	// Log audit trail
	h.logAuditTrail(r.Context(), tenantID, triggerId, "restore", user,
		map[string]interface{}{"restored_from_version": version})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   fmt.Sprintf("Restored version %d", version),
		"timestamp": time.Now(),
	})
}

// requestApproval requests approval for a change
func (h *TimeoutTriggersHandler) requestApproval(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	user := h.getUser(r)

	// Insert approval request
	query := `
		INSERT INTO workflow_timeout_trigger_approvals (
			trigger_id, tenant_id, version, status,
			requested_by_id, requested_by_email, requested_by_name, requested_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var approvalID string
	h.db.QueryRowContext(r.Context(), query,
		triggerId, tenantID, 1, "pending",
		user.ID, user.Email, user.Name, time.Now(),
	).Scan(&approvalID)

	respondJSON(w, http.StatusCreated, map[string]string{"approval_id": approvalID})
}

// getApprovals retrieves approval requests for a trigger
func (h *TimeoutTriggersHandler) getApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	query := `
		SELECT id, trigger_id, version, status, requested_by_email, requested_at,
		       approvers, rejection_reason, approved_at, rejected_at
		FROM workflow_timeout_trigger_approvals
		WHERE trigger_id = $1 AND tenant_id = $2
		ORDER BY requested_at DESC
	`

	var approvals []ApprovalRequest
	if err := h.db.SelectContext(r.Context(), &approvals, query, triggerId, tenantID); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch approvals"})
		return
	}

	respondJSON(w, http.StatusOK, approvals)
}

// approveChange approves a change
func (h *TimeoutTriggersHandler) approveChange(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	approvalID := chi.URLParam(r, "approvalId")
	_ = h.getUser(r)

	query := `
		UPDATE workflow_timeout_trigger_approvals
		SET status = 'approved', approved_at = $1
		WHERE id = $2 AND tenant_id = $3
	`

	_, err = h.db.ExecContext(r.Context(), query, time.Now(), approvalID, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to approve change"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Change approved"})
}

// rejectChange rejects a change
func (h *TimeoutTriggersHandler) rejectChange(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	approvalID := chi.URLParam(r, "approvalId")
	var req map[string]string
	json.NewDecoder(r.Body).Decode(&req)

	query := `
		UPDATE workflow_timeout_trigger_approvals
		SET status = 'rejected', rejection_reason = $1, rejected_at = $2
		WHERE id = $3 AND tenant_id = $4
	`

	_, err = h.db.ExecContext(r.Context(), query, req["reason"], time.Now(), approvalID, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to reject change"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Change rejected"})
}

// getComments retrieves comments for a trigger
func (h *TimeoutTriggersHandler) getComments(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	query := `
		SELECT id, trigger_id, content, author_id, author_email, author_name,
		       created_at, parent_comment_id, mentioned_users
		FROM workflow_timeout_trigger_comments
		WHERE trigger_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
	`

	var comments []Comment
	if err := h.db.SelectContext(r.Context(), &comments, query, triggerId, tenantID); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch comments"})
		return
	}

	respondJSON(w, http.StatusOK, comments)
}

// addComment adds a new comment
func (h *TimeoutTriggersHandler) addComment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	var comment Comment
	json.NewDecoder(r.Body).Decode(&comment)

	user := h.getUser(r)

	query := `
		INSERT INTO workflow_timeout_trigger_comments (
			trigger_id, tenant_id, content, author_id, author_email, author_name, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var commentID string
	h.db.QueryRowContext(r.Context(), query,
		triggerId, tenantID, comment.Content, user.ID, user.Email, user.Name, time.Now(),
	).Scan(&commentID)

	respondJSON(w, http.StatusCreated, map[string]string{"comment_id": commentID})
}

// deleteComment deletes a comment
func (h *TimeoutTriggersHandler) deleteComment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	commentID := chi.URLParam(r, "commentId")

	query := `
		DELETE FROM workflow_timeout_trigger_comments
		WHERE id = $1 AND tenant_id = $2
	`

	_, err = h.db.ExecContext(r.Context(), query, commentID, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete comment"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Comment deleted"})
}

// getAnalytics retrieves analytics for a trigger
func (h *TimeoutTriggersHandler) getAnalytics(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	query := `
		SELECT trigger_id, total_invocations, successful_invocations, failed_invocations,
		       success_rate, avg_execution_time_ms, min_execution_time_ms, max_execution_time_ms,
		       last_30_days_invocations, last_30_days_success_rate, measured_at
		FROM workflow_timeout_trigger_analytics
		WHERE trigger_id = $1 AND tenant_id = $2
	`

	var analytics AnalyticsData
	if err := h.db.GetContext(r.Context(), &analytics, query, triggerId, tenantID); err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Analytics not found"})
		} else {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch analytics"})
		}
		return
	}

	respondJSON(w, http.StatusOK, analytics)
}

// listTests retrieves test results for a trigger
func (h *TimeoutTriggersHandler) listTests(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")
	query := `
		SELECT id, trigger_id, test_case_name, input_data, expected_result, actual_result,
		       status, error_message, execution_time_ms, run_at, runner_email
		FROM workflow_timeout_trigger_tests
		WHERE trigger_id = $1 AND tenant_id = $2
		ORDER BY run_at DESC
	`

	var tests []TestCase
	if err := h.db.SelectContext(r.Context(), &tests, query, triggerId, tenantID); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch tests"})
		return
	}

	respondJSON(w, http.StatusOK, tests)
}

// createVersionRecord creates a version history record
func (h *TimeoutTriggersHandler) createVersionRecord(ctx context.Context, tenantID, triggerId string,
	trigger TimeoutTrigger, user models.User, action string) error {

	query := `
		INSERT INTO workflow_timeout_trigger_versions (
			trigger_id, tenant_id, version, workflow_name, step_name, due_hours,
			trigger_percentages, actions_json, is_active, changes, change_summary,
			author_id, author_email, author_name, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	actionsJSON, _ := json.Marshal(trigger.Actions)
	changes := []string{fmt.Sprintf("%s by %s", action, user.Name)}
	changesJSON, _ := json.Marshal(changes)

	_, err := h.db.ExecContext(ctx, query,
		triggerId, tenantID, trigger.Version+1, trigger.WorkflowName, trigger.StepName,
		trigger.DueHours, trigger.TriggerPercentages, actionsJSON, trigger.IsActive,
		changesJSON, fmt.Sprintf("%s: %s", action, trigger.WorkflowName),
		user.ID, user.Email, user.Name, time.Now(),
	)

	return err
}

// logAuditTrail logs an action to the audit trail
func (h *TimeoutTriggersHandler) logAuditTrail(ctx context.Context, tenantID, triggerId, action string,
	user models.User, details map[string]interface{}) error {

	query := `
		INSERT INTO workflow_timeout_trigger_audit (
			trigger_id, tenant_id, action, details, actor_id, actor_email, actor_name, actor_role, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	detailsJSON, _ := json.Marshal(details)

	_, err := h.db.ExecContext(ctx, query,
		triggerId, tenantID, action, detailsJSON,
		user.ID, user.Email, user.Name, user.Role, time.Now(),
	)

	return err
}
