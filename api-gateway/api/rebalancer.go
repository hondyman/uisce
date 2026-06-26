package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/api-gateway/abac"
	"go.temporal.io/sdk/client"
)

type RebalancePlan struct {
	PortfolioID   string  `json:"portfolioId" binding:"required"`
	CurrentDrift  string  `json:"currentDrift" binding:"required"`
	ExpectedDrift string  `json:"expectedDrift" binding:"required"`
	TaxSavings    string  `json:"taxSavings" binding:"required"`
	Rationale     string  `json:"rationale"`
	Trades        []Trade `json:"trades"`
	Confidence    float64 `json:"confidence"`
}

type Trade struct {
	Action string  `json:"action" binding:"required"`
	Symbol string  `json:"symbol" binding:"required"`
	Shares int     `json:"shares" binding:"required"`
	Value  float64 `json:"value" binding:"required"`
}

func RegisterRebalancerRoutes(r *gin.Engine, tc client.Client) {
	// Execute UMA rebalancing workflow
	r.POST("/portfolio/:id/rebalance", func(c *gin.Context) {
		if !abac.Evaluate(c, "rebalance", "portfolio") {
			c.JSON(403, nil)
			return
		}

		var req RebalancePlan
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid rebalance plan"})
			return
		}

		workflowRun, err := tc.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{TaskQueue: "default"}, "UMAAlpha", req.PortfolioID, req)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to start rebalancing workflow"})
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

	// Get rebalancer portfolio list
	r.GET("/rebalancer/portfolios", func(c *gin.Context) {
		if !abac.Evaluate(c, "read", "portfolio") {
			c.JSON(403, nil)
			return
		}

		// Return mock portfolio list - in production this queries the database
		portfolios := []map[string]interface{}{
			{
				"id":             "port-1",
				"clientId":       "client-1",
				"clientName":     "James Howlett",
				"aum":            2500000,
				"drift":          8.5,
				"holdings":       42,
				"status":         "high-drift",
				"lastRebalanced": "Mar 15, 2024",
				"taxSaved":       12000,
			},
		}

		c.JSON(200, gin.H{"portfolios": portfolios})
	})

	// Propose AI rebalance plan
	r.POST("/portfolio/:id/propose-rebalance", func(c *gin.Context) {
		if !abac.Evaluate(c, "analyze", "portfolio") {
			c.JSON(403, nil)
			return
		}

		// In production, this would fetch portfolio data and call AI optimization
		// For now, return mock proposal
		proposal := map[string]interface{}{
			"portfolioId":   c.Param("id"),
			"currentDrift":  8.5,
			"expectedDrift": 0.5,
			"taxSavings":    1200,
			"confidence":    0.95,
			"rationale":     "Rebalancing to reduce overweight exposure in the tech sector, capitalizing on tax-loss harvesting opportunities.",
			"trades": []Trade{
				{Action: "SELL", Symbol: "AAPL", Shares: 150, Value: 25500},
				{Action: "BUY", Symbol: "MSFT", Shares: 60, Value: 24000},
			},
		}

		c.JSON(200, proposal)
	})
}
