package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"go.temporal.io/sdk/client"
)

type RebalanceAPI struct {
	temporal client.Client
}

type SimulationParameters struct {
	PortfolioID        string
	StartDate          time.Time
	EndDate            time.Time
	RebalanceFrequency string
}

type SimulationResult struct {
	Success bool        `json:"success"`
	Summary string      `json:"summary,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	// Connect to Temporal using centralized helper (env-driven + retries)
	temporalClient, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatal(err)
	}
	defer temporalClient.Close()

	api := &RebalanceAPI{
		temporal: temporalClient,
	}

	r := gin.Default()
	r.Use(authMiddleware())

	r.POST("/api/portfolio/:id/rebalance", api.TriggerRebalance)
	r.GET("/api/portfolio/:id/rebalance-plans", api.GetRebalancePlans)
	r.GET("/api/rebalance/status/:workflow_id", api.GetWorkflowStatus)
	r.POST("/api/portfolio/:id/simulate", api.TriggerSimulation)
	r.GET("/api/simulation/:workflow_id", api.GetSimulationStatus)
	r.POST("/api/portfolio/:id/risk", api.TriggerRiskWorkflow)
	r.POST("/api/portfolio/:id/attribute", api.TriggerAttributionWorkflow)

	r.Run(":8080")
}

// POST /api/portfolio/:id/simulate
func (api *RebalanceAPI) TriggerSimulation(c *gin.Context) {
	portfolioID := c.Param("id")

	var params struct {
		StartDate          string `json:"start_date" binding:"required"`
		EndDate            string `json:"end_date" binding:"required"`
		RebalanceFrequency string `json:"rebalance_frequency"`
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	startDate, err1 := time.Parse("2006-01-02", params.StartDate)
	endDate, err2 := time.Parse("2006-01-02", params.EndDate)

	if err1 != nil || err2 != nil {
		c.JSON(400, gin.H{"error": "Invalid date format. Use YYYY-MM-DD."})
		return
	}

	wfParams := SimulationParameters{
		PortfolioID:        portfolioID,
		StartDate:          startDate,
		EndDate:            endDate,
		RebalanceFrequency: params.RebalanceFrequency,
	}

	workflowID := fmt.Sprintf("simulate-%s-%d", portfolioID, time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "rebalancing",
	}

	_, err := api.temporal.ExecuteWorkflow(
		c.Request.Context(),
		workflowOptions,
		"SimulateRebalanceWorkflow",
		wfParams,
	)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{
		"workflow_id": workflowID,
		"status":      "initiated",
	})
}

// GET /api/simulation/:workflow_id
func (api *RebalanceAPI) GetSimulationStatus(c *gin.Context) {
	workflowID := c.Param("workflow_id")

	resp, err := api.temporal.DescribeWorkflowExecution(c.Request.Context(), workflowID, "")
	if err != nil {
		c.JSON(404, gin.H{"error": "workflow not found"})
		return
	}

	status := resp.WorkflowExecutionInfo.Status.String()

	if status == "Completed" {
		var result SimulationResult
		err := api.temporal.GetWorkflow(c.Request.Context(), workflowID, "").Get(c.Request.Context(), &result)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to get workflow result"})
			return
		}
		c.JSON(200, gin.H{"status": status, "result": result})
	} else {
		c.JSON(200, gin.H{"status": status})
	}
}

// POST /api/portfolio/:id/rebalance
func (api *RebalanceAPI) TriggerRebalance(c *gin.Context) {
	portfolioID := c.Param("id")
	userID := c.GetString("user_id")
	tenantID := c.GetString("tenant_id")

	// Start Temporal workflow
	workflowID := fmt.Sprintf("rebalance-%s-%d", portfolioID, time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "rebalancing",
	}

	// Pass user context to workflow
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)
	ctx = context.WithValue(ctx, "tenant_id", tenantID)

	workflowRun, err := api.temporal.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"RebalanceAlphaWorkflow",
		portfolioID,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{
		"workflow_id": workflowRun.GetID(),
		"status":      "initiated",
	})
}

// GET /api/portfolio/:id/rebalance-plans
func (api *RebalanceAPI) GetRebalancePlans(c *gin.Context) {
	portfolioID := c.Param("id")

	// Query handled by Hasura GraphQL subscription
	c.JSON(200, gin.H{
		"message": "Use Hasura GraphQL subscription for real-time plans",
		"query":   fmt.Sprintf("subscription { rebalance_plans(where: {portfolio_id: {_eq: \"%s\"}}) { ... } }", portfolioID),
	})
}

// GET /api/rebalance/status/:workflow_id
func (api *RebalanceAPI) GetWorkflowStatus(c *gin.Context) {
	workflowID := c.Param("workflow_id")

	// Query workflow status
	describeResp, err := api.temporal.DescribeWorkflowExecution(
		c.Request.Context(),
		workflowID,
		"",
	)
	if err != nil {
		c.JSON(404, gin.H{"error": "workflow not found"})
		return
	}

	c.JSON(200, gin.H{
		"workflow_id": workflowID,
		"status":      describeResp.WorkflowExecutionInfo.Status.String(),
	})
}

// POST /api/portfolio/:id/risk
func (api *RebalanceAPI) TriggerRiskWorkflow(c *gin.Context) {
	if !abacEvaluate(c, "risk", "portfolio") { // ABAC check placeholder
		c.JSON(403, gin.H{"error": "forbidden"})
		return
	}

	portfolioID := c.Param("id")
	workflowID := fmt.Sprintf("risk-%s-%d", portfolioID, time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "rebalancing", // Using the same task queue
	}

	_, err := api.temporal.ExecuteWorkflow(
		c.Request.Context(),
		workflowOptions,
		"RiskAlphaWorkflow",
		portfolioID,
	)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"status": "risk analysis initiated", "workflow_id": workflowID})
}

// POST /api/portfolio/:id/attribute
func (api *RebalanceAPI) TriggerAttributionWorkflow(c *gin.Context) {
	if !abacEvaluate(c, "attribute", "portfolio") { // ABAC check placeholder
		c.JSON(403, gin.H{"error": "forbidden"})
		return
	}

	portfolioID := c.Param("id")
	workflowID := fmt.Sprintf("attr-%s-%d", portfolioID, time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "rebalancing",
	}

	_, err := api.temporal.ExecuteWorkflow(
		c.Request.Context(),
		workflowOptions,
		"AttributionAlphaWorkflow",
		portfolioID,
	)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"status": "attribution initiated", "workflow_id": workflowID})
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract JWT from Authorization header
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		// Verify JWT and extract claims (implement your JWT logic)
		userID, tenantID := verifyJWT(token)

		c.Set("user_id", userID)
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

func verifyJWT(_ string) (string, string) {
	// Implement JWT verification
	return "user_123", "tenant_abc"
}

// Placeholder for ABAC check
func abacEvaluate(_ *gin.Context, _, _ string) bool {
	// In a real implementation, this would check user roles and policies.
	return true
}
