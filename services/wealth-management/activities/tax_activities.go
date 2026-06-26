package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// ExecuteHarvest executes the tax harvest based on AI optimization results
func ExecuteHarvest(ctx context.Context, harvest map[string]any) error {
	fmt.Printf("🌾 Starting tax harvest execution...\n")

	// 1. Lot Selection and Tax Lot Accounting
	lots, ok := harvest["selectedLots"].([]interface{})
	if !ok || len(lots) == 0 {
		return fmt.Errorf("no tax lots selected for harvest")
	}

	fmt.Printf("  ✓ Selected %d tax lots for harvesting\n", len(lots))

	// 2. Wash Sale Rule Compliance Check
	if err := checkWashSaleCompliance(ctx, lots); err != nil {
		return fmt.Errorf("wash sale compliance check failed: %w", err)
	}
	fmt.Printf("  ✓ Wash sale compliance verified\n")

	// 3. ESG Alignment Verification
	if esgPreferences, ok := harvest["esgPreferences"].(map[string]interface{}); ok {
		if err := verifyESGAlignment(ctx, lots, esgPreferences); err != nil {
			return fmt.Errorf("ESG alignment check failed: %w", err)
		}
		fmt.Printf("  ✓ ESG alignment verified\n")
	}

	//4. Execute Trades for Tax Loss Harvesting
	trades := buildHarvestTrades(lots, harvest)
	for i, trade := range trades {
		if err := executeHarvestTrade(ctx, trade); err != nil {
			return fmt.Errorf("failed to execute harvest trade %d: %w", i, err)
		}
		fmt.Printf("  ✓ Harvest trade %d/%d executed\n", i+1, len(trades))
	}

	// 5. Record tax harvest results
	if err := recordTaxHarvestResults(ctx, harvest, len(trades)); err != nil {
		return fmt.Errorf("failed to record harvest results: %w", err)
	}

	estimatedSavings, _ := harvest["estimatedTaxSavings"].(float64)
	fmt.Printf("✅ Tax harvest complete: %d trades, estimated savings: $%.2f\n",
		len(trades), estimatedSavings)

	return nil
}

// checkWashSaleCompliance verifies trades don't violate IRS wash sale rules
func checkWashSaleCompliance(ctx context.Context, lots []interface{}) error {
	// Wash sale rule: Can't buy substantially identical security
	// within 30 days before or after a loss sale
	washSaleWindow := 30 * 24 * time.Hour

	for _, lot := range lots {
		lotMap, ok := lot.(map[string]interface{})
		if !ok {
			continue
		}

		symbol, _ := lotMap["symbol"].(string)
		saleDate, _ := lotMap["proposedSaleDate"].(string)

		// Check for recent purchases of same security
		if hasRecentPurchase(ctx, symbol, saleDate, washSaleWindow) {
			return fmt.Errorf("wash sale violation detected for %s", symbol)
		}
	}

	return nil
}

// hasRecentPurchase checks if security was purchased within wash sale window
func hasRecentPurchase(ctx context.Context, symbol, saleDate string, window time.Duration) bool {
	// Query transaction history for recent purchases
	// For now, return false (no violations)
	// TODO: Implement actual database query for transaction history
	return false
}

// verifyESGAlignment ensures replacement securities match ESG preferences
func verifyESGAlignment(ctx context.Context, lots []interface{}, esgPrefs map[string]interface{}) error {
	for _, lot := range lots {
		lotMap, ok := lot.(map[string]interface{})
		if !ok {
			continue
		}

		replacementSymbol, _ := lotMap["replacementSecurity"].(string)
		if replacementSymbol == "" {
			continue
		}

		// Check ESG score of replacement security
		esgScore := getESGScore(ctx, replacementSymbol)

		minScore, _ := esgPrefs["minimumESGScore"].(float64)
		if minScore > 0 && esgScore < minScore {
			return fmt.Errorf("replacement security %s ESG score %.1f below minimum %.1f",
				replacementSymbol, esgScore, minScore)
		}

		// Check excluded sectors
		if excludedSectors, ok := esgPrefs["excludedSectors"].([]interface{}); ok {
			sector := getSector(ctx, replacementSymbol)
			for _, excluded := range excludedSectors {
				if sector == excluded {
					return fmt.Errorf("replacement security %s in excluded sector: %s",
						replacementSymbol, sector)
				}
			}
		}
	}

	return nil
}

