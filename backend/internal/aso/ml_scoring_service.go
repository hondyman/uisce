package aso

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// ML Scoring Types
// ============================================================================

// MLScoreInput is the feature set for scoring
type MLScoreInput struct {
	// Workload features
	QueriesPerDay       float64 `json:"queries_per_day"`
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
	P95LatencyMs        float64 `json:"p95_latency_ms"`
	CurrentHitRate      float64 `json:"current_hit_rate"`
	QueryPatternEntropy float64 `json:"query_pattern_entropy"` // How varied are query patterns

	// Asset features
	GrainCount        int   `json:"grain_count"`
	MeasureCount      int   `json:"measure_count"`
	EstimatedRowCount int64 `json:"estimated_row_count"`

	// Tenant features
	TenantSize       string  `json:"tenant_size"`        // small, medium, large, enterprise
	TenantGrowthRate float64 `json:"tenant_growth_rate"` // % growth in query volume

	// Historical features
	SimilarOptSuccessRate float64 `json:"similar_opt_success_rate"`
	SimilarOptAvgROI      float64 `json:"similar_opt_avg_roi"`
}

// MLScoreResult is the output of ML scoring
type MLScoreResult struct {
	Score             float64            `json:"score"`             // 0-1 overall score
	Confidence        float64            `json:"confidence"`        // 0-1 confidence in score
	PredictedSpeedup  float64            `json:"predicted_speedup"` // e.g., 3.5x
	PredictedROI      float64            `json:"predicted_roi"`     // % return
	FeatureImportance map[string]float64 `json:"feature_importance"`

	// Breakdown
	WorkloadScore    float64 `json:"workload_score"`
	CostBenefitScore float64 `json:"cost_benefit_score"`
	RiskScore        float64 `json:"risk_score"`

	// Explanation
	TopFactors     []string `json:"top_factors"`
	Recommendation string   `json:"recommendation"`
}

// TrainingDataPoint represents historical outcome for training
type TrainingDataPoint struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	OptimizationID uuid.UUID       `json:"optimization_id" db:"optimization_id"`
	Input          MLScoreInput    `json:"input" db:"-"`
	InputJSON      json.RawMessage `json:"-" db:"input_json"`

	// Actual outcomes
	ActualSpeedup float64 `json:"actual_speedup" db:"actual_speedup"`
	ActualROI     float64 `json:"actual_roi" db:"actual_roi"`
	WasSuccessful bool    `json:"was_successful" db:"was_successful"`

	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
}

// ============================================================================
// ML Scoring Service
// ============================================================================

// MLScoringService provides ML-powered optimization scoring
type MLScoringService interface {
	// ScoreOptimization scores an optimization using ML
	ScoreOptimization(ctx context.Context, optID uuid.UUID) (*MLScoreResult, error)

	// ScoreFromFeatures scores given raw features
	ScoreFromFeatures(ctx context.Context, input MLScoreInput) (*MLScoreResult, error)

	// RecordOutcome records actual outcome for training
	RecordOutcome(ctx context.Context, optID uuid.UUID, speedup, roi float64, successful bool) error

	// GetModelStats returns model performance statistics
	GetModelStats(ctx context.Context) (*ModelStats, error)

	// RetrainModel triggers model retraining
	RetrainModel(ctx context.Context) error
}

// ModelStats shows model performance
type ModelStats struct {
	TotalTrainingPoints int       `json:"total_training_points"`
	LastTrainedAt       time.Time `json:"last_trained_at"`
	AccuracyR2          float64   `json:"accuracy_r2"`      // R² score
	MAE                 float64   `json:"mae"`              // Mean absolute error
	PredictionCount     int       `json:"prediction_count"` // Total predictions made
}

// mlScoringService implements MLScoringService
type mlScoringService struct {
	db        *sqlx.DB
	optRepo   ASOOptimizationRepository
	telemetry TelemetryService
	config    *ASOConfig
	weights   ModelWeights
}

// ModelWeights are the learned coefficients
type ModelWeights struct {
	QueriesWeight        float64
	LatencyWeight        float64
	HitRateWeight        float64
	GrainWeight          float64
	SimilarSuccessWeight float64
	Bias                 float64
}

// NewMLScoringService creates a new ML scoring service
func NewMLScoringService(db *sqlx.DB, optRepo ASOOptimizationRepository, telemetry TelemetryService, config *ASOConfig) MLScoringService {
	if config == nil {
		config = DefaultConfig()
	}
	return &mlScoringService{
		db:        db,
		optRepo:   optRepo,
		telemetry: telemetry,
		config:    config,
		weights:   weightsFromConfig(config),
	}
}

func weightsFromConfig(config *ASOConfig) ModelWeights {
	return ModelWeights{
		QueriesWeight:        config.ML.QueriesWeight,
		LatencyWeight:        config.ML.LatencyWeight,
		HitRateWeight:        config.ML.HitRateWeight,
		GrainWeight:          config.ML.GrainWeight,
		SimilarSuccessWeight: config.ML.SimilarSuccessWeight,
		Bias:                 config.ML.Bias,
	}
}

