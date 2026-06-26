package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/backend/internal/bundles"
	"github.com/hondyman/semlayer/backend/internal/logging"
)

func main() {
	r := gin.Default()
	// Initialize DB if available (reads POLICY_DB_URL or ../config.yaml)
	if _, err := bundles.InitDBFromConfig("../config.yaml"); err != nil {
		logging.GetLogger().Sugar().Warnf("Warning: failed to init DB for bundles PoC: %v", err)
	}
	api := r.Group("/api")
	api.POST("/bundles/generate", gin.WrapF(bundles.GenerateBundlesHandler))
	api.POST("/bundles/analyze", gin.WrapF(bundles.TriggerAnalyzeHandler))
	api.GET("/bundles/proposals", gin.WrapF(bundles.ListProposalsHandler))

	api.GET("/bundles/candidates", func(c *gin.Context) {
		// return persisted candidates when DB is configured, otherwise fall back to in-memory miner
		db, _ := bundles.InitDBFromConfig("../config.yaml")
		if db != nil {
			list, err := bundles.ListPersistedCandidates(db, "t1", 100)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"candidates": list})
				return
			}
		}
		c.Request.Header.Set("Content-Type", "application/json")
		c.JSON(http.StatusOK, gin.H{"candidates": bundles.MineCandidates(bundles.SampleEnts(), bundles.SampleEvents(), "t1", 1)})
	})

	// Proposal action endpoints (approve/apply/reject)
	api.POST("/bundles/proposals/:id/approve", gin.WrapF(bundles.ApproveProposalHandler))
	api.POST("/bundles/proposals/:id/apply", gin.WrapF(bundles.ApplyProposalHandler))
	api.POST("/bundles/proposals/:id/reject", gin.WrapF(bundles.RejectProposalHandler))

	// Admin guardrail CRUD
	admin := api.Group("/admin")
	// apply simple admin auth middleware (gin variant)
	admin.Use(func(c *gin.Context) {
		// allow health/read operations
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}
		// allow if header role is admin
		if strings.ToLower(c.Request.Header.Get("X-User-Role")) == "admin" {
			c.Next()
			return
		}
		// allow if Authorization Bearer token matches env ADMIN_API_KEY
		authHeader := c.Request.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if token != "" && os.Getenv("ADMIN_API_KEY") != "" && token == os.Getenv("ADMIN_API_KEY") {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
	})
	admin.POST("/guardrails", gin.WrapF(bundles.CreateGuardrailHandler))
	admin.GET("/guardrails", gin.WrapF(bundles.ListGuardrailsHandler))
	admin.GET("/guardrails/load", gin.WrapF(bundles.ForceLoadGuardrailsHandler))
	admin.PUT("/guardrails/:id", gin.WrapF(bundles.UpdateGuardrailHandler))
	admin.DELETE("/guardrails/:id", gin.WrapF(bundles.DeleteGuardrailHandler))
	// Reload in-memory guardrail cache
	admin.POST("/guardrails/reload", gin.WrapF(bundles.ReloadGuardrailsHandler))
	admin.GET("/guardrails/cache", gin.WrapF(bundles.GetGuardrailsCacheHandler))

	logging.GetLogger().Sugar().Info("starting bundles PoC server at :8085")
	if err := r.Run(":8085"); err != nil {
		logging.GetLogger().Sugar().Fatal(err)
	}
}
