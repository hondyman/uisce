package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"calendar-service/internal/database"

	"github.com/sirupsen/logrus"
)

type AnomalyDetector struct {
	dbClient     *database.Client
	alertService AnomalyAlertService
	logger       *logrus.Entry
	config       AnomalyDetectorConfig
}

type AnomalyDetectorConfig struct {
	DBClient       *database.Client
	AlertService   AnomalyAlertService
	Logger         *logrus.Entry
	CheckInterval  time.Duration
	SystemTenantID string
}

type AnomalyAlertService interface {
	TriggerAlert(ctx context.Context, anomalyID string, anomalyType string, severity string, description string) error
}

func NewAnomalyDetector(cfg AnomalyDetectorConfig) *AnomalyDetector {
	return &AnomalyDetector{
		dbClient:     cfg.DBClient,
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
	timeThreshold := time.Now().Add(-15 * time.Minute)

	var total int
	var failed int

	totalQuery := `SELECT COUNT(*) FROM sync_jobs WHERE created_at > $1`
	if err := ad.dbClient.Pool().QueryRow(ctx, totalQuery, timeThreshold).Scan(&total); err != nil {
		return err
	}

	failedQuery := `SELECT COUNT(*) FROM sync_jobs WHERE created_at > $1 AND status = 'failed'`
	if err := ad.dbClient.Pool().QueryRow(ctx, failedQuery, timeThreshold).Scan(&failed); err != nil {
		return err
	}

	if total > 50 && float64(failed)/float64(total) > 0.15 {
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

		ad.createAnomaly(ctx, ad.config.SystemTenantID, "sync_failure_spike", "critical",
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
	metricsJSON, _ := json.Marshal(metrics)
	thresholdJSON, _ := json.Marshal(threshold)

	var anomalyID string
	query := `INSERT INTO anomalies (id, tenant_id, anomaly_type, severity, description, metrics, threshold_violated, detection_method, confidence_score, created_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW()) RETURNING id`

	var tenantIDVal interface{}
	if tenantID != "" && tenantID != ad.config.SystemTenantID {
		tenantIDVal = tenantID
	}

	err := ad.dbClient.Pool().QueryRow(ctx, query, tenantIDVal, anomalyType, severity, description, string(metricsJSON), string(thresholdJSON), detectionMethod, confidence).Scan(&anomalyID)
	if err != nil {
		ad.logger.WithError(err).Error("Failed to store anomaly in database")
		return
	}

	ad.logger.WithField("anomaly_id", anomalyID).Info("Anomaly recorded successfully")

	if ad.alertService != nil {
		if err := ad.alertService.TriggerAlert(ctx, anomalyID, anomalyType, severity, description); err != nil {
			ad.logger.WithError(err).Error("Failed to trigger alert for anomaly")
		}
	}
}
