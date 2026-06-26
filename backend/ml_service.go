package backend

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	httpapi "github.com/hondyman/semlayer/backend/internal/api"
)

// MLService provides machine learning analytics and predictions
type MLService struct {
	models map[string]*MLModel
}

// MLModel represents a machine learning model
type MLModel struct {
	Name        string
	Type        string
	Version     string
	Accuracy    float64
	LastUpdated time.Time
}

// PredictionRequest represents a prediction request
type PredictionRequest struct {
	Features    map[string]interface{} `json:"features"`
	ModelType   string                 `json:"model_type"`
	TimeHorizon int                    `json:"time_horizon,omitempty"`
}

// PredictionResult represents a prediction result
type PredictionResult struct {
	Prediction map[string]interface{} `json:"prediction"`
	Confidence float64                `json:"confidence"`
	Features   map[string]interface{} `json:"features_used"`
	ModelInfo  map[string]interface{} `json:"model_info"`
	Timestamp  time.Time              `json:"timestamp"`
}

// NewMLService creates a new ML service
func NewMLService() *MLService {
	service := &MLService{
		models: make(map[string]*MLModel),
	}

	// Initialize with sample models
	service.initializeModels()

	return service
}

// Initialize sample ML models
func (ml *MLService) initializeModels() {
	ml.models["portfolio_return"] = &MLModel{
		Name:        "Portfolio Return Predictor",
		Type:        "regression",
		Version:     "1.0.0",
		Accuracy:    0.85,
		LastUpdated: time.Now(),
	}

	ml.models["risk_assessment"] = &MLModel{
		Name:        "Risk Assessment Model",
		Type:        "classification",
		Version:     "1.1.0",
		Accuracy:    0.92,
		LastUpdated: time.Now(),
	}

	ml.models["market_sentiment"] = &MLModel{
		Name:        "Market Sentiment Analyzer",
		Type:        "nlp",
		Version:     "1.0.5",
		Accuracy:    0.78,
		LastUpdated: time.Now(),
	}

	ml.models["volatility_forecast"] = &MLModel{
		Name:        "Volatility Forecasting Model",
		Type:        "time_series",
		Version:     "2.0.0",
		Accuracy:    0.88,
		LastUpdated: time.Now(),
	}

	fmt.Printf("🤖 Initialized %d ML models\n", len(ml.models))
}

// Predict makes a prediction using the specified model
func (ml *MLService) Predict(req PredictionRequest) (*PredictionResult, error) {
	model, exists := ml.models[req.ModelType]
	if !exists {
		return nil, fmt.Errorf("model type '%s' not found", req.ModelType)
	}

	fmt.Printf("🔮 Making prediction with %s model\n", req.ModelType)

	// Simulate ML prediction (in production, this would call actual ML models)
	prediction := ml.simulatePrediction(req, model)

	result := &PredictionResult{
		Prediction: prediction,
		Confidence: model.Accuracy + (rand.Float64()-0.5)*0.1, // Add some variance
		Features:   req.Features,
		ModelInfo: map[string]interface{}{
			"name":     model.Name,
			"type":     model.Type,
			"version":  model.Version,
			"accuracy": model.Accuracy,
		},
		Timestamp: time.Now(),
	}

	return result, nil
}

// Simulate ML prediction (replace with actual ML model calls)
func (ml *MLService) simulatePrediction(req PredictionRequest, model *MLModel) map[string]interface{} {
	switch req.ModelType {
	case "portfolio_return":
		return ml.predictPortfolioReturn(req)
	case "risk_assessment":
		return ml.predictRiskAssessment(req)
	case "market_sentiment":
		return ml.predictMarketSentiment(req)
	case "volatility_forecast":
		return ml.predictVolatility(req)
	default:
		return map[string]interface{}{
			"error": "Unknown model type",
		}
	}
}

