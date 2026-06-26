package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// Attribution Handler
// ============================================================================

type AttributionHandler struct {
	db   *sqlx.DB
	tc   client.Client
	abac interface{} // ABAC engine
}

func NewAttributionHandler(db *sqlx.DB, tc client.Client, abac interface{}) *AttributionHandler {
	return &AttributionHandler{
		db:   db,
		tc:   tc,
		abac: abac,
	}
}

// POST /api/portfolio/:id/attribute - Trigger AI-powered performance attribution
func (h *AttributionHandler) TriggerAttributionAlpha(c *gin.Context) {
	portfolioID := c.Param("id")

	// ABAC Check
	if !h.evaluateABAC(c, "attribute", "portfolio") {
		c.JSON(http.StatusForbidden, gin.H{"error": "ABAC denied"})
		return
	}

	// Start Temporal workflow
	workflowID := "attribution-alpha-" + uuid.New().String()
	_, err := h.tc.ExecuteWorkflow(
		c.Request.Context(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: "alpha",
		},
		"AttributionAlpha",
		portfolioID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "alpha initiated", "workflow_id": workflowID})
}

// evaluateABAC performs ABAC authorization check
func (h *AttributionHandler) evaluateABAC(c *gin.Context, action, resourceType string) bool {
	if h.abac == nil {
		return true
	}

	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	userID := c.GetHeader("X-User-ID")

	if tenantID == "" || userID == "" {
		return false
	}

	roles := c.GetHeader("X-User-Roles")
	return strings.Contains(roles, "admin") || strings.Contains(roles, "advisor")
}
