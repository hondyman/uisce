package orm

import (
	"context"
	"database/sql"
)

// InsertExecutionQuality writes the post-trade TCA row for a fill.
func InsertExecutionQuality(ctx context.Context, tx *sql.Tx, tenantID string,
	executionID string, m ExecutionQualityMetrics) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.execution_quality (
			execution_id, arrival_price, arrival_price_time,
			interval_vwap, market_vwap, slippage_bps, slippage_amount,
			implementation_shortfall_bps, implementation_shortfall_amount,
			opportunity_cost_bps, opportunity_cost_amount,
			participation_rate_pct, benchmark_ref, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (tenant_id, execution_id) DO UPDATE SET
			arrival_price = EXCLUDED.arrival_price,
			interval_vwap = EXCLUDED.interval_vwap,
			market_vwap = EXCLUDED.market_vwap,
			slippage_bps = EXCLUDED.slippage_bps,
			implementation_shortfall_bps = EXCLUDED.implementation_shortfall_bps
	`, executionID, m.ArrivalPrice, m.ArrivalPriceTime,
		m.IntervalVWAP, m.MarketVWAP, m.SlippageBps, m.SlippageAmt,
		m.ISBps, m.ISAmt, m.OppCostBps, m.OppCostAmt,
		m.ParticipationPct, m.BenchmarkRef, tenantID)
	return err
}

type ExecutionQualityMetrics struct {
	ArrivalPrice     sql.NullFloat64
	ArrivalPriceTime sql.NullTime
	IntervalVWAP     sql.NullFloat64
	MarketVWAP       sql.NullFloat64
	SlippageBps      sql.NullFloat64
	SlippageAmt      sql.NullFloat64
	ISBps            sql.NullFloat64
	ISAmt            sql.NullFloat64
	OppCostBps       sql.NullFloat64
	OppCostAmt       sql.NullFloat64
	ParticipationPct sql.NullFloat64
	BenchmarkRef     sql.NullString
}

// InsertOrderBenchmark records a benchmark price snapshot for TCA.
func InsertOrderBenchmark(ctx context.Context, tx *sql.Tx, tenantID,
	orderID, benchmarkType string, t sql.NullTime,
	price, qty sql.NullFloat64, ccy sql.NullString) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.order_benchmark (
			order_id, benchmark_type, benchmark_time,
			benchmark_price, benchmark_quantity, benchmark_currency, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, orderID, benchmarkType, t, price, qty, ccy, tenantID)
	return err
}
