package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/api-gateway/abac"
	"go.temporal.io/sdk/client"
)

type ScenarioRequest struct {
	Scenario string `json:"scenario" binding:"required"`
}

func RegisterScenarioAnalysisRoutes(r *gin.Engine, tc client.Client) {
	r.POST("/portfolio/:id/scenario", func(c *gin.Context) {
		if !abac.Evaluate(c, "analyze", "portfolio") {
			c.JSON(403, nil)
			return
		}

		var req ScenarioRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		workflowRun, err := tc.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{TaskQueue: "default"}, "ScenarioAnalysis", c.Param("id"), req.Scenario)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to start workflow"})
			return
		}

		var result map[string]any
		err = workflowRun.Get(context.Background(), &result)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get workflow result"})
			return
		}

		c.JSON(200, result)
	})
}
