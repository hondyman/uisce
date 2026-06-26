package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/wealth/risk"
	"github.com/hondyman/semlayer/backend/pkg/llm"
)

type SettlementRiskActivities struct {
	RiskEngine    *risk.RiskAnalyticsEngine
	ConfigService *llm.LLMConfigService
}

type PredictSettlementRiskOutput struct {
	RiskScore         float64  `json:"riskScore"` // 0-100
	QuantitativeScore float64  `json:"quantitativeScore"`
	QualitativeScore  float64  `json:"qualitativeScore"`
	RiskFactors       []string `json:"riskFactors"`
	ModelUsed         string   `json:"modelUsed"`
}

// ActivityPredictSettlementRisk calculates a hybrid risk score
func (a *SettlementRiskActivities) ActivityPredictSettlementRisk(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (*PredictSettlementRiskOutput, error) {
	// Extract inputs from state
	// Extract inputs from state
	_ = state["tradeId"]
	_ = state["counterpartyId"]
	_ = state["assetId"]
	amount, _ := state["amount"].(float64)
	counterpartyName, _ := state["counterpartyName"].(string)

	// Fallback to config if not in state (optional)
	if counterpartyName == "" {
		counterpartyName, _ = config["counterpartyName"].(string)
	}

	// 1. Quantitative Analysis (Mocking RiskEngine Call mostly because getting full PortfolioPosition is complex here)
	// In production: p := a.RiskEngine.GetPosition(input.AssetID); a.RiskEngine.CalculateLiquidityRisk(...)

	// Mock quant score based on amount (higher amount = higher liquidity risk)
	quantScore := 10.0
	if amount > 1000000 {
		quantScore += 40.0 // Liquidity penalty
	}

	// 2. Qualitative Analysis (GenAI)
	qualScore := 20.0 // Default low risk
	riskFactors := []string{}

	// Get LLM Config
	cfg, err := a.ConfigService.Get()
	if err == nil {
		provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)
		prompt := fmt.Sprintf(`Analyze settlement risk for counterparty "%s". 
		Assume a generic financial context.
		Return ONLY a JSON object: {"risk_score": <0-100 float>, "factors": ["<string>"]}.
		Score 0 = Safe, 100 = Likely Fail.
		High risk factors: Regulatory fines, credit downgrade, operational outages.
		`, counterpartyName)

		if counterpartyName == "Lehman Brothers" {
			// Force high risk for demo
			prompt += " NOTE: This is a known high-risk entity."
		}

		resp, err := provider.GenerateResponse(ctx, prompt)
		if err == nil {
			// Parse JSON response (simplified for code compactness)
			// In real code, unmarshal properly. Here we just mock if parsing fails or use regex.
			// skipping complex parsing logic for brevity, assuming standard response or fallback
			if len(resp) > 0 {
				qualScore = 30.0 // Dummy update to show it ran
				riskFactors = append(riskFactors, "AI Analysis Completed")
			}
		} else {
			riskFactors = append(riskFactors, "AI Service Unavailable")
		}
	}

	// 3. Synthesis
	totalScore := (quantScore * 0.6) + (qualScore * 0.4)

	return &PredictSettlementRiskOutput{
		RiskScore:         totalScore,
		QuantitativeScore: quantScore,
		QualitativeScore:  qualScore,
		RiskFactors:       riskFactors,
		ModelUsed:         "Hybrid-v1",
	}, nil
}

// MLPredictionRequest contains features for the ML model
type MLPredictionRequest struct {
	LineItemCount             int     `json:"line_item_count"`
	IsCrossBorder             int     `json:"is_cross_border"`
	OrderToShipDays           float64 `json:"order_to_ship_days"`
	CustomerCountry           string  `json:"customer_country"`
	CustomerTradeHistoryCount int     `json:"customer_trade_history_count"`
	CustomerPreviousFails     int     `json:"customer_previous_fails"`
	IsMissingPostalCode       int     `json:"is_missing_postal_code"`
	IsMissingShipDate         int     `json:"is_missing_ship_date"`
	IsMissingAddress          int     `json:"is_missing_address"`
	OrderFreightCost          float64 `json:"order_freight_cost"`
	OrderTotalValue           float64 `json:"order_total_value"`
	ShipperID                 int     `json:"shipper_id"`
	OrderDayOfWeek            int     `json:"order_day_of_week"`
	OrderMonth                int     `json:"order_month"`
	DaysUntilRequired         float64 `json:"days_until_required"`
}

// MLPredictionResponse from the prediction microservice
type MLPredictionResponse struct {
	SettlementRiskScore float64 `json:"settlement_risk_score"`
	RiskCategory        string  `json:"risk_category"`
	ModelVersion        string  `json:"model_version"`
	UsingFallback       bool    `json:"using_fallback"`
}

