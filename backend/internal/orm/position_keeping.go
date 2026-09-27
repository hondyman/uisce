package orm

import (
	"context"
	"database/sql"
	"time"
)

// OpenPositionLot inserts a new tax lot when a buy fill settles.
func OpenPositionLot(ctx context.Context, tx *sql.Tx, tenantID string, l PositionLot) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO orm.position_lot (
			position_id, account_id, security_id,
			lot_open_date, lot_acquisition_type,
			quantity_open, quantity_remaining,
			cost_basis_per_unit, cost_basis_total,
			tax_lot_method, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$6,$7,$8,$9,$10)
		RETURNING id
	`, l.PositionID, l.AccountID, l.SecurityID,
		l.LotOpenDate, l.AcquisitionType,
		l.Quantity, l.CostPerUnit, l.CostTotal,
		l.TaxLotMethod, tenantID).Scan(&id)
	return id, err
}

// ClosePositionLots consumes open lots for a sell fill. Caller decides lot
// selection order (FIFO / LIFO / HIFO / specific-ID) before invoking.
func ClosePositionLots(ctx context.Context, tx *sql.Tx, tenantID string,
	lotIDs []string, qty float64) error {
	for _, id := range lotIDs {
		if _, err := tx.ExecContext(ctx, `
			UPDATE orm.position_lot
			SET quantity_closed = LEAST(quantity_open, quantity_closed + $1),
			    quantity_remaining = GREATEST(0, quantity_open - quantity_closed - $1),
			    is_open = (quantity_open - quantity_closed - $1) > 0,
			    closed_date = CASE WHEN quantity_open - quantity_closed - $1 <= 0
			                       THEN CURRENT_DATE ELSE closed_date END,
			    updated_at = now()
			WHERE id = $2 AND tenant_id = $3
		`, qty, id, tenantID); err != nil {
			return err
		}
	}
	return nil
}

// UpsertCashBalance updates the day's balance for an account/currency.
func UpsertCashBalance(ctx context.Context, tx *sql.Tx, tenantID string, b CashBalance) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.cash_balance (
			account_id, currency, as_of_date, as_of_time, balance_type,
			settled_cash, unsettled_cash, projected_cash,
			margin_requirement, buying_power, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (tenant_id, account_id, currency, as_of_date, balance_type)
		DO UPDATE SET
			as_of_time = EXCLUDED.as_of_time,
			settled_cash = EXCLUDED.settled_cash,
			unsettled_cash = EXCLUDED.unsettled_cash,
			projected_cash = EXCLUDED.projected_cash,
			margin_requirement = EXCLUDED.margin_requirement,
			buying_power = EXCLUDED.buying_power
	`, b.AccountID, b.Currency, b.AsOfDate, b.AsOfTime, b.BalanceType,
		b.SettledCash, b.UnsettledCash, b.ProjectedCash,
		b.MarginRequirement, b.BuyingPower, tenantID)
	return err
}

type PositionLot struct {
	PositionID      string
	AccountID       string
	SecurityID      string
	LotOpenDate     time.Time
	AcquisitionType string
	Quantity        float64
	CostPerUnit     sql.NullFloat64
	CostTotal       sql.NullFloat64
	TaxLotMethod    string
}

type CashBalance struct {
	AccountID         string
	Currency          string
	AsOfDate          time.Time
	AsOfTime          sql.NullTime
	BalanceType       string
	SettledCash       float64
	UnsettledCash     float64
	ProjectedCash     sql.NullFloat64
	MarginRequirement float64
	BuyingPower       sql.NullFloat64
}
