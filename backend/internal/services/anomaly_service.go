package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// AnomalyService provides methods for anomaly detection and retrieval.
type AnomalyService struct {
	db *sqlx.DB
}

// NewAnomalyService creates a new AnomalyService.
func NewAnomalyService(db *sqlx.DB) *AnomalyService {
	return &AnomalyService{db: db}
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
// This is a mock implementation of the engine logic.
func (s *AnomalyService) DetectAnomalies(ctx context.Context, datasourceID, tableName, metric string) error {
	// In a real system, this would fetch time series data, apply statistical models (e.g., Z-score, IQR),
	// and write any found anomalies to the `explorer_anomaly` table.
	logging.GetLogger().Sugar().Infof("Anomaly detection engine ran for metric '%s' on table '%s'", metric, tableName)
	return nil
}
