package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/pkg/workflows"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   TITAN MDM GOLDEN RECORD DEMO")
	fmt.Println("==================================================")

	mdmActivities := workflows.NewMDMActivities()
	ctx := context.Background()

	// 1. Scenario: Updating data that MATCHES the Golden Record
	// "CP-123" has risk_rating=HIGH in mock MDM. Proposing HIGH is fine.
	fmt.Println("\n[Case 1] Validating Consistent Data Update...")
	req1 := workflows.MDMValidationRequest{
		EntityType: "Counterparty",
		EntityID:   "CP-123",
		Attributes: map[string]interface{}{
			"risk_rating": "HIGH",
			"country":     "US",
		},
	}
	res1, err := mdmActivities.ActivityValidateGoldenRecord(ctx, req1)
	if err != nil {
		log.Fatalf("Case 1 Failed: %v", err)
	}
	fmt.Printf(" -> Result: %v\n", res1)
	fmt.Println(" -> ✅ ALLOWED (Matching Truth)")

	// 2. Scenario: Data Drift Attempt
	// Proposing risk_rating="LOW" when Golden Source says "HIGH".
	fmt.Println("\n[Case 2] Attempting Data Drift (Updating Risk to LOW)...")
	req2 := workflows.MDMValidationRequest{
		EntityType: "Counterparty",
		EntityID:   "CP-123",
		Attributes: map[string]interface{}{
			"risk_rating": "LOW",
			"country":     "US",
		},
	}
	_, err = mdmActivities.ActivityValidateGoldenRecord(ctx, req2)
	if err == nil {
		fmt.Println(" -> ❌ FAILURE: Violation was not detected!")
	} else {
		fmt.Printf(" -> 🛑 BLOCKED: %v\n", err)
		fmt.Println(" -> ✅ SUCCESS (Drift Prevented)")
	}

	// 3. Scenario: Unknown Entity
	fmt.Println("\n[Case 3] Validating Unknown Entity...")
	req3 := workflows.MDMValidationRequest{
		EntityType: "Counterparty",
		EntityID:   "CP-999",
		Attributes: map[string]interface{}{"foo": "bar"},
	}
	_, err = mdmActivities.ActivityValidateGoldenRecord(ctx, req3)
	if err != nil {
		fmt.Printf(" -> 🛑 ERROR: %v\n", err)
	}
}
