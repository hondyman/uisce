package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/marketdata"
)

func main() {
	fmt.Println("🚀 Starting Shadow Mode Rebalancing Cycle...")

	// 1. Initialize Market Data Provider
	provider := marketdata.NewMockProvider()
	ctx := context.Background()

	// 2. Simulate Portfolio Analysis
	portfolio := []string{"AAPL", "GOOG", "MSFT", "TSLA", "SPY"}
	fmt.Printf("📊 Analyzing Portfolio: %v\n", portfolio)

	totalValue := 0.0
	for _, symbol := range portfolio {
		quote, err := provider.GetLatestPrice(ctx, symbol)
		if err != nil {
			log.Printf("Failed to get price for %s: %v", symbol, err)
			continue
		}
		// Assume 10 shares of each
		value := quote.Price * 10
		totalValue += value
		fmt.Printf("   - %s: $%.2f (Timestamp: %s)\n", symbol, quote.Price, quote.Timestamp.Format(time.RFC3339))
	}
	fmt.Printf("💰 Total Portfolio Value: $%.2f\n", totalValue)

	// 3. Simulate Drift Detection (Mock)
	fmt.Println("🔍 Checking for Drift...")
	// Mock target allocation: Equal weight (20% each)
	// Mock drift logic: If any asset deviates > 5%
	driftDetected := false
	if totalValue > 0 {
		for _, symbol := range portfolio {
			quote, _ := provider.GetLatestPrice(ctx, symbol)
			currentWeight := (quote.Price * 10) / totalValue
			targetWeight := 0.20
			diff := currentWeight - targetWeight
			if diff > 0.05 || diff < -0.05 {
				fmt.Printf("   ⚠️ Drift Detected for %s: Current Weight %.1f%% (Target 20%%)\n", symbol, currentWeight*100)
				driftDetected = true
			}
		}
	}

	if !driftDetected {
		fmt.Println("✅ Portfolio is balanced.")
	} else {
		fmt.Println("⚡ Generating Rebalancing Trades (Shadow Mode - No Execution)...")
		// Mock trade generation
		fmt.Println("   - SELL AAPL 2 shares")
		fmt.Println("   - BUY TSLA 1 share")
	}

	fmt.Println("🏁 Shadow Mode Cycle Complete.")
}
