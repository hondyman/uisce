package activities

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// AIScenario is an activity that calls an AI service to analyze a portfolio under a given scenario.
func AIScenario(ctx context.Context, portfolioID string, scenario string) (map[string]any, error) {
	prompt := `Analyze portfolio ` + portfolioID + ` under the following scenario: "` + scenario + `". Provide projected AUM, Sharpe ratio, risk score, and asset allocation.`
	resp, _ := http.Post("https://api.x.ai/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"grok-beta","messages":[{"role":"user","content":"`+prompt+`"}]}`))
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}
