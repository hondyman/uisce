package tlh

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// WashSaleRegistry manages wash sale detection and enforcement
type WashSaleRegistry struct {
	db *sqlx.DB
}

func NewWashSaleRegistry(db *sqlx.DB) *WashSaleRegistry {
	return &WashSaleRegistry{db: db}
}

// CheckWashSale checks if selling a security would trigger a wash sale
// Returns true if a wash sale would be triggered (i.e., a purchase occurred in the window)
func (r *WashSaleRegistry) CheckWashSale(ctx context.Context, householdID string, ticker string, tradeDate time.Time) (bool, error) {
	// Window: 30 days before and 30 days after
	// Since we are checking BEFORE a sale, we look 30 days back for any purchases
	// We also need to check if we are "locked" from a previous sale (forward guard)

	startWindow := tradeDate.AddDate(0, 0, -30)
	endWindow := tradeDate.AddDate(0, 0, 30)

	// 1. Check for recent purchases (Backward Scan)
	// Query 'trades' table for BUYs of this ticker in this household
	// Note: This requires joining accounts -> households
	// TODO: Replace SQL with Hasura GraphQL query:
	// query CheckRecentPurchases($householdId: uuid!, $ticker: String!, $start: timestamptz!, $end: timestamptz!) {
	//   trades_aggregate(where: {
	//     account: {household_id: {_eq: $householdId}},
	//     ticker: {_eq: $ticker},
	//     side: {_eq: "BUY"},
	//     trade_date: {_gte: $start, _lte: $end}
	//   }) {
	//     aggregate {
	//       count
	//     }
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		SELECT COUNT(*) 
		FROM trades t
		JOIN accounts a ON t.account_id = a.id
		WHERE a.household_id = $1
		  AND t.ticker = $2
		  AND t.side = 'BUY'
		  AND t.trade_date >= $3
		  AND t.trade_date <= $4
	`

	var count int
	if err := r.db.GetContext(ctx, &count, query, householdID, ticker, startWindow, endWindow); err != nil {
		return false, fmt.Errorf("failed to check wash sales: %w", err)
	}

	if count > 0 {
		return true, nil
	}

	// 2. Check for active Wash Sale Locks (Forward Guard)
	// If we previously sold this stock at a loss, we might have recorded a lock
	// (This logic depends on how we persist locks - using wash_sales table expiration)
	// TODO: Replace SQL with Hasura GraphQL query:
	// query CheckWashSaleLocks($householdId: uuid!, $ticker: String!, $date: timestamptz!) {
	//   wash_sales_aggregate(where: {
	//     tax_lot: {
	//       ticker: {_eq: $ticker},
	//       account: {household_id: {_eq: $householdId}}
	//     },
	//     expiration_date: {_gte: $date}
	//   }) {
	//     aggregate {
	//       count
	//     }
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	lockQuery := `
		SELECT COUNT(*)
		FROM wash_sales w
		JOIN tax_lots l ON w.original_lot_id = l.id
		JOIN accounts a ON l.account_id = a.id
		WHERE a.household_id = $1
		  AND l.ticker = $2
		  AND w.expiration_date >= $3
	`

	var lockCount int
	if err := r.db.GetContext(ctx, &lockCount, lockQuery, householdID, ticker, tradeDate); err != nil {
		return false, fmt.Errorf("failed to check wash sale locks: %w", err)
	}

	return lockCount > 0, nil
}

// RecordWashSale records a disallowed loss
func (r *WashSaleRegistry) RecordWashSale(ctx context.Context, originalLotID string, replacementLotID *string, lossAmount float64) error {
	// Expiration is 30 days from now
	expiration := time.Now().AddDate(0, 0, 30)

	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation InsertWashSale($object: wash_sales_insert_input!) {
	//   insert_wash_sales_one(object: $object) {
	//     id
	//     original_lot_id
	//   }
	// }
	// Variables: {"object": {"original_lot_id": "...", "replacement_lot_id": "...",
	//   "disallowed_loss": 1234.56, "wash_date": "...", "expiration_date": "..."}}
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		INSERT INTO wash_sales (original_lot_id, replacement_lot_id, disallowed_loss, wash_date, expiration_date)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query, originalLotID, replacementLotID, lossAmount, time.Now(), expiration)
	return err
}
