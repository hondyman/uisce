package ai_routing

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// PredictiveRoutingModel provides ML-based predictions
type PredictiveRoutingModel struct {
	modelEndpoint       string
	modelVersion        string
	confidenceThreshold float64
	httpClient          *http.Client
	cache               map[string]*PredictionResult
	lastUpdateTime      time.Time
}

// NewPredictiveRoutingModel creates a new prediction model
func NewPredictiveRoutingModel(endpoint string) *PredictiveRoutingModel {
	return &PredictiveRoutingModel{
		modelEndpoint:       endpoint,
		modelVersion:        "v1",
		confidenceThreshold: 0.5,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		cache:          make(map[string]*PredictionResult),
		lastUpdateTime: time.Now(),
	}
}

// Predict generates predictions for available branches
func (m *PredictiveRoutingModel) Predict(features Features, branches []Branch) PredictionResult {
	featureVector := m.prepareFeatureVector(features)

	// Try to call ML service, fallback to heuristic if unavailable
	predictions := m.callPredictionAPI(featureVector, branches)
	if len(predictions) == 0 {
		log.Println("ML prediction service unavailable, using heuristic model")
		return m.heuristicPredict(features, branches)
	}

	// Find best prediction
	var bestPrediction PredictionResult
	maxScore := 0.0

	for _, pred := range predictions {
		// Composite score: 70% success rate, 30% speed
		speedScore := 1.0 / (pred.EstimatedDuration + 1.0)
		score := (pred.PredictedSuccessRate * 0.7) + (speedScore * 0.3)

		if score > maxScore && pred.Confidence > m.confidenceThreshold {
			maxScore = score
			bestPrediction = pred
		}
	}

	return bestPrediction
}

// prepareFeatureVector converts features to ML input format
func (m *PredictiveRoutingModel) prepareFeatureVector(features Features) MLFeatureVector {
	businessHours := isBusinessHours(features.Timestamp)
	isWeekend := features.Timestamp.Weekday() > 4 // Friday = 5

	return MLFeatureVector{
		OrderAmount:           features.OrderAmount,
		CustomerLTV:           features.CustomerLTV,
		HistoricalOrderCount:  features.HistoricalOrderCount,
		AvgOrderValue:         features.AvgOrderValue,
		DaysSinceLastOrder:    features.DaysSinceLastOrder,
		RiskScore:             features.RiskScore,
		CustomerTier_VIP:      boolToInt(features.OrderAmount > 10000),
		CustomerTier_Standard: boolToInt(features.OrderAmount <= 10000),
		PaymentMethod_Card:    boolToInt(features.OrderAmount > 0), // placeholder
		PaymentMethod_Wire:    boolToInt(features.OrderAmount > 0), // placeholder
		HourOfDay:             features.Timestamp.Hour(),
		DayOfWeek:             int(features.Timestamp.Weekday()),
		IsWeekend:             boolToInt(isWeekend),
		IsBusinessHours:       boolToInt(businessHours),
		CurrentQueueDepth:     0,   // Will be filled from context
		SystemLoad:            0.5, // placeholder
		SeasonalFactor:        getSeasonalFactor(features.Timestamp),
	}
}

// callPredictionAPI sends request to ML service
func (m *PredictiveRoutingModel) callPredictionAPI(features MLFeatureVector, branches []Branch) []PredictionResult {
	if m.modelEndpoint == "" {
		return nil
	}

	payload := map[string]interface{}{
		"features":      features,
		"branches":      branches,
		"model_version": m.modelVersion,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal prediction request: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", m.modelEndpoint+"/predict", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Failed to create prediction request: %v", err)
		return nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		log.Printf("ML service unavailable: %v", err)
		return nil
	}
	defer resp.Body.Close()

	var predictions []PredictionResult
	if err := json.NewDecoder(resp.Body).Decode(&predictions); err != nil {
		log.Printf("Failed to decode ML response: %v", err)
		return nil
	}

	return predictions
}

// heuristicPredict provides fallback prediction when ML service is unavailable
func (m *PredictiveRoutingModel) heuristicPredict(features Features, branches []Branch) PredictionResult {
	if len(branches) == 0 {
		return PredictionResult{}
	}

	var bestBranch Branch
	bestScore := 0.0

	for _, branch := range branches {
		// Score based on: success rate, duration, capacity
		capacityScore := 1.0 - (float64(branch.CurrentLoad) / float64(branch.Capacity))
		score := (branch.SuccessRate * 0.5) + (capacityScore * 0.3) + ((1.0 / (branch.AvgDuration + 1.0)) * 0.2)

		if score > bestScore {
			bestScore = score
			bestBranch = branch
		}
	}

	return PredictionResult{
		BranchID:             bestBranch.ID,
		PredictedSuccessRate: bestBranch.SuccessRate,
		EstimatedDuration:    bestBranch.AvgDuration,
		Confidence:           0.6, // Lower confidence for heuristic
		FeatureImportance: map[string]float64{
			"success_rate": 0.5,
			"capacity":     0.3,
			"duration":     0.2,
		},
	}
}

// ExplainPrediction returns feature importance for a prediction
func (m *PredictiveRoutingModel) ExplainPrediction(features Features, branchID string) map[string]float64 {
	if m.modelEndpoint == "" {
		return map[string]float64{
			"order_amount":     0.3,
			"customer_tier":    0.25,
			"risk_score":       0.2,
			"time_of_day":      0.15,
			"historical_order": 0.1,
		}
	}

	payload := map[string]interface{}{
		"features":  m.prepareFeatureVector(features),
		"branch_id": branchID,
		"explain":   true,
	}

	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", m.modelEndpoint+"/explain", bytes.NewBuffer(body))
	if err != nil {
		return map[string]float64{}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return map[string]float64{}
	}
	defer resp.Body.Close()

	var explanation map[string]float64
	json.NewDecoder(resp.Body).Decode(&explanation)

	return explanation
}

// UpdateModelVersion updates the model being used
func (m *PredictiveRoutingModel) UpdateModelVersion(newVersion string, newEndpoint string) {
	m.modelVersion = newVersion
	m.modelEndpoint = newEndpoint
	m.cache = make(map[string]*PredictionResult) // Clear cache on update
	log.Printf("Updated predictive model to version %s at %s", newVersion, newEndpoint)
}

// Helper functions

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isBusinessHours(t time.Time) bool {
	hour := t.Hour()
	return hour >= 9 && hour < 17 && t.Weekday() < 5 // Mon-Fri 9AM-5PM
}

func getSeasonalFactor(t time.Time) float64 {
	month := t.Month()
	// Higher factor during peak seasons
	switch {
	case month >= 11 || month == 1:
		return 1.2 // Holiday season
	case month == 3 || month == 9:
		return 1.1 // Q-end
	default:
		return 1.0
	}
}
