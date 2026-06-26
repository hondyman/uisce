-- ============================================================================
-- PORTFOLIO ANALYSIS FUNCTIONS FOR WORKDAY-STYLE DRILL-DOWN
-- Deployed to: wealth_app PostgreSQL (local)
-- ============================================================================

-- 1. PORTFOLIO DRILL-DOWN (Asset Class → Sector → Industry → Security)
CREATE OR REPLACE FUNCTION analyze_portfolio_drill_down(
    p_portfolio_id UUID,
    p_dimension TEXT DEFAULT 'asset_class',
    p_level INTEGER DEFAULT 1,
    p_as_of_date DATE DEFAULT CURRENT_DATE,
    p_tenant_id UUID DEFAULT NULL,
    p_datasource_id UUID DEFAULT NULL
)
RETURNS TABLE (
    dimension_value TEXT,
    dimension_level INTEGER,
    position_count BIGINT,
    market_value NUMERIC,
    cost_basis NUMERIC,
    unrealized_gain_loss NUMERIC,
    gain_loss_pct NUMERIC,
    weight_pct NUMERIC,
    has_children BOOLEAN
) AS $$
DECLARE
    v_portfolio_total NUMERIC;
BEGIN
    -- Get total portfolio value
    SELECT COALESCE(SUM(current_value), 0) INTO v_portfolio_total
    FROM holdings
    WHERE portfolio_id = p_portfolio_id
      AND as_of_date = p_as_of_date;

    IF v_portfolio_total = 0 THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT 
        CASE p_dimension
            WHEN 'asset_class' THEN COALESCE(h.asset_class, 'UNCLASSIFIED')
            WHEN 'sector' THEN COALESCE(h.sector, 'UNCLASSIFIED')
            WHEN 'geography' THEN COALESCE(h.country, 'UNCLASSIFIED')
            WHEN 'security' THEN h.ticker || ' - ' || h.name
            ELSE 'Other'
        END AS dimension_value,
        p_level AS dimension_level,
        COUNT(*)::BIGINT AS position_count,
        SUM(h.current_value) AS market_value,
        SUM(h.cost_basis) AS cost_basis,
        SUM(h.current_value - h.cost_basis) AS unrealized_gain_loss,
        CASE 
            WHEN SUM(h.cost_basis) > 0 
            THEN ((SUM(h.current_value - h.cost_basis) / SUM(h.cost_basis)) * 100)
            ELSE 0 
        END AS gain_loss_pct,
        (SUM(h.current_value) / v_portfolio_total * 100) AS weight_pct,
        TRUE AS has_children
    FROM holdings h
    WHERE h.portfolio_id = p_portfolio_id
      AND h.as_of_date = p_as_of_date
      AND h.current_value > 0
    GROUP BY 
        CASE p_dimension
            WHEN 'asset_class' THEN COALESCE(h.asset_class, 'UNCLASSIFIED')
            WHEN 'sector' THEN COALESCE(h.sector, 'UNCLASSIFIED')
            WHEN 'geography' THEN COALESCE(h.country, 'UNCLASSIFIED')
            WHEN 'security' THEN h.ticker || ' - ' || h.name
            ELSE 'Other'
        END
    ORDER BY market_value DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- 2. HOUSEHOLD AGGREGATION (All positions across all accounts)
CREATE OR REPLACE FUNCTION aggregate_household_holdings(
    p_household_id UUID,
    p_as_of_date DATE DEFAULT CURRENT_DATE,
    p_tenant_id UUID DEFAULT NULL,
    p_datasource_id UUID DEFAULT NULL
)
RETURNS TABLE (
    security_id UUID,
    ticker TEXT,
    security_name TEXT,
    asset_class TEXT,
    total_market_value NUMERIC,
    total_cost_basis NUMERIC,
    total_unrealized_gain_loss NUMERIC,
    weight_pct NUMERIC,
    account_count BIGINT,
    position_count BIGINT
) AS $$
DECLARE
    v_household_total NUMERIC;
