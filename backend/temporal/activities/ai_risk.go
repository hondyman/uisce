package activities

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// AIRiskScore is an activity that calls an AI service to score portfolio risk.
func AIRiskScore(ctx context.Context, portfolioID string) (map[string]any, error) {
	resp, _ := http.Post("https://api.x.ai/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"grok-beta","messages":[{"role":"user","content":"Score risk for portfolio `+portfolioID+`: VaR, concentration, ESG, liquidity, counterparty, geopolitical."}]}`))
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}
