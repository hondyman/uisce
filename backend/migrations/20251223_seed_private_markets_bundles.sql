-- Migration: seed private_markets_bundles with sample bundles

CREATE TABLE IF NOT EXISTS public.private_markets_bundles (
    bundle_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    audience TEXT,
    version TEXT,
    modules JSONB DEFAULT '[]',
    metrics JSONB DEFAULT '[]',
    governance JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO private_markets_bundles (bundle_id, name, audience, version, modules, metrics, governance) VALUES
('lp_private_markets_bundle', 'LP Private Markets Bundle', 'lp', '1.0.0',
 '[{"id": "fund-selector", "name": "Fund Selector", "type": "selector", "config": {"multiSelect": true}}, {"id": "irr-curve", "name": "IRR Curve Chart", "type": "chart", "config": {"timeRange": "5y"}}, {"id": "j-curve", "name": "J-Curve Plot", "type": "chart", "config": {"showBenchmark": true}}, {"id": "benchmark-comparison", "name": "Benchmark Comparison", "type": "comparison", "config": {"indices": ["S&P 500", "NASDAQ"]}}, {"id": "liquidity-panel", "name": "Liquidity Panel", "type": "panel", "config": {"showProjections": true}}]'::jsonb,
 '[{"id": "tvpi", "name": "TVPI", "type": "ratio", "formula": "(distributions + residual_value) / paid_in_capital"}, {"id": "irr", "name": "IRR", "type": "percentage", "formula": "XIRR(cash_flows, dates)"}, {"id": "pme", "name": "PME", "type": "ratio", "formula": "PME(cash_flows, benchmark)"}]'::jsonb,
 '{"status": "active", "steward_group": "data-stewards", "schema_hash": "abc123", "sla": {"refresh_frequency": "daily", "max_latency": "4h"}}'::jsonb),

('gp_private_markets_bundle', 'GP Private Markets Bundle', 'gp', '1.0.0',
 '[{"id": "deployment-pacing", "name": "Deployment Pacing Chart", "type": "chart", "config": {"targetPacing": "24months"}}, {"id": "irr-nav-tracking", "name": "IRR/NAV Tracking", "type": "tracking", "config": {"frequency": "quarterly"}}, {"id": "fee-analysis", "name": "Fee Analysis", "type": "analysis", "config": {"feeTypes": ["management", "performance"]}}, {"id": "value-attribution", "name": "Value Attribution", "type": "attribution", "config": {"methodology": "brinson"}}, {"id": "exit-analysis", "name": "Exit Analysis", "type": "analysis", "config": {"exitTypes": ["ipo", "merger", "sale"]}}]'::jsonb,
 '[{"id": "dpi", "name": "DPI", "type": "ratio", "formula": "distributions / paid_in_capital"}, {"id": "rvpi", "name": "RVPI", "type": "ratio", "formula": "residual_value / paid_in_capital"}, {"id": "tvpi", "name": "TVPI", "type": "ratio", "formula": "dpi + rvpi"}]'::jsonb,
 '{"status": "active", "steward_group": "gp-stewards", "schema_hash": "def456", "sla": {"refresh_frequency": "weekly", "max_latency": "24h"}}'::jsonb),

('fof_private_markets_bundle', 'FoF Private Markets Bundle', 'fof', '1.0.0',
 '[{"id": "portfolio-overview", "name": "Portfolio Overview", "type": "overview", "config": {"groupBy": "strategy"}}, {"id": "manager-performance", "name": "Manager Performance", "type": "performance", "config": {"benchmark": true}}, {"id": "allocation-analysis", "name": "Allocation Analysis", "type": "analysis", "config": {"dimensions": ["geography", "vintage"]}}, {"id": "risk-attribution", "name": "Risk Attribution", "type": "attribution", "config": {"method": "factor"}}]'::jsonb,
 '[{"id": "portfolio-irr", "name": "Portfolio IRR", "type": "percentage", "formula": "weighted_average(irr)"}, {"id": "diversification", "name": "Diversification Score", "type": "score", "formula": "1 - concentration_ratio"}, {"id": "alpha", "name": "Alpha vs Benchmark", "type": "percentage", "formula": "irr - benchmark_irr"}]'::jsonb,
 '{"status": "active", "steward_group": "fof-stewards", "schema_hash": "ghi789", "sla": {"refresh_frequency": "monthly", "max_latency": "48h"}}'::jsonb)
ON CONFLICT (bundle_id) DO NOTHING;
