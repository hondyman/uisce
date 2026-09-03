package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/mdm"
	"github.com/hondyman/uisce/backend/pkg/workflows"
)

// TITAN MDM Golden Record Demo. Masters a real counterparty record into
// catalog_mdm.golden_records_ledger via mdm.UniversalMasteringEngine, then
// runs it through workflows.MDMActivities.ActivityValidateGoldenRecord —
// the same activity a real BP workflow node dispatches to — proving the
// full mastering-then-validation path against real storage rather than the
// hardcoded fake data this demo previously exercised.
func main() {
	fmt.Println("==================================================")
	fmt.Println("   TITAN MDM GOLDEN RECORD DEMO")
	fmt.Println("==================================================")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	engine := mdm.NewUniversalMasteringEngine(db)
	mdmActivities := workflows.NewMDMActivities(engine)
	ctx := context.Background()

	tenantID := uuid.New()
	entityType := "Counterparty"
	entityID := "CP-123"

	fmt.Println("\n[Setup] Mastering a golden record via MasterAndSealRecord...")
	_, err = engine.MasterAndSealRecord(ctx, tenantID, mdm.VendorFeedRecord{
		TenantID:     tenantID,
		DomainKey:    entityType,
		VendorSource: "BLOOMBERG",
		// CUSIP forces MasterAndSealRecord's master_entity_sid to entityID
		// (see its masterSID fallback chain) so the demo can look it back up.
		Identifiers:   map[string]string{"CUSIP": entityID},
		Attributes:    map[string]interface{}{"risk_rating": "HIGH", "kyc_status": "APPROVED", "country": "US"},
		EffectiveTime: time.Now(),
	})
	if err != nil {
		log.Fatalf("mastering failed: %v", err)
	}
	fmt.Println(" -> Golden record sealed for", entityType, entityID)

	config := map[string]interface{}{
		"tenant_id":   tenantID.String(),
		"entity_type": entityType,
		"entity_id":   entityID,
	}

	// 1. Scenario: Updating data that MATCHES the Golden Record
	fmt.Println("\n[Case 1] Validating Consistent Data Update...")
	res1, err := mdmActivities.ActivityValidateGoldenRecord(ctx, config, map[string]interface{}{
		"attributes": map[string]interface{}{"risk_rating": "HIGH", "country": "US"},
	})
	if err != nil {
		log.Fatalf("Case 1 Failed: %v", err)
	}
	fmt.Printf(" -> Result: %v\n", res1)
	fmt.Println(" -> ✅ ALLOWED (Matching Truth)")

	// 2. Scenario: Data Drift Attempt
	fmt.Println("\n[Case 2] Attempting Data Drift (Updating Risk to LOW)...")
	_, err = mdmActivities.ActivityValidateGoldenRecord(ctx, config, map[string]interface{}{
		"attributes": map[string]interface{}{"risk_rating": "LOW", "country": "US"},
	})
	if err == nil {
		fmt.Println(" -> ❌ FAILURE: Violation was not detected!")
	} else {
		fmt.Printf(" -> 🛑 BLOCKED: %v\n", err)
		fmt.Println(" -> ✅ SUCCESS (Drift Prevented)")
	}

	// 3. Scenario: Entity never mastered
	fmt.Println("\n[Case 3] Validating Never-Mastered Entity...")
	res3, err := mdmActivities.ActivityValidateGoldenRecord(ctx, map[string]interface{}{
		"tenant_id":   tenantID.String(),
		"entity_type": entityType,
		"entity_id":   "CP-999",
	}, map[string]interface{}{"attributes": map[string]interface{}{"foo": "bar"}})
	if err != nil {
		fmt.Printf(" -> 🛑 ERROR: %v\n", err)
	} else {
		fmt.Printf(" -> Result: %v (no golden record exists yet, not a violation)\n", res3)
	}
}
