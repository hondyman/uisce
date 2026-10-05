-- 20261218_005_core_rule_library_seed.up.sql
--
-- Core Gold-Copy Rule Library v1 — 50 Rules Seed Migration
-- Seeded under dynamic master gold-copy tenant
-- Categorized into licensable rulesets: CORE_REGULATORY, MARKET_CONDUCT, INSTITUTIONAL_CONTROLS
-- Rules with scenario corpus coverage are ACTIVE; un-corpus'd rules are PROVISIONAL.

CREATE OR REPLACE FUNCTION compliance.seed_core_rule(
    p_rule_code TEXT,
    p_name TEXT,
    p_phase TEXT,
    p_severity TEXT,
    p_priority INT,
    p_citation TEXT,
    p_jurisdictions TEXT[],
    p_params JSONB,
    p_ast JSONB,
    p_status TEXT DEFAULT 'PROVISIONAL'
) RETURNS UUID AS $$
DECLARE
    v_id UUID;
    v_phase TEXT := p_phase;
    v_gold_tenant UUID;
BEGIN
    IF v_phase = 'PRE' THEN v_phase := 'PRE_TRADE';
    ELSIF v_phase = 'POST' THEN v_phase := 'POST_TRADE';
    END IF;

    -- Resolve gold-copy tenant ID
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        BEGIN
            SELECT public.uisce_gold_copy_tenant_id() INTO v_gold_tenant;
        EXCEPTION WHEN OTHERS THEN
            v_gold_tenant := '00000000-0000-4000-a000-000000000000'::uuid;
        END;
    END IF;
    IF v_gold_tenant IS NULL THEN
        v_gold_tenant := '00000000-0000-4000-a000-000000000000'::uuid;
    END IF;

    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(),
        v_gold_tenant,
        'inherit',
        p_rule_code,
        p_name,
        v_phase,
        p_severity,
        p_priority,
        true,
        p_citation,
        p_jurisdictions,
        p_params,
        p_ast,
        '2026-01-01T00:00:00Z',
        'CORE_LIB_V1',
        p_status
    )
    ON CONFLICT (tenant_id, rule_code) DO UPDATE SET
        name = EXCLUDED.name,
        rule_phase = EXCLUDED.rule_phase,
        severity = EXCLUDED.severity,
        priority = EXCLUDED.priority,
        citation = EXCLUDED.citation,
        jurisdictions = EXCLUDED.jurisdictions,
        parameter_thresholds = EXCLUDED.parameter_thresholds,
        ast_condition = EXCLUDED.ast_condition,
        source_version = EXCLUDED.source_version,
        effective_from = EXCLUDED.effective_from,
        library_status = EXCLUDED.library_status,
        updated_at = now()
    RETURNING id INTO v_id;

    RETURN v_id;
END $$ LANGUAGE plpgsql;

-- ====================================================================
-- A. CONCENTRATION & DIVERSIFICATION (6)
-- ====================================================================
SELECT compliance.seed_core_rule('UCITS_ISSUER_5',
  'UCITS 5% Single-Issuer Concentration', 'BOTH', 'HARD_BLOCK', 100,
  'UCITS Directive 2009/65/EC Annex IV, implementing Directive 2010/43/EU Art. 44; ESMA Guidelines ESMA/2014/937',
  ARRAY['EU'],
  '{"issuer_limit_pct":"0.050000","lookthrough":true,"aggregate_across_accounts":true}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.issuer_exposure_pct"},"right":{"type":"PARAM","name":"issuer_limit_pct"}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('UCITS_ISSUER_10_EXCEPTION',
  'UCITS 10% Post-Acquisition Exception', 'POST_TRADE', 'APPROVAL_REQUIRED', 90,
  'UCITS Annex IV (5->10 exception, 6-month divestment window)',
  ARRAY['EU'],
  '{"grandfather_days":180,"iss10_limit_pct":"0.100000"}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.issuer_exposure_pct"},"right":{"type":"PARAM","name":"iss10_limit_pct"}},{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.days_over_5pct"},"right":{"type":"PARAM","name":"grandfather_days"}}]}',
  'ACTIVE');

