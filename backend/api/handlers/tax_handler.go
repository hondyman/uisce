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
// Tax Handler
// ============================================================================

type TaxHandler struct {
	db   *sqlx.DB
	tc   client.Client
	abac interface{} // ABAC engine
}

func NewTaxHandler(db *sqlx.DB, tc client.Client, abac interface{}) *TaxHandler {
	return &TaxHandler{
		db:   db,
		tc:   tc,
		abac: abac,
	}
}

// POST /api/uma/{id}/tax - Trigger AI-powered tax optimization
func (h *TaxHandler) TriggerTaxHarvest(c *gin.Context) {
	umaID := c.Param("id")

	// ABAC Check
	if !h.evaluateABAC(c, "tax", "uma") {
		c.JSON(http.StatusForbidden, gin.H{"error": "ABAC denied"})
		return
	}

	// Start Temporal workflow
	workflowID := "tax-harvest-" + uuid.New().String()
	_, err := h.tc.ExecuteWorkflow(
		c.Request.Context(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: "tax",
		},
		"TaxHarvest",
		umaID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":      "tax optimization initiated",
		"workflow_id": workflowID,
		"uma_id":      umaID,
	})
}

// evaluateABAC performs ABAC authorization check
func (h *TaxHandler) evaluateABAC(c *gin.Context, action, resourceType string) bool {
	if h.abac == nil {
		env := os.Getenv("ENVIRONMENT")
		if env == "production" || env == "prod" {
			return false // Fail closed in production for security
		}
		return true // Development mode
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
