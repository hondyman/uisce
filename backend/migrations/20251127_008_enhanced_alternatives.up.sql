-- Enhanced alternative investments schema
-- Multi-asset class support with performance metrics

-- Add asset class enum
DO $$ BEGIN
    CREATE TYPE alt_asset_class AS ENUM (
        'PRIVATE_EQUITY',
        'VENTURE_CAPITAL',
        'HEDGE_FUND',
        'REAL_ESTATE',
        'PRIVATE_CREDIT',
        'INFRASTRUCTURE',
        'COLLECTIBLES',
        'COMMODITIES'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Enhance alternative_investments table
ALTER TABLE alternative_investments
ADD COLUMN IF NOT EXISTS asset_class alt_asset_class,
ADD COLUMN IF NOT EXISTS vintage_year INTEGER,
ADD COLUMN IF NOT EXISTS fund_strategy VARCHAR(100),
ADD COLUMN IF NOT EXISTS geography VARCHAR(100), -- 'North America', 'Europe', 'Asia', 'Global'
ADD COLUMN IF NOT EXISTS industry_focus VARCHAR(100), -- 'Technology', 'Healthcare', 'Energy', etc.

-- Performance metrics (IRR, MOIC, PME, quartile ranking)
ADD COLUMN IF NOT EXISTS performance_metrics JSONB DEFAULT '{}'::jsonb,
-- Example: {"irr": 0.185, "moic": 2.4, "dpi": 0.8, "rvpi": 1.6, "tvpi": 2.4, "pme_ks": 1.15, "vintage_quartile": 1}

-- Industry-specific KPIs
ADD COLUMN IF NOT EXISTS industry_kpis JSONB DEFAULT '{}'::jsonb;
-- Example for PE: {"revenue_growth": 0.25, "ebitda_margin": 0.18, "leverage_ratio": 3.2}
-- Example for VC: {"arr": 5000000, "burn_multiple": 1.5, "magic_number": 0.8}
-- Example for RE: {"occupancy_rate": 0.95, "noi_growth": 0.08, "cap_rate": 0.065}

-- Capital call forecasting
DO $$
DECLARE
  ref_col TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'alternative_investments') THEN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'id') THEN
      ref_col := 'id';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'investment_id') THEN
      ref_col := 'investment_id';
    ELSE
      ref_col := NULL;
    END IF;
  ELSE
    ref_col := NULL;
  END IF;

  IF ref_col IS NOT NULL THEN
    EXECUTE format($f$
      CREATE TABLE IF NOT EXISTS capital_call_forecasts (
          forecast_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
          investment_id UUID NOT NULL REFERENCES alternative_investments(%I),

          -- Forecast details
          forecasted_call_date DATE NOT NULL,
          estimated_amount DECIMAL(15,2) NOT NULL,
          confidence_score DECIMAL(3,2), -- ML model confidence (0.0 - 1.0)
          forecast_method VARCHAR(50), -- 'HISTORICAL_PATTERN', 'GP_GUIDANCE', 'ML_PREDICTIVE'

          -- Cash planning
          liquidity_check_status VARCHAR(50), -- 'SUFFICIENT', 'MARGINAL', 'INSUFFICIENT'
          available_liquid_cash DECIMAL(15,2),
          recommended_funding_source UUID REFERENCES accounts(id),

          -- Alert management
          days_notice_before_due INTEGER DEFAULT 14,
          alert_sent BOOLEAN DEFAULT FALSE,
          alert_sent_at TIMESTAMPTZ,

          -- Actual outcome (for model training)
          actual_call_date DATE,
          actual_amount DECIMAL(15,2),
          forecast_accuracy_pct DECIMAL(5,2),

          created_at TIMESTAMPTZ DEFAULT NOW(),
          updated_at TIMESTAMPTZ DEFAULT NOW()
      );
    $f$, ref_col);
  ELSE
    -- CREATE TABLE IF NOT EXISTS without a foreign key if the referenced column is missing
    CREATE TABLE IF NOT EXISTS capital_call_forecasts (
        forecast_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        investment_id UUID NOT NULL,

        -- Forecast details
        forecasted_call_date DATE NOT NULL,
        estimated_amount DECIMAL(15,2) NOT NULL,
        confidence_score DECIMAL(3,2), -- ML model confidence (0.0 - 1.0)
        forecast_method VARCHAR(50), -- 'HISTORICAL_PATTERN', 'GP_GUIDANCE', 'ML_PREDICTIVE'

        -- Cash planning
        liquidity_check_status VARCHAR(50), -- 'SUFFICIENT', 'MARGINAL', 'INSUFFICIENT'
        available_liquid_cash DECIMAL(15,2),
        recommended_funding_source UUID,

        -- Alert management
        days_notice_before_due INTEGER DEFAULT 14,
        alert_sent BOOLEAN DEFAULT FALSE,
        alert_sent_at TIMESTAMPTZ,

        -- Actual outcome (for model training)
        actual_call_date DATE,
        actual_amount DECIMAL(15,2),
        forecast_accuracy_pct DECIMAL(5,2),

        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    );
  END IF;