SELECT compliance.seed_core_rule('UCITS_ISSUER_40',
  'UCITS 40% Basket Issuer Limit', 'POST_TRADE', 'HARD_BLOCK', 90,
  'UCITS Directive 2009/65/EC Art. 52 (5/10/40 concentration limit)',
  ARRAY['EU'],
  '{"iss40_limit_pct":"0.400000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.issuer_exposure_pct"},"right":{"type":"PARAM","name":"iss40_limit_pct"}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('UCITS_DEPOSIT_20',
  'UCITS 20% Deposit Institution Limit', 'BOTH', 'HARD_BLOCK', 95,
  'UCITS Annex IV Art. 45',
  ARRAY['EU'],
  '{"deposit_limit_pct":"0.200000","exception_pct":"0.300000","exception_days":180}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"deposit.institution_exposure_pct"},"right":{"type":"PARAM","name":"deposit_limit_pct"}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('ACT40_DIV_75_5',
  '40-Act Diversification 75/5', 'BOTH', 'SOFT_WARNING', 80,
  'Investment Company Act 1940 §8(b)(1)',
  ARRAY['US'],
  '{"compliant_assets_pct":"0.750000","issuer_limit_pct":"0.050000"}',
  '{"type":"OR","children":[{"type":"COMPARISON","op":"LT","left":{"type":"METRIC","path":"fund.compliant_assets_pct"},"right":{"type":"PARAM","name":"compliant_assets_pct"}},{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.issuer_exposure_pct"},"right":{"type":"PARAM","name":"issuer_limit_pct"}}]}',
  'ACTIVE');

SELECT compliance.seed_core_rule('FOF_20',
  'UCITS Fund-of-Funds 20% Target Fund Exposure', 'BOTH', 'HARD_BLOCK', 95,
  'ESMA/2013/1208 Guidelines on ETFs and other UCITS issues',
  ARRAY['EU'],
  '{"fof_limit_pct":"0.200000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.target_fund_exposure_pct"},"right":{"type":"PARAM","name":"fof_limit_pct"}}',
  'ACTIVE');