BEGIN
    -- Get total household value
    SELECT COALESCE(SUM(h.current_value), 0) INTO v_household_total
    FROM holdings h
    JOIN portfolios p ON h.portfolio_id = p.id
    WHERE p.household_id = p_household_id
      AND h.as_of_date = p_as_of_date;

    IF v_household_total = 0 THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT 
        h.security_id,
        h.ticker,
        h.name,
        h.asset_class,
        SUM(h.current_value) AS total_market_value,
        SUM(h.cost_basis) AS total_cost_basis,
        SUM(h.current_value - h.cost_basis) AS total_unrealized_gain_loss,
        (SUM(h.current_value) / v_household_total * 100) AS weight_pct,
        COUNT(DISTINCT h.portfolio_id)::BIGINT AS account_count,
        COUNT(*)::BIGINT AS position_count
    FROM holdings h
    JOIN portfolios p ON h.portfolio_id = p.id
    WHERE p.household_id = p_household_id
      AND h.as_of_date = p_as_of_date
      AND h.current_value > 0
    GROUP BY 
        h.security_id,
        h.ticker,
        h.name,
        h.asset_class
    ORDER BY total_market_value DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- 3. REAL-TIME PERFORMANCE CALCULATION
CREATE OR REPLACE FUNCTION calculate_portfolio_performance(
    p_portfolio_id UUID,
    p_start_date DATE,
    p_end_date DATE DEFAULT CURRENT_DATE,
    p_tenant_id UUID DEFAULT NULL,
    p_datasource_id UUID DEFAULT NULL
)
RETURNS TABLE (
    period_name TEXT,
    start_value NUMERIC,
    end_value NUMERIC,
    net_cash_flows NUMERIC,
    total_return_pct NUMERIC,
    time_weighted_return_pct NUMERIC,
    days_held INTEGER
) AS $$
DECLARE
    v_start_value NUMERIC;
    v_end_value NUMERIC;
    v_cash_flows NUMERIC;
BEGIN
    -- Get starting value
    SELECT COALESCE(SUM(current_value), 0) INTO v_start_value
    FROM holdings
    WHERE portfolio_id = p_portfolio_id
      AND as_of_date = p_start_date;

    -- Get ending value
    SELECT COALESCE(SUM(current_value), 0) INTO v_end_value
    FROM holdings
    WHERE portfolio_id = p_portfolio_id
      AND as_of_date = p_end_date;

    -- Get net cash flows (deposits/withdrawals)
    SELECT COALESCE(SUM(
        CASE 
            WHEN transaction_type IN ('DEPOSIT', 'CONTRIBUTION') THEN amount
            WHEN transaction_type IN ('WITHDRAWAL', 'DISTRIBUTION') THEN -amount
            ELSE 0
        END
    ), 0) INTO v_cash_flows
    FROM transactions
    WHERE portfolio_id = p_portfolio_id
      AND transaction_date BETWEEN p_start_date AND p_end_date;

    RETURN QUERY
    SELECT 
        to_char(p_start_date, 'YYYY-MM-DD') || ' to ' || to_char(p_end_date, 'YYYY-MM-DD') AS period_name,
        v_start_value,
        v_end_value,
        v_cash_flows,
        CASE 
            WHEN v_start_value > 0 
            THEN ((v_end_value - v_start_value) / v_start_value * 100)
            ELSE 0 
        END AS total_return_pct,
        CASE 
            WHEN v_start_value > 0 
            THEN ((v_end_value - v_start_value - v_cash_flows) / v_start_value * 100)
            ELSE 0 
        END AS time_weighted_return_pct,
        (p_end_date - p_start_date)::INTEGER AS days_held;
END;
$$ LANGUAGE plpgsql STABLE;

-- 4. CONCENTRATION RISK ANALYSIS
CREATE OR REPLACE FUNCTION analyze_concentration_risk(
    p_portfolio_id UUID,
    p_dimension TEXT DEFAULT 'security',
    p_threshold_pct NUMERIC DEFAULT 10.0,
    p_as_of_date DATE DEFAULT CURRENT_DATE,
    p_tenant_id UUID DEFAULT NULL,
    p_datasource_id UUID DEFAULT NULL
)
RETURNS TABLE (
    dimension_value TEXT,
    concentration_pct NUMERIC,
    market_value NUMERIC,
    risk_level TEXT,
    exceeds_threshold BOOLEAN
) AS $$
DECLARE
    v_portfolio_total NUMERIC;
    v_concentration NUMERIC;
