package ml

import (
	"time"
)

// PredictionInput represents the features for failure prediction
type PredictionInput struct {
	ChainID             string             `json:"chain_id"`
	Region              string             `json:"region"`
	TenantID            string             `json:"tenant_id"`
	HealthScore         float64            `json:"health_score"`
	ActiveConflicts     int                `json:"active_conflicts"`
	P99Latency          float64            `json:"p99_latency_ms"`
	LastIncidentTime    *time.Time         `json:"last_incident_time,omitempty"`
	ResolvedConflict24h int                `json:"resolved_conflicts_24h"`
	SLAComplianceScore  float64            `json:"sla_compliance_score"`
	DailyMessageCount   int64              `json:"daily_message_count"`
	ErrorRate           float64            `json:"error_rate"`
	CrossRegionLatency  float64            `json:"cross_region_latency_ms"`
	ConsensusTimeouts   int                `json:"consensus_timeouts_24h"`
	ReplicationLag      int64              `json:"replication_lag_ms"`
	CustomFeatures      map[string]float64 `json:"custom_features,omitempty"`
}

// Prediction represents a failure probability prediction
type Prediction struct {
	ChainID            string          `json:"chain_id"`
	Region             string          `json:"region"`
	TenantID           string          `json:"tenant_id"`
	FailureProbability float64         `json:"failure_probability"`
	Confidence         float64         `json:"confidence"`
	RiskLevel          string          `json:"risk_level"` // low, medium, high, critical
	PredictedAt        time.Time       `json:"predicted_at"`
	Horizon            int             `json:"horizon_hours"` // 1, 6, 24
	TopRiskFactors     []RiskFactor    `json:"top_risk_factors"`
	ModelVersion       string          `json:"model_version"`
	Explainability     *Explainability `json:"explainability,omitempty"`
}

// RiskFactor represents a contributing factor to failure probability
type RiskFactor struct {
	Name         string  `json:"name"`
	Contribution float64 `json:"contribution"` // 0-1, how much it contributes to risk
	CurrentValue float64 `json:"current_value"`
	Threshold    float64 `json:"threshold"`
	Direction    string  `json:"direction"` // "increasing", "decreasing", "stable"
}

// Explainability contains SHAP-based explanations for predictions
type Explainability struct {
	SHAPValues         map[string]float64     `json:"shap_values"`         // Feature name -> SHAP value
	BaseValue          float64                `json:"base_value"`          // Model's base prediction
	FeatureImportance  map[string]float64     `json:"feature_importance"`  // Normalized importance scores
	FeatureValues      map[string]interface{} `json:"feature_values"`      // Actual feature values
	InteractionPairs   []InteractionPair      `json:"interaction_pairs"`   // Top feature interactions
	LocalContributions []LocalContribution    `json:"local_contributions"` // Per-feature detailed explanation
	ExplanationType    string                 `json:"explanation_type"`    // "shap_kernel", "shap_tree", "lime"
	ComputationTime    float64                `json:"computation_time_ms"`
}

// InteractionPair represents interaction between two features
type InteractionPair struct {
	Feature1    string  `json:"feature_1"`
	Feature2    string  `json:"feature_2"`
	Interaction float64 `json:"interaction"` // SHAP interaction value
}

// LocalContribution represents detailed explanation for a single feature
type LocalContribution struct {
	Feature      string        `json:"feature"`
	SHAPValue    float64       `json:"shap_value"`
	AbsShapValue float64       `json:"abs_shap_value"`
	ActualValue  interface{}   `json:"actual_value"`
	Range        *FeatureRange `json:"range,omitempty"`
	Impact       string        `json:"impact"`     // "positive", "negative", "neutral"
	Percentile   float64       `json:"percentile"` // Where feature falls in distribution (0-100)
}

// FeatureRange represents the distribution range of a feature
type FeatureRange struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"std_dev"`
	Q1     float64 `json:"q1"`     // 25th percentile
	Median float64 `json:"median"` // 50th percentile
	Q3     float64 `json:"q3"`     // 75th percentile
}

// ModelMetrics tracks model performance
type ModelMetrics struct {
	ModelVersion       string                 `json:"model_version"`
	TrainedAt          time.Time              `json:"trained_at"`
	Accuracy           float64                `json:"accuracy"`
	Precision          float64                `json:"precision"`
	Recall             float64                `json:"recall"`
	F1Score            float64                `json:"f1_score"`
	AUC                float64                `json:"auc"`
	SpiryProximity     map[string]interface{} `json:"spiry_proximity"` // Custom metrics
	FeatureImportances map[string]float64     `json:"global_feature_importances"`
}

// PredictionBatch represents multiple predictions for batch processing
type PredictionBatch struct {
	TenantID   string               `json:"tenant_id"`
	Region     string               `json:"region"`
	Horizon    int                  `json:"horizon_hours"`
	Inputs     []PredictionInput    `json:"inputs"`
	Timestamps map[string]time.Time `json:"timestamps"` // Optional: custom timestamps per input
}

// PredictionBatchResult contains batch prediction results
type PredictionBatchResult struct {
	BatchID         string            `json:"batch_id"`
	TenantID        string            `json:"tenant_id"`
	Region          string            `json:"region"`
	Horizon         int               `json:"horizon_hours"`
	Predictions     []Prediction      `json:"predictions"`
	Errors          map[string]string `json:"errors"` // chainID -> error message
	ProcessedAt     time.Time         `json:"processed_at"`
	ComputationTime float64           `json:"computation_time_ms"`
}

// AnomalyScore represents an anomaly detection score
type AnomalyScore struct {
	ChainID         string                 `json:"chain_id"`
	Region          string                 `json:"region"`
	Score           float64                `json:"score"` // 0-1, higher = more anomalous
	IsAnomaly       bool                   `json:"is_anomaly"`
	AnomalyType     string                 `json:"anomaly_type"`     // "latency_spike", "error_rate_spike", etc.
	DetectionMethod string                 `json:"detection_method"` // "isolation_forest", "lof", "autoencoder"
	DetectedAt      time.Time              `json:"detected_at"`
	Context         map[string]interface{} `json:"context"`
}

// ModelConfiguration represents ML model configuration
type ModelConfiguration struct {
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Type            string                 `json:"type"` // "xgboost", "neural_net", "ensemble"
	ModelPath       string                 `json:"model_path"`
	SHAPType        string                 `json:"shap_type"` // "kernel", "tree", "lime"
	FeatureNames    []string               `json:"feature_names"`
	FeatureCount    int                    `json:"feature_count"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	Thresholds      map[string]float64     `json:"thresholds"` // risk level thresholds
}
