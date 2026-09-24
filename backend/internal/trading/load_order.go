package trading

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LoadOrder loads one CRIMS orm."order" row fenced by id + tenant_id.
//
// Unifies MCP omsLoadCRIMSOrder and handlers.OMSFIXCommandHandler.loadOrder
// (SL commit 5/5). Pre-flight diff:
//
//	SQL WHERE: identical (id=$1::uuid AND tenant_id=$2::uuid)
//	SELECT:    identical columns
//	qty/leaves / symbol-default: equivalent (NullString empty → "AAPL")
//	DELTA:     none on the fence — chose HTTP GetContext/struct scan shape
//	           as the single implementation (clearer, not a richer predicate).
func LoadOrder(ctx context.Context, tenantID uuid.UUID, orderID string) (*Order, error) {
	db, err := OpenCRIMS(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return LoadOrderDB(ctx, sqlx.NewDb(db, "postgres"), tenantID, orderID)
}

// LoadOrderDB is the testable core (same predicate as LoadOrder).
func LoadOrderDB(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, orderID string) (*Order, error) {
	if db == nil {
		return nil, fmt.Errorf("crims db is nil")
	}
	var row struct {
		ID     string          `db:"id"`
		Side   string          `db:"side"`
		Qty    sql.NullFloat64 `db:"target_qty"`
		Leaves sql.NullFloat64 `db:"leaves_qty"`
		Price  sql.NullFloat64 `db:"limit_price"`
		SecID  sql.NullString  `db:"sec_id"`
		Status sql.NullString  `db:"status"`
	}
	err := db.GetContext(ctx, &row, `
		SELECT id::text, side, target_qty, leaves_qty, limit_price, sec_id::text, status
		FROM orm."order"
		WHERE id = $1::uuid AND tenant_id = $2::uuid
	`, orderID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("order %s not found on crims.orm: %w", orderID, err)
	}
	qty := row.Qty.Float64
	if row.Leaves.Valid && row.Leaves.Float64 > 0 {
		qty = row.Leaves.Float64
	}
	symbol := row.SecID.String
	if symbol == "" {
		symbol = "AAPL"
	}
	return &Order{
		OrderID:  row.ID,
		Symbol:   symbol,
		Quantity: qty,
		Side:     row.Side,
		Price:    row.Price.Float64,
		Status:   row.Status.String,
	}, nil
}