END$$;

-- PME (Public Market Equivalent) benchmarking
DO $$
DECLARE
  ref_col TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'alternative_investments') THEN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'id') THEN
      ref_col := 'id';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'investment_id') THEN
      ref_col := 'investment_id';
    ELSE
      ref_col := NULL;
    END IF;
  ELSE
    ref_col := NULL;
  END IF;

  IF ref_col IS NOT NULL THEN
    EXECUTE format($f$
      CREATE TABLE IF NOT EXISTS alt_investment_benchmarks (
          benchmark_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
          investment_id UUID NOT NULL REFERENCES alternative_investments(%I),

          -- PME calculations
          pme_kaplan_schoar DECIMAL(6,4), -- Ratio > 1.0 means outperformed public markets
          pme_direct_alpha DECIMAL(15,2), -- Dollar value of outperformance

          -- Benchmark index used
          benchmark_index VARCHAR(100), -- 'S&P 500', 'Russell 2000', 'MSCI World'
          benchmark_start_date DATE,
          benchmark_end_date DATE,

          -- Peer comparison
          peer_group_name VARCHAR(200),
          peer_median_irr DECIMAL(6,4),
          peer_top_quartile_irr DECIMAL(6,4),
          fund_percentile_rank INTEGER, -- 1-100

          calculation_date TIMESTAMPTZ DEFAULT NOW(),
          created_at TIMESTAMPTZ DEFAULT NOW()
      );
    $f$, ref_col);
  ELSE
    CREATE TABLE IF NOT EXISTS alt_investment_benchmarks (
        benchmark_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        investment_id UUID NOT NULL,

        -- PME calculations
        pme_kaplan_schoar DECIMAL(6,4), -- Ratio > 1.0 means outperformed public markets
        pme_direct_alpha DECIMAL(15,2), -- Dollar value of outperformance

        -- Benchmark index used
        benchmark_index VARCHAR(100), -- 'S&P 500', 'Russell 2000', 'MSCI World'
        benchmark_start_date DATE,
        benchmark_end_date DATE,

        -- Peer comparison
        peer_group_name VARCHAR(200),
        peer_median_irr DECIMAL(6,4),
        peer_top_quartile_irr DECIMAL(6,4),
        fund_percentile_rank INTEGER, -- 1-100

        calculation_date TIMESTAMPTZ DEFAULT NOW(),
        created_at TIMESTAMPTZ DEFAULT NOW()
    );
  END IF;
END$$;

-- Enhanced capital calls tracking with liquidity checks
ALTER TABLE capital_calls
ADD COLUMN IF NOT EXISTS liquidity_check_passed BOOLEAN,
ADD COLUMN IF NOT EXISTS recommended_action TEXT,
ADD COLUMN IF NOT EXISTS days_advance_notice INTEGER;

-- Function to check capital call liquidity
CREATE OR REPLACE FUNCTION check_capital_call_liquidity()
RETURNS TRIGGER AS $$
DECLARE
    v_client_id UUID;
    v_total_liquid_cash DECIMAL(15,2);
    v_liquidity_ratio DECIMAL(5,2);
BEGIN
    -- Get client ID
    SELECT client_id INTO v_client_id
    FROM alternative_investments
    WHERE id = NEW.investment_id;
    
    -- Calculate total liquid cash
    SELECT COALESCE(SUM(balance), 0) INTO v_total_liquid_cash
    FROM accounts
    WHERE client_id = v_client_id
      AND account_type IN ('CHECKING', 'SAVINGS', 'MONEY_MARKET')
      AND is_active = TRUE;
    
    -- Calculate liquidity ratio
    v_liquidity_ratio := v_total_liquid_cash / NULLIF(NEW.amount_requested, 0);
    
    -- Determine liquidity status
    NEW.liquidity_check_passed := CASE
        WHEN v_liquidity_ratio >= 1.5 THEN TRUE -- Sufficient buffer
        WHEN v_liquidity_ratio >= 1.1 THEN TRUE -- Marginal but OK
        ELSE FALSE -- Insufficient
    END;
    
    -- Set recommended action
    IF v_liquidity_ratio < 1.0 THEN
        NEW.recommended_action := format(
            'URGENT: Liquidate $%s from portfolio. Available cash: $%s, Required: $%s',
            (NEW.amount_requested - v_total_liquid_cash),
            v_total_liquid_cash,
            NEW.amount_requested
        );
        
        -- Create high-priority alert
        INSERT INTO advisor_alerts (alert_type, priority, client_id, message, created_at)
        VALUES (
            'CAPITAL_CALL_SHORTFALL',
            'HIGH',
            v_client_id,
            format('Client has insufficient liquidity for capital call of $%s due on %s. Available: $%s',
                   NEW.amount_requested, NEW.due_date, v_total_liquid_cash),
            NOW()
        );
    ELSIF v_liquidity_ratio < 1.5 THEN
        NEW.recommended_action := 'CAUTION: Low liquidity buffer. Consider maintaining additional cash reserves.';
    ELSE
        NEW.recommended_action := 'Sufficient liquidity confirmed.';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS trigger_capital_call_liquidity_check ON capital_calls;
