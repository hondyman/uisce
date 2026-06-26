-- Migration: 016_alternative_assets.sql
-- Description: Creates tables for Alternative Assets and Valuation Events using JSONB for polymorphism.

-- 1. Create alternative_assets table
CREATE TABLE IF NOT EXISTS alternative_assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    asset_type TEXT NOT NULL, -- e.g., 'PrivateEquity', 'RealEstate', 'HedgeFund', 'Art', 'Crypto'
    common_attributes JSONB DEFAULT '{}'::jsonb, -- Shared attributes (Custodian, Currency, etc.)
    specific_attributes JSONB DEFAULT '{}'::jsonb, -- Polymorphic attributes (Vintage Year, Address, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT ux_alternative_assets_slug UNIQUE (tenant_id, slug)
);

-- Index for efficient JSONB querying on specific attributes
CREATE INDEX IF NOT EXISTS idx_alternative_assets_specifics ON alternative_assets USING GIN (specific_attributes);
CREATE INDEX IF NOT EXISTS idx_alternative_assets_tenant ON alternative_assets (tenant_id);
CREATE INDEX IF NOT EXISTS idx_alternative_assets_type ON alternative_assets (asset_type);

-- 2. Create valuation_events table (Append-Only Log)
CREATE TABLE IF NOT EXISTS valuation_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES alternative_assets(asset_id) ON DELETE CASCADE,
    event_date DATE NOT NULL,
    event_type TEXT NOT NULL, -- 'NAV_Update', 'Capital_Call', 'Distribution', 'Revaluation'
    amount NUMERIC(19, 4) NOT NULL, -- The value or cash flow amount
    currency TEXT NOT NULL DEFAULT 'USD',
    source TEXT, -- e.g., 'Manual_Entry', 'Custodian_Feed'
    metadata JSONB DEFAULT '{}'::jsonb, -- Context (e.g., PDF reference)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_valuation_events_asset_date ON valuation_events (asset_id, event_date DESC);

-- 3. Create view for Daily NAV (LOCF - Last Observation Carried Forward)
-- Note: This is a heavy view. In production, consider materializing this or using the daily_nav_cache approach.
CREATE OR REPLACE VIEW view_alternative_assets_daily_nav AS
WITH date_spine AS (
    -- Generate a spine for the last 10 years (adjust as needed)
    SELECT generate_series(
        (CURRENT_DATE - INTERVAL '10 years')::date, 
        CURRENT_DATE, 
        '1 day'::interval
    )::date AS report_date
),
asset_dates AS (
    -- Cross join assets with dates to ensure every asset has a row for every day
    SELECT 
        a.asset_id,
        a.tenant_id,
        d.report_date
    FROM alternative_assets a
    CROSS JOIN date_spine d
    WHERE d.report_date >= a.created_at::date -- Optimization: Don't project before creation
),
latest_valuations AS (
    SELECT
        ad.asset_id,
        ad.report_date,
        -- Window function to find the most recent NAV update on or before the report date
        (
            SELECT amount 
            FROM valuation_events ve 
            WHERE ve.asset_id = ad.asset_id 
              AND ve.event_date <= ad.report_date 
              AND ve.event_type IN ('NAV_Update', 'Revaluation')
            ORDER BY ve.event_date DESC, ve.created_at DESC
            LIMIT 1
        ) as carried_nav
    FROM asset_dates ad
)
SELECT
    lv.asset_id,
    lv.report_date,
    COALESCE(lv.carried_nav, 0) as nav
FROM latest_valuations lv
WHERE lv.carried_nav IS NOT NULL;
