package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sharedtypes "github.com/hondyman/semlayer/libs/shared-types"
	"github.com/hondyman/semlayer/services/semantic-engine/internal/services"
)

// HandlerConfig holds configuration for API handlers
type HandlerConfig struct {
	SemanticService *services.SemanticService
}

// Handler provides HTTP API endpoints for semantic operations
type Handler struct {
	config HandlerConfig
}

// NewHandler creates a new API handler instance
func NewHandler(config HandlerConfig) *Handler {
	return &Handler{
		config: config,
	}
}

// SetupRoutes configures the HTTP routes for the semantic service
func SetupRoutes(r *gin.Engine, handler *Handler) {
	r.POST("/api/v1/semantic/calculate", handler.CalculateSemanticModel)
	r.GET("/api/v1/semantic/mappings", handler.GetSemanticMappings)
	r.GET("/health", handler.HealthCheck)
}

// CalculateSemanticModel handles semantic model calculation requests
func (h *Handler) CalculateSemanticModel(c *gin.Context) {
	var request sharedtypes.SemanticCalculationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	result, err := h.config.SemanticService.CalculateSemanticModel(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSemanticMappings handles semantic mappings retrieval requests
func (h *Handler) GetSemanticMappings(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	datasourceID := c.Query("datasource_id")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing tenant_id or datasource_id parameters"})
		return
	}

	mappings, err := h.config.SemanticService.GetSemanticMappings(c.Request.Context(), tenantID, datasourceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   mappings,
	})
}

// HealthCheck provides a health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "semantic-engine",
	})
}
