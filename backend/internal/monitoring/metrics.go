package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PredictionMetrics holds Prometheus metrics for ML predictions
type PredictionMetrics struct {
	// Prediction latency (milliseconds)
	PredictionLatency prometheus.Histogram

	// Prediction counts by risk level
	PredictionsByRiskLevel prometheus.CounterVec

	// Prediction batch sizes
	BatchPredictionSize prometheus.Histogram

	// Batch prediction latency
	BatchPredictionLatency prometheus.Histogram

	// SHAP computation latency
	SHAPComputeLatency prometheus.Histogram

	// Model errors
	PredictionErrors prometheus.CounterVec

	// Model versions in use
	ActiveModelVersion prometheus.GaugeVec

	// Model metrics (AUC, F1, etc)
	ModelAUC prometheus.GaugeVec
	ModelF1  prometheus.GaugeVec

	// Feature importance tracking
	FeatureImportance prometheus.GaugeVec

	// Prediction drift detection
	InputDriftDetected prometheus.CounterVec

	// Anomaly detection counts
	AnomaliesDetected prometheus.CounterVec

	// Cache hit/miss for SHAP
	SHAPCacheHits   prometheus.Counter
	SHAPCacheMisses prometheus.Counter

	// Model retraining
	ModelRetrainingDuration  prometheus.Histogram
	ModelRetrainingFailures  prometheus.Counter
	ModelRetrainingSuccesses prometheus.Counter

	// API endpoint metrics
	APIRequestDuration prometheus.HistogramVec
	APIErrors          prometheus.CounterVec
}

// NewPredictionMetrics creates and registers all prediction metrics
func NewPredictionMetrics() *PredictionMetrics {
	return &PredictionMetrics{
		// Prediction latency (50ms, 100ms, 200ms, 500ms, 1s, 2s buckets)
		PredictionLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semlayer_prediction_latency_ms",
			Help:    "Latency of individual predictions in milliseconds",
			Buckets: []float64{10, 25, 50, 100, 200, 500, 1000, 2000},
		}),

		// Risk level distribution
		PredictionsByRiskLevel: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "semlayer_predictions_by_risk_level_total",
			Help: "Total predictions by risk level",
		}, []string{"risk_level", "region", "tenant_id"}),

		// Batch size distribution
		BatchPredictionSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semlayer_batch_prediction_size",
			Help:    "Number of predictions in batch requests",
			Buckets: []float64{1, 5, 10, 50, 100, 500, 1000},
		}),

		// Batch prediction latency (100ms, 500ms, 1s, 5s buckets)
		BatchPredictionLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semlayer_batch_prediction_latency_ms",
			Help:    "Latency of batch predictions in milliseconds",
			Buckets: []float64{100, 500, 1000, 2000, 5000, 10000},
		}),

		// SHAP computation latency
		SHAPComputeLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semlayer_shap_compute_latency_ms",
			Help:    "SHAP computation latency in milliseconds",
			Buckets: []float64{10, 50, 100, 200, 500, 1000, 2000},
		}),

		// Prediction errors
		PredictionErrors: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "semlayer_prediction_errors_total",
			Help: "Total prediction errors",
		}, []string{"error_type", "region", "tenant_id"}),

		// Active model versions
		ActiveModelVersion: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "semlayer_active_model_version",
			Help: "Currently active model version (1=active, 0=inactive)",
		}, []string{"model_version", "region", "tenant_id"}),

		// Model AUC
		ModelAUC: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "semlayer_model_auc",
			Help: "Model AUC score",
		}, []string{"model_version", "region"}),

		// Model F1 score
		ModelF1: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "semlayer_model_f1_score",
			Help: "Model F1 score",
		}, []string{"model_version", "region"}),

		// Feature importance
		FeatureImportance: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "semlayer_feature_importance",
			Help: "Feature importance score",
		}, []string{"feature_name", "model_version", "region"}),

		// Input drift detection
		InputDriftDetected: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "semlayer_input_drift_detected_total",
			Help: "Total input drift detection events",
		}, []string{"feature_name", "region", "tenant_id"}),

		// Anomaly detection
		AnomaliesDetected: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "semlayer_anomalies_detected_total",
			Help: "Total anomalies detected",
		}, []string{"anomaly_type", "region", "tenant_id"}),

		// SHAP caching
		SHAPCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semlayer_shap_cache_hits_total",
			Help: "Total SHAP cache hits",
		}),

		SHAPCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semlayer_shap_cache_misses_total",
			Help: "Total SHAP cache misses",
		}),

		// Model retraining
		ModelRetrainingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semlayer_model_retraining_duration_seconds",
			Help:    "Model retraining duration in seconds",
			Buckets: []float64{60, 300, 600, 1200, 1800, 3600},
		}),

		ModelRetrainingFailures: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semlayer_model_retraining_failures_total",
			Help: "Total model retraining failures",
		}),

		ModelRetrainingSuccesses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semlayer_model_retraining_successes_total",
			Help: "Total successful model retraining operations",
		}),

		// API metrics
		APIRequestDuration: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "semlayer_api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
		}, []string{"method", "endpoint", "status"}),

		APIErrors: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "semlayer_api_errors_total",
			Help: "Total API errors",
		}, []string{"method", "endpoint", "status_code"}),
	}
}