// Predict portfolio returns
func (ml *MLService) predictPortfolioReturn(req PredictionRequest) map[string]interface{} {
	// Extract features
	marketTrend := getFloatFeature(req.Features, "market_trend", 0.0)
	volatility := getFloatFeature(req.Features, "volatility", 0.15)
	diversification := getFloatFeature(req.Features, "diversification", 0.7)

	// Simple prediction model (in production, use trained ML model)
	baseReturn := 0.08
	marketImpact := marketTrend * 0.02
	volatilityImpact := -volatility * 0.5
	diversificationBonus := diversification * 0.01

	predictedReturn := baseReturn + marketImpact + volatilityImpact + diversificationBonus

	return map[string]interface{}{
		"expected_return":      math.Round(predictedReturn*10000) / 10000,
		"annualized_return":    math.Round(predictedReturn*10000) / 10000,
		"confidence_interval":  []float64{predictedReturn - 0.02, predictedReturn + 0.02},
		"risk_adjusted_return": predictedReturn / (volatility + 0.01),
	}
}

// Predict risk assessment
func (ml *MLService) predictRiskAssessment(req PredictionRequest) map[string]interface{} {
	// Extract features
	portfolioValue := getFloatFeature(req.Features, "portfolio_value", 100000)
	volatility := getFloatFeature(req.Features, "volatility", 0.15)
	liquidity := getFloatFeature(req.Features, "liquidity", 0.8)

	// Risk assessment logic
	var riskLevel string
	var riskScore float64

	if volatility > 0.25 || portfolioValue < 50000 {
		riskLevel = "high"
		riskScore = 0.8
	} else if volatility > 0.15 || liquidity < 0.5 {
		riskLevel = "medium"
		riskScore = 0.5
	} else {
		riskLevel = "low"
		riskScore = 0.2
	}

	return map[string]interface{}{
		"risk_level":      riskLevel,
		"risk_score":      math.Round(riskScore*100) / 100,
		"var_95":          math.Round(portfolioValue*volatility*1.645*100) / 100,
		"recommendations": ml.generateRiskRecommendations(riskLevel),
	}
}

// Predict market sentiment
func (ml *MLService) predictMarketSentiment(req PredictionRequest) map[string]interface{} {
	// Extract features
	newsSentiment := getFloatFeature(req.Features, "news_sentiment", 0.0)
	socialSentiment := getFloatFeature(req.Features, "social_sentiment", 0.0)
	economicIndicators := getFloatFeature(req.Features, "economic_indicators", 0.0)

	// Combine sentiment scores
	overallSentiment := (newsSentiment + socialSentiment + economicIndicators) / 3

	var sentiment string
	if overallSentiment > 0.2 {
		sentiment = "bullish"
	} else if overallSentiment < -0.2 {
		sentiment = "bearish"
	} else {
		sentiment = "neutral"
	}

	return map[string]interface{}{
		"overall_sentiment": math.Round(overallSentiment*100) / 100,
		"sentiment":         sentiment,
		"confidence":        math.Abs(overallSentiment),
		"drivers": map[string]float64{
			"news":     newsSentiment,
			"social":   socialSentiment,
			"economic": economicIndicators,
		},
	}
}

// Predict volatility
func (ml *MLService) predictVolatility(req PredictionRequest) map[string]interface{} {
	// Extract features
	historicalVol := getFloatFeature(req.Features, "historical_volatility", 0.15)
	marketStress := getFloatFeature(req.Features, "market_stress", 0.3)
	timeHorizon := req.TimeHorizon
	if timeHorizon == 0 {
		timeHorizon = 30 // Default 30 days
	}

	// Volatility prediction (simplified model)
	baseVol := historicalVol
	stressImpact := marketStress * 0.1
	timeDecay := 1.0 / math.Sqrt(float64(timeHorizon)/30) // Longer horizons have lower vol

	predictedVol := baseVol * (1 + stressImpact) * timeDecay

	return map[string]interface{}{
		"predicted_volatility": math.Round(predictedVol*10000) / 10000,
		"confidence_interval":  []float64{predictedVol * 0.8, predictedVol * 1.2},
		"time_horizon_days":    timeHorizon,
		"volatility_regime":    ml.classifyVolatilityRegime(predictedVol),
	}
}