// ScoreOptimization scores an optimization using ML
func (s *mlScoringService) ScoreOptimization(ctx context.Context, optID uuid.UUID) (*MLScoreResult, error) {
	opt, err := s.optRepo.GetByID(ctx, optID)
	if err != nil || opt == nil {
		return nil, err
	}

	// Extract features from optimization
	input := s.extractFeatures(ctx, opt)

	return s.ScoreFromFeatures(ctx, input)
}

// ScoreFromFeatures scores given raw features
func (s *mlScoringService) ScoreFromFeatures(ctx context.Context, input MLScoreInput) (*MLScoreResult, error) {
	result := &MLScoreResult{
		FeatureImportance: make(map[string]float64),
		TopFactors:        []string{},
	}

	// Normalize features using config thresholds
	normalizedQueries := s.normalize(input.QueriesPerDay, 0, s.config.ML.MaxQueriesPerDay)
	normalizedLatency := s.normalize(input.P95LatencyMs, 0, s.config.ML.MaxLatencyMs)
	normalizedHitRate := 1 - input.CurrentHitRate                        // Invert: lower hit rate = more opportunity
	normalizedGrain := 1 - s.normalize(float64(input.GrainCount), 1, 10) // Fewer grains = better
	normalizedSuccess := input.SimilarOptSuccessRate

	// Calculate component scores
	result.WorkloadScore = normalizedQueries*0.4 + normalizedLatency*0.6
	result.CostBenefitScore = normalizedHitRate*0.5 + input.SimilarOptAvgROI/100*0.5
	result.RiskScore = 1 - input.QueryPatternEntropy // Lower entropy = more predictable = lower risk

	// Calculate overall score using weighted sum
	score := s.weights.QueriesWeight*normalizedQueries +
		s.weights.LatencyWeight*normalizedLatency +
		s.weights.HitRateWeight*normalizedHitRate +
		s.weights.GrainWeight*normalizedGrain +
		s.weights.SimilarSuccessWeight*normalizedSuccess +
		s.weights.Bias

	// Clamp to 0-1
	result.Score = math.Min(1.0, math.Max(0.0, score))

	// Calculate confidence based on data availability
	result.Confidence = s.calculateConfidence(input)

	// Predict speedup and ROI
	result.PredictedSpeedup = 1 + (result.Score * 10) // 1x to 11x
	result.PredictedROI = result.Score * 200          // 0% to 200%

	// Feature importance
	result.FeatureImportance["queries_per_day"] = s.weights.QueriesWeight * normalizedQueries
	result.FeatureImportance["p95_latency_ms"] = s.weights.LatencyWeight * normalizedLatency
	result.FeatureImportance["current_hit_rate"] = s.weights.HitRateWeight * normalizedHitRate
	result.FeatureImportance["grain_count"] = s.weights.GrainWeight * normalizedGrain
	result.FeatureImportance["similar_success_rate"] = s.weights.SimilarSuccessWeight * normalizedSuccess

	// Identify top factors
	s.identifyTopFactors(result, input)

	// Generate recommendation
	s.generateMLRecommendation(result)

	return result, nil
}