// RecordPrediction records metrics for a single prediction
func (m *PredictionMetrics) RecordPrediction(latencyMs float64, riskLevel string, region string, tenantID string) {
	m.PredictionLatency.Observe(latencyMs)
	m.PredictionsByRiskLevel.WithLabelValues(riskLevel, region, tenantID).Inc()
}

// RecordBatchPrediction records metrics for batch predictions
func (m *PredictionMetrics) RecordBatchPrediction(batchSize int, latencyMs float64, region string, tenantID string) {
	m.BatchPredictionSize.Observe(float64(batchSize))
	m.BatchPredictionLatency.Observe(latencyMs)
}

// RecordSHAPComputation records SHAP computation metrics
func (m *PredictionMetrics) RecordSHAPComputation(latencyMs float64) {
	m.SHAPComputeLatency.Observe(latencyMs)
}

// RecordPredictionError records a prediction error
func (m *PredictionMetrics) RecordPredictionError(errType string, region string, tenantID string) {
	m.PredictionErrors.WithLabelValues(errType, region, tenantID).Inc()
}

// SetActiveModel records the current active model version
func (m *PredictionMetrics) SetActiveModel(version string, region string, tenantID string, isActive bool) {
	value := 0.0
	if isActive {
		value = 1.0
	}
	m.ActiveModelVersion.WithLabelValues(version, region, tenantID).Set(value)
}

// UpdateModelMetrics updates model performance metrics
func (m *PredictionMetrics) UpdateModelMetrics(version string, region string, auc float64, f1 float64) {
	m.ModelAUC.WithLabelValues(version, region).Set(auc)
	m.ModelF1.WithLabelValues(version, region).Set(f1)
}

// UpdateFeatureImportance updates feature importance metrics
func (m *PredictionMetrics) UpdateFeatureImportance(featureName string, version string, region string, importance float64) {
	m.FeatureImportance.WithLabelValues(featureName, version, region).Set(importance)
}

// RecordDriftDetection records input drift detection
func (m *PredictionMetrics) RecordDriftDetection(featureName string, region string, tenantID string) {
	m.InputDriftDetected.WithLabelValues(featureName, region, tenantID).Inc()
}

// RecordAnomaly records anomaly detection
func (m *PredictionMetrics) RecordAnomaly(anomalyType string, region string, tenantID string) {
	m.AnomaliesDetected.WithLabelValues(anomalyType, region, tenantID).Inc()
}

// RecordModelRetraining records model retraining metrics
func (m *PredictionMetrics) RecordModelRetainingSuccess(durationSeconds float64) {
	m.ModelRetrainingDuration.Observe(durationSeconds)
	m.ModelRetrainingSuccesses.Inc()
}

