-- Tax Planning & Optimization Engine Schema
-- Phase 5: Integrated Tax Optimization

-- ===========================
-- TAX OPTIMIZATION OPPORTUNITIES TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS tax_optimization_opportunities (
    opportunity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    opportunity_type VARCHAR(100) NOT NULL CHECK (opportunity_type IN (
        'TAX_LOSS_HARVEST',
        'ROTH_CONVERSION',
        'CHARITABLE_DONATION',
        'ASSET_LOCATION',
        'MUNICIPAL_BOND_SWAP',
        'CAPITAL_GAIN_HARVEST',
        'QUALIFIED_DIVIDEND_OPTIMIZATION',
        'ESTATE_TAX_PLANNING'
    )),
    detected_date DATE DEFAULT CURRENT_DATE,
    
    -- Opportunity details
    estimated_tax_savings DECIMAL(10,2),
    implementation_complexity VARCHAR(20) CHECK (implementation_complexity IN (
        'LOW',
        'MEDIUM',
        'HIGH'
    )),
    time_sensitivity VARCHAR(50) CHECK (time_sensitivity IN (
        'BEFORE_YEAR_END',
        'BEFORE_RMD_AGE',
        'BEFORE_TAX_LAW_CHANGE',
        'ANYTIME'
    )),
    
    -- Actions required
    recommended_actions JSONB, -- Array of action steps
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'IDENTIFIED' CHECK (status IN (
        'IDENTIFIED',
        'PRESENTED_TO_CLIENT',
        'APPROVED',
        'IMPLEMENTED',
        'DECLINED',
        'EXPIRED'
    )),
    advisor_notes TEXT,
    
    -- Implementation tracking
    presented_date DATE,
    approved_date DATE,
    implemented_date DATE,
    actual_tax_savings DECIMAL(10,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tax_opp_client ON tax_optimization_opportunities(client_id);
CREATE INDEX idx_tax_opp_type ON tax_optimization_opportunities(opportunity_type);
CREATE INDEX idx_tax_opp_status ON tax_optimization_opportunities(status);
CREATE INDEX idx_tax_opp_time_sensitive ON tax_optimization_opportunities(time_sensitivity, detected_date) 
    WHERE status IN ('IDENTIFIED', 'PRESENTED_TO_CLIENT', 'APPROVED');

-- ===========================
-- TAX OPPORTUNITY DETECTION FUNCTION
-- ===========================
CREATE OR REPLACE FUNCTION detect_tax_loss_harvesting_opportunities()
RETURNS INTEGER AS $$
DECLARE
    opportunities_created INTEGER := 0;
BEGIN
    -- Detect tax-loss harvesting (unrealized losses > $3k)
    INSERT INTO tax_optimization_opportunities (
        client_id, 
        opportunity_type, 
        estimated_tax_savings, 
        implementation_complexity,
        time_sensitivity,
        recommended_actions
    )
    SELECT 
        c.client_id,
        'TAX_LOSS_HARVEST',
        ABS(SUM(h.unrealized_gain_loss)) * 0.37, -- Assume 37% tax bracket
        'LOW',
        CASE 
            WHEN EXTRACT(MONTH FROM CURRENT_DATE) >= 10 THEN 'BEFORE_YEAR_END'
            ELSE 'ANYTIME'
        END,
        jsonb_build_object(
            'positions_to_sell', jsonb_agg(h.ticker),
            'total_loss', SUM(h.unrealized_gain_loss),
            'wash_sale_warning', 'Avoid repurchasing within 30 days'
        )
    FROM portfolio_holdings h
    JOIN clients c ON h.client_id = c.client_id
    WHERE h.unrealized_gain_loss < -3000
      AND EXTRACT(MONTH FROM CURRENT_DATE) >= 10 -- Q4 tax planning season
    GROUP BY c.client_id
    HAVING SUM(h.unrealized_gain_loss) < -3000
    ON CONFLICT DO NOTHING;
    
    GET DIAGNOSTICS opportunities_created = ROW_COUNT;
    
    RETURN opportunities_created;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION detect_roth_conversion_opportunities()
RETURNS INTEGER AS $$
DECLARE
    opportunities_created INTEGER := 0;
BEGIN
    -- Detect Roth conversion opportunities (low income years)
    INSERT INTO tax_optimization_opportunities (
        client_id,
        opportunity_type,
        estimated_tax_savings,
        implementation_complexity,
        time_sensitivity,
        recommended_actions
    )
    SELECT 
        c.client_id,
        'ROTH_CONVERSION_OPPORTUNITY',
        50000 * (0.37 - 0.24), -- Savings from converting in 24% vs future 37% bracket
        'MEDIUM',
        'BEFORE_YEAR_END',
        jsonb_build_object(
            'conversion_amount', 50000,
            'current_bracket', '24%',
            'projected_future_bracket', '37%',
            'rationale', 'Income down 30%+ from average'
        )
    FROM clients c
    WHERE c.current_year_income < c.average_annual_income * 0.7 -- Income down 30%
      AND c.age < 60 -- Time for growth
      AND EXISTS (
          SELECT 1 FROM accounts 
          WHERE client_id = c.client_id 
            AND account_type = 'TRADITIONAL_IRA'
      )
    ON CONFLICT DO NOTHING;
    
    GET DIAGNOSTICS opportunities_created = ROW_COUNT;
    
    RETURN opportunities_created;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE tax_optimization_opportunities IS 'Proactively detected tax optimization opportunities with savings estimates';
COMMENT ON FUNCTION detect_tax_loss_harvesting_opportunities IS 'Automatically detect tax-loss harvesting opportunities in Q4';
COMMENT ON FUNCTION detect_roth_conversion_opportunities IS 'Automatically detect Roth conversion opportunities during low-income years';
