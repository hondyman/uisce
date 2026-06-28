package main

import (
	"context"
	"encoding/json"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// AI Attribution Activity (xAI)
func (a *RebalanceActivities) AIAttribution(ctx context.Context, portfolioID string) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Attributing performance for portfolio", "portfolioID", portfolioID)

	prompt := fmt.Sprintf(`Attribute performance for portfolio %s using the Brinson-Fachler model. Your response must be a valid JSON object with the keys "alpha" (float), "sector_attribution" (map of sector to float), and "summary" (string). Example: {"alpha": 1.2, "sector_attribution": {"Technology": 0.8, "Healthcare": 0.4}, "summary": "Outperformance driven by strong stock selection in Technology."}.`, portfolioID)

	response, err := a.callXAI(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI attribution response: %w", err)
	}

	return result, nil
}

// Execute Attribution Activity (Placeholder)
func (a *RebalanceActivities) ExecuteAttribution(ctx context.Context, attrResult map[string]interface{}) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing attribution", "alpha", attrResult["alpha"])
	// In a real implementation, this might involve generating detailed reports or persisting model results.
	return nil
}

// Update Attribution Status Activity
func (a *RebalanceActivities) UpdateAttributionStatus(ctx context.Context, portfolioID string, attrResult map[string]interface{}) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating attribution status in Hasura", "portfolioID", portfolioID)

	sectorAttributionJSON, err := json.Marshal(attrResult["sector_attribution"])
	if err != nil {
		return fmt.Errorf("failed to marshal sector attribution: %w", err)
	}

	mutation := `
		mutation UpdateAttribution($id: uuid!, $alpha: numeric!, $sector: jsonb!, $summary: String!) {
			update_portfolios_by_pk(pk_columns: {id: $id}, _set: {alpha: $alpha, sector_attribution: $sector, rebalance_status: $summary}) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"id":      portfolioID,
		"alpha":   attrResult["alpha"],
		"sector":  string(sectorAttributionJSON),
		"summary": attrResult["summary"],
	}

	return a.hasuraMutate(ctx, mutation, variables)
}