// RecordModelRetrainingFailure records a failed retraining
func (m *PredictionMetrics) RecordModelRetrainingFailure() {
	m.ModelRetrainingFailures.Inc()
}

// RecordAPIRequest records an API request
func (m *PredictionMetrics) RecordAPIRequest(method string, endpoint string, status string, durationSeconds float64) {
	m.APIRequestDuration.WithLabelValues(method, endpoint, status).Observe(durationSeconds)
}

// RecordAPIError records an API error
func (m *PredictionMetrics) RecordAPIError(method string, endpoint string, statusCode string) {
	m.APIErrors.WithLabelValues(method, endpoint, statusCode).Inc()
}

// DriftDetectionMetrics holds metrics for input distribution drift
type DriftDetectionMetrics struct {
	// Current feature statistics
	FeatureMean   map[string]float64
	FeatureStdDev map[string]float64

	// Historical feature statistics
	HistoricalMean   map[string]float64
	HistoricalStdDev map[string]float64

	// Kolmogorov-Smirnov test statistic
	KSStatistic map[string]float64

	// Wasserstein distance
	WassersteinDistance map[string]float64

	// Population Stability Index
	PSI map[string]float64
}

// NewDriftDetectionMetrics creates drift detection metrics
func NewDriftDetectionMetrics() *DriftDetectionMetrics {
	return &DriftDetectionMetrics{
		FeatureMean:         make(map[string]float64),
		FeatureStdDev:       make(map[string]float64),
		HistoricalMean:      make(map[string]float64),
		HistoricalStdDev:    make(map[string]float64),
		KSStatistic:         make(map[string]float64),
		WassersteinDistance: make(map[string]float64),
		PSI:                 make(map[string]float64),
	}
}

// ComputeDriftMetrics computes drift metrics for a feature
func (d *DriftDetectionMetrics) ComputeDriftMetrics(feature string, currentMean float64, currentStdDev float64) float64 {
	d.FeatureMean[feature] = currentMean
	d.FeatureStdDev[feature] = currentStdDev

	historicalMean := d.HistoricalMean[feature]
	historicalStdDev := d.HistoricalStdDev[feature]

	// Compute Population Stability Index (PSI)
	// PSI = sum((current_pct - historical_pct) * ln(current_pct / historical_pct))
	// For continuous variables, approximate using mean and stddev
	if historicalStdDev > 0 {
		psi := ((currentMean - historicalMean) / historicalStdDev) * ((currentMean - historicalMean) / historicalStdDev)
		if psi > 0.25 {
			// Significant drift detected
			d.PSI[feature] = psi
		}
	}

	return d.PSI[feature]
}

// IsDriftDetected returns whether significant drift is detected
func (d *DriftDetectionMetrics) IsDriftDetected() bool {
	driftThreshold := 0.25 // PSI threshold

	for _, psi := range d.PSI {
		if psi > driftThreshold {
			return true
		}
	}

	return false
}

// ModelDriftMetrics tracks model performance drift
type ModelDriftMetrics struct {
	BaselineAUC   float64
	CurrentAUC    float64
	AUCDriftRatio float64

	BaselineF1   float64
	CurrentF1    float64
	F1DriftRatio float64

	PredictionDistributionShift float64
	ConfidenceDrift             float64

	DriftDetectedAt *string
	DriftSeverity   string // "low", "medium", "high", "critical"
}

// DetectModelDrift detects if model performance has drifted
func (m *ModelDriftMetrics) DetectModelDrift() bool {
	// AUC drift > 1% indicates potential drift
	m.AUCDriftRatio = 1.0 - (m.CurrentAUC / m.BaselineAUC)
	if m.AUCDriftRatio > 0.01 {
		m.DriftSeverity = "high"
		return true
	}

	// F1 drift > 1% indicates potential drift
	m.F1DriftRatio = 1.0 - (m.CurrentF1 / m.BaselineF1)
	if m.F1DriftRatio > 0.01 {
		m.DriftSeverity = "high"
		return true
	}

	m.DriftSeverity = "low"
	return false
}