CREATE TRIGGER trigger_capital_call_liquidity_check
    BEFORE INSERT OR UPDATE ON capital_calls
    FOR EACH ROW
    EXECUTE FUNCTION check_capital_call_liquidity();

-- Enhanced document tracking with AI extraction status
ALTER TABLE alt_investment_documents
ADD COLUMN IF NOT EXISTS ai_extraction_status VARCHAR(50) DEFAULT 'PENDING',
ADD COLUMN IF NOT EXISTS extraction_confidence DECIMAL(3,2),
ADD COLUMN IF NOT EXISTS extracted_data JSONB DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS extraction_errors TEXT[];

-- Index for AI document processing queue
CREATE INDEX IF NOT EXISTS idx_alt_docs_ai_pending 
ON alt_investment_documents(ai_extraction_status, uploaded_at)
WHERE ai_extraction_status = 'PENDING';

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_alt_inv_asset_class ON alternative_investments(asset_class);
CREATE INDEX IF NOT EXISTS idx_alt_inv_vintage ON alternative_investments(vintage_year);
CREATE INDEX IF NOT EXISTS idx_capital_call_forecasts_date ON capital_call_forecasts(forecasted_call_date);
CREATE INDEX IF NOT EXISTS idx_capital_call_forecasts_investment ON capital_call_forecasts(investment_id);

