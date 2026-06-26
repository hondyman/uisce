package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/workflows"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   TITAN IMMUTABLE LEDGER DEMO")
	fmt.Println("==================================================")

	// 1. Connect DB
	db, err := sqlx.Connect("postgres", "postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable")
	if err != nil {
		log.Fatalf("DB Connection failed: %v", err)
	}

	// 2. Clear Ledger for Demo (Optional, but clean)
	tenantID := uuid.New().String()
	fmt.Printf("[Setup] Using Tenant ID: %s\n", tenantID)

	// 3. Insert 3 Valid Records via DurableLedgerWrite
	ledgerActivities := workflows.NewLedgerActivities(db)
	ctx := context.Background()

	actorID := uuid.New().String()

	fmt.Println("\n[Action] Inserting Record 1 (Trade Initiated)...")
	rec1Hash, err := ledgerActivities.DurableLedgerWrite(ctx, workflows.LedgerRecord{
		TenantID:        tenantID,
		TransactionType: "TRADE_INIT",
		ActorID:         actorID,
		Payload:         json.RawMessage(`{"amount": 100, "symbol": "AAPL"}`),
	})
	if err != nil {
		log.Fatalf("Insert 1 failed: %v", err)
	}
	fmt.Printf(" -> Hash: %s\n", rec1Hash)

	fmt.Println("[Action] Inserting Record 2 (Compliance Check)...")
	rec2Hash, err := ledgerActivities.DurableLedgerWrite(ctx, workflows.LedgerRecord{
		TenantID:        tenantID,
		TransactionType: "COMPLIANCE_PASS",
		ActorID:         uuid.New().String(), // Was "system-bot"
		Payload:         json.RawMessage(`{"status": "APPROVED"}`),
	})
	if err != nil {
		log.Fatalf("Insert 2 failed: %v", err)
	}
	fmt.Printf(" -> Hash: %s\n", rec2Hash)

	fmt.Println("[Action] Inserting Record 3 (Trade Executed)...")
	rec3Hash, err := ledgerActivities.DurableLedgerWrite(ctx, workflows.LedgerRecord{
		TenantID:        tenantID,
		TransactionType: "TRADE_EXEC",
		ActorID:         actorID,
		Payload:         json.RawMessage(`{"fill_price": 150.00}`),
	})
	if err != nil {
		log.Fatalf("Insert 3 failed: %v", err)
	}
	fmt.Printf(" -> Hash: %s\n", rec3Hash)

	// 4. Verify Integrity (Should Pass)
	fmt.Println("\n[Audit] Verifying Chain Integrity...")
	if verifyChain(db, tenantID) {
		fmt.Println(" -> ✅ CHAIN VALID")
	} else {
		fmt.Println(" -> ❌ CHAIN BROKEN")
	}

	// 5. Simulate Tampering Attack
	fmt.Println("\n[Attack] Tampering with Record 2 (Changing Payload in DB)...")
	_, err = db.Exec(`
		UPDATE audit_ledger 
		SET payload = '{"status": "REJECTED"}' 
		WHERE hash = $1
	`, rec2Hash)
	if err != nil {
		log.Fatalf("Tamper failed: %v", err)
	}
	fmt.Println(" -> Database Row Modified Directly!")

	// 6. Verify Integrity (Should Fail)
	fmt.Println("\n[Audit] Verifying Chain Integrity After Attack...")
	if verifyChain(db, tenantID) {
		fmt.Println(" -> ✅ CHAIN VALID (Unexpected!)")
	} else {
		fmt.Println(" -> ❌ CHAIN BROKEN (Attack Detected!)")
	}
}

// verifyChain reads all records for a tenant and re-computes hashes
func verifyChain(db *sqlx.DB, tenantID string) bool {
	var records []workflows.LedgerRecord
	err := db.Select(&records, `
		SELECT * FROM audit_ledger 
		WHERE tenant_id = $1 
		ORDER BY created_at ASC
	`, tenantID)
	if err != nil {
		log.Printf("Fetch failed: %v", err)
		return false
	}

	expectedPrevHash := "0000000000000000000000000000000000000000000000000000000000000000"

	for i, r := range records {
		// 1. Check Linkage
		if r.PreviousHash != expectedPrevHash {
			fmt.Printf("   [Break] Record %d (ID %s): PreviousHash Mismatch!\n", i, r.ID)
			fmt.Printf("           Expected: %s\n", expectedPrevHash)
			fmt.Printf("           Actual:   %s\n", r.PreviousHash)
			return false
		}

		// 2. Check Data Integrity
		payloadStr := string(r.Payload)
		dataToHash := fmt.Sprintf("%s:%s:%s:%s", r.PreviousHash, r.TransactionType, r.ActorID, payloadStr)
		hash := sha256.Sum256([]byte(dataToHash))
		computedHash := fmt.Sprintf("%x", hash)

		if computedHash != r.Hash {
			fmt.Printf("   [Break] Record %d (ID %s): Hash Integration Check Failed!\n", i, r.ID)
			fmt.Printf("           Stored:   %s\n", r.Hash)
			fmt.Printf("           Computed: %s\n", computedHash)
			fmt.Printf("           Payload:  %s\n", payloadStr) // Debug
			return false
		}

		// Advance
		expectedPrevHash = r.Hash
	}

	return true
}
