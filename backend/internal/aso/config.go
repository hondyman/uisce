package aso

import (
	"os"
	"strconv"
	"time"
)

// ============================================================================
// ASO Configuration - Production Settings
// ============================================================================

// ASOConfig contains all configurable parameters for ASO
type ASOConfig struct {
	// Cost Configuration
	Cost CostConfigSettings `json:"cost"`

	// Anomaly Detection
	Anomaly AnomalyConfigSettings `json:"anomaly"`

	// Experiments
	Experiment ExperimentConfigSettings `json:"experiment"`

	// ML Scoring
	ML MLConfigSettings `json:"ml"`

	// Simulation
	Simulation SimulationConfigSettings `json:"simulation"`

	// Self-Healing
	Healing HealingConfigSettings `json:"healing"`

	// Feature Flags
	Features FeatureFlags `json:"features"`
}

// CostConfigSettings configures cost calculations
type CostConfigSettings struct {
	// Cloud pricing ($/unit)
	ComputeCostPerMsPerQuery float64 `json:"compute_cost_per_ms_per_query"` // Default: $0.00001
	StorageCostPerGBPerMonth float64 `json:"storage_cost_per_gb_per_month"` // Default: $0.023 (S3)
	RefreshCostPerMs         float64 `json:"refresh_cost_per_ms"`           // Default: $0.00005

	// Overrides per cloud provider
	CloudProvider string `json:"cloud_provider"` // aws, gcp, azure
}

// AnomalyConfigSettings configures anomaly detection
type AnomalyConfigSettings struct {
	// Miss rate thresholds
	MissRateSpikeThreshold float64 `json:"miss_rate_spike_threshold"` // Default: 1.5 (50% increase)
	MissRateHighThreshold  float64 `json:"miss_rate_high_threshold"`  // Default: 0.5 (50% misses)

	// Latency thresholds
	LatencyRegressionThreshold float64 `json:"latency_regression_threshold"` // Default: 1.2 (20% regression)

	// Refresh failure thresholds
	RefreshFailureAlertCount    int `json:"refresh_failure_alert_count"`    // Default: 3
	RefreshFailureCriticalCount int `json:"refresh_failure_critical_count"` // Default: 5

	// Scan intervals
	ScanIntervalMinutes int `json:"scan_interval_minutes"` // Default: 60
}

// ExperimentConfigSettings configures A/B testing
type ExperimentConfigSettings struct {
	// Defaults
	DefaultMinDuration       time.Duration `json:"default_min_duration"`       // Default: 24h
	DefaultMaxDuration       time.Duration `json:"default_max_duration"`       // Default: 7d
	DefaultMinSampleSize     int           `json:"default_min_sample_size"`    // Default: 1000
	DefaultSignificanceLevel float64       `json:"default_significance_level"` // Default: 0.05
	DefaultMinImprovement    float64       `json:"default_min_improvement"`    // Default: 10%
	DefaultTrafficPercent    float64       `json:"default_traffic_percent"`    // Default: 10%

	// Limits
	MaxConcurrentExperiments int `json:"max_concurrent_experiments"` // Default: 5
}

// MLConfigSettings configures ML scoring
type MLConfigSettings struct {
	// Model weights
	QueriesWeight        float64 `json:"queries_weight"`         // Default: 0.25
	LatencyWeight        float64 `json:"latency_weight"`         // Default: 0.30
	HitRateWeight        float64 `json:"hit_rate_weight"`        // Default: 0.15
	GrainWeight          float64 `json:"grain_weight"`           // Default: 0.10
	SimilarSuccessWeight float64 `json:"similar_success_weight"` // Default: 0.20
	Bias                 float64 `json:"bias"`                   // Default: 0.1

	// Normalization ranges
	MaxQueriesPerDay float64 `json:"max_queries_per_day"` // Default: 10000
	MaxLatencyMs     float64 `json:"max_latency_ms"`      // Default: 5000

	// Confidence thresholds
	MinSamplesForConfidence int `json:"min_samples_for_confidence"` // Default: 30

	// Recommendation thresholds
	StrongRecommendThreshold float64 `json:"strong_recommend_threshold"` // Default: 0.8
	RecommendThreshold       float64 `json:"recommend_threshold"`        // Default: 0.6
	CautionThreshold         float64 `json:"caution_threshold"`          // Default: 0.4
}

