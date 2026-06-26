package main

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/governance"
	"github.com/hondyman/semlayer/backend/pkg/workflows"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   TITAN PRE-TRADE COMPLIANCE DEMO")
	fmt.Println("==================================================")

	// Initialize Engine
	engine := governance.NewGovernanceEngine(nil)

	// Test Case 1: Safe Trade
	safeTrade := map[string]interface{}{
		"symbol":   "AAPL",
		"amount":   500000, // < 1M
		"exposure": 100000,
	}
	fmt.Printf("[Test 1] Validating Safe Trade (Amount: %v)...\n", safeTrade["amount"])
	res1, err := engine.ValidateTransaction(context.Background(), "default-tenant", safeTrade)
	printResult(res1, err)

	// Test Case 2: High Value Trade (Blocked)
	start := map[string]interface{}{
		"symbol":              "GOOGL",
		"amount":              1500000, // > 1M
		"compliance_approved": false,
	}
	fmt.Printf("\n[Test 2] Validating High-Value Trade (Amount: %v)...\n", start["amount"])
	res2, err := engine.ValidateTransaction(context.Background(), "default-tenant", start)
	printResult(res2, err)

	// Test Case 3: Restricted Symbol (Blocked)
	restricted := map[string]interface{}{
		"symbol": "BAD",
		"amount": 100,
	}
	fmt.Printf("\n[Test 3] Validating Restricted Symbol (Symbol: %v)...\n", restricted["symbol"])
	res3, err := engine.ValidateTransaction(context.Background(), "default-tenant", restricted)
	printResult(res3, err)

	// 4. Graph Execution Demo (Mocked)
	// Showing how it integrates into the workflow graph
	fmt.Println("\n--------------------------------------------------")
	fmt.Println("   WORKFLOW INTEGRATION CHECK")
	fmt.Println("--------------------------------------------------")

	// Create Activities Wrapper
	activities := workflows.NewComplianceActivities(engine)

	// Run Activity Directly (Simulating Workflow Engine call)
	fmt.Println("[Engine] Executing ActivityCheckCompliance for Restricted Trade...")
	_, err = activities.ActivityCheckCompliance(context.Background(), nil, restricted)
	if err != nil {
		fmt.Printf(" -> STOPPED: %v\n", err)
	} else {
		fmt.Println(" -> ALLOWED (Unexpected!)")
	}
}

func printResult(res *governance.ValidationResult, err error) {
	if err != nil {
		fmt.Printf(" -> ERROR: %v\n", err)
		return
	}
	if res.Allowed {
		fmt.Println(" -> ✅ ALLOWED")
	} else {
		fmt.Printf(" -> 🛑 BLOCKED: %v\n", res.Reasons)
	}
}
