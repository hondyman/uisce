package trading

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LoadOrder loads one orm."order" row from the CALLING TENANT's own database,
// fenced by id + tenant_id.
//
// The tenant is the argument, as it always was, but it is no longer a row filter
// over one shared database: it is placed in the context as the caller tenant, so
// the router resolves that tenant's own orm datasource and the fence becomes a
// statement about which database was opened, not which rows came back. A
// datasource belonging to another tenant is refused outright (ADR-030).
func LoadOrder(ctx context.Context, tenantID uuid.UUID, orderID string) (*Order, error) {
	conn, err := ormDB(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return LoadOrderDB(ctx, sqlx.NewDb(conn, "postgres"), tenantID, orderID)
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
