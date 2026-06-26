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

// ExecuteAttribution executes the performance attribution analysis
func ExecuteAttribution(ctx context.Context, attr map[string]any) error {
	fmt.Printf("📊 Starting performance attribution analysis...\n")

	// Extract attribution parameters
	portfolioID, _ := attr["portfolioID"].(string)
	startDate, _ := attr["startDate"].(string)
	endDate, _ := attr["endDate"].(string)

	if portfolioID == "" || startDate == "" || endDate == "" {
		return fmt.Errorf("missing required attribution parameters")
	}

	// 1. Calculate Total Return
	totalReturn, err := calculateTotalReturn(ctx, portfolioID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to calculate total return: %w", err)
	}
	fmt.Printf("  ✓ Total return: %.2f%%\n", totalReturn*100)

	// 2. Calculate Benchmark Return
	benchmarkID, _ := attr["benchmarkID"].(string)
	benchmarkReturn, err := calculateBenchmarkReturn(ctx, benchmarkID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to calculate benchmark return: %w", err)
	}
	fmt.Printf("  ✓ Benchmark return: %.2f%%\n", benchmarkReturn*100)

	// 3. Perform Attribution Analysis
	attributionResults := performBrinsonAttribution(ctx, portfolioID, benchmarkID, startDate, endDate)

	// 4. Calculate Factor Contributions
	factorContributions := calculateFactorContributions(ctx, portfolioID, startDate, endDate)

	// 5. Store Attribution Results
	results := map[string]interface{}{
		"portfolioID":         portfolioID,
		"startDate":           startDate,
		"endDate":             endDate,
		"totalReturn":         totalReturn,
		"benchmarkReturn":     benchmarkReturn,
		"activeReturn":        totalReturn - benchmarkReturn,
		"allocationEffect":    attributionResults["allocation"],
		"selectionEffect":     attributionResults["selection"],
		"interactionEffect":   attributionResults["interaction"],
		"factorContributions": factorContributions,
		"calculatedAt":        time.Now().Format(time.RFC3339),
	}

	if err := storeAttributionResults(ctx, results); err != nil {
		return fmt.Errorf("failed to store attribution results: %w", err)
	}

	activeReturn := totalReturn - benchmarkReturn
	fmt.Printf("✅ Attribution complete: Active return: %.2f%%, Allocation: %.2f%%, Selection: %.2f%%\n",
		activeReturn*100,
		attributionResults["allocation"].(float64)*100,
		attributionResults["selection"].(float64)*100)

	return nil
}

// calculateTotalReturn computes the time-weighted return for the portfolio
func calculateTotalReturn(ctx context.Context, portfolioID, startDate, endDate string) (float64, error) {
	// Query portfolio valuations at start and end dates
	// Calculate time-weighted return accounting for cash flows
	// Formula: TWR = (End Value / Begin Value)^(1/years) - 1

	// TODO: Implement actual valuation query from database
	// For now, return a mock return
	return 0.0847, nil // 8.47% return
}

// calculateBenchmarkReturn computes the return for the benchmark index
func calculateBenchmarkReturn(ctx context.Context, benchmarkID, startDate, endDate string) (float64, error) {
	// Query benchmark index values
	// TODO: Implement actual benchmark data query
	return 0.0723, nil // 7.23% return
}

// performBrinsonAttribution performs Brinson-Fachler attribution analysis
func performBrinsonAttribution(ctx context.Context, portfolioID, benchmarkID, startDate, endDate string) map[string]interface{} {
	// Brinson Attribution decomposes excess return into:
	// 1. Allocation Effect: Return from sector/asset class weights
	// 2. Selection Effect: Return from security selection within sectors
	// 3. Interaction Effect: Combined effect of allocation and selection

	// TODO: Implement actual Brinson attribution calculation with:
	// - Portfolio and benchmark sector weights
	// - Portfolio and benchmark sector returns
	// - Calculate weighted contributions

	// Mock results
	return map[string]interface{}{
		"allocation":  0.0045,  // +0.45% from allocation
		"selection":   0.0079,  // +0.79% from selection
		"interaction": -0.0001, // -0.01% interaction
	}
}

// calculateFactorContributions analyzes return attribution by risk factors
func calculateFactorContributions(ctx context.Context, portfolioID, startDate, endDate string) map[string]interface{} {
	// Factor attribution uses regression analysis to attribute returns to:
	// - Market factor (Beta)
	// - Size factor (SMB - Small Minus Big)
	// - Value factor (HML - High Minus Low)
	// - Momentum factor
	// - Quality factor
	// - Low volatility factor

	// TODO: Implement actual factor analysis with:
	// - Factor exposures calculated via regression
	// - Factor returns for the period
	// - Contribution = Exposure × Factor Return

	// Mock factor contributions
	return map[string]interface{}{
		"market":        0.0523,  // 5.23% from market beta
		"size":          0.0012,  // 0.12% from size tilt
		"value":         -0.0008, // -0.08% from value tilt
		"momentum":      0.0145,  // 1.45% from momentum
		"quality":       0.0089,  // 0.89% from quality
		"lowVolatility": 0.0031,  // 0.31% from low vol
		"idiosyncratic": 0.0055,  // 0.55% stock-specific
	}
}

// storeAttributionResults saves attribution analysis to database via Hasura
func storeAttributionResults(ctx context.Context, results map[string]interface{}) error {
	hasuraURL := os.Getenv("HASURA_GRAPHQL_URL")
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080/v1/graphql"
	}

	mutation := `
		mutation StoreAttribution($data: performance_attribution_insert_input!) {
			insert_performance_attribution_one(object: $data) {
				id
				calculated_at
			}
		}
	`

	variables := map[string]interface{}{
		"data": results,
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
		// Log error but don't fail the attribution
		fmt.Printf("  ⚠️  Failed to store attribution results: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("Hasura error: %s", result.Errors[0].Message)
	}

	fmt.Printf("  ✓ Attribution results stored to database\n")
	return nil
}
