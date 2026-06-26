package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// AutomationHandler handles API requests for the automation engine.
type AutomationHandler struct {
	service *services.AutomationService
}

// NewAutomationHandler creates a new AutomationHandler.
func NewAutomationHandler(service *services.AutomationService) *AutomationHandler {
	return &AutomationHandler{service: service}
}

// HandleRunAutomationCycle manually triggers an automation cycle.
func (h *AutomationHandler) HandleRunAutomationCycle(c *gin.Context) {
	actorID := "current_admin"
	logs, err := h.service.RunAutomationCycle(c.Request.Context(), actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run automation cycle"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "completed", "logs": logs})
}

// HandleListAutomationLogs retrieves recent automation logs.
func (h *AutomationHandler) HandleListAutomationLogs(c *gin.Context) {
	logs, err := h.service.ListAutomationLogs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list automation logs"})
		return
	}
	c.JSON(http.StatusOK, logs)
}

// HandlePauseAutomation pauses the automation engine.
func (h *AutomationHandler) HandlePauseAutomation(c *gin.Context) {
	err := h.service.PauseAutomation(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pause automation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "paused"})
}

// HandleResumeAutomation resumes the automation engine.
func (h *AutomationHandler) HandleResumeAutomation(c *gin.Context) {
	err := h.service.ResumeAutomation(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resume automation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "resumed"})
}

// HandleListAutomationPolicies lists all automation policies.
func (h *AutomationHandler) HandleListAutomationPolicies(c *gin.Context) {
	policies, err := h.service.ListAutomationPolicies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list automation policies"})
		return
	}
	c.JSON(http.StatusOK, policies)
}