// SimulationConfigSettings configures what-if simulations
type SimulationConfigSettings struct {
	// Default speedup estimates by optimization type
	DefaultPreAggSpeedup    float64 `json:"default_preagg_speedup"`    // Default: 5.0
	DefaultRollbackSlowdown float64 `json:"default_rollback_slowdown"` // Default: 3.0

	// Confidence levels
	BaseConfidence            float64 `json:"base_confidence"`              // Default: 0.5
	HighVolumeConfidenceBoost float64 `json:"high_volume_confidence_boost"` // Default: 0.2
	HistoryConfidenceBoost    float64 `json:"history_confidence_boost"`     // Default: 0.15

	// Risk thresholds
	HighLatencyRegressionPct float64 `json:"high_latency_regression_pct"` // Default: 20
	HighCostIncreasePct      float64 `json:"high_cost_increase_pct"`      // Default: 50
	LargeStorageIncreaseMB   int64   `json:"large_storage_increase_mb"`   // Default: 1024
}

// HealingConfigSettings configures self-healing
type HealingConfigSettings struct {
	// Retry settings
	MaxRetries    int           `json:"max_retries"`     // Default: 3
	RetryBackoff  time.Duration `json:"retry_backoff"`   // Default: 5m
	MaxRetryDelay time.Duration `json:"max_retry_delay"` // Default: 1h

	// Auto-actions
	AutoRetryRefresh     bool `json:"auto_retry_refresh"`      // Default: true
	AutoRebuildOnRegress bool `json:"auto_rebuild_on_regress"` // Default: false
	AutoAdjustInterval   bool `json:"auto_adjust_interval"`    // Default: true
}

// FeatureFlags controls feature availability
type FeatureFlags struct {
	EnableABTesting    bool `json:"enable_ab_testing"`
	EnableMLScoring    bool `json:"enable_ml_scoring"`
	EnableSimulation   bool `json:"enable_simulation"`
	EnableSelfHealing  bool `json:"enable_self_healing"`
	EnableAutoApply    bool `json:"enable_auto_apply"`
	EnableCostTracking bool `json:"enable_cost_tracking"`
}

