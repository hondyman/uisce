package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// UMA Handler
// ============================================================================

type UMAHandler struct {
	db   *sqlx.DB
	tc   client.Client
	abac interface{} // ABAC engine
}

func NewUMAHandler(db *sqlx.DB, tc client.Client, abac interface{}) *UMAHandler {
	return &UMAHandler{
		db:   db,
		tc:   tc,
		abac: abac,
	}
}

// POST /api/uma/:id/alpha - Trigger AI-powered UMA rebalance
func (h *UMAHandler) TriggerUMAAlpha(c *gin.Context) {
	umaID := c.Param("id")

	// ABAC Check
	if !h.evaluateABAC(c, "alpha", "uma") {
		c.JSON(http.StatusForbidden, gin.H{"error": "ABAC denied"})
		return
	}

	// Start Temporal workflow
	workflowID := "uma-alpha-" + uuid.New().String()
	_, err := h.tc.ExecuteWorkflow(
		c.Request.Context(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: "alpha",
		},
		"UMAAlpha",
		umaID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "alpha initiated", "workflow_id": workflowID})
}

// evaluateABAC performs ABAC authorization check
func (h *UMAHandler) evaluateABAC(c *gin.Context, action, resourceType string) bool {
	if h.abac == nil {
		env := os.Getenv("ENVIRONMENT")
		if env == "production" || env == "prod" {
			return false // Fail closed in production
		}
		// If no ABAC service configured, default to allow (for development)
		return true
	}

	// Extract user context
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	userID := c.GetHeader("X-User-ID")

	if tenantID == "" || userID == "" {
		// Missing required headers
		return false
	}

	// Build ABAC evaluation request
	request := map[string]string{
		"action":       action,
		"resourceType": resourceType,
		"tenantID":     tenantID,
		"userID":       userID,
	}
	_ = request // TODO: Use when calling actual ABAC service

	// For now, using simple role-based check
	roles := c.GetHeader("X-User-Roles")
	if roles == "" {
		return false
	}

	// Allow if user has 'admin' or 'advisor' role
	return strings.Contains(roles, "admin") || strings.Contains(roles, "advisor")
}