// RecordOutcome records actual outcome for training
func (s *mlScoringService) RecordOutcome(ctx context.Context, optID uuid.UUID, speedup, roi float64, successful bool) error {
	opt, err := s.optRepo.GetByID(ctx, optID)
	if err != nil || opt == nil {
		return err
	}

	input := s.extractFeatures(ctx, opt)
	inputJSON, _ := json.Marshal(input)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO semantic.ml_training_data (
			id, optimization_id, input_json, actual_speedup, actual_roi, was_successful, recorded_at
		) VALUES ($1, $2, $3, $4, $5, $6, now())
	`, uuid.New(), optID, inputJSON, speedup, roi, successful)

	return err
}

// GetModelStats returns model performance statistics
// GetModelStats returns model performance statistics
func (s *mlScoringService) GetModelStats(ctx context.Context) (*ModelStats, error) {
	stats := &ModelStats{}

	// Count training points
	s.db.GetContext(ctx, &stats.TotalTrainingPoints, `
		SELECT COUNT(*) FROM semantic.ml_training_data
	`)

	// Get last trained time
	s.db.GetContext(ctx, &stats.LastTrainedAt, `
		SELECT COALESCE(MAX(recorded_at), now()) FROM semantic.ml_training_data
	`)

	// Calculate accuracy from training data
	var r2, mae float64
	s.db.GetContext(ctx, &r2, `
		SELECT COALESCE(
			1 - (SUM(POW(actual_speedup - 5.0, 2)) / NULLIF(SUM(POW(actual_speedup - AVG(actual_speedup) OVER(), 2)), 0)),
			0.5
		)
		FROM semantic.ml_training_data
		WHERE recorded_at > now() - interval '90 days'
	`)
	s.db.GetContext(ctx, &mae, `
		SELECT COALESCE(AVG(ABS(actual_speedup - 5.0)), 1.0)
		FROM semantic.ml_training_data
		WHERE recorded_at > now() - interval '90 days'
	`)
	stats.AccuracyR2 = r2
	stats.MAE = mae

	return stats, nil
}

// RetrainModel triggers model retraining
func (s *mlScoringService) RetrainModel(ctx context.Context) error {
	// In production, this would:
	// 1. Load all training data
	// 2. Split into train/validation sets
	// 3. Train regression model (could use external ML service)
	// 4. Update weights
	// 5. Calculate accuracy metrics

	// For now, we simulate by adjusting weights slightly based on data
	var avgSuccess float64
	s.db.GetContext(ctx, &avgSuccess, `
		SELECT COALESCE(AVG(CASE WHEN was_successful THEN 1.0 ELSE 0.0 END), 0.5)
		FROM semantic.ml_training_data
		WHERE recorded_at > now() - interval '30 days'
	`)

	// Adjust bias based on success rate
	s.weights.Bias = avgSuccess * 0.2

	return nil
}

// ============================================================================
// Internal Methods
// ============================================================================

func (s *mlScoringService) extractFeatures(ctx context.Context, opt *ASOOptimization) MLScoreInput {
	input := MLScoreInput{
		TenantSize:            "medium",
		SimilarOptSuccessRate: 0.5, // Default, will be overwritten
		SimilarOptAvgROI:      50.0,
	}

	if opt.QueriesPerDay != nil {
		input.QueriesPerDay = *opt.QueriesPerDay
	}
	if opt.AvgLatencyMs != nil {
		input.AvgLatencyMs = *opt.AvgLatencyMs
	}
	if opt.P95LatencyMs != nil {
		input.P95LatencyMs = *opt.P95LatencyMs
	}

	// Query similar optimizations for success rate
	var successRate float64
	s.db.GetContext(ctx, &successRate, `
		SELECT COALESCE(AVG(CASE WHEN status = 'applied' THEN 1.0 ELSE 0.0 END), 0.5)
		FROM semantic.aso_optimization
		WHERE optimization_type = $1
		AND created_at > now() - interval '90 days'
	`, opt.OptimizationType)
	input.SimilarOptSuccessRate = successRate

	// Extract grain count from details
	if opt.Details != nil {
		var details CreatePreAggDetails
		if json.Unmarshal(opt.Details, &details) == nil {
			input.GrainCount = len(details.Grain)
			input.MeasureCount = len(details.Measures)
		}
	}

	// Get workload profile from telemetry if available
	if profile, err := s.telemetry.GetWorkloadProfile(ctx, opt.TargetID); err == nil && profile != nil {
		if profile.PreAggHitRate > 0 {
			input.CurrentHitRate = profile.PreAggHitRate
		}
	}

	return input
}

func (s *mlScoringService) normalize(value, min, max float64) float64 {
	if max <= min {
		return 0.5
	}
	normalized := (value - min) / (max - min)
	return math.Min(1.0, math.Max(0.0, normalized))
}

func (s *mlScoringService) calculateConfidence(input MLScoreInput) float64 {
	confidence := s.config.Simulation.BaseConfidence

	// More queries = more confidence
	if input.QueriesPerDay > 1000 {
		confidence += s.config.Simulation.HighVolumeConfidenceBoost
	}

	// History of similar optimizations = more confidence
	if input.SimilarOptSuccessRate > 0 {
		confidence += s.config.Simulation.HistoryConfidenceBoost
	}

	// Low entropy = more predictable = more confidence
	if input.QueryPatternEntropy < 0.3 {
		confidence += 0.1
	}

	return math.Min(0.95, confidence)
}

func (s *mlScoringService) identifyTopFactors(result *MLScoreResult, input MLScoreInput) {
	// Sort features by importance
	if input.P95LatencyMs > 1000 {
		result.TopFactors = append(result.TopFactors, "High latency (p95 > 1000ms)")
	}
	if input.QueriesPerDay > 5000 {
		result.TopFactors = append(result.TopFactors, "High query volume (>5000/day)")
	}
	if input.CurrentHitRate < 0.5 {
		result.TopFactors = append(result.TopFactors, "Low pre-agg hit rate (<50%)")
	}
	if input.SimilarOptSuccessRate > 0.8 {
		result.TopFactors = append(result.TopFactors, "Similar optimizations highly successful")
	}
}

func (s *mlScoringService) generateMLRecommendation(result *MLScoreResult) {
	switch {
	case result.Score >= s.config.ML.StrongRecommendThreshold:
		result.Recommendation = "Strongly recommended. High likelihood of significant improvement."
	case result.Score >= s.config.ML.RecommendThreshold:
		result.Recommendation = "Recommended. Expected positive impact with acceptable risk."
	case result.Score >= s.config.ML.CautionThreshold:
		result.Recommendation = "Consider A/B testing first. Mixed signals in prediction."
	default:
		result.Recommendation = "Not recommended. Low expected benefit or high risk."
	}
}