BEGIN
    -- Get total portfolio value
    SELECT COALESCE(SUM(current_value), 0) INTO v_portfolio_total
    FROM holdings
    WHERE portfolio_id = p_portfolio_id
      AND as_of_date = p_as_of_date;

    IF v_portfolio_total = 0 THEN
        RETURN;
    END IF;

    RETURN QUERY
    WITH concentration_data AS (
        SELECT 
            CASE p_dimension
                WHEN 'security' THEN h.ticker || ' - ' || h.name
                WHEN 'sector' THEN COALESCE(h.sector, 'UNCLASSIFIED')
                WHEN 'issuer' THEN COALESCE(h.issuer, 'UNCLASSIFIED')
                WHEN 'asset_class' THEN COALESCE(h.asset_class, 'UNCLASSIFIED')
                ELSE 'Other'
            END AS dim_value,
            SUM(h.current_value) AS value,
            (SUM(h.current_value) / v_portfolio_total * 100) AS conc_pct
        FROM holdings h
        WHERE h.portfolio_id = p_portfolio_id
          AND h.as_of_date = p_as_of_date
          AND h.current_value > 0
        GROUP BY 
            CASE p_dimension
                WHEN 'security' THEN h.ticker || ' - ' || h.name
                WHEN 'sector' THEN COALESCE(h.sector, 'UNCLASSIFIED')
                WHEN 'issuer' THEN COALESCE(h.issuer, 'UNCLASSIFIED')
                WHEN 'asset_class' THEN COALESCE(h.asset_class, 'UNCLASSIFIED')
                ELSE 'Other'
            END
    )
    SELECT 
        cd.dim_value,
        cd.conc_pct,
        cd.value,
        CASE 
            WHEN cd.conc_pct >= 40 THEN 'HIGH'
            WHEN cd.conc_pct >= 25 THEN 'MEDIUM'
            ELSE 'LOW'
        END AS risk_level,
        cd.conc_pct > p_threshold_pct AS exceeds_threshold
    FROM concentration_data cd
    WHERE cd.conc_pct > 1.0  -- Only show >1% positions
    ORDER BY cd.conc_pct DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- 5. SCENARIO MODELING (WHAT-IF ANALYSIS)
CREATE OR REPLACE FUNCTION model_portfolio_scenario(
    p_portfolio_id UUID,
    p_scenario_changes JSONB,  -- {"AAPL": {"new_shares": 100, "new_price": 180}, ...}
    p_as_of_date DATE DEFAULT CURRENT_DATE,
    p_tenant_id UUID DEFAULT NULL,
    p_datasource_id UUID DEFAULT NULL
)
RETURNS TABLE (
    ticker TEXT,
    security_name TEXT,
    current_shares NUMERIC,
    current_price NUMERIC,
    current_value NUMERIC,
    scenario_shares NUMERIC,
    scenario_price NUMERIC,
    scenario_value NUMERIC,
    value_change NUMERIC,
    value_change_pct NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        h.ticker,
        h.name,
        h.shares,
        h.current_value / NULLIF(h.shares, 0) AS current_price,
        h.current_value,
        COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_shares'::TEXT,
            h.shares::TEXT
        )::NUMERIC AS scenario_shares,
        COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_price'::TEXT,
            (h.current_value / NULLIF(h.shares, 0))::TEXT
        )::NUMERIC AS scenario_price,
        COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_shares'::TEXT,
            h.shares::TEXT
        )::NUMERIC * COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_price'::TEXT,
            (h.current_value / NULLIF(h.shares, 0))::TEXT
        )::NUMERIC AS scenario_value,
        (COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_shares'::TEXT,
            h.shares::TEXT
        )::NUMERIC * COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_price'::TEXT,
            (h.current_value / NULLIF(h.shares, 0))::TEXT
        )::NUMERIC) - h.current_value AS value_change,
        (((COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_shares'::TEXT,
            h.shares::TEXT
        )::NUMERIC * COALESCE(
            (p_scenario_changes->>h.ticker::TEXT)::JSONB->>'new_price'::TEXT,
            (h.current_value / NULLIF(h.shares, 0))::TEXT
        )::NUMERIC) - h.current_value) / NULLIF(h.current_value, 0) * 100) AS value_change_pct
    FROM holdings h
    WHERE h.portfolio_id = p_portfolio_id
      AND h.as_of_date = p_as_of_date
    ORDER BY ABS(value_change) DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- Grant permissions (adjust as needed for your security model)
GRANT EXECUTE ON FUNCTION analyze_portfolio_drill_down TO PUBLIC;
GRANT EXECUTE ON FUNCTION aggregate_household_holdings TO PUBLIC;
GRANT EXECUTE ON FUNCTION calculate_portfolio_performance TO PUBLIC;
GRANT EXECUTE ON FUNCTION analyze_concentration_risk TO PUBLIC;
GRANT EXECUTE ON FUNCTION model_portfolio_scenario TO PUBLIC;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_holdings_portfolio_date ON holdings(portfolio_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_holdings_security ON holdings(security_id);
CREATE INDEX IF NOT EXISTS idx_transactions_portfolio_date ON transactions(portfolio_id, transaction_date);
CREATE INDEX IF NOT EXISTS idx_portfolios_household ON portfolios(household_id);