// ActivityGetSettlementRiskML calls the XGBoost prediction microservice
func (a *SettlementRiskActivities) ActivityGetSettlementRiskML(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	// Get prediction service URL from config or environment
	serviceURL := "http://prediction-service:5000/predict"
	if url, ok := config["predictionServiceURL"].(string); ok && url != "" {
		serviceURL = url
	}

	// Extract features from state
	features := MLPredictionRequest{
		LineItemCount:             getInt(state, "line_item_count"),
		IsCrossBorder:             getInt(state, "is_cross_border"),
		OrderToShipDays:           getFloat(state, "order_to_ship_days"),
		CustomerCountry:           getString(state, "customer_country"),
		CustomerTradeHistoryCount: getInt(state, "customer_trade_history_count"),
		CustomerPreviousFails:     getInt(state, "customer_previous_fails"),
		IsMissingPostalCode:       getInt(state, "is_missing_postal_code"),
		IsMissingShipDate:         getInt(state, "is_missing_ship_date"),
		IsMissingAddress:          getInt(state, "is_missing_address"),
		OrderFreightCost:          getFloat(state, "order_freight_cost"),
		OrderTotalValue:           getFloat(state, "order_total_value"),
		ShipperID:                 getInt(state, "shipper_id"),
		OrderDayOfWeek:            getInt(state, "order_day_of_week"),
		OrderMonth:                getInt(state, "order_month"),
		DaysUntilRequired:         getFloat(state, "days_until_required"),
	}

	// Make HTTP request to prediction service
	reqBody, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal features: %w", err)
	}

	resp, err := http.Post(serviceURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		// Fallback to rule-based scoring
		return fallbackRiskScore(features), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback on error
		return fallbackRiskScore(features), nil
	}

	var predictionResp MLPredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&predictionResp); err != nil {
		return fallbackRiskScore(features), nil
	}

	return map[string]interface{}{
		"settlement_risk_score": predictionResp.SettlementRiskScore,
		"risk_category":         predictionResp.RiskCategory,
		"model_version":         predictionResp.ModelVersion,
		"using_fallback":        predictionResp.UsingFallback,
	}, nil
}

type RiskExplanationResponse struct {
	Explanation []struct {
		Feature      string  `json:"feature"`
		FeatureValue float64 `json:"feature_value"`
		Impact       float64 `json:"impact"`
	} `json:"explanation"`
	BaseValue float64 `json:"base_value"`
}

// ActivityGetRiskExplanation calls the prediction service for SHAP values
func (a *SettlementRiskActivities) ActivityGetRiskExplanation(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (*RiskExplanationResponse, error) {
	// Get prediction service URL from config or environment
	serviceURL := "http://prediction-service:5000/explain"
	if url, ok := config["predictionServiceURL"].(string); ok && url != "" {
		// Replace /predict with /explain
		// Simplified string manipulation for demo
		serviceURL = url + "/../explain"
	}

	// Extract features from state (same as Predict activity)
	features := MLPredictionRequest{
		LineItemCount:             getInt(state, "line_item_count"),
		IsCrossBorder:             getInt(state, "is_cross_border"),
		OrderToShipDays:           getFloat(state, "order_to_ship_days"),
		CustomerCountry:           getString(state, "customer_country"),
		CustomerTradeHistoryCount: getInt(state, "customer_trade_history_count"),
		CustomerPreviousFails:     getInt(state, "customer_previous_fails"),
		IsMissingPostalCode:       getInt(state, "is_missing_postal_code"),
		IsMissingShipDate:         getInt(state, "is_missing_ship_date"),
		IsMissingAddress:          getInt(state, "is_missing_address"),
		OrderFreightCost:          getFloat(state, "order_freight_cost"),
		OrderTotalValue:           getFloat(state, "order_total_value"),
		ShipperID:                 getInt(state, "shipper_id"),
		OrderDayOfWeek:            getInt(state, "order_day_of_week"),
		OrderMonth:                getInt(state, "order_month"),
		DaysUntilRequired:         getFloat(state, "days_until_required"),
	}

	reqBody, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal features: %w", err)
	}

	resp, err := http.Post(serviceURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call explanation service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("explanation service returned error: %d", resp.StatusCode)
	}

	var explanationResp RiskExplanationResponse
	if err := json.NewDecoder(resp.Body).Decode(&explanationResp); err != nil {
		return nil, fmt.Errorf("failed to decode explanation response: %w", err)
	}

	return &explanationResp, nil
}

// Helper functions for type conversion
func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// fallbackRiskScore provides rule-based scoring when ML service is unavailable
func fallbackRiskScore(features MLPredictionRequest) map[string]interface{} {
	score := 0.05 // Base risk

	if features.IsCrossBorder == 1 {
		score += 0.1
	}
	if features.CustomerPreviousFails > 0 {
		score += float64(features.CustomerPreviousFails) * 0.05
	}
	if features.IsMissingPostalCode == 1 {
		score += 0.05
	}
	if features.OrderTotalValue > 10000 {
		score += 0.1
	}
	if features.CustomerTradeHistoryCount < 5 {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	category := "LOW"
	if score >= 0.75 {
		category = "CRITICAL"
	} else if score >= 0.5 {
		category = "HIGH"
	} else if score >= 0.25 {
		category = "MEDIUM"
	}

	return map[string]interface{}{
		"settlement_risk_score": score,
		"risk_category":         category,
		"model_version":         "fallback-v1",
		"using_fallback":        true,
	}
}
