package services

import (
	"context"
	"time"

	"calendar-service/internal/database"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type CostTracker struct {
	dbClient *database.Client
	logger   *logrus.Entry
}

func NewCostTracker(db *database.Client, logger *logrus.Entry) *CostTracker {
	return &CostTracker{
		dbClient: db,
		logger:   logger.WithField("component", "cost_tracker"),
	}
}

func (ct *CostTracker) RecordSyncCost(ctx context.Context, tenantID, syncJobID string, apiCalls int, computeSeconds, storageMB, transferMB float64) error {
	apiCallCost := int(float64(apiCalls) * 0.01)
	computeCost := int(computeSeconds * 0.001667)
	storageCost := int(storageMB * 0.000023)
	transferCost := int(transferMB * 0.009)
	totalCost := apiCallCost + computeCost + storageCost + transferCost

	query := `
		INSERT INTO sync_cost_tracking (id, tenant_id, sync_job_id, api_calls, api_call_cost_cents,
			compute_time_seconds, compute_cost_cents, storage_mb, storage_cost_cents,
			data_transfer_mb, data_transfer_cost_cents, total_cost_cents, sync_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := ct.dbClient.Pool().Exec(ctx, query,
		uuid.New(), tenantID, syncJobID, apiCalls, apiCallCost,
		computeSeconds, computeCost, storageMB, storageCost,
		transferMB, transferCost, totalCost, time.Now().Format("2006-01-02"),
	)

	if err != nil {
		ct.logger.WithError(err).Error("Failed to record sync cost")
	}

	return err
}
