package orm

import (
	"context"
	"database/sql"
	"time"
)

// InsertQuote writes a top-of-book snapshot. Called from the market data
// fan-out goroutine for each venue update.
func InsertQuote(ctx context.Context, db *sql.DB, tenantID string, q Quote) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO orm.quote (
			security_id, venue_mic, quote_time,
			bid_price, bid_size, ask_price, ask_size,
			mid_price, last_price, last_size,
			spread_bps, quote_currency, is_indicative, source, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`, q.SecurityID, q.VenueMIC, q.QuoteTime,
		q.BidPrice, q.BidSize, q.AskPrice, q.AskSize,
		q.MidPrice, q.LastPrice, q.LastSize,
		q.SpreadBps, q.Currency, q.IsIndicative, q.Source, tenantID)
	return err
}

// UpsertMarketSnapshot writes the EOD OHLC snapshot for a security.
func UpsertMarketSnapshot(ctx context.Context, tx *sql.Tx, tenantID string, s MarketSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.market_data_snapshot (
			security_id, as_of_date,
			open_price, high_price, low_price, close_price,
			vwap, volume, trade_count, adjusted_close,
			currency, source, is_final, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (tenant_id, security_id, as_of_date)
		DO UPDATE SET
			close_price = EXCLUDED.close_price,
			vwap = EXCLUDED.vwap,
			volume = EXCLUDED.volume,
			trade_count = EXCLUDED.trade_count,
			adjusted_close = EXCLUDED.adjusted_close,
			is_final = EXCLUDED.is_final
	`, s.SecurityID, s.AsOfDate,
		s.Open, s.High, s.Low, s.Close,
		s.VWAP, s.Volume, s.TradeCount, s.AdjustedClose,
		s.Currency, s.Source, s.IsFinal, tenantID)
	return err
}

type Quote struct {
	SecurityID   string
	VenueMIC     sql.NullString
	QuoteTime    time.Time
	BidPrice     sql.NullFloat64
	BidSize      sql.NullFloat64
	AskPrice     sql.NullFloat64
	AskSize      sql.NullFloat64
	MidPrice     sql.NullFloat64
	LastPrice    sql.NullFloat64
	LastSize     sql.NullFloat64
	SpreadBps    sql.NullFloat64
	Currency     sql.NullString
	IsIndicative bool
	Source       sql.NullString
}

type MarketSnapshot struct {
	SecurityID    string
	AsOfDate      time.Time
	Open, High    sql.NullFloat64
	Low, Close    sql.NullFloat64
	VWAP          sql.NullFloat64
	Volume        sql.NullFloat64
	TradeCount    sql.NullInt64
	AdjustedClose sql.NullFloat64
	Currency      sql.NullString
	Source        sql.NullString
	IsFinal       bool
}
