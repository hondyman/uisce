package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ABACCheck performs ABAC authorization check for UMA rebalance
func ABACCheck(ctx context.Context, action, resourceType, resourceID string) (bool, error) {
	// Call ABAC governance service for permission check
	abacURL := os.Getenv("ABAC_SERVICE_URL")
	if abacURL == "" {
		abacURL = "http://localhost:8083" // Default governance service URL
	}

	request := map[string]interface{}{
		"action":       action,
		"resourceType": resourceType,
		"resourceID":   resourceID,
		"tenantID":     getTenantFromContext(ctx),
		"userID":       getUserFromContext(ctx),
	}

	body, _ := json.Marshal(request)
	resp, err := http.Post(abacURL+"/api/abac/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		// If ABAC service is unavailable, log and allow (fail open for now)
		fmt.Printf("ABAC service unavailable, granting permission: %v\n", err)
		return true, nil
	}
	defer resp.Body.Close()

	var result struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode ABAC response: %w", err)
	}

	fmt.Printf("ABAC Check: action=%s, resource=%s:%s, allowed=%v, reason=%s\n",
		action, resourceType, resourceID, result.Allowed, result.Reason)

	return result.Allowed, nil
}

// ExecuteTrades executes the trades based on the harvest plan
func ExecuteTrades(ctx context.Context, harvest map[string]any) error {
	// Extract trade details from harvest plan
	trades, ok := harvest["trades"].([]interface{})
	if !ok {
		return fmt.Errorf("no trades found in harvest plan")
	}

	fmt.Printf("Executing %d trades for UMA rebalance\n", len(trades))

	for i, trade := range trades {
		tradeMap, ok := trade.(map[string]interface{})
		if !ok {
			continue
		}

		// Execute trade via trading API
		err := executeSingleTrade(ctx, tradeMap)
		if err != nil {
			return fmt.Errorf("failed to execute trade %d: %w", i, err)
		}

		fmt.Printf("  ✓ Trade %d executed: %v\n", i+1, tradeMap)
	}

	return nil
}

// executeSingleTrade executes a single trade
func executeSingleTrade(ctx context.Context, trade map[string]interface{}) error {
	tradingURL := os.Getenv("TRADING_API_URL")
	if tradingURL == "" {
		// If no trading API configured, just log the trade
		fmt.Printf("    [SIMULATED] Trade: %+v\n", trade)
		return nil
	}

	body, _ := json.Marshal(trade)
	resp, err := http.Post(tradingURL+"/api/trades", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("trading API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("trade execution failed with status %d", resp.StatusCode)
	}

	return nil
}

// HasuraUpdate updates the Hasura database with rebalance results
func HasuraUpdate(ctx context.Context, update map[string]any) error {
	hasuraURL := os.Getenv("HASURA_GRAPHQL_URL")
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080/v1/graphql" // Default Hasura URL
	}

	hasuraSecret := os.Getenv("HASURA_ADMIN_SECRET")
	if hasuraSecret == "" {
		hasuraSecret = "your-secret-key"
	}

	// Build GraphQL mutation based on update type
	updateType, _ := update["type"].(string)
	entityID, _ := update["entityID"].(string)

	var mutation string
	var variables map[string]interface{}

	switch updateType {
	case "rebalance_complete":
		mutation = `
			mutation UpdateRebalance($id: uuid!, $status: String!, $results: jsonb!) {
				update_rebalance_executions_by_pk(
					pk_columns: {id: $id},
					_set: {status: $status, results: $results, completed_at: "now()"}
				) {
					id
					status
				}
			}
		`
		variables = map[string]interface{}{
			"id":      entityID,
			"status":  "completed",
			"results": update["results"],
		}

	case "portfolio_update":
		mutation = `
			mutation UpdatePortfolio($id: uuid!, $holdings: jsonb!) {
				update_portfolios_by_pk(
					pk_columns: {id: $id},
					_set: {holdings: $holdings, updated_at: "now()"}
				) {
					id
					updated_at
				}
			}
		`
		variables = map[string]interface{}{
			"id":       entityID,
			"holdings": update["holdings"],
		}

	default:
		return fmt.Errorf("unknown update type: %s", updateType)
	}

	request := map[string]interface{}{
		"query":     mutation,
		"variables": variables,
	}

	body, _ := json.Marshal(request)
	req, _ := http.NewRequestWithContext(ctx, "POST", hasuraURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", hasuraSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Hasura request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode Hasura response: %w", err)
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("Hasura mutation error: %s", result.Errors[0].Message)
	}

	fmt.Printf("Hasura updated successfully: type=%s, entityID=%s\n", updateType, entityID)
	return nil
}

// Helper functions to extract context values
func getTenantFromContext(ctx context.Context) string {
	if tenantID := ctx.Value("tenantID"); tenantID != nil {
		if tid, ok := tenantID.(string); ok {
			return tid
		}
	}
	return "default"
}

func getUserFromContext(ctx context.Context) string {
	if userID := ctx.Value("userID"); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return "system"
}
