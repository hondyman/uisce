package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/backend/pkg/ui"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// UI HANDLERS - Workday-Style Metadata-Driven Forms
// ============================================================================

type UIHandler struct {
	uiGenerator *ui.UIGenerator
	db          *sqlx.DB
}

// NewUIHandler creates a new UI handler
func NewUIHandler(db *sqlx.DB) *UIHandler {
	return &UIHandler{
		uiGenerator: ui.NewUIGenerator(db),
		db:          db,
	}
}

// ============================================================================
// GET /api/ui/forms/:layoutId
// Returns the complete form definition for a given layout
// ============================================================================

// GetFormDefinition returns form metadata and validation rules
// @Summary Get Form Definition
// @Description Load complete form definition with all metadata, fields, and validation rules
// @Tags UI
// @Param layoutId path string true "Layout ID"
// @Param tenant_id query string true "Tenant ID"
// @Param datasource_id query string true "Datasource ID"
// @Produce json
// @Success 200 {object} ui.FormDefinition
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /ui/forms/{layoutId} [get]
func (h *UIHandler) GetFormDefinition(c *gin.Context) {
	layoutID := c.Param("layoutId")
	tenantID := c.GetString("tenant_id")
	// datasourceID := c.GetString("datasource_id") // For future audit

	if layoutID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "layoutId is required"})
		return
	}

	// Generate form definition
	formDef, err := h.uiGenerator.GetFormDefinition(c.Request.Context(), layoutID)
	if err != nil {
		// Check if not found
		if err.Error() == "sql: no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "layout not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to load form: %v", err)})
		return
	}

	// Verify tenant ownership (security check)
	if formDef.BusinessObject.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, formDef)
}

// ============================================================================
// POST /api/ui/validate
// Validates form data against all business rules
// ============================================================================

type ValidateRequest struct {
	BOID string                 `json:"bo_id" binding:"required"`
	Data map[string]interface{} `json:"data" binding:"required"`
}

// ValidateFormData validates form submission data
// @Summary Validate Form Data
// @Description Execute all validation rules for a business object
// @Tags UI
// @Param request body ValidateRequest true "Validation Request"
// @Param tenant_id query string true "Tenant ID"
// @Param datasource_id query string true "Datasource ID"
// @Produce json
// @Success 200 {object} ui.ValidationResult
// @Failure 400 {object} map[string]string
// @Router /ui/validate [post]
func (h *UIHandler) ValidateFormData(c *gin.Context) {
	var req ValidateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	datasourceID := c.GetString("datasource_id")

	// Validate data
	result, err := h.uiGenerator.ValidateFormData(c.Request.Context(), req.BOID, req.Data, tenantID, datasourceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("validation failed: %v", err)})
		return
	}

	// Log validation for audit
	_ = tenantID // Use for logging

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// POST /api/ui/save
// Saves form data without triggering business processes
// ============================================================================

type SaveRequest struct {
	BOID string                 `json:"bo_id" binding:"required"`
	Data map[string]interface{} `json:"data" binding:"required"`
}

// SaveFormData saves form data to database
// @Summary Save Form Data
// @Description Save form data after validation (without triggering BP)
// @Tags UI
// @Param request body SaveRequest true "Save Request"
// @Param tenant_id query string true "Tenant ID"
// @Param datasource_id query string true "Datasource ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /ui/save [post]
func (h *UIHandler) SaveFormData(c *gin.Context) {
	var req SaveRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	// 1. Validate
	validation, err := h.uiGenerator.ValidateFormData(c.Request.Context(), req.BOID, req.Data, tenantID, c.GetString("datasource_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !validation.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "validation failed",
			"validation": validation,
		})
		return
	}

	// 2. Save to database
	// In production: Generate unique ID, store in appropriate table
	recordID := generateUUID()

	// Store form submission
	err = h.storeFormSubmission(c.Request.Context(), tenantID, req.BOID, recordID, req.Data, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"record_id": recordID,
		"status":    "saved",
		"message":   "Form data saved successfully",
	})
}

