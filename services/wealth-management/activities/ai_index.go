package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AIIndexOptimize performs advanced AI-powered direct indexing optimization
func AIIndexOptimize(ctx context.Context, indexID string) (map[string]any, error) {
	// Call xAI API for comprehensive direct indexing optimization analysis
	resp, err := http.Post("https://api.x.ai/v1/chat/completions", "application/json",
		strings.NewReader(fmt.Sprintf(`{
			"model": "grok-beta",
			"messages": [{
				"role": "user",
				"content": "Optimize direct index %s: holdings, drift minimization, tax lots, ESG alignment, cash flow forecasting, household impact. Provide detailed rebalancing recommendations, tax efficiency calculations, and ESG scoring."
			}]
		}`, indexID)))
	if err != nil {
		return nil, fmt.Errorf("failed to call xAI API: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode xAI response: %w", err)
	}

	return result, nil
}
