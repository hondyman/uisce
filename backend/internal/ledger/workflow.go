package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TradeEvent represents the incoming business event
type TradeEvent struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	AccountID      string    `json:"account_id"`
	AssetID        string    `json:"asset_id"`
	Quantity       float64   `json:"quantity"`
	TradeDate      time.Time `json:"trade_date"`
	SettlementDate time.Time `json:"settlement_date"`
}

// LedgerEntry represents a row in the ledger_entries table
type LedgerEntry struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	BasisID        string    `json:"basis_id"` // IBOR, ABOR, PBOR
	AccountID      string    `json:"account_id"`
	AssetID        string    `json:"asset_id"`
	Quantity       float64   `json:"quantity"`
	ValidFrom      time.Time `json:"valid_from"`
	ValidTo        time.Time `json:"valid_to"`
	TransactionRef string    `json:"transaction_ref"`
}

// PostingRule defines how a trade transforms into a ledger entry
type PostingRule struct {
	BasisID string `json:"basis_id"`
	Timing  string `json:"timing"` // TradeDate, SettlementDate
}

// LedgerUpdateWorkflow implements the Prism Pattern (Multi-Book)
func LedgerUpdateWorkflow(ctx workflow.Context, trade TradeEvent) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting LedgerUpdateWorkflow (Multi-Book)", "TradeID", trade.ID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: time.Second * 10,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Generate Ledger Entries (Fan-Out)
	var entries []LedgerEntry
	err := workflow.ExecuteActivity(ctx, GenerateLedgerEntriesActivity, trade).Get(ctx, &entries)
	if err != nil {
		return err
	}

	// Step 2: Write Entries to Ledger (Batch Transaction)
	// We use a single activity to ensure atomicity of all books.
	err = workflow.ExecuteActivity(ctx, WriteLedgerBatchActivity, entries).Get(ctx, nil)
	if err != nil {
		return err
	}

	logger.Info("LedgerUpdateWorkflow Completed Successfully", "EntriesGenerated", len(entries))
	return nil
}

// --- Activities ---

// GenerateLedgerEntriesActivity fetches rules and fans out the trade
func GenerateLedgerEntriesActivity(ctx context.Context, trade TradeEvent) ([]LedgerEntry, error) {
	fmt.Printf("Generating Ledger Entries for Trade: %s\n", trade.ID)

	// 1. Fetch Rules (Mocking the DB fetch from meta_posting_rules)
	// In production: SELECT rules_json FROM meta_posting_rules WHERE event_type = 'Trade'
	rules := []PostingRule{
		{BasisID: "IBOR", Timing: "TradeDate"},
		{BasisID: "ABOR", Timing: "SettlementDate"},
	}

	var entries []LedgerEntry

	for _, rule := range rules {
		validFrom := trade.TradeDate
		if rule.Timing == "SettlementDate" {
			validFrom = trade.SettlementDate
		}

		entry := LedgerEntry{
			ID:             uuid.New().String(),
			TenantID:       trade.TenantID,
			BasisID:        rule.BasisID,
			AccountID:      trade.AccountID,
			AssetID:        trade.AssetID,
			Quantity:       trade.Quantity,
			ValidFrom:      validFrom,
			ValidTo:        time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), // Infinity
			TransactionRef: trade.ID,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// WriteLedgerBatchActivity writes all entries in a single transaction
func WriteLedgerBatchActivity(ctx context.Context, entries []LedgerEntry) error {
	fmt.Printf("Writing Batch of %d Ledger Entries\n", len(entries))
	// Mock DB Transaction
	// BEGIN;
	// INSERT INTO ledger_entries ...
	// COMMIT;
	for _, e := range entries {
		fmt.Printf("  - Writing Entry [Basis: %s]: %f %s (ValidFrom: %s)\n", e.BasisID, e.Quantity, e.AssetID, e.ValidFrom)
	}
	return nil
}

// Deprecated Activities (kept for interface compatibility if needed, or removed)
func WriteDraftEntry(ctx context.Context, entry LedgerEntry) (string, error) { return "", nil }
func ValidatePositions(ctx context.Context, entry LedgerEntry) error { return nil }
func ConfirmEntry(ctx context.Context, id string) error { return nil }
func DeleteDraftEntry(ctx context.Context, id string) error { return nil }
