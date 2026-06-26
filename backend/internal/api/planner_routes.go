package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/backend/internal/planner"
)

// PlannerHandler handles query planning endpoints
type PlannerHandler struct {
	planner *planner.Planner
	store   *planner.Store
}

// NewPlannerHandler creates a new planner handler
func NewPlannerHandler(p *planner.Planner, store *planner.Store) *PlannerHandler {
	return &PlannerHandler{
		planner: p,
		store:   store,
	}
}

// RegisterRoutes registers planner API routes
func (h *PlannerHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")

	// Plan a query
	api.POST("/plan", h.PlanQuery)

	// Get plan details (explain)
	api.GET("/plan/:plan_id/explain", h.ExplainPlan)

	// Get plan details (raw)
	api.GET("/plan/:plan_id", h.GetPlan)

	// Get plans for a semantic target
	api.GET("/plan/target/:semantic_target", h.GetPlansForTarget)

	// Get SLO compliance
	api.GET("/planner/slo/:query_type", h.GetSLO)

	// Get recent plans
	api.GET("/plans", h.GetRecentPlans)
}

// PlanQuery handles POST /api/v1/plan
// @Summary Plan a semantic query
// @Description Generate an execution plan for a semantic query
// @Tags Planner
// @Accept json
// @Produce json
// @Param request body planner.QueryRequest true "Query request"
// @Success 200 {object} planner.QueryPlan
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/plan [post]
func (h *PlannerHandler) PlanQuery(c *gin.Context) {
	var req planner.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request
	if req.QueryType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query_type is required"})
		return
	}
	if req.SemanticTarget == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "semantic_target is required"})
		return
	}

	// Set defaults
	if req.ConsistencyLevel == "" {
		req.ConsistencyLevel = "region_preferred"
	}
	if req.Priority == "" {
		req.Priority = "interactive"
	}

	// Plan the query
	plan, err := h.planner.Plan(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Emit metrics
	c.Set("planner_plan_id", plan.PlanID)
	c.Set("planner_regions", len(plan.SelectedRegions))

	c.JSON(http.StatusOK, plan)
}

// ExplainPlan handles GET /api/v1/plan/:plan_id/explain
// @Summary Get detailed explanation of a query plan
// @Description Retrieve human-readable explanation of planning decisions
// @Tags Planner
// @Produce json
// @Param plan_id path string true "Plan ID"
// @Success 200 {object} planner.ExplainPlan
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/plan/{plan_id}/explain [get]
func (h *PlannerHandler) ExplainPlan(c *gin.Context) {
	planID := c.Param("plan_id")

	explain, err := h.planner.GetExplainPlan(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if explain == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	c.JSON(http.StatusOK, explain)
}

// GetPlan handles GET /api/v1/plan/:plan_id
// @Summary Get plan details
// @Tags Planner
// @Produce json
// @Param plan_id path string true "Plan ID"
// @Success 200 {object} planner.PlannerDecision
// @Router /api/v1/plan/{plan_id} [get]
func (h *PlannerHandler) GetPlan(c *gin.Context) {
	planID := c.Param("plan_id")

	decision, err := h.store.GetDecision(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if decision == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	c.JSON(http.StatusOK, decision)
}

// GetPlansForTarget handles GET /api/v1/plan/target/:semantic_target
// @Summary Get recent plans for a semantic target
// @Tags Planner
// @Produce json
// @Param semantic_target path string true "Semantic target"
// @Param limit query int false "Limit (default 10)"
// @Success 200 {array} planner.PlannerDecision
// @Router /api/v1/plan/target/{semantic_target} [get]
func (h *PlannerHandler) GetPlansForTarget(c *gin.Context) {
	target := c.Param("semantic_target")
	limit := 10

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed < 100 {
			limit = parsed
		}
	}

	decisions, err := h.store.GetDecisionsForTarget(c.Request.Context(), target, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if decisions == nil {
		decisions = []planner.PlannerDecision{}
	}

	c.JSON(http.StatusOK, decisions)
}

// GetSLO handles GET /api/v1/planner/slo/:query_type
// @Summary Get planner SLO compliance for a query type
// @Tags Planner
// @Produce json
// @Param query_type path string true "Query type (feature, metric, ts, drift, importance, discovery)"
// @Param hours path int false "Hours back (default 24)"
// @Success 200 {object} planner.SLOCompliance
// @Router /api/v1/planner/slo/{query_type} [get]
func (h *PlannerHandler) GetSLO(c *gin.Context) {
	queryType := c.Param("query_type")
	hoursBack := 24

	if h := c.Query("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			hoursBack = parsed
		}
	}

	slo, err := h.store.GetSLOCompliance(c.Request.Context(), queryType, hoursBack)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if slo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no SLO data"})
		return
	}

	c.JSON(http.StatusOK, slo)
}

// GetRecentPlans handles GET /api/v1/plans
// @Summary Get recently made planner decisions
// @Tags Planner
// @Produce json
// @Param limit query int false "Limit (default 20)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {array} planner.PlannerDecision
// @Router /api/v1/plans [get]
func (h *PlannerHandler) GetRecentPlans(c *gin.Context) {
	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed < 100 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	decisions, err := h.store.GetRecentDecisions(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if decisions == nil {
		decisions = []planner.PlannerDecision{}
	}

	c.JSON(http.StatusOK, decisions)
}
