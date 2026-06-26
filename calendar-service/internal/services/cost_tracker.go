package services

import (
	"context"
	"time"

	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

// CostTracker monitors and records sync costs
type CostTracker struct {
	hasuraClient *hasura.Client
	logger       *logrus.Entry
}

// NewCostTracker creates a new cost tracker
func NewCostTracker(hasuraClient *hasura.Client, logger *logrus.Entry) *CostTracker {
	return &CostTracker{
		hasuraClient: hasuraClient,
		logger:       logger.WithField("component", "cost_tracker"),
	}
}

// RecordSyncCost logs the cost of a completed sync job
func (ct *CostTracker) RecordSyncCost(ctx context.Context, tenantID, syncJobID string, apiCalls int, computeSeconds, storageMB, transferMB float64) error {
	// Calculate actual costs (mock logic for demo)
	apiCallCost := int(float64(apiCalls) * 0.01)
	computeCost := int(computeSeconds * 0.001667)
	storageCost := int(storageMB * 0.000023)
	transferCost := int(transferMB * 0.009)
	totalCost := apiCallCost + computeCost + storageCost + transferCost

	mutation := `
	mutation RecordCost($input: sync_cost_tracking_insert_input!) {
		insert_sync_cost_tracking_one(object: $input) {
			id
		}
	}
	`

	input := map[string]interface{}{
		"tenant_id":                tenantID,
		"sync_job_id":              syncJobID,
		"api_calls":                apiCalls,
		"api_call_cost_cents":      apiCallCost,
		"compute_time_seconds":     computeSeconds,
		"compute_cost_cents":       computeCost,
		"storage_mb":               storageMB,
		"storage_cost_cents":       storageCost,
		"data_transfer_mb":         transferMB,
		"data_transfer_cost_cents": transferCost,
		"total_cost_cents":         totalCost,
		"sync_date":                time.Now().Format("2006-01-02"),
	}

	return ct.hasuraClient.Mutate(ctx, mutation, map[string]interface{}{
		"input": input,
	}, &struct{}{})
}
