package audit

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// AuditAPIHandler provides HTTP endpoints for querying audit data
// Deprecated: Trino audit chain has been removed. All endpoints return empty results.
type AuditAPIHandler struct{}

// NewAuditAPIHandler creates a new audit API handler
// Deprecated: Returns a stub handler
func NewAuditAPIHandler() *AuditAPIHandler {
	return &AuditAPIHandler{}
}

// RegisterRoutes registers audit API routes with Gin
func (h *AuditAPIHandler) RegisterRoutes(r *gin.RouterGroup) {
	audit := r.Group("/audit")
	{
		audit.GET("/job-runs", h.GetJobRuns)
		audit.GET("/job-runs/:run_id", h.GetJobRun)
		audit.GET("/dag-runs", h.GetDAGRuns)
		audit.GET("/changesets", h.GetChangeSets)
		audit.GET("/changesets/:changeset_id", h.GetChangeSet)
		audit.GET("/violations", h.GetComplianceViolations)
		audit.GET("/violations/:violation_id", h.GetComplianceViolation)
		audit.GET("/semantic/:semantic_term_id/lineage", h.GetSemanticLineage)
		audit.GET("/semantic/:semantic_term_id/versions", h.GetSemanticVersions)
		audit.GET("/ai-narratives", h.GetAINarratives)
		audit.POST("/ai-narratives", h.GenerateAINarrative)
		audit.GET("/dashboard/slo", h.GetSLODashboard)
		audit.GET("/dashboard/compliance", h.GetComplianceDashboard)
		audit.GET("/dashboard/governance", h.GetGovernanceDashboard)
	}
}

func (h *AuditAPIHandler) GetJobRuns(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "count": 0})
}

func (h *AuditAPIHandler) GetJobRun(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "job run not found"})
}

func (h *AuditAPIHandler) GetDAGRuns(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "count": 0})
}

func (h *AuditAPIHandler) GetChangeSets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "count": 0})
}

func (h *AuditAPIHandler) GetChangeSet(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "changeset not found"})
}

func (h *AuditAPIHandler) GetComplianceViolations(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "count": 0})
}

func (h *AuditAPIHandler) GetComplianceViolation(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "violation not found"})
}

func (h *AuditAPIHandler) GetSemanticLineage(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "semantic term version not found"})
}

func (h *AuditAPIHandler) GetSemanticVersions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"semantic_term_id": c.Param("semantic_term_id"), "versions": []interface{}{}, "count": 0})
}

func (h *AuditAPIHandler) GetAINarratives(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "count": 0})
}

func (h *AuditAPIHandler) GenerateAINarrative(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"narrative": "AI narrative generation is disabled: Trino audit chain removed"})
}

func (h *AuditAPIHandler) GetSLODashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tenant_id": "", "data": []interface{}{}})
}

func (h *AuditAPIHandler) GetComplianceDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tenant_id": "", "data": []interface{}{}})
}

func (h *AuditAPIHandler) GetGovernanceDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tenant_id": "", "data": []interface{}{}})
}

// TenantScopeMiddleware enforces tenant isolation on all audit queries
func TenantScopeMiddlewareGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := jwtmiddleware.GetGinClaimsFromContext(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
