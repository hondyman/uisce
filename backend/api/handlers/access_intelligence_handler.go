package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/models"
)

// AccessIntelligenceHandler handles API requests for the unified access intelligence service.
type AccessIntelligenceHandler struct {
	service *services.AccessIntelligenceService
}

// NewAccessIntelligenceHandler creates a new AccessIntelligenceHandler.
func NewAccessIntelligenceHandler(service *services.AccessIntelligenceService) *AccessIntelligenceHandler {
	return &AccessIntelligenceHandler{service: service}
}

// HandleGetEffectiveClaims retrieves all effective claims for a user.
func (h *AccessIntelligenceHandler) HandleGetEffectiveClaims(c *gin.Context) {
	userID := c.Query("user_id")
	tenantID := c.Query("tenant_id")
	if userID == "" || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and tenant_id are required"})
		return
	}
	claims, err := h.service.GetEffectiveClaims(c.Request.Context(), userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get effective claims"})
		return
	}
	c.JSON(http.StatusOK, claims)
}

// HandleGrantClaim grants a claim to a user.
func (h *AccessIntelligenceHandler) HandleGrantClaim(c *gin.Context) {
	var req models.GrantClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	actorID := "current_admin"
	claim, conflict, err := h.service.GrantClaim(c.Request.Context(), req, actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant claim", "details": err.Error()})
		return
	}
	if conflict != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Claim conflict detected", "conflict": conflict})
		return
	}
	c.JSON(http.StatusCreated, claim)
}

// HandleAssignBundle assigns a claim bundle to a user.
func (h *AccessIntelligenceHandler) HandleAssignBundle(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		BundleID string `json:"bundle_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	actorID := "current_admin"
	err := h.service.AssignBundle(c.Request.Context(), req.UserID, req.BundleID, actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign bundle"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "assigned"})
}

// HandleEvaluateAccess performs a real-time access check.
func (h *AccessIntelligenceHandler) HandleEvaluateAccess(c *gin.Context) {
	var req models.EvaluateAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	response, err := h.service.EvaluateAccess(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate access"})
		return
	}
	c.JSON(http.StatusOK, response)
}

// HandleRefreshClaimsCache invalidates a user's claims cache.
func (h *AccessIntelligenceHandler) HandleRefreshClaimsCache(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		TenantID string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	err := h.service.RefreshClaimsCache(c.Request.Context(), req.UserID, req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh cache"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cache_invalidated"})
}

// HandleGetDecisionTrace retrieves a decision trace.
func (h *AccessIntelligenceHandler) HandleGetDecisionTrace(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

// HandleGetDecisionExplanation retrieves a decision explanation.
func (h *AccessIntelligenceHandler) HandleGetDecisionExplanation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

// HandleSimulateAccess performs a what-if access simulation.
func (h *AccessIntelligenceHandler) HandleSimulateAccess(c *gin.Context) {
	var req models.SimulateAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	response, err := h.service.SimulateAccess(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to simulate access"})
		return
	}
	c.JSON(http.StatusOK, response)
}

// HandleGetGovernanceCockpitSnapshot retrieves the snapshot for the governance cockpit.
func (h *AccessIntelligenceHandler) HandleGetGovernanceCockpitSnapshot(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		tenantID = "default_tenant"
	}
	snapshot, err := h.service.GetGovernanceCockpitSnapshot(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get governance cockpit snapshot"})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}
