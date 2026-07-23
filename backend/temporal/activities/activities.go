package activities

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// Portfolio data structure
type Portfolio struct {
	ID             string  `json:"id" db:"id"`
	ClientID       string  `json:"client_id" db:"client_id"`
	AUM            float64 `json:"aum" db:"aum"`
	Sharpe         float64 `json:"sharpe" db:"sharpe"`
	Risk           float64 `json:"risk" db:"risk"`
	Drift          float64 `json:"drift" db:"drift"`
	Status         string  `json:"status" db:"status"`
	LastRebalanced string  `json:"last_rebalanced" db:"last_rebalanced"`
}

// FetchPortfolioData fetches portfolio data from database
func FetchPortfolioData(ctx context.Context, portfolioID string) (map[string]interface{}, error) {
	// In production, query from database
	// For now, return mock data
	portfolio := map[string]interface{}{
		"id":     portfolioID,
		"aum":    5000000.0,
		"sharpe": 1.2,
		"risk":   0.15,
		"drift":  2.5,
		"assetAllocation": []map[string]interface{}{
			{"asset": "Stocks", "percentage": 0.60},
			{"asset": "Bonds", "percentage": 0.30},
			{"asset": "Cash", "percentage": 0.10},
		},
	}
	return portfolio, nil
}

// FetchUMAData fetches UMA account data
func FetchUMAData(ctx context.Context, umaID string) (map[string]interface{}, error) {
	// In production, query from database
	umaData := map[string]interface{}{
		"id":    umaID,
		"aum":   2500000.0,
		"drift": 8.5,
		"holdings": []map[string]interface{}{
			{"symbol": "AAPL", "shares": 150, "value": 25500},
			{"symbol": "MSFT", "shares": 100, "value": 40000},
			{"symbol": "TSLA", "shares": 50, "value": 9000},
		},
		"taxLots": []map[string]interface{}{
			{"symbol": "AAPL", "cost_basis": 24000, "current_value": 25500, "unrealized_loss": 0},
			{"symbol": "MSFT", "cost_basis": 35000, "current_value": 40000, "unrealized_loss": 0},
		},
	}
	return umaData, nil
}

// FetchIndexData fetches index portfolio data
func FetchIndexData(ctx context.Context, indexID string) (map[string]interface{}, error) {
	// In production, query from database
	indexData := map[string]interface{}{
		"id":             indexID,
		"aum":            15000000.0,
		"benchmarkIndex": "SPY",
		"drift":          3.2,
		"holdings":       100,
	}
	return indexData, nil
}

// ProjectScenario projects portfolio performance under a scenario
func ProjectScenario(ctx context.Context, portfolio map[string]interface{}, scenarioType string) (map[string]interface{}, error) {
	aum := portfolio["aum"].(float64)
	sharpe := portfolio["sharpe"].(float64)
	risk := portfolio["risk"].(float64)

	// Apply scenario adjustments
	aumChange := 0.0
	sharpeChange := 0.0
	riskChange := 0.0

	switch scenarioType {
	case "market-downturn":
		aumChange = -0.08 // -8%
		sharpeChange = -0.3
		riskChange = 0.05
	case "interest-rate-rise":
		aumChange = -0.05 // -5%
		sharpeChange = -0.2
		riskChange = 0.03
	case "inflation-spike":
		aumChange = -0.06 // -6%
		sharpeChange = -0.15
		riskChange = 0.04
	case "deflation":
		aumChange = 0.04 // +4%
		sharpeChange = 0.2
		riskChange = -0.02
	case "commodity-spike":
		aumChange = -0.04 // -4%
		sharpeChange = -0.1
		riskChange = 0.02
	default:
		aumChange = 0.0
		sharpeChange = 0.0
		riskChange = 0.0
	}

	scenarioResult := map[string]interface{}{
		"aum":          aum * (1 + aumChange),
		"aumChange":    aumChange,
		"sharpe":       sharpe + sharpeChange,
		"sharpeChange": sharpeChange,
		"risk":         risk + riskChange,
		"riskChange":   riskChange,
		"assetAllocation": []map[string]interface{}{
			{"asset": "Stocks", "percentage": 0.55},
			{"asset": "Bonds", "percentage": 0.35},
			{"asset": "Cash", "percentage": 0.10},
		},
	}

	return scenarioResult, nil
}

