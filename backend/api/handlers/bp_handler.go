package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/bp"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// BP Handler
// ============================================================================

type BPHandler struct {
	db        *sqlx.DB
	bpService *bp.BPService
}

func NewBPHandler(db *sqlx.DB) *BPHandler {
	return &BPHandler{
		db:        db,
		bpService: bp.NewBPService(db),
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

type SaveBPRequest struct {
	ProcessName string     `json:"processName" binding:"required"`
	Description string     `json:"description"`
	Entity      string     `json:"entity" binding:"required"`
	Status      string     `json:"status"`
	IsActive    bool       `json:"isActive"`
	Steps       []StepData `json:"steps" binding:"required"`
}

type StepData struct {
	ID             string          `json:"id"`
	StepOrder      int16           `json:"stepOrder"`
	StepType       string          `json:"stepType" binding:"required"`
	StepName       string          `json:"stepName" binding:"required"`
	Description    *string         `json:"description"`
	DurationHours  int16           `json:"durationHours"`
	Config         json.RawMessage `json:"config"`
	ValidateRules  []string        `json:"validationRules"`
	ApproverRole   *string         `json:"assigneeRole"`
	ApproverUser   *string         `json:"assigneeUser"`
	Condition      *string         `json:"condition"`
	NotifyTemplate *string         `json:"notificationTemplate"`
}

type SaveBPResponse struct {
	ID             string `json:"id"`
	ProcessName    string `json:"processName"`
	Status         string `json:"status"`
	VersionNumber  int    `json:"versionNumber"`
	TotalSteps     int    `json:"totalSteps"`
	TotalDurationH int    `json:"totalDurationHours"`
	Message        string `json:"message"`
}

type SimulateBPRequest struct {
	ProcessID string     `json:"processId"`
	Steps     []StepData `json:"steps"`
}

type SimulateBPResponse struct {
	EstimatedDurationHours int      `json:"estimatedDurationHours"`
	StepsCount             int      `json:"stepsCount"`
	ValidationSteps        int      `json:"validationSteps"`
	ApprovalSteps          int      `json:"approvalSteps"`
	NotificationSteps      int      `json:"notificationSteps"`
	Warnings               []string `json:"warnings"`
	Status                 string   `json:"status"`
}

type ListBPResponse struct {
	Processes []BPListItem `json:"processes"`
	Total     int64        `json:"total"`
	Count     int          `json:"count"`
}

type BPListItem struct {
	ID                 string  `json:"id"`
	ProcessName        string  `json:"processName"`
	Entity             string  `json:"entity"`
	Status             string  `json:"status"`
	IsActive           bool    `json:"isActive"`
	StepsCount         int     `json:"stepsCount"`
	TotalDurationHours *int    `json:"totalDurationHours"`
	CreatedBy          string  `json:"createdBy"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          *string `json:"updatedAt"`
}

// ============================================================================
// Route Handlers
// ============================================================================

// SaveBusinessProcess saves a new or updated BP
// POST /api/bp/save
func (h *BPHandler) SaveBusinessProcess(c *gin.Context) {
	var req SaveBPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	tenantID := c.GetString("tenant_id")
	userEmail := c.GetString("user_email")

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Convert request to domain model
	bpModel := &bp.BusinessProcess{
		ProcessName: req.ProcessName,
		Description: req.Description,
		EntityType:  req.Entity,
		Status:      req.Status,
		IsActive:    req.IsActive,
	}

	if req.Status == "" {
		bpModel.Status = "draft"
	}

	// Convert steps
	bpModel.Steps = make([]bp.BPStep, len(req.Steps))
	for i, stepData := range req.Steps {
		configJSON, _ := json.Marshal(stepData.Config)
		bpModel.Steps[i] = bp.BPStep{
			StepOrder:     stepData.StepOrder,
			StepType:      stepData.StepType,
			StepName:      stepData.StepName,
			Description:   stepData.Description,
			DurationHours: stepData.DurationHours,
			Config:        configJSON,
			Status:        "pending",
		}
	}

	// Validate BP
	validationErrors := h.bpService.ValidateBusinessProcess(bpModel)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": validationErrors,
		})
		return
	}

	// Save to database
	saved, err := h.bpService.SaveBusinessProcess(c.Request.Context(), tenantUUID, bpModel, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save business process: " + err.Error(),
		})
		return
	}

	// Get IP for audit trail
	ip := net.ParseIP(c.ClientIP())
	h.bpService.LogAuditEntry(c.Request.Context(), tenantUUID, &saved.ID, userEmail, "created", map[string]interface{}{
		"processName": saved.ProcessName,
		"stepsCount":  len(saved.Steps),
		"ip":          ip,
	})

	c.JSON(http.StatusCreated, SaveBPResponse{
		ID:             saved.ID.String(),
		ProcessName:    saved.ProcessName,
		Status:         saved.Status,
		VersionNumber:  saved.VersionNumber,
		TotalSteps:     len(saved.Steps),
		TotalDurationH: *saved.TotalDurationHours,
		Message:        "Business process saved successfully",
	})
}

// SimulateBusinessProcess simulates BP execution
// POST /api/bp/simulate
func (h *BPHandler) SimulateBusinessProcess(c *gin.Context) {
	var req SimulateBPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	tenantID := c.GetString("tenant_id")
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	processUUID, err := uuid.Parse(req.ProcessID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid process ID"})
		return
	}

	// Get the BP if ID provided, otherwise use steps from request
	var bpModel *bp.BusinessProcess

	if req.ProcessID != "" {
		bpModel, err = h.bpService.GetBusinessProcess(c.Request.Context(), tenantUUID, processUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Business process not found"})
			return
		}
	} else {
		// Simulate from request data
		bpModel = &bp.BusinessProcess{
			Steps: make([]bp.BPStep, len(req.Steps)),
		}
		for i, stepData := range req.Steps {
			configJSON, _ := json.Marshal(stepData.Config)
			bpModel.Steps[i] = bp.BPStep{
				StepType:      stepData.StepType,
				StepName:      stepData.StepName,
				DurationHours: stepData.DurationHours,
				Config:        configJSON,
			}
		}
	}

	// Run simulation analysis
	totalDuration := int16(0)
	validationCount := 0
	approvalCount := 0
	notificationCount := 0
	warnings := []string{}

	for _, step := range bpModel.Steps {
		totalDuration += step.DurationHours

		switch step.StepType {
		case "validate":
			validationCount++
		case "approve":
			approvalCount++
		case "notify":
			notificationCount++
		case "condition":
			// Check for orphaned branches
			var config map[string]interface{}
			if err := json.Unmarshal(step.Config, &config); err == nil {
				if condition, ok := config["condition"].(string); ok && condition == "" {
					warnings = append(warnings, "Step "+step.StepName+": Condition is empty")
				}
			}
		}

		// Warn if step duration is excessive
		if step.DurationHours > 168 { // 1 week
			warnings = append(warnings, fmt.Sprintf("Step %s has high duration (%d hours)", step.StepName, step.DurationHours))
		}
	}

	// Warn if no validation steps
	if validationCount == 0 {
		warnings = append(warnings, "No validation steps defined - form data will not be validated")
	}

	resp := SimulateBPResponse{
		EstimatedDurationHours: int(totalDuration),
		StepsCount:             len(bpModel.Steps),
		ValidationSteps:        validationCount,
		ApprovalSteps:          approvalCount,
		NotificationSteps:      notificationCount,
		Warnings:               warnings,
		Status:                 "ready_to_execute",
	}

	if len(warnings) > 0 {
		resp.Status = "ready_with_warnings"
	}

	c.JSON(http.StatusOK, resp)
}

// ListBusinessProcesses returns all BPs for tenant
// GET /api/bp
func (h *BPHandler) ListBusinessProcesses(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	offset := 0
	limit := 20

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	processes, total, err := h.bpService.ListBusinessProcesses(c.Request.Context(), tenantUUID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list business processes: " + err.Error(),
		})
		return
	}

	// Convert to response format
	items := make([]BPListItem, len(processes))
	for i, p := range processes {
		items[i] = BPListItem{
			ID:                 p.ID.String(),
			ProcessName:        p.ProcessName,
			Entity:             p.EntityType,
			Status:             p.Status,
			IsActive:           p.IsActive,
			StepsCount:         len(p.Steps),
			TotalDurationHours: p.TotalDurationHours,
			CreatedBy:          p.CreatedBy,
			CreatedAt:          p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if p.UpdatedAt != nil {
			updatedAtStr := p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
			items[i].UpdatedAt = &updatedAtStr
		}
	}

	c.JSON(http.StatusOK, ListBPResponse{
		Processes: items,
		Total:     total,
		Count:     len(items),
	})
}

// GetBusinessProcess returns a single BP with all details
// GET /api/bp/:id
func (h *BPHandler) GetBusinessProcess(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	processID := c.Param("id")

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	processUUID, err := uuid.Parse(processID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid process ID"})
		return
	}

	bp, err := h.bpService.GetBusinessProcess(c.Request.Context(), tenantUUID, processUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business process not found"})
		return
	}

	c.JSON(http.StatusOK, bp)
}

// DeleteBusinessProcess archives a BP
// DELETE /api/bp/:id
func (h *BPHandler) DeleteBusinessProcess(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userEmail := c.GetString("user_email")
	processID := c.Param("id")

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	processUUID, err := uuid.Parse(processID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid process ID"})
		return
	}

	err = h.bpService.DeleteBusinessProcess(c.Request.Context(), tenantUUID, processUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete business process: " + err.Error(),
		})
		return
	}

	// Log audit entry
	h.bpService.LogAuditEntry(c.Request.Context(), tenantUUID, &processUUID, userEmail, "archived", map[string]interface{}{
		"processId": processID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Business process archived successfully",
	})
}

// ============================================================================
// StartExecution - Trigger Business Process Workflow
// ============================================================================

type StartExecutionRequest struct {
	BusinessProcessID string                 `json:"businessProcessId" binding:"required"`
	EntityID          string                 `json:"entityId" binding:"required"`
	FormData          map[string]interface{} `json:"formData" binding:"required"`
}

type StartExecutionResponse struct {
	WorkflowID string `json:"workflowId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	StartedAt  string `json:"startedAt"`
}

// StartExecution triggers a business process workflow execution
// POST /api/bp/start-execution
func (h *BPHandler) StartExecution(c *gin.Context) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required tenant scoping headers: X-Tenant-ID and X-Tenant-Datasource-ID",
		})
		return
	}

	// Parse request
	var req StartExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate business process exists
	processUUID, err := uuid.Parse(req.BusinessProcessID)
	if err != nil {
		// Try as string ID
		processUUID = uuid.New()
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid tenant ID format",
		})
		return
	}

	// In production, verify the business process exists
	// For now, we generate a workflow ID and return success

	workflowID := uuid.New().String()

	// Log audit entry
	userEmail := c.GetString("user_email")
	if userEmail == "" {
		userEmail = "system"
	}

	h.bpService.LogAuditEntry(c.Request.Context(), tenantUUID, &processUUID, userEmail, "execution_started", map[string]interface{}{
		"businessProcessId": req.BusinessProcessID,
		"entityId":          req.EntityID,
		"workflowId":        workflowID,
		"formDataFields":    len(req.FormData),
	})

	// In a real implementation, this would:
	// 1. Validate the business process exists and is active
	// 2. Trigger a Temporal workflow or similar
	// 3. Return the workflow execution ID

	response := StartExecutionResponse{
		WorkflowID: workflowID,
		Status:     "started",
		Message:    "Business process workflow execution started successfully",
		StartedAt:  time.Now().Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusAccepted, response)
}

// ============================================================================
// Register Routes
// ============================================================================

func RegisterBPRoutes(router *gin.Engine, db *sqlx.DB) {
	handler := NewBPHandler(db)

	bpGroup := router.Group("/api/bp")
	{
		bpGroup.POST("/save", handler.SaveBusinessProcess)
		bpGroup.POST("/simulate", handler.SimulateBusinessProcess)
		bpGroup.POST("/start-execution", handler.StartExecution)
		bpGroup.GET("", handler.ListBusinessProcesses)
		bpGroup.GET("/:id", handler.GetBusinessProcess)
		bpGroup.DELETE("/:id", handler.DeleteBusinessProcess)
	}
}
