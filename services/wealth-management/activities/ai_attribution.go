package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AIAttribution performs AI-powered performance attribution for portfolios
func AIAttribution(ctx context.Context, portfolioID string) (map[string]any, error) {
	// Call xAI API for performance attribution analysis
	resp, err := http.Post("https://api.x.ai/v1/chat/completions", "application/json",
		strings.NewReader(fmt.Sprintf(`{
			"model": "grok-beta",
			"messages": [{
				"role": "user",
				"content": "Attribute performance for portfolio %s: Brinson-Fachler, sector, security, interaction, currency, ESG impact."
			}]
		}`, portfolioID)))
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
