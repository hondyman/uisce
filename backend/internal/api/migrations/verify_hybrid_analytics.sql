-- Verification Script for Hybrid Analytics (StarRocks)
-- Run this on the StarRocks instance to verify schema creation.

-- 1. Verify Tables Exist
EXISTS TABLE trades_stream;
EXISTS TABLE compliance_events;
EXISTS TABLE audit_log;
EXISTS TABLE daily_pnl_agg;
EXISTS TABLE compliance_stats_agg;

-- 2. Verify Materialized Views Exist
EXISTS TABLE mv_daily_pnl;
EXISTS TABLE mv_compliance_stats;

-- 3. Verify Partitioning (Check system.parts)
SELECT table, partition, rows FROM system.parts 
WHERE table IN ('trades_stream', 'compliance_events') 
ORDER BY table, partition;

-- 4. Test Ingestion (Mock)
INSERT INTO trades_stream (event_time, trade_id, portfolio_id, desk_id, symbol, side, quantity, price, notional, currency, basis_id)
VALUES (now(), generateUUIDv4(), generateUUIDv4(), 'DESK-A', 'AAPL', 'Buy', 100, 150.00, 15000.00, 'USD', 'IBOR');

-- 5. Verify Aggregation (MV should update)
-- Wait a moment for MV to catch up (async)
SELECT * FROM daily_pnl_agg WHERE desk_id = 'DESK-A';
