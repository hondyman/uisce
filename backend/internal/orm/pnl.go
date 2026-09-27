package orm

import (
	"context"
	"database/sql"
	"time"
)

// SnapshotPnL writes an intraday P&L row for an account.
func SnapshotPnL(ctx context.Context, tx *sql.Tx, tenantID string, p PnLSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.pnl_intraday (
			account_id, portfolio_id, as_of_timestamp, currency,
			realized_pnl, unrealized_pnl, total_pnl,
			income, commissions, fees, net_pnl, market_value, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, p.AccountID, p.PortfolioID, p.AsOf, p.Currency,
		p.Realized, p.Unrealized, p.Total,
		p.Income, p.Commissions, p.Fees, p.NetPnL, p.MarketValue, tenantID)
	return err
}

type PnLSnapshot struct {
	AccountID   string
	PortfolioID sql.NullString
	AsOf        time.Time
	Currency    string
	Realized    float64
	Unrealized  float64
	Total       float64
	Income      float64
	Commissions float64
	Fees        float64
	NetPnL      sql.NullFloat64
	MarketValue sql.NullFloat64
}