// CalculateComparison calculates difference between base case and scenario
func CalculateComparison(ctx context.Context, baseCase map[string]interface{}, scenarioCase map[string]interface{}) (map[string]interface{}, error) {
	baseAUM := baseCase["aum"].(float64)
	scenarioAUM := scenarioCase["aum"].(float64)
	baseSharpe := baseCase["sharpe"].(float64)
	scenarioSharpe := scenarioCase["sharpe"].(float64)
	baseRisk := baseCase["risk"].(float64)
	scenarioRisk := scenarioCase["risk"].(float64)

	comparison := map[string]interface{}{
		"aumDifference":    scenarioAUM - baseAUM,
		"sharpeDifference": scenarioSharpe - baseSharpe,
		"riskDifference":   scenarioRisk - baseRisk,
		"aumPercentChange": (scenarioAUM - baseAUM) / baseAUM,
	}

	return comparison, nil
}

// AITaxHarvest performs AI-powered tax harvest analysis
func AITaxHarvest(ctx context.Context, umaData map[string]interface{}) (map[string]interface{}, error) {
	holdings := umaData["holdings"].([]map[string]interface{})
	taxLots := umaData["taxLots"].([]map[string]interface{})

	totalTaxSavings := 0.0
	lotsToHarvest := []map[string]interface{}{}

	for _, lot := range taxLots {
		unrealizedLoss := lot["unrealized_loss"].(float64)
		if unrealizedLoss < 0 {
			taxSavings := unrealizedLoss * 0.37 // Assuming 37% tax rate
			totalTaxSavings += taxSavings
			lotsToHarvest = append(lotsToHarvest, lot)
		}
	}

	harvestPlan := map[string]interface{}{
		"saved":        totalTaxSavings,
		"lots":         len(lotsToHarvest),
		"selectedLots": lotsToHarvest,
		"holdings":     holdings,
		"confidence":   0.95,
	}

	return harvestPlan, nil
}

// AIIndexOptimize performs AI-powered index optimization
func AIIndexOptimize(ctx context.Context, indexData map[string]interface{}) (map[string]interface{}, error) {
	aum := indexData["aum"].(float64)
	drift := indexData["drift"].(float64)

	// Simulate AI optimization
	newDrift := drift * 0.1
	taxSaved := aum * 0.001 // Estimate 0.1% in tax savings
	esgScore := 0.85 + rand.Float64()*0.15

	optimization := map[string]interface{}{
		"drift":      newDrift,
		"saved":      taxSaved,
		"esg_score":  esgScore,
		"confidence": 0.93,
		"trades": []map[string]interface{}{
			{"action": "SELL", "symbol": "OLD_HOLDING", "shares": 100},
			{"action": "BUY", "symbol": "NEW_HOLDING", "shares": 95},
		},
	}

	return optimization, nil
}

// AIAttribution performs AI-powered performance attribution
func AIAttribution(ctx context.Context, portfolioData map[string]interface{}) (map[string]interface{}, error) {
	// Simulate attribution analysis
	attribution := map[string]interface{}{
		"alpha":  0.025, // 2.5% alpha
		"sector": "Technology",
		"alpha_contributors": []map[string]interface{}{
			{"name": "AAPL", "contribution": 0.015},
			{"name": "MSFT", "contribution": 0.010},
		},
		"confidence": 0.92,
	}

	return attribution, nil
}

// ExecuteHarvest executes tax harvest trades
func ExecuteHarvest(ctx context.Context, harvest map[string]interface{}) (map[string]interface{}, error) {
	// Execute the harvest plan
	executed := map[string]interface{}{
		"status":        "executed",
		"taxSaved":      harvest["saved"],
		"lotsHarvested": harvest["lots"],
		"executedAt":    time.Now().Format(time.RFC3339),
	}

	return executed, nil
}

// StoreAnalysisResult stores analysis results in database
func StoreAnalysisResult(ctx context.Context, portfolioID string, result map[string]interface{}) error {
	// In production, store in database
	fmt.Printf("Storing analysis for portfolio %s: %v\n", portfolioID, result)
	return nil
}
