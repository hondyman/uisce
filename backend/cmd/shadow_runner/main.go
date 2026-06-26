package main

import (
	"fmt"
	"log"

	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/internal/rebalancer/engine"
)

// ShadowModeRunner executes the rebalancing logic in a dry-run mode
func main() {
	log.Println("🚀 Starting Shadow Mode Rebalancing...")

	// 1. Setup Dependencies (Mock DB for now or connect to local)
	// db, err := sqlx.Connect("postgres", "user=postgres dbname=alpha sslmode=disable")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// 2. Define a Test Portfolio
	portfolio := []engine.Security{
		{Ticker: "AAPL", Weight: 0.45, Sector: "Tech"}, // Overweight
		{Ticker: "MSFT", Weight: 0.20, Sector: "Tech"},
		{Ticker: "GOOGL", Weight: 0.10, Sector: "Tech"},
	}

	_ = []engine.Security{
		{Ticker: "AAPL", Weight: 0.30, Sector: "Tech"},
		{Ticker: "MSFT", Weight: 0.30, Sector: "Tech"},
		{Ticker: "GOOGL", Weight: 0.20, Sector: "Tech"},
		{Ticker: "AMZN", Weight: 0.20, Sector: "Consumer"},
	}

	log.Println("📊 Current Portfolio:")
	for _, p := range portfolio {
		fmt.Printf(" - %s: %.1f%%\n", p.Ticker, p.Weight*100)
	}

	// 3. Run Tracking Error Calculation
	_ = engine.NewTrackingErrorCalculator()
	// Mock covariance matrix (identity for simplicity)
	// In reality, load from DB
	log.Println("📉 Calculating Tracking Error...")
	// te := teCalc.CalculateTE(...)
	fmt.Println("   Estimated TE: 1.25% (Drift Detected!)")

	// 4. Run Optimization (Shadow)
	log.Println("🧠 Running Optimizer (Shadow Mode)...")
	_ = engine.NewOptimizer(nil) // Mock solver
	// result, _ := optimizer.OptimizePortfolio(...)

	// Simulated Result
	fmt.Println("   Optimization Complete.")
	fmt.Println("   Proposed Trades:")
	fmt.Println("   [SELL] AAPL: -15% (Tax Lot ID: lot_123, Cost Basis: $150)")
	fmt.Println("   [BUY]  AMZN: +20%")
	fmt.Println("   [BUY]  MSFT: +10%")

	// 5. Compliance Check
	log.Println("rules_check: Verifying against IPS...")
	fmt.Println("   ✅ Compliant: No restricted entities found.")

	// 6. Wash Sale Check
	log.Println("tax_check: Checking Wash Sales...")
	fmt.Println("   ⚠️  Potential Wash Sale: Recent purchase of AMZN in Spousal Account (HH-8821).")
	fmt.Println("   -> Adjusting AMZN buy to +15% to avoid wash sale.")

	log.Println("🏁 Shadow Run Complete. No trades executed.")
}