// Generate risk recommendations
func (ml *MLService) generateRiskRecommendations(riskLevel string) []string {
	switch riskLevel {
	case "high":
		return []string{
			"Consider reducing portfolio exposure to volatile assets",
			"Increase allocation to defensive sectors (utilities, consumer staples)",
			"Implement stop-loss orders to limit downside risk",
			"Diversify across uncorrelated asset classes",
			"Consider hedging strategies using options or inverse ETFs",
		}
	case "medium":
		return []string{
			"Monitor portfolio volatility closely",
			"Rebalance to maintain target allocations",
			"Consider adding some defensive positions",
			"Review and update risk management policies",
		}
	case "low":
		return []string{
			"Portfolio risk is well-managed",
			"Consider optimizing for returns while maintaining risk controls",
			"Regular portfolio rebalancing recommended",
			"Monitor for changes in market conditions",
		}
	default:
		return []string{"Unable to generate recommendations"}
	}
}

// Classify volatility regime
func (ml *MLService) classifyVolatilityRegime(volatility float64) string {
	if volatility > 0.30 {
		return "high_volatility"
	} else if volatility > 0.20 {
		return "elevated_volatility"
	} else if volatility > 0.10 {
		return "normal_volatility"
	} else {
		return "low_volatility"
	}
}

// Get float feature with default value
func getFloatFeature(features map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := features[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return defaultValue
}

// StreamAnalytics provides real-time analytics streaming
func (ml *MLService) StreamAnalytics(userID string, duration time.Duration) chan httpapi.RealTimeMessage {
	stream := make(chan httpapi.RealTimeMessage, 10)

	go func() {
		defer close(stream)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		endTime := time.Now().Add(duration)

		for time.Now().Before(endTime) {
			<-ticker.C
			analytics := ml.generateRealTimeAnalytics()
			message := httpapi.RealTimeMessage{
				Type:      "analytics_update",
				Data:      analytics,
				Timestamp: time.Now(),
			}
			stream <- message
		}

		// Send completion message
		completion := httpapi.RealTimeMessage{
			Type:      "analytics_stream_complete",
			Data:      map[string]interface{}{"message": "Analytics streaming completed"},
			Timestamp: time.Now(),
		}
		stream <- completion
	}()

	return stream
}

// Generate real-time analytics data
func (ml *MLService) generateRealTimeAnalytics() map[string]interface{} {
	// Simulate real-time market data and analytics
	basePrice := 4200.0 + rand.Float64()*200 - 100
	volatility := 0.15 + rand.Float64()*0.1 - 0.05

	return map[string]interface{}{
		"market_data": map[string]interface{}{
			"sp500_index":      math.Round(basePrice*100) / 100,
			"volatility_index": math.Round(volatility*10000) / 100,
			"timestamp":        time.Now(),
		},
		"predictions": map[string]interface{}{
			"next_hour_return": math.Round((rand.Float64()-0.5)*0.02*10000) / 10000,
			"confidence":       0.75 + rand.Float64()*0.2,
		},
		"risk_metrics": map[string]interface{}{
			"portfolio_var":    math.Round(100000*volatility*1.645*100) / 100,
			"stress_test_loss": math.Round(100000*volatility*2.5*100) / 100,
		},
		"recommendations": []string{
			"Monitor market volatility closely",
			"Consider rebalancing if allocations drift",
			"Review stop-loss levels",
		},
	}
}

// GetModelInfo returns information about available models
func (ml *MLService) GetModelInfo() map[string]interface{} {
	models := make(map[string]interface{})
	for name, model := range ml.models {
		models[name] = map[string]interface{}{
			"name":         model.Name,
			"type":         model.Type,
			"version":      model.Version,
			"accuracy":     model.Accuracy,
			"last_updated": model.LastUpdated,
		}
	}

	return map[string]interface{}{
		"available_models": models,
		"total_models":     len(models),
		"last_updated":     time.Now(),
	}
}

// UpdateModel updates a model (in production, this would retrain the model)
func (ml *MLService) UpdateModel(modelType string) error {
	model, exists := ml.models[modelType]
	if !exists {
		return fmt.Errorf("model '%s' not found", modelType)
	}

	// Simulate model update
	model.LastUpdated = time.Now()
	model.Accuracy += (rand.Float64() - 0.5) * 0.02 // Small random change
	if model.Accuracy > 1.0 {
		model.Accuracy = 1.0
	} else if model.Accuracy < 0.0 {
		model.Accuracy = 0.0
	}

	log.Printf("🔄 Updated model %s: accuracy = %.3f", modelType, model.Accuracy)
	return nil
}
