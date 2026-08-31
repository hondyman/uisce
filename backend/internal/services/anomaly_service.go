package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/platform_intelligence/exceptions"
	"github.com/hondyman/uisce/backend/models"
	"github.com/jmoiron/sqlx"
)

// AnomalyService provides methods for anomaly detection and retrieval.
type AnomalyService struct {
	db           *sqlx.DB
	exceptionHub *exceptions.ExceptionAggregator
}

// NewAnomalyService creates a new AnomalyService.
func NewAnomalyService(db *sqlx.DB) *AnomalyService {
	return &AnomalyService{db: db}
}

// SetExceptionHub wires this service into the shared platform exception hub
// so anomalies it detects also flow into the tenant-isolated exceptions
// console (dedup, autofix policy, AI assist all apply from there on).
func (s *AnomalyService) SetExceptionHub(hub *exceptions.ExceptionAggregator) {
	s.exceptionHub = hub
}

// ListAnomalies retrieves detected anomalies based on filters.
// This is a mock implementation.
func (s *AnomalyService) ListAnomalies(ctx context.Context, datasourceID, metric string) ([]models.Anomaly, error) {
	// In a real implementation, this would query the `explorer_anomaly` table.
	// For now, we return mock data if the metric is 'churn_rate'.
	if metric != "churn_rate" {
		return []models.Anomaly{}, nil
	}

	expectedRange, _ := json.Marshal(map[string]float64{"min": 4.5, "max": 5.5})

	return []models.Anomaly{
		{
			ID:            uuid.New(),
			DatasourceID:  datasourceID,
			TableName:     "subscriptions",
			Metric:        "churn_rate",
			TimeGrain:     "daily",
			Timestamp:     time.Now().AddDate(0, 0, -3),
			Value:         7.2,
			ExpectedRange: expectedRange,
			AnomalyType:   "spike",
			Severity:      "high",
			Explanation:   "Daily churn rate of 7.2% is significantly higher than the expected range of 4.5-5.5% (3.5 standard deviations above mean).",
			DetectedAt:    time.Now().AddDate(0, 0, -2),
		},
	}, nil
}

// DetectAnomalies runs the detection engine for a given metric and time range.
// This is a mock implementation of the engine logic. tenantID scopes the
// resulting exception into the tenant-isolated platform exception hub; pass
// uuid.Nil to skip publishing (e.g. for anonymous/local runs).
func (s *AnomalyService) DetectAnomalies(ctx context.Context, tenantID uuid.UUID, datasourceID, tableName, metric string) error {
	// In a real system, this would fetch time series data, apply statistical models (e.g., Z-score, IQR),
	// and write any found anomalies to the `explorer_anomaly` table.
	logging.GetLogger().Sugar().Infof("Anomaly detection engine ran for metric '%s' on table '%s'", metric, tableName)

	// Wire into the shared platform exception hub so this detector's
	// findings dedup/auto-fix/AI-assist alongside every other source. The
	// statistical model itself is unchanged (still a stub) — this only
	// adds the publish call the plan asks for.
	if s.exceptionHub != nil && tenantID != uuid.Nil {
		_, err := s.exceptionHub.Publish(ctx, exceptions.Exception{
			TenantID:    tenantID,
			Type:        exceptions.ExceptionDataQuality,
			Severity:    "medium",
			Source:      "table:" + tableName,
			Description: "Anomaly detection engine ran for metric '" + metric + "' on table '" + tableName + "'",
			Evidence:    []string{"datasource:" + datasourceID, "metric:" + metric},
		})
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to publish exception hub signal: %v", err)
		}
	}
	return nil
}
