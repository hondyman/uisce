package orm

import (
	"context"
	"database/sql"
	"time"
)

// UpsertFXExposure writes/updates the day's exposure for an account-currency.
func UpsertFXExposure(ctx context.Context, tx *sql.Tx, tenantID string, e FXExposure) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.fx_exposure (
			account_id, portfolio_id, currency, as_of_date,
			gross_exposure, net_exposure, hedged_exposure, unhedged_exposure,
			hedge_ratio_pct, base_currency, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (tenant_id, account_id, currency, as_of_date)
		DO UPDATE SET
			gross_exposure = EXCLUDED.gross_exposure,
			net_exposure = EXCLUDED.net_exposure,
			hedged_exposure = EXCLUDED.hedged_exposure,
			unhedged_exposure = EXCLUDED.unhedged_exposure,
			hedge_ratio_pct = EXCLUDED.hedge_ratio_pct
	`, e.AccountID, e.PortfolioID, e.Currency, e.AsOfDate,
		e.Gross, e.Net, e.Hedged, e.Unhedged,
		e.HedgeRatio, e.BaseCurrency, tenantID)
	return err
}

// InsertFXHedge registers a hedge overlay.
func InsertFXHedge(ctx context.Context, tx *sql.Tx, tenantID string, h FXHedge) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO orm.fx_hedge (
			account_id, portfolio_id, currency,
			hedge_instrument_type, hedge_instrument_id,
			notional_amount, notional_currency, hedge_ratio_pct,
			effective_date, maturity_date, is_active, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,true,$11)
		RETURNING id
	`, h.AccountID, h.PortfolioID, h.Currency,
		h.InstrumentType, h.InstrumentID,
		h.Notional, h.NotionalCcy, h.HedgeRatio,
		h.EffectiveDate, h.MaturityDate, tenantID).Scan(&id)
	return id, err
}

type FXExposure struct {
	AccountID    string
	PortfolioID  sql.NullString
	Currency     string
	AsOfDate     time.Time
	Gross, Net   sql.NullFloat64
	Hedged       sql.NullFloat64
	Unhedged     sql.NullFloat64
	HedgeRatio   sql.NullFloat64
	BaseCurrency sql.NullString
}

type FXHedge struct {
	AccountID      string
	PortfolioID    sql.NullString
	Currency       string
	InstrumentType sql.NullString
	InstrumentID   sql.NullString
	Notional       float64
	NotionalCcy    string
	HedgeRatio     sql.NullFloat64
	EffectiveDate  time.Time
	MaturityDate   sql.NullTime
}