-- ====================================================================
-- B. ELIGIBILITY, RESTRICTED & SANCTIONS LISTS (6)
-- ====================================================================
SELECT compliance.seed_core_rule('SEC_144A_ELIGIBILITY',
  'SEC Rule 144A QIB Eligibility', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'SEC Rule 144A (Resale of Restricted Securities to Qualified Institutional Buyers)',
  ARRAY['US'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.restricted_144a"},"right":{"type":"LITERAL","value":true}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"account.qib_status"},"right":{"type":"LITERAL","value":false}}]}',
  'ACTIVE');

SELECT compliance.seed_core_rule('REG_S_OFFSHORE_ONLY',
  'Regulation S Offshore Offering Jurisdictional Gate', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'Securities Act 1933 Regulation S §5',
  ARRAY['US', 'GLOBAL'],
  '{"domestic_jurisdictions":["US"]}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.reg_s"},"right":{"type":"LITERAL","value":true}},{"type":"IN_LIST","target":{"type":"METRIC","path":"order.jurisdiction"},"list":{"type":"PARAM","name":"domestic_jurisdictions"}}]}',
  'ACTIVE');

SELECT compliance.seed_core_rule('RESTRICTED_LIST_BLOCK',
  'House Restricted Security Trading Block', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'House / Tenant Restricted Security Governance Policy',
  ARRAY['GLOBAL'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.in_restricted_list"},"right":{"type":"LITERAL","value":true}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('INSIDER_LIST_MAR',
  'EU MAR Insider List Dealing Prohibition', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'EU Market Abuse Regulation (MAR) Regulation (EU) No 596/2014 Art. 11 & Art. 18',
  ARRAY['EU'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.in_insider_list"},"right":{"type":"LITERAL","value":true}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('SANCTIONS_ISSUER_BLOCK',
  'OFAC / EU Sanctioned Issuer Block', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'OFAC Specially Designated Nationals / EU Regulation 833/2014 Sanctions',
  ARRAY['US', 'EU', 'UN'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.issuer_is_sanctioned"},"right":{"type":"LITERAL","value":true}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('COUNTRY_EXPOSURE_LIMIT',
  'Sovereign / Country Concentration Limit', 'BOTH', 'SOFT_WARNING', 70,
  'House Sovereign & Country Exposure Risk Policy',
  ARRAY['GLOBAL'],
  '{"country_limit_pct":"0.150000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.country_exposure_pct"},"right":{"type":"PARAM","name":"country_limit_pct"}}',
  'ACTIVE');

-- ====================================================================
-- C. TRADING CONDUCT & MARKET ABUSE (8)
-- ====================================================================
SELECT compliance.seed_core_rule('REG_M_RULE_105',
  'SEC Reg M Rule 105 Short Sale Prior to Offering', 'PRE_TRADE', 'HARD_BLOCK', 95,
  'SEC Regulation M Rule 105 (17 CFR § 242.105)',
  ARRAY['US'],
  '{"restricted_window_days":5}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.side"},"right":{"type":"LITERAL","value":"SHORT"}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"offering.is_covered"},"right":{"type":"LITERAL","value":true}},{"type":"COMPARISON","op":"LTE","left":{"type":"METRIC","path":"order.days_before_pricing"},"right":{"type":"PARAM","name":"restricted_window_days"}}]}',
  'ACTIVE');

SELECT compliance.seed_core_rule('WASH_SALE_1091',
  'IRC §1091 Wash Sale 30-Day Window Block', 'BOTH', 'HARD_BLOCK', 90,
  'Internal Revenue Code (IRC) §1091 (Loss from wash sales of stock or securities)',
  ARRAY['US'],
  '{"window_days":30}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.side"},"right":{"type":"LITERAL","value":"BUY"}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"position.realized_loss_30d"},"right":{"type":"LITERAL","value":true}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"position.same_beneficial_owner"},"right":{"type":"LITERAL","value":true}}]}',
  'ACTIVE');

SELECT compliance.seed_core_rule('FRONT_RUNNING_CLIENT_ORDER',
  'Front-Running Pending Client Order Prohibition', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'SEC Investment Advisers Act §206(1) / EU MAR Art. 14 Prohibition of Insider Dealing',
  ARRAY['GLOBAL'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.is_principal_or_employee"},"right":{"type":"LITERAL","value":true}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"market.pending_client_orders_exist"},"right":{"type":"LITERAL","value":true}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('FREERIDING_REG_T',
  'Federal Reserve Reg T Freeriding Prohibition', 'PRE_TRADE', 'HARD_BLOCK', 95,
  'Federal Reserve Regulation T §220.8(c)(1) 90-Day Cash Account Restriction',
  ARRAY['US'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.side"},"right":{"type":"LITERAL","value":"SELL"}},{"type":"COMPARISON","op":"LT","left":{"type":"METRIC","path":"security.settled_qty"},"right":{"type":"METRIC","path":"order.qty"}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"cash.unpaid"},"right":{"type":"LITERAL","value":true}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('LAYERING_SPOOF_PATTERN',
  'High Order-to-Trade Ratio & Spoofing Pattern', 'POST_TRADE', 'SOFT_WARNING', 85,
  'ESMA RTS 6 / MiFID II Algorithmic Trading Controls & Manipulation Indicators',
  ARRAY['EU', 'GLOBAL'],
  '{"max_cancel_rate":"0.900000","max_otr":"50.000000"}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"trading.cancel_rate_5min"},"right":{"type":"PARAM","name":"max_cancel_rate"}},{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"trading.order_to_trade_ratio"},"right":{"type":"PARAM","name":"max_otr"}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('EXECUTION_PRICE_DEVIATION',
  'Execution Arrival Price Deviation Surveillance', 'POST_TRADE', 'SOFT_WARNING', 75,
  'MiFID II RTS 27/28 Best Execution Quality Assessment',
  ARRAY['EU', 'GLOBAL'],
  '{"max_bench_deviation":"0.020000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"execution.arrival_price_deviation_pct"},"right":{"type":"PARAM","name":"max_bench_deviation"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('SHORT_SALE_LOCATE',
  'Reg SHO 203(b) Short Sale Locate Requirement', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'SEC Regulation SHO Rule 203(b)(1) Locate Requirement',
  ARRAY['US'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.side"},"right":{"type":"LITERAL","value":"SHORT"}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"locate.valid"},"right":{"type":"LITERAL","value":false}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.easy_to_borrow"},"right":{"type":"LITERAL","value":false}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('LULD_PRICE_BAND',
  'SEC LULD Rule 201 Limit Price Band Gate', 'PRE_TRADE', 'HARD_BLOCK', 95,
  'SEC Limit Up-Limit Down (LULD) Plan / Rule 201',
  ARRAY['US'],
  '{}',
  '{"type":"OR","children":[{"type":"COMPARISON","op":"LT","left":{"type":"METRIC","path":"order.limit_price"},"right":{"type":"METRIC","path":"luld.lower_band"}},{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"order.limit_price"},"right":{"type":"METRIC","path":"luld.upper_band"}}]}',
  'PROVISIONAL');

-- ====================================================================
-- D. ORDER-LEVEL PRE-TRADE CONTROLS (8)
-- ====================================================================
SELECT compliance.seed_core_rule('FAT_FINGER_NOTIONAL',
  'Maximum Notional Fat-Finger Limit', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'SEC Rule 15c3-5 Market Access Pre-Trade Capital Threshold',
  ARRAY['GLOBAL'],
  '{"max_notional":"50000000.000000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"order.notional"},"right":{"type":"PARAM","name":"max_notional"}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('FAT_FINGER_ADV_RATIO',
  'Maximum ADV 20-Day Participation Multiple', 'PRE_TRADE', 'SOFT_WARNING', 80,
  'House Market Impact & Order Size Protocol',
  ARRAY['GLOBAL'],
  '{"adv_multiple":"0.250000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"order.adv_ratio"},"right":{"type":"PARAM","name":"adv_multiple"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('PRICE_COLLAR_PCT',
  'Price Collar Percentage Threshold', 'PRE_TRADE', 'HARD_BLOCK', 95,
  'House Price Collar & Volatility Protection Mandate',
  ARRAY['GLOBAL'],
  '{"collar_pct":"0.100000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"order.price_deviation_pct"},"right":{"type":"PARAM","name":"collar_pct"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('DUPLICATE_ORDER_WINDOW',
  'Duplicate Order Burst Rate Control', 'PRE_TRADE', 'SOFT_WARNING', 70,
  'House Erratic & Automated Duplicate Order Guardrail',
  ARRAY['GLOBAL'],
  '{"window_seconds":60}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"order.duplicate_count"},"right":{"type":"LITERAL","value":0}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('SELF_TRADE_PREVENT',
  'Self-Trade Prevention (Wash Execution Filter)', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'FINRA Rule 5210 Self-Trades / House Wash Trade Prevention',
  ARRAY['GLOBAL'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.pending_opposing_order"},"right":{"type":"LITERAL","value":true}}',
  'ACTIVE');

SELECT compliance.seed_core_rule('ORDER_RATE_LIMIT',
  'Order Ingress Rate Limit (Max Orders/Min)', 'PRE_TRADE', 'HARD_BLOCK', 95,
  'SEC Rule 15c3-5 Ingress Order Throttling',
  ARRAY['GLOBAL'],
  '{"max_opm":100}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"account.orders_per_minute"},"right":{"type":"PARAM","name":"max_opm"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('ODD_LOT_ABOVE_MIN',
  'Odd-Lot Market Order Warning', 'PRE_TRADE', 'SOFT_WARNING', 50,
  'House Odd-Lot Best Execution Protocol',
  ARRAY['GLOBAL'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"LT","left":{"type":"METRIC","path":"order.qty"},"right":{"type":"METRIC","path":"security.round_lot"}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.penny_stock"},"right":{"type":"LITERAL","value":false}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('SHORT_POSITION_RESTRICTION',
  'Short Sale Restriction (SSR) Circuit Breaker', 'PRE_TRADE', 'APPROVAL_REQUIRED', 90,
  'SEC Regulation SHO Rule 201 Alternative Uptick Rule Trigger',
  ARRAY['US'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"security.short_sale_restricted"},"right":{"type":"LITERAL","value":true}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.side"},"right":{"type":"LITERAL","value":"SHORT"}}]}',
  'PROVISIONAL');

-- ====================================================================
-- E. POSITION, LEVERAGE & LIQUIDITY (6)
-- ====================================================================
SELECT compliance.seed_core_rule('LEVERAGE_VAR_COMMIT',
  'VaR & Commitment Leverage Ratio Cap', 'POST_TRADE', 'APPROVAL_REQUIRED', 85,
  'UCITS Directive 2009/65/EC Annex IV Art. 48 Global Exposure (VaR / Commitment)',
  ARRAY['EU'],
  '{"max_leverage_var":"0.200000","max_commitment_ratio":"2.000000"}',
  '{"type":"OR","children":[{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.var_leverage_ratio"},"right":{"type":"PARAM","name":"max_leverage_var"}},{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.commitment_ratio"},"right":{"type":"PARAM","name":"max_commitment_ratio"}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('ILLIQUID_ASSET_LIMIT',
  'Open-End Fund Illiquid Asset Cap (15%)', 'BOTH', 'SOFT_WARNING', 80,
  'SEC Rule 22e-4 Open-End Fund Liquidity Risk Management Program',
  ARRAY['US'],
  '{"max_illiquid_pct":"0.150000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"fund.illiquid_assets_pct"},"right":{"type":"PARAM","name":"max_illiquid_pct"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('LIQUIDITY_BUCKET_DAYS',
  'Days to Liquidate 50% Portfolio Threshold', 'BOTH', 'SOFT_WARNING', 75,
  'ESMA Liquidity Guidelines for UCITS and AIFs',
  ARRAY['EU'],
  '{"max_liquid_days":7}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.days_to_liquidate_50pct"},"right":{"type":"PARAM","name":"max_liquid_days"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('MARGIN_HOUSE_LIMIT',
  'Margin Utilization House Capacity Limit', 'PRE_TRADE', 'HARD_BLOCK', 90,
  'House Margin Policy & Federal Reserve Regulation T',
  ARRAY['GLOBAL'],
  '{"house_margin_limit":"0.700000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"account.margin_utilization_pct"},"right":{"type":"PARAM","name":"house_margin_limit"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('SECTOR_CONCENTRATION',
  'Sector & Industry Concentration Limit', 'BOTH', 'SOFT_WARNING', 70,
  'House Sector Exposure Mandate',
  ARRAY['GLOBAL'],
  '{"max_sector_pct":"0.250000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.sector_exposure_pct"},"right":{"type":"PARAM","name":"max_sector_pct"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('SINGLE_POSITION_NAV',
  'Single Security NAV Concentration Limit', 'BOTH', 'SOFT_WARNING', 75,
  'House Single Position Weight Mandate',
  ARRAY['GLOBAL'],
  '{"max_single_nav_pct":"0.100000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"position.nav_weight_pct"},"right":{"type":"PARAM","name":"max_single_nav_pct"}}',
  'PROVISIONAL');

-- ====================================================================
-- F. ALLOCATION, BEST EXECUTION & VENUE (4)
-- ====================================================================
SELECT compliance.seed_core_rule('PRO_RATA_ALLOCATION_FAIRNESS',
  'Pro Rata Trade Allocation Fairness & Price Dispersion', 'POST_TRADE', 'SOFT_WARNING', 85,
  'SEC Advisers Act Rule 204-2 / Fair Allocation Exam Guidance',
  ARRAY['US', 'GLOBAL'],
  '{"max_deviation":"0.020000","max_dispersion":"0.001000"}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"allocation.ratio_deviation"},"right":{"type":"PARAM","name":"max_deviation"}},{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"allocation.price_dispersion"},"right":{"type":"PARAM","name":"max_dispersion"}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('AGGREGATION_ELIGIBILITY',
  'Order Block Aggregation Eligibility Verification', 'PRE_TRADE', 'SOFT_WARNING', 65,
  'House Order Aggregation & Client Fairness Protocol',
  ARRAY['GLOBAL'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.can_be_aggregated"},"right":{"type":"LITERAL","value":false}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"order.parent_is_block"},"right":{"type":"LITERAL","value":true}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('VENUE_APPROVED_LIST',
  'Execution Venue Best-Ex Policy Approved List', 'PRE_TRADE', 'HARD_BLOCK', 90,
  'MiFID II RTS 28 Execution Venue Selection & Governance',
  ARRAY['EU', 'GLOBAL'],
  '{"approved_venues":["XNYS","XNAS","XLON","XFRA","XPAR"]}',
  '{"type":"IN_LIST","target":{"type":"METRIC","path":"order.venue"},"list":{"type":"PARAM","name":"approved_venues"},"negate":true}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('EXECUTION_WITHIN_SPREAD',
  'Execution Within Prevailing Bid/Ask Spread', 'POST_TRADE', 'SOFT_WARNING', 70,
  'MiFID II RTS 27 Best Execution Quality Benchmark',
  ARRAY['EU', 'GLOBAL'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"execution.outside_spread"},"right":{"type":"LITERAL","value":true}}',
  'PROVISIONAL');

-- ====================================================================
-- G. PERSONAL / EMPLOYEE TRADING (4)
-- ====================================================================
SELECT compliance.seed_core_rule('PT_PRECLEARANCE_REQUIRED',
  'Personal Account Dealing Preclearance Gate', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'SEC Investment Company Act Rule 17j-1 / House Code of Ethics',
  ARRAY['US', 'GLOBAL'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"NEQ","left":{"type":"METRIC","path":"order.employee_id"},"right":{"type":"LITERAL","value":null}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"employee.preclearance_valid"},"right":{"type":"LITERAL","value":false}}]}',
  'ACTIVE');

SELECT compliance.seed_core_rule('PT_BLACKOUT_PERIOD',
  'Employee Blackout Window Dealing Prohibition', 'PRE_TRADE', 'HARD_BLOCK', 100,
  'SEC Rule 17j-1 / House Personal Trading Blackout Policy',
  ARRAY['US', 'GLOBAL'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"NEQ","left":{"type":"METRIC","path":"order.employee_id"},"right":{"type":"LITERAL","value":null}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"employee.blackout_active"},"right":{"type":"LITERAL","value":true}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('PT_MIN_HOLDING_30D',
  'Employee Minimum 30-Day Holding Period', 'PRE_TRADE', 'HARD_BLOCK', 90,
  'House Code of Ethics Personal Trading Holding Period Mandate',
  ARRAY['GLOBAL'],
  '{"min_holding_days":30}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"NEQ","left":{"type":"METRIC","path":"order.employee_id"},"right":{"type":"LITERAL","value":null}},{"type":"COMPARISON","op":"LT","left":{"type":"METRIC","path":"employee.days_since_purchase"},"right":{"type":"PARAM","name":"min_holding_days"}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('PT_ACCESS_PERSON_RECON',
  'Access Person Broker Statement Reconciliation Discrepancy', 'POST_TRADE', 'SOFT_WARNING', 80,
  'SEC Rule 17j-1 Access Person Holdings Reporting & Audit',
  ARRAY['US'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"employee.statement_discrepancy"},"right":{"type":"LITERAL","value":true}}',
  'PROVISIONAL');

-- ====================================================================
-- H. REPORTING, SETTLEMENT & OTC (8)
-- ====================================================================
SELECT compliance.seed_core_rule('LARGE_TRADE_THRESHOLD',
  'MiFID II RTS 22 Large in Scale (LIS) Trade Flagging', 'POST_TRADE', 'HARD_BLOCK', 85,
  'MiFID II RTS 22 Large in Scale Pre/Post Trade Transparency',
  ARRAY['EU'],
  '{"large_trade_threshold":"500000.000000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"order.notional"},"right":{"type":"PARAM","name":"large_trade_threshold"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('TXN_REPORT_COMPLETENESS',
  'Transaction Report Mandatory Fields & LEI Validation', 'POST_TRADE', 'HARD_BLOCK', 100,
  'MiFID II RTS 22 Annex I Transaction Reporting Completeness',
  ARRAY['EU', 'UK'],
  '{}',
  '{"type":"OR","children":[{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"report.missing_fields_count"},"right":{"type":"LITERAL","value":0}},{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"report.lei_invalid"},"right":{"type":"LITERAL","value":true}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('TXN_REPORT_TIMELINESS',
  'Transaction Reporting T+1 Timeliness Window', 'POST_TRADE', 'SOFT_WARNING', 80,
  'MiFID II RTS 22 / FCA Transaction Reporting Timeliness Standards',
  ARRAY['EU', 'UK'],
  '{"max_reporting_delay_minutes":1440}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"report.minutes_since_execution"},"right":{"type":"PARAM","name":"max_reporting_delay_minutes"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('SETTLEMENT_FAIL_AGING',
  'CSDR Settlement Fail Aging & Buy-in Escalation', 'POST_TRADE', 'APPROVAL_REQUIRED', 85,
  'CSDR Art. 7 Settlement Discipline & Mandatory Buy-in Regime',
  ARRAY['EU'],
  '{"max_fail_days":3}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"settlement.fail_age_days"},"right":{"type":"PARAM","name":"max_fail_days"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('COUNTERPARTY_OTC_LIMIT',
  'Bilateral OTC Counterparty Credit Exposure Limit', 'PRE_TRADE', 'HARD_BLOCK', 95,
  'EMIR Art. 11 / House Bilateral OTC Counterparty Risk Mandate',
  ARRAY['EU', 'GLOBAL'],
  '{"counterparty_limit":"25000000.000000"}',
  '{"type":"COMPARISON","op":"GT","left":{"type":"METRIC","path":"otc.counterparty_exposure"},"right":{"type":"PARAM","name":"counterparty_limit"}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('EMIR_FIELD_VALIDITY',
  'EMIR ISO 20022 Schema Field Validation', 'POST_TRADE', 'HARD_BLOCK', 100,
  'EMIR Regulatory Technical Standards (RTS) ISO 20022 XML Validation',
  ARRAY['EU'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"report.emir_valid"},"right":{"type":"LITERAL","value":false}}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('FX_SETTLEMENT_CURRENCY_MATCH',
  'FX Settlement Base Currency Alignment Guardrail', 'PRE_TRADE', 'SOFT_WARNING', 60,
  'House Multi-Currency Settlement Risk Protocol',
  ARRAY['GLOBAL'],
  '{}',
  '{"type":"AND","children":[{"type":"COMPARISON","op":"NEQ","left":{"type":"METRIC","path":"order.currency"},"right":{"type":"METRIC","path":"account.base_currency"}},{"type":"COMPARISON","op":"NEQ","left":{"type":"METRIC","path":"order.currency"},"right":{"type":"METRIC","path":"security.currency"}}]}',
  'PROVISIONAL');

SELECT compliance.seed_core_rule('CROSS_BORDER_CLIENT_ELIGIBILITY',
  'Cross-Border Client Distribution Eligibility (KIID / PRIIPs)', 'PRE_TRADE', 'HARD_BLOCK', 95,
  'UCITS / AIFMD National Private Placement & Cross-Border Passporting Requirements',
  ARRAY['GLOBAL'],
  '{}',
  '{"type":"COMPARISON","op":"EQ","left":{"type":"METRIC","path":"account.distribution_eligible"},"right":{"type":"LITERAL","value":false}}',
  'PROVISIONAL');

-- ====================================================================
-- LICENSABLE RULESET BUNDLE MEMBERSHIP
-- ====================================================================
-- 1. Core Regulatory Pack
INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
SELECT 'CORE_REGULATORY', id FROM compliance.compliance_rule
WHERE source_version = 'CORE_LIB_V1'
  AND rule_code IN (
    'UCITS_ISSUER_5', 'UCITS_ISSUER_10_EXCEPTION', 'UCITS_ISSUER_40',
    'UCITS_DEPOSIT_20', 'ACT40_DIV_75_5', 'FOF_20',
    'SEC_144A_ELIGIBILITY', 'REG_S_OFFSHORE_ONLY', 'RESTRICTED_LIST_BLOCK',
    'INSIDER_LIST_MAR', 'SANCTIONS_ISSUER_BLOCK', 'COUNTRY_EXPOSURE_LIMIT',
    'LARGE_TRADE_THRESHOLD', 'TXN_REPORT_COMPLETENESS', 'TXN_REPORT_TIMELINESS',
    'SETTLEMENT_FAIL_AGING', 'CROSS_BORDER_CLIENT_ELIGIBILITY'
  )
ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

-- 2. Market Conduct Pack
INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
SELECT 'MARKET_CONDUCT', id FROM compliance.compliance_rule
WHERE source_version = 'CORE_LIB_V1'
  AND rule_code IN (
    'REG_M_RULE_105', 'WASH_SALE_1091', 'FRONT_RUNNING_CLIENT_ORDER',
    'FREERIDING_REG_T', 'LAYERING_SPOOF_PATTERN', 'EXECUTION_PRICE_DEVIATION',
    'SHORT_SALE_LOCATE', 'LULD_PRICE_BAND', 'PRO_RATA_ALLOCATION_FAIRNESS',
    'AGGREGATION_ELIGIBILITY', 'VENUE_APPROVED_LIST', 'EXECUTION_WITHIN_SPREAD'
  )
ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

-- 3. Institutional Controls Pack
INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
SELECT 'INSTITUTIONAL_CONTROLS', id FROM compliance.compliance_rule
WHERE source_version = 'CORE_LIB_V1'
  AND rule_code IN (
    'FAT_FINGER_NOTIONAL', 'FAT_FINGER_ADV_RATIO', 'PRICE_COLLAR_PCT',
    'DUPLICATE_ORDER_WINDOW', 'SELF_TRADE_PREVENT', 'ORDER_RATE_LIMIT',
    'ODD_LOT_ABOVE_MIN', 'SHORT_POSITION_RESTRICTION', 'LEVERAGE_VAR_COMMIT',
    'ILLIQUID_ASSET_LIMIT', 'LIQUIDITY_BUCKET_DAYS', 'MARGIN_HOUSE_LIMIT',
    'SECTOR_CONCENTRATION', 'SINGLE_POSITION_NAV', 'PT_PRECLEARANCE_REQUIRED',
    'PT_BLACKOUT_PERIOD', 'PT_MIN_HOLDING_30D', 'PT_ACCESS_PERSON_RECON',
    'COUNTERPARTY_OTC_LIMIT', 'EMIR_FIELD_VALIDITY', 'FX_SETTLEMENT_CURRENCY_MATCH'
  )
ON CONFLICT (ruleset_code, rule_id) DO NOTHING;
