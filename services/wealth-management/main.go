package main

import (
	"log"

	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"github.com/hondyman/semlayer/services/wealth-management/activities"
	"github.com/hondyman/semlayer/services/wealth-management/workflows"
	"go.temporal.io/sdk/worker"
)

func main() {
	log.Println("🚀 Starting Wealth Management Service...")

	// Create Temporal client using centralized helper (env-driven + retries)
	c, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, "wealth-management", worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.UMAAlpha)
	w.RegisterWorkflow(workflows.AttributionAlpha)
	w.RegisterWorkflow(workflows.TaxHarvest)
	w.RegisterWorkflow(workflows.IndexAlpha)

	// Register activities
	w.RegisterActivity(activities.ABACCheck)
	w.RegisterActivity(activities.ExecuteTrades)
	w.RegisterActivity(activities.HasuraUpdate)
	w.RegisterActivity(activities.AITaxHarvest)
	w.RegisterActivity(activities.AIAttribution)
	w.RegisterActivity(activities.AIIndexOptimize)

	log.Println("✅ Wealth Management Service started successfully")
	log.Println("📊 Registered workflows: UMA Alpha, Attribution Alpha, Tax Harvest, Direct Indexing Alpha")

	// Start worker
	err = w.Start()
	if err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}
}
