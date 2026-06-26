package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/api-gateway/abac"
	"go.temporal.io/sdk/client"
)

func RegisterOptimizeAlphaRoutes(r *gin.Engine, tc client.Client) {
	r.POST("/portfolio/:id/optimize", func(c *gin.Context) {
		if !abac.Evaluate(c, "optimize", "portfolio") {
			c.JSON(403, nil)
			return
		}
		_, err := tc.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{TaskQueue: "alpha"}, "OptimizeAlpha", c.Param("id"))
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to start workflow"})
			return
		}
		c.JSON(202, gin.H{"status": "alpha initiated"})
	})
}