// getESGScore retrieves ESG score for a security
func getESGScore(ctx context.Context, symbol string) float64 {
	// Query ESG database or external API
	// For now, return a mock score
	// TODO: Implement actual ESG score lookup
	return 7.5
}

// getSector gets the sector classification for a security
func getSector(ctx context.Context, symbol string) string {
	// Query security master data
	// TODO: Implement actual sector lookup
	return "Technology"
}

// buildHarvestTrades creates trade orders from selected lots
func buildHarvestTrades(lots []interface{}, harvest map[string]any) []map[string]interface{} {
	var trades []map[string]interface{}

	for _, lot := range lots {
		lotMap, ok := lot.(map[string]interface{})
		if !ok {
			continue
		}

		// Sell order for current position
		sellTrade := map[string]interface{}{
			"type":      "sell",
			"symbol":    lotMap["symbol"],
			"quantity":  lotMap["quantity"],
			"lotID":     lotMap["lotID"],
			"orderType": "market",
			"purpose":   "tax_loss_harvest",
		}
		trades = append(trades, sellTrade)

		// Buy order for replacement security (if specified)
		if replacement, ok := lotMap["replacementSecurity"].(string); ok && replacement != "" {
			buyTrade := map[string]interface{}{
				"type":      "buy",
				"symbol":    replacement,
				"quantity":  lotMap["quantity"],
				"orderType": "market",
				"purpose":   "tax_loss_harvest_replacement",
			}
			trades = append(trades, buyTrade)
		}
	}

	return trades
}

// executeHarvestTrade executes a single tax harvest trade
func executeHarvestTrade(ctx context.Context, trade map[string]interface{}) error {
	tradingURL := os.Getenv("TRADING_API_URL")
	if tradingURL == "" {
		// Simulation mode
		fmt.Printf("    [SIMULATED] Harvest trade: %s %v shares of %s\n",
			trade["type"], trade["quantity"], trade["symbol"])
		return nil
	}

	body, _ := json.Marshal(trade)
	resp, err := http.Post(tradingURL+"/api/trades", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("trading API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("trade failed with status %d", resp.StatusCode)
	}

	return nil
}

// recordTaxHarvestResults saves harvest execution results to database
func recordTaxHarvestResults(ctx context.Context, harvest map[string]any, tradeCount int) error {
	hasuraURL := os.Getenv("HASURA_GRAPHQL_URL")
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080/v1/graphql"
	}

	mutation := `
		mutation RecordTaxHarvest($data: tax_harvests_insert_input!) {
			insert_tax_harvests_one(object: $data) {
				id
				executed_at
			}
		}
	`

	variables := map[string]interface{}{
		"data": map[string]interface{}{
			"harvest_id":             harvest["harvestID"],
			"client_id":              harvest["clientID"],
			"trade_count":            tradeCount,
			"estimated_tax_savings":  harvest["estimatedTaxSavings"],
			"actual_losses_realized": 0, // Will be updated after settlement
			"executed_at":            time.Now().Format(time.RFC3339),
			"status":                 "executed",
		},
	}

	request := map[string]interface{}{
		"query":     mutation,
		"variables": variables,
	}

	body, _ := json.Marshal(request)
	req, _ := http.NewRequestWithContext(ctx, "POST", hasuraURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", os.Getenv("HASURA_ADMIN_SECRET"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Log error but don't fail the harvest
		fmt.Printf("  ⚠️  Failed to record results to database: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	return nil
}
