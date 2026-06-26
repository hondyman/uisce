-- Migration 006: Semantic Layer Enhancements

-- 1. Portfolios Table (Core Entity)
CREATE TABLE IF NOT EXISTS portfolios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    manager_id UUID, -- User ID of the manager
    region TEXT, -- For ABAC (e.g., 'EU', 'US')
    strategy TEXT, -- For ABAC (e.g., 'Equity', 'FixedIncome')
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Market Prices Table (For NAV Calculation)
CREATE TABLE IF NOT EXISTS market_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    asset_id TEXT NOT NULL,
    price DECIMAL(20, 8) NOT NULL,
    currency TEXT DEFAULT 'USD',
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
-- Index for finding latest price
CREATE INDEX idx_market_prices_asset_valid ON market_prices (asset_id, valid_from DESC);

-- 3. Computed Field: calculate_nav
-- This function will be exposed as a computed field 'nav' on the 'portfolios' table in Hasura.
-- It sums the value of all positions associated with the portfolio.
-- Note: We assume 'account_id' in position_versions maps to 'portfolios.id' for simplicity in this demo,
-- or we add a portfolio_id to position_versions. Let's assume position_versions.account_id = portfolios.id for now.

CREATE OR REPLACE FUNCTION calculate_nav(portfolio_row portfolios)
RETURNS DECIMAL AS $$
DECLARE
    total_nav DECIMAL := 0;
BEGIN
    -- Sum (Quantity * Latest Price) for all positions in this portfolio
    -- We use the 'current' valid time (infinity) for positions
    SELECT COALESCE(SUM(p.quantity * mp.price), 0)
    INTO total_nav
    FROM position_versions p
    JOIN LATERAL (
        SELECT price 
        FROM market_prices 
        WHERE asset_id = p.asset_id 
        ORDER BY valid_from DESC 
        LIMIT 1
    ) mp ON true
    WHERE p.account_id = portfolio_row.id::text -- Casting UUID to text to match position_versions.account_id schema
      AND p.valid_time @> NOW() -- Valid now
      AND p.system_time @> NOW(); -- Known now

    RETURN total_nav;
END;
$$ LANGUAGE plpgsql STABLE;

-- 4. Dynamic ABAC (Row-Level Security)
-- Enable RLS on Portfolios
ALTER TABLE portfolios ENABLE ROW LEVEL SECURITY;

-- Policy: Managers can see their own portfolios
CREATE POLICY manager_access ON portfolios
    FOR SELECT
    USING (
        manager_id::text = current_setting('hasura.user.id', true)
    );

-- Policy: Region-based access (Dynamic Attribute)
-- Users can see portfolios if the portfolio's region is in the user's allowed regions.
-- We assume 'hasura.user.allowed_regions' is a JSON array or CSV string passed in session variables.
-- Example: '{"EU", "US"}'
CREATE POLICY region_access ON portfolios
    FOR SELECT
    USING (
        region = ANY(
            string_to_array(current_setting('hasura.user.allowed_regions', true), ',')
        )
        OR
        current_setting('hasura.user.role', true) = 'admin'
    );

-- Grant access to Hasura role (postgres role used by Hasura)
-- GRANT SELECT ON portfolios TO hasura_user; -- Uncomment if specific role needed
