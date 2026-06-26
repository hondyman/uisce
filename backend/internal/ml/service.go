package ml

import (
	"context"
	"math"
	"time"
)

// Predictor handles failure probability predictions
type Predictor interface {
	Predict(ctx context.Context, input *PredictionInput) (*Prediction, error)
	PredictBatch(ctx context.Context, batch *PredictionBatch) (*PredictionBatchResult, error)
	GetModelMetrics(ctx context.Context) (*ModelMetrics, error)
	DetectAnomalies(ctx context.Context, input *PredictionInput) ([]AnomalyScore, error)
}

// Service combines prediction and explainability
type Service struct {
	predictor  Predictor
	explainer  Explainer
	config     *ServiceConfig
	cache      map[string]*Prediction
	cacheMutex chan bool
}

// ServiceConfig holds ML service configuration
type ServiceConfig struct {
	ModelVersion           string
	EnableExplainability   bool
	EnableAnomalyDetection bool
	DefaultHorizon         int // hours for prediction
	CacheSize              int
	PredictionThresholds   map[string]float64 // risk level thresholds
}

// Explainer interface for generating explanations
type Explainer interface {
	Explain(ctx context.Context, input *PredictionInput, modelPath string, prediction *Prediction) (interface{}, error)
	ExplainBatch(ctx context.Context, inputs []PredictionInput, modelPath string) (map[string]interface{}, error)
}

// NewService creates a new ML service
func NewService(predictor Predictor, explainer Explainer, config *ServiceConfig) *Service {
	return &Service{
		predictor:  predictor,
		explainer:  explainer,
		config:     config,
		cache:      make(map[string]*Prediction),
		cacheMutex: make(chan bool, 1),
	}
}

// GetPrediction returns a prediction with optional explanation
func (s *Service) GetPrediction(ctx context.Context, input *PredictionInput) (*Prediction, error) {
	// Try to get from cache
	s.cacheMutex <- true
	cached, exists := s.cache[getCacheKey(input)]
	<-s.cacheMutex

	if exists {
		return cached, nil
	}

	// Get prediction
	prediction, err := s.predictor.Predict(ctx, input)
	if err != nil {
		return nil, err
	}

	// Add risk level
	prediction.RiskLevel = s.getRiskLevel(prediction.FailureProbability)

	// Get top risk factors
	prediction.TopRiskFactors = s.getTopRiskFactors(input)

	// Get explainability if enabled
	if s.config.EnableExplainability && s.explainer != nil {
		explain, err := s.explainer.Explain(ctx, input, "", prediction)
		if err == nil {
			prediction.Explainability = explain.(*Explainability)
		}
	}

	// Cache
	s.cacheMutex <- true
	s.cache[getCacheKey(input)] = prediction
	<-s.cacheMutex

	return prediction, nil
}

// GetPredictionBatch returns batch predictions with explanations
func (s *Service) GetPredictionBatch(ctx context.Context, batch *PredictionBatch) (*PredictionBatchResult, error) {
	startTime := time.Now()

	// Get predictions
	result, err := s.predictor.PredictBatch(ctx, batch)
	if err != nil {
		return nil, err
	}

	// Add risk levels and factors
	for i := range result.Predictions {
		result.Predictions[i].RiskLevel = s.getRiskLevel(result.Predictions[i].FailureProbability)
		result.Predictions[i].TopRiskFactors = s.getTopRiskFactors(&batch.Inputs[i])
	}

	// Get explainability for batch if enabled
	if s.config.EnableExplainability && s.explainer != nil {
		explains, err := s.explainer.ExplainBatch(ctx, batch.Inputs, "")
		if err == nil {
			for i, chainID := range batchChainIDs(batch) {
				if explain, exists := explains[chainID]; exists {
					result.Predictions[i].Explainability = explain.(*Explainability)
				}
			}
		}
	}

	result.ComputationTime = float64(time.Since(startTime).Milliseconds())

	return result, nil
}

// GetAnomalies detects anomalies in chain metrics
func (s *Service) GetAnomalies(ctx context.Context, input *PredictionInput) ([]AnomalyScore, error) {
	if !s.config.EnableAnomalyDetection {
		return []AnomalyScore{}, nil
	}

	anomalies, err := s.predictor.DetectAnomalies(ctx, input)
	if err != nil {
		return nil, err
	}

	return anomalies, nil
}

// GetModelMetrics returns current model performance metrics
func (s *Service) GetModelMetrics(ctx context.Context) (*ModelMetrics, error) {
	return s.predictor.GetModelMetrics(ctx)
}

// Helpers

func (s *Service) getRiskLevel(failureProb float64) string {
	thresholds := s.config.PredictionThresholds
	if thresholds == nil {
		thresholds = map[string]float64{
			"high":   0.7,
			"medium": 0.4,
			"low":    0.1,
		}
	}

	if failureProb >= thresholds["high"] {
		return "critical"
	}
	if failureProb >= thresholds["medium"] {
		return "high"
	}
	if failureProb >= thresholds["low"] {
		return "medium"
	}
	return "low"
}

func (s *Service) getTopRiskFactors(input *PredictionInput) []RiskFactor {
	factors := []RiskFactor{
		{
			Name:         "Health Score",
			Contribution: (1.0 - input.HealthScore) * 0.3,
			CurrentValue: input.HealthScore,
			Threshold:    0.85,
			Direction:    getDirection(input.HealthScore, 0.85),
		},
		{
			Name:         "Active Conflicts",
			Contribution: math.Min(float64(input.ActiveConflicts)/10.0*0.25, 1.0),
			CurrentValue: float64(input.ActiveConflicts),
			Threshold:    5.0,
			Direction:    getDirectionInt(input.ActiveConflicts, 5),
		},
		{
			Name:         "P99 Latency",
			Contribution: math.Min(input.P99Latency/1000.0*0.2, 1.0),
			CurrentValue: input.P99Latency,
			Threshold:    500.0,
			Direction:    getDirection(input.P99Latency, 500.0),
		},
		{
			Name:         "Error Rate",
			Contribution: input.ErrorRate * 0.15,
			CurrentValue: input.ErrorRate,
			Threshold:    0.05,
			Direction:    getDirection(input.ErrorRate, 0.05),
		},
	}

	// Sort by contribution (descending)
	sort := func(i, j int) bool {
		return factors[i].Contribution > factors[j].Contribution
	}

	// Quick sort inline
	for i := 0; i < len(factors); i++ {
		for j := i + 1; j < len(factors); j++ {
			if sort(j, i) {
				factors[i], factors[j] = factors[j], factors[i]
			}
		}
	}

	// Return top 4
	if len(factors) > 4 {
		factors = factors[:4]
	}

	return factors
}

func getDirection(current, threshold float64) string {
	if current < threshold {
		return "increasing"
	}
	return "decreasing"
}

func getDirectionInt(current, threshold int) string {
	if current < threshold {
		return "decreasing"
	}
	return "increasing"
}

func getCacheKey(input *PredictionInput) string {
	return input.ChainID + ":" + input.Region + ":" + input.TenantID
}

func batchChainIDs(batch *PredictionBatch) []string {
	ids := make([]string, len(batch.Inputs))
	for i, input := range batch.Inputs {
		ids[i] = input.ChainID
	}
	return ids
}