// ============================================================================
// POST /api/ui/submit
// Validates, saves, and triggers business process
// ============================================================================

type SubmitRequest struct {
	BOID string                 `json:"bo_id" binding:"required"`
	Data map[string]interface{} `json:"data" binding:"required"`
	BPID string                 `json:"bp_id"` // Optional: Business Process ID to trigger
}

// SubmitFormData submits form data for business process execution
// @Summary Submit Form for Approval
// @Description Validate, save, and trigger business process workflow
// @Tags UI
// @Param request body SubmitRequest true "Submit Request"
// @Param tenant_id query string true "Tenant ID"
// @Param datasource_id query string true "Datasource ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /ui/submit [post]
func (h *UIHandler) SubmitFormData(c *gin.Context) {
	var req SubmitRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	datasourceID := c.GetString("datasource_id")
	userID := c.GetString("user_id")

	// 1. Validate
	validation, err := h.uiGenerator.ValidateFormData(c.Request.Context(), req.BOID, req.Data, tenantID, datasourceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !validation.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "validation failed",
			"validation": validation,
		})
		return
	}

	// 2. Save
	recordID := generateUUID()
	err = h.storeFormSubmission(c.Request.Context(), tenantID, req.BOID, recordID, req.Data, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save: %v", err)})
		return
	}

	// 3. Trigger Business Process (if specified)
	workflowID := ""
	if req.BPID != "" {
		workflowID, err = h.triggerBusinessProcess(c.Request.Context(), tenantID, datasourceID, req.BPID, recordID, req.Data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to trigger workflow: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"record_id":   recordID,
		"workflow_id": workflowID,
		"status":      "submitted",
		"message":     "Form submitted successfully",
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (h *UIHandler) storeFormSubmission(
	ctx context.Context,
	tenantID string,
	boID string,
	recordID string,
	data map[string]interface{},
	userID string,
	validationPassed bool,
) error {
	// In production: Store in form_submissions table
	// This is a placeholder implementation

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal form data: %w", err)
	}

	query := `
		INSERT INTO form_submissions (
			tenant_id, bo_id, layout_id, submission_id, submitted_by, 
			form_data, form_data_hash, validation_passed, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	// Calculate hash
	hash := calculateHash(string(dataJSON))

	// Get layout ID from business object
	var layoutID string
	layoutQuery := `SELECT layout_id FROM business_objects WHERE id = $1 AND tenant_id = $2`
	err = h.db.GetContext(ctx, &layoutID, layoutQuery, boID, tenantID)
	if err != nil {
		// Fallback to default if not found
		layoutID = "default_layout"
	}

	_, err = h.db.ExecContext(ctx, query,
		tenantID,
		boID,
		layoutID,
		recordID,
		userID,
		dataJSON,
		hash,
		validationPassed,
		"pending",
	)

	return err
}

func (h *UIHandler) triggerBusinessProcess(
	ctx context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ map[string]interface{},
) (string, error) {
	// Integrate with Temporal workflow engine
	// Load BP definition and start workflow

	// For now, use placeholder until Temporal client is injected
	// In production, this would:
	// 1. Load BP definition from database
	// 2. Call h.temporalClient.ExecuteWorkflow(...)
	// 3. Return actual workflow ID

	workflowID := generateUUID()
	// TODO: Replace with actual Temporal integration once client is available
	// workflowID, err := h.tc.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
	//     ID: workflowID,
	//     TaskQueue: "bp_engine",
	// }, "DynamicBPWorkflow", bpInput)

	return workflowID, nil
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func generateUUID() string {
	// In production: Use github.com/google/uuid
	return "generated-uuid-placeholder"
}

func calculateHash(_ string) string {
	// In production: Calculate SHA-256 hash
	return "hash-placeholder"
}