// DefaultConfig returns production-safe defaults
func DefaultConfig() *ASOConfig {
	return &ASOConfig{
		Cost: CostConfigSettings{
			ComputeCostPerMsPerQuery: getEnvFloat("ASO_COST_COMPUTE_PER_MS", 0.00001),
			StorageCostPerGBPerMonth: getEnvFloat("ASO_COST_STORAGE_PER_GB", 0.023),
			RefreshCostPerMs:         getEnvFloat("ASO_COST_REFRESH_PER_MS", 0.00005),
			CloudProvider:            getEnv("ASO_CLOUD_PROVIDER", "aws"),
		},
		Anomaly: AnomalyConfigSettings{
			MissRateSpikeThreshold:      getEnvFloat("ASO_ANOMALY_MISS_SPIKE", 1.5),
			MissRateHighThreshold:       getEnvFloat("ASO_ANOMALY_MISS_HIGH", 0.5),
			LatencyRegressionThreshold:  getEnvFloat("ASO_ANOMALY_LATENCY_REGRESS", 1.2),
			RefreshFailureAlertCount:    getEnvInt("ASO_ANOMALY_REFRESH_ALERT", 3),
			RefreshFailureCriticalCount: getEnvInt("ASO_ANOMALY_REFRESH_CRITICAL", 5),
			ScanIntervalMinutes:         getEnvInt("ASO_ANOMALY_SCAN_INTERVAL", 60),
		},
		Experiment: ExperimentConfigSettings{
			DefaultMinDuration:       time.Duration(getEnvInt("ASO_EXP_MIN_HOURS", 24)) * time.Hour,
			DefaultMaxDuration:       time.Duration(getEnvInt("ASO_EXP_MAX_DAYS", 7)) * 24 * time.Hour,
			DefaultMinSampleSize:     getEnvInt("ASO_EXP_MIN_SAMPLES", 1000),
			DefaultSignificanceLevel: getEnvFloat("ASO_EXP_SIGNIFICANCE", 0.05),
			DefaultMinImprovement:    getEnvFloat("ASO_EXP_MIN_IMPROVE", 10.0),
			DefaultTrafficPercent:    getEnvFloat("ASO_EXP_TRAFFIC_PCT", 10.0),
			MaxConcurrentExperiments: getEnvInt("ASO_EXP_MAX_CONCURRENT", 5),
		},
		ML: MLConfigSettings{
			QueriesWeight:            getEnvFloat("ASO_ML_QUERIES_WEIGHT", 0.25),
			LatencyWeight:            getEnvFloat("ASO_ML_LATENCY_WEIGHT", 0.30),
			HitRateWeight:            getEnvFloat("ASO_ML_HITRATE_WEIGHT", 0.15),
			GrainWeight:              getEnvFloat("ASO_ML_GRAIN_WEIGHT", 0.10),
			SimilarSuccessWeight:     getEnvFloat("ASO_ML_SIMILAR_WEIGHT", 0.20),
			Bias:                     getEnvFloat("ASO_ML_BIAS", 0.1),
			MaxQueriesPerDay:         getEnvFloat("ASO_ML_MAX_QUERIES", 10000),
			MaxLatencyMs:             getEnvFloat("ASO_ML_MAX_LATENCY", 5000),
			MinSamplesForConfidence:  getEnvInt("ASO_ML_MIN_SAMPLES", 30),
			StrongRecommendThreshold: getEnvFloat("ASO_ML_STRONG_THRESH", 0.8),
			RecommendThreshold:       getEnvFloat("ASO_ML_RECOMMEND_THRESH", 0.6),
			CautionThreshold:         getEnvFloat("ASO_ML_CAUTION_THRESH", 0.4),
		},
		Simulation: SimulationConfigSettings{
			DefaultPreAggSpeedup:      getEnvFloat("ASO_SIM_SPEEDUP", 5.0),
			DefaultRollbackSlowdown:   getEnvFloat("ASO_SIM_SLOWDOWN", 3.0),
			BaseConfidence:            getEnvFloat("ASO_SIM_BASE_CONF", 0.5),
			HighVolumeConfidenceBoost: getEnvFloat("ASO_SIM_VOL_BOOST", 0.2),
			HistoryConfidenceBoost:    getEnvFloat("ASO_SIM_HIST_BOOST", 0.15),
			HighLatencyRegressionPct:  getEnvFloat("ASO_SIM_LAT_REGRESS", 20),
			HighCostIncreasePct:       getEnvFloat("ASO_SIM_COST_INCREASE", 50),
			LargeStorageIncreaseMB:    int64(getEnvInt("ASO_SIM_STORAGE_MB", 1024)),
		},
		Healing: HealingConfigSettings{
			MaxRetries:           getEnvInt("ASO_HEAL_MAX_RETRIES", 3),
			RetryBackoff:         time.Duration(getEnvInt("ASO_HEAL_BACKOFF_MIN", 5)) * time.Minute,
			MaxRetryDelay:        time.Duration(getEnvInt("ASO_HEAL_MAX_DELAY_MIN", 60)) * time.Minute,
			AutoRetryRefresh:     getEnvBool("ASO_HEAL_AUTO_RETRY", true),
			AutoRebuildOnRegress: getEnvBool("ASO_HEAL_AUTO_REBUILD", false),
			AutoAdjustInterval:   getEnvBool("ASO_HEAL_AUTO_ADJUST", true),
		},
		Features: FeatureFlags{
			EnableABTesting:    getEnvBool("ASO_FEATURE_AB_TESTING", true),
			EnableMLScoring:    getEnvBool("ASO_FEATURE_ML_SCORING", true),
			EnableSimulation:   getEnvBool("ASO_FEATURE_SIMULATION", true),
			EnableSelfHealing:  getEnvBool("ASO_FEATURE_SELF_HEALING", true),
			EnableAutoApply:    getEnvBool("ASO_FEATURE_AUTO_APPLY", false), // Disabled by default for safety
			EnableCostTracking: getEnvBool("ASO_FEATURE_COST_TRACKING", true),
		},
	}
}

// ============================================================================
// Environment Helpers
// ============================================================================

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
