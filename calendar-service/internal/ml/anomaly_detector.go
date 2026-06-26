package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

// AnomalyDetector identifies anomalies in calendar syncs and system performance
type AnomalyDetector struct {
	hasuraClient *hasura.Client
	alertService AnomalyAlertService
	logger       *logrus.Entry
	config       AnomalyDetectorConfig
}

// AnomalyDetectorConfig holds configuration
type AnomalyDetectorConfig struct {
	HasuraClient  *hasura.Client
	AlertService  AnomalyAlertService
	Logger        *logrus.Entry
	CheckInterval time.Duration
}

// AnomalyAlertService interface for dependency injection
type AnomalyAlertService interface {
	TriggerAlert(ctx context.Context, anomalyID string, anomalyType string, severity string, description string) error
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(cfg AnomalyDetectorConfig) *AnomalyDetector {
	return &AnomalyDetector{
		hasuraClient: cfg.HasuraClient,
		alertService: cfg.AlertService,
		logger:       cfg.Logger.WithField("component", "anomaly_detector"),
		config:       cfg,
	}
}

// StartMonitoring starts the anomaly detection background process
func (ad *AnomalyDetector) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(ad.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ad.logger.Info("Stopping anomaly detector")
			return
		case <-ticker.C:
			ad.runChecks(ctx)
		}
	}
}

// runChecks runs all anomaly detection checks
func (ad *AnomalyDetector) runChecks(ctx context.Context) {
	ad.logger.Debug("Running anomaly detection checks")

	// Check for sync failure spikes
	if err := ad.checkSyncFailureSpikes(ctx); err != nil {
		ad.logger.WithError(err).Error("Failed to check sync failure spikes")
	}

	// Check for API quota usage anomalies
	if err := ad.checkAPIQuotaAnomalies(ctx); err != nil {
		ad.logger.WithError(err).Error("Failed to check API quota anomalies")
	}

	// Check for latency spikes
	if err := ad.checkLatencySpikes(ctx); err != nil {
		ad.logger.WithError(err).Error("Failed to check latency spikes")
	}
}

func (ad *AnomalyDetector) checkSyncFailureSpikes(ctx context.Context) error {
	// Query recent sync jobs (last 15 mins) and compare failure rate to threshold
	query := `
	query GetRecentSyncStats {
		sync_jobs_aggregate(where: {
			created_at: {_gt: $time_threshold}
		}) {
			aggregate {
				count
			}
		}
		failed_syncs: sync_jobs_aggregate(where: {
			created_at: {_gt: $time_threshold},
			status: {_eq: "failed"}
		}) {
			aggregate {
				count
			}
		}
	}
	`

	timeThreshold := time.Now().Add(-15 * time.Minute).Format(time.RFC3339)

	var result struct {
		Total struct {
			Aggregate struct {
				Count int `json:"count"`
			} `json:"aggregate"`
		} `json:"sync_jobs_aggregate"`
		Failed struct {
			Aggregate struct {
				Count int `json:"count"`
			} `json:"aggregate"`
		} `json:"failed_syncs"`
	}

	if err := ad.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"time_threshold": timeThreshold,
	}, &result); err != nil {
		return err
	}

	total := result.Total.Aggregate.Count
	failed := result.Failed.Aggregate.Count

	if total > 50 && float64(failed)/float64(total) > 0.15 {
		// Greater than 15% failure rate
		ad.logger.Warn("Sync failure spike detected")

		metrics := map[string]interface{}{
			"total_syncs":  total,
			"failed_syncs": failed,
			"failure_rate": float64(failed) / float64(total),
		}

		threshold := map[string]interface{}{
			"max_failure_rate": 0.15,
			"min_syncs":        50,
		}

		// Create anomaly in DB
		ad.createAnomaly(ctx, "00000000-0000-0000-0000-000000000000", "sync_failure_spike", "critical",
			fmt.Sprintf("High sync failure rate detected: %.2f%%", float64(failed)/float64(total)*100),
			metrics, threshold, "threshold_based", 0.95)
	}

	return nil
}

func (ad *AnomalyDetector) checkAPIQuotaAnomalies(ctx context.Context) error {
	// Placeholder for API quota check logic
	return nil
}

func (ad *AnomalyDetector) checkLatencySpikes(ctx context.Context) error {
	// Placeholder for latency check logic
	return nil
}

// createAnomaly stores anomaly in the database and triggers alert
func (ad *AnomalyDetector) createAnomaly(ctx context.Context, tenantID, anomalyType, severity, description string, metrics, threshold map[string]interface{}, detectionMethod string, confidence float64) {
	mutation := `
	mutation CreateAnomaly($input: anomalies_insert_input!) {
		insert_anomalies_one(object: $input) {
			id
		}
	}
	`

	metricsJSON, _ := json.Marshal(metrics)
	thresholdJSON, _ := json.Marshal(threshold)

	var tenantIDVal *string
	if tenantID != "00000000-0000-0000-0000-000000000000" && tenantID != "" {
		tenantIDVal = &tenantID
	}

	input := map[string]interface{}{
		"anomaly_type":       anomalyType,
		"severity":           severity,
		"description":        description,
		"metrics":            string(metricsJSON),
		"threshold_violated": string(thresholdJSON),
		"detection_method":   detectionMethod,
		"confidence_score":   confidence,
	}

	if tenantIDVal != nil {
		input["tenant_id"] = *tenantIDVal
	}

	var result struct {
		InsertAnomaliesOne struct {
			ID string `json:"id"`
		} `json:"insert_anomalies_one"`
	}

	if err := ad.hasuraClient.Mutate(ctx, mutation, map[string]interface{}{
		"input": input,
	}, &result); err != nil {
		ad.logger.WithError(err).Error("Failed to store anomaly in database")
		return
	}

	anomalyID := result.InsertAnomaliesOne.ID
	ad.logger.WithField("anomaly_id", anomalyID).Info("Anomaly recorded successfully")

	// Trigger Alert
	if ad.alertService != nil {
		if err := ad.alertService.TriggerAlert(ctx, anomalyID, anomalyType, severity, description); err != nil {
			ad.logger.WithError(err).Error("Failed to trigger alert for anomaly")
		}
	}
}