-- View: Alternative investments performance summary
DO $$
DECLARE
  ai_id_col TEXT := 'id';
  fund_col TEXT := NULL;
  has_commitment BOOL := FALSE;
  has_funded BOOL := FALSE;
  has_unfunded BOOL := FALSE;
  has_perf BOOL := FALSE;
  has_inception BOOL := FALSE;
  has_current_nav BOOL := FALSE;
  sql TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'id') THEN
    ai_id_col := 'id';
  ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'investment_id') THEN
    ai_id_col := 'investment_id';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'fund_name') THEN
    fund_col := 'fund_name';
  ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'name') THEN
    fund_col := 'name';
  END IF;

  has_commitment := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'commitment_amount');
  has_funded := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'funded_amount');
  has_unfunded := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'unfunded_commitment');
  has_perf := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'performance_metrics');
  has_inception := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'inception_date');
  has_current_nav := EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'current_nav');

  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'alternative_investments')
     AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'alt_investment_benchmarks')
     AND fund_col IS NOT NULL AND has_perf THEN

    sql := 'CREATE OR REPLACE VIEW alt_investments_performance_summary AS SELECT ' || format('ai.%I AS investment_id, ', ai_id_col);
    sql := sql || format('COALESCE(ai.%I, '''') AS fund_name, ', fund_col);
    sql := sql || 'ai.asset_class, ai.vintage_year, ';
    sql := sql || CASE WHEN has_commitment THEN 'COALESCE(ai.commitment_amount, 0) AS commitment_amount, ' ELSE '0::numeric AS commitment_amount, ' END;
    sql := sql || CASE WHEN has_funded THEN 'COALESCE(ai.funded_amount, 0) AS funded_amount, ' ELSE '0::numeric AS funded_amount, ' END;
    sql := sql || CASE WHEN has_unfunded THEN 'COALESCE(ai.unfunded_commitment, 0) AS unfunded_commitment, ' ELSE '0::numeric AS unfunded_commitment, ' END;
    sql := sql || CASE WHEN has_perf THEN '(ai.performance_metrics->>''irr'')::DECIMAL AS irr, (ai.performance_metrics->>''moic'')::DECIMAL AS moic, (ai.performance_metrics->>''tvpi'')::DECIMAL AS tvpi, (ai.performance_metrics->>''dpi'')::DECIMAL AS dpi, (ai.performance_metrics->>''rvpi'')::DECIMAL AS rvpi, ' ELSE '0::numeric AS irr, 0::numeric AS moic, 0::numeric AS tvpi, 0::numeric AS dpi, 0::numeric AS rvpi, ' END;
    sql := sql || 'b.pme_kaplan_schoar, b.fund_percentile_rank, ';
    sql := sql || CASE WHEN has_inception THEN 'CASE WHEN ai.inception_date IS NOT NULL AND EXTRACT(YEAR FROM AGE(NOW(), ai.inception_date)) < 3 THEN ''INVESTMENT_PHASE'' WHEN ai.inception_date IS NOT NULL AND EXTRACT(YEAR FROM AGE(NOW(), ai.inception_date)) BETWEEN 3 AND 7 THEN ''HARVESTING_PHASE'' ELSE ''MATURE'' END AS j_curve_position, ' ELSE '''UNKNOWN'' AS j_curve_position, ' END;
    sql := sql || CASE WHEN has_current_nav THEN 'ai.current_nav ' ELSE 'NULL::numeric AS current_nav ' END;
    sql := sql || format('FROM alternative_investments ai LEFT JOIN alt_investment_benchmarks b ON ai.%I = b.investment_id;', ai_id_col);

    EXECUTE 'DROP VIEW IF EXISTS alt_investments_performance_summary';
    EXECUTE sql;
  ELSE
    CREATE OR REPLACE VIEW alt_investments_performance_summary AS
      SELECT NULL::uuid AS investment_id, ''::text AS fund_name, NULL::text AS asset_class, NULL::int AS vintage_year, 0::numeric AS commitment_amount, 0::numeric AS funded_amount, 0::numeric AS unfunded_commitment, 0::numeric AS irr, 0::numeric AS moic, 0::numeric AS tvpi, 0::numeric AS dpi, 0::numeric AS rvpi, NULL::numeric AS pme_kaplan_schoar, NULL::int AS fund_percentile_rank, 'UNKNOWN' AS j_curve_position, NULL::numeric AS current_nav WHERE FALSE;
  END IF;
END$$;

-- View: Upcoming capital calls with liquidity status
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'capital_calls')
     AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'alternative_investments')
     AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'clients')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'capital_calls' AND column_name = 'investment_id')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'capital_calls' AND column_name = 'due_date')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'id')
     AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alternative_investments' AND column_name = 'client_id')
     AND (EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'client_name') OR EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'name')) THEN

    DROP VIEW IF EXISTS upcoming_capital_calls_with_liquidity;
    CREATE OR REPLACE VIEW upcoming_capital_calls_with_liquidity AS
    SELECT
      cc.call_id,
      cc.investment_id,
      ai.fund_name AS fund_name,
      ai.client_id AS client_id,
      COALESCE(c.client_name, c.name) AS client_name,
      cc.call_date,
      cc.due_date,
      cc.amount_requested,
      cc.liquidity_check_passed,
      cc.recommended_action,
      EXTRACT(DAY FROM (cc.due_date - NOW())) AS days_until_due,
      ccf.confidence_score AS forecast_confidence
    FROM capital_calls cc
    JOIN alternative_investments ai ON cc.investment_id = ai.id
    JOIN clients c ON ai.client_id = c.id
    LEFT JOIN capital_call_forecasts ccf ON cc.investment_id = ccf.investment_id
    WHERE cc.payment_status != 'PAID' AND cc.due_date >= NOW()
    ORDER BY cc.due_date ASC;
  ELSE
    CREATE OR REPLACE VIEW upcoming_capital_calls_with_liquidity AS
      SELECT NULL::uuid AS call_id, NULL::uuid AS investment_id, ''::text AS fund_name, NULL::uuid AS client_id, ''::text AS client_name, NULL::date AS call_date, NULL::date AS due_date, 0::numeric AS amount_requested, FALSE AS liquidity_check_passed, ''::text AS recommended_action, NULL::int AS days_until_due, NULL::numeric AS forecast_confidence WHERE FALSE;
  END IF;
END$$;
COMMENT ON TABLE capital_call_forecasts IS 'ML-powered capital call predictions with liquidity analysis';
COMMENT ON TABLE alt_investment_benchmarks IS 'PME benchmarking and peer group comparisons';
COMMENT ON COLUMN alternative_investments.performance_metrics IS 'IRR, MOIC, DPI, RVPI, TVPI, PME metrics as JSON';
COMMENT ON COLUMN alternative_investments.industry_kpis IS 'Asset-class specific KPIs (revenue growth, occupancy, ARR, etc.)';
