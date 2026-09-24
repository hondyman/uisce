-- backend/db/migrations/20260903_drill_down_metadata.up.sql
CREATE SCHEMA IF NOT EXISTS semantic_drill;

CREATE TABLE IF NOT EXISTS semantic_drill.calculation_drill_paths (
    path_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    aggregated_metric_key VARCHAR(100) NOT NULL, -- e.g. "portfolio_xirr", "total_nav"
    target_bo_key VARCHAR(100) NOT NULL,         -- e.g. "TaxLotCashFlows", "PositionMaster"
    target_page_key VARCHAR(100),                -- e.g. "tax_lot_analyzer"
    target_table VARCHAR(150) NOT NULL,          -- e.g. "mdm.tax_lot_cash_flows"
    default_columns JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_metric_drill UNIQUE (tenant_id, aggregated_metric_key)
);

CREATE INDEX IF NOT EXISTS idx_drill_paths_lookup 
ON semantic_drill.calculation_drill_paths (tenant_id, aggregated_metric_key);
