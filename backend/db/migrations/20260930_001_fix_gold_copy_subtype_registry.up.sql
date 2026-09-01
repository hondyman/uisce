-- 20260930_001_fix_gold_copy_subtype_registry.up.sql
-- Fix oms.subtype_registry to use the actual gold_copy tenant.
-- The seed file hardcoded 00000000-0000-0000-0000-000000000001; this migration
-- re-upserts all 22 oms rows using the dynamically-resolved gold_copy tenant.
-- Also seeds the 16 missing altinv / cash_flow / master subtypes.

BEGIN;

-- ---------------------------------------------------------------------------
-- Resolve gold_copy tenant (idempotent)
-- ---------------------------------------------------------------------------
DO $$
DECLARE
    gct UUID;
BEGIN
    SELECT id INTO gct FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gct IS NULL THEN
        -- Fall back to the well-known system tenant used throughout this project
        gct := '00000000-0000-0000-0000-000000000001'::UUID;
    END IF;

    -- ---------------------------------------------------------------------------
    -- Phase A: Upsert the original 22 oms.* rows against the real gold-copy tenant
    -- ---------------------------------------------------------------------------
    INSERT INTO oms.subtype_registry
        (tenant_id, root_object, subtype_code, display_name, field_allowlist, is_active)
    VALUES
        -- Account Subtypes
        (gct, 'account', 'institutional',     'Institutional Account',
         '["sponsor_id", "mandate_type", "erisa_flag", "fee_schedule_code"]'::jsonb, true),
        (gct, 'account', 'retail_wealth',     'Retail / Wealth Account',
         '["tax_id_type", "citizenship", "margin_agreement_flag", "accredited_investor_status"]'::jsonb, true),
        (gct, 'account', 'sma',              'Separately Managed Account (SMA)',
         '["sponsor_firm", "model_strategy_id", "overlay_manager_id", "rebalance_frequency"]'::jsonb, true),
        (gct, 'account', 'trust_estate',      'Trust & Estate Account',
         '["trust_type", "grantor_name", "trustee_signatory_id", "dissolution_date"]'::jsonb, true),
        (gct, 'account', 'qualified_retirement', 'Qualified Retirement Plan',
         '["plan_type", "vesting_schedule_code", "rmd_eligible_flag", "custodian_bank_id"]'::jsonb, true),
        (gct, 'account', 'corporate_treasury','Corporate Treasury Account',
         '["corporate_entity_id", "treasury_signatory_group", "wire_limit_daily"]'::jsonb, true),

        -- Position Subtypes
        (gct, 'position', 'settled_long',      'Settled Long Position',
         '["custody_account_id", "settled_shares", "cost_basis_method"]'::jsonb, true),
        (gct, 'position', 'short_borrowed',   'Short / Borrowed Position',
         '["prime_broker_id", "borrow_rate_bps", "locate_id"]'::jsonb, true),
        (gct, 'position', 'derivative_exposure', 'Derivative Open Interest',
         '["underlying_security_id", "notional_amount", "unrealized_pnl"]'::jsonb, true),
        (gct, 'position', 'pledged_collateral','Pledged Collateral Position',
         '["pledged_to_party", "haircut_pct"]'::jsonb, true),
        (gct, 'position', 'unsettled_pipeline','Unsettled Pipeline Position',
         '["trade_date_shares", "pending_settlement_cash"]'::jsonb, true),

        -- Security Subtypes
        (gct, 'security', 'equity',           'Common & Preferred Equity',
         '["ticker", "isin", "voting_rights_type"]'::jsonb, true),
        (gct, 'security', 'sovereign_debt',  'Sovereign & Government Debt',
         '["coupon_rate", "maturity_date", "day_count_convention"]'::jsonb, true),
        (gct, 'security', 'corporate_debt',  'Corporate Debt',
         '["credit_rating_sp", "call_date", "seniority_level"]'::jsonb, true),
        (gct, 'security', 'structured_abs_mbs','Securitized ABS / MBS / CLO',
         '["pool_number", "factor_current", "tranche_tier"]'::jsonb, true),
        (gct, 'security', 'etd_derivative',   'Exchange-Traded Derivative',
         '["contract_size", "strike_price", "put_call_indicator"]'::jsonb, true),
        (gct, 'security', 'otc_derivative',   'Over-The-Counter Derivative',
         '["isda_agreement_id", "fixed_rate", "floating_index_name"]'::jsonb, true),

        -- Trade Order Subtypes
        (gct, 'trade_order', 'block_parent',  'Block / Parent Allocation',
         '["allocation_profile_id", "total_requested_quantity", "average_price"]'::jsonb, true),
        (gct, 'trade_order', 'dma_execution', 'Direct Market Access (DMA)',
         '["execution_algo_id", "venue_id", "liquidity_flag"]'::jsonb, true),
        (gct, 'trade_order', 'otc_bilateral', 'Bilateral OTC Trade',
         '["counterparty_dealer_id", "confirmation_status"]'::jsonb, true),
        (gct, 'trade_order', 'fx_spot_forward','FX Spot / Forward',
         '["base_currency", "quote_currency", "fx_rate", "value_date"]'::jsonb, true),
        (gct, 'trade_order', 'primary_auction','Primary Auction Syndication',
         '["syndicate_manager_id", "concession_amount", "allotment_shares"]'::jsonb, true)

    ON CONFLICT (tenant_id, root_object, subtype_code) DO UPDATE
        SET display_name    = EXCLUDED.display_name,
            field_allowlist = EXCLUDED.field_allowlist,
            is_active       = EXCLUDED.is_active;

    -- ---------------------------------------------------------------------------
    -- Phase B: Seed the 6 altinv.alternative_investment subtypes
    -- ---------------------------------------------------------------------------
    INSERT INTO oms.subtype_registry
        (tenant_id, root_object, subtype_code, display_name, field_allowlist, is_active)
    VALUES
        (gct, 'alternative_investment', 'private_equity',  'Private Equity Fund',
         '["vintage_year", "committed_capital", "called_capital", "unfunded_commitment", "dpi", "rvpi"]'::jsonb, true),
        (gct, 'alternative_investment', 'venture_capital', 'Venture Capital Fund',
         '["vintage_year", "round_series", "pro_rata_rights_flag", "lead_investor_name", "post_money_valuation"]'::jsonb, true),
        (gct, 'alternative_investment', 'hedge_fund',     'Hedge Fund',
         '["high_water_mark_nav", "lockup_period_months", "redemption_notice_days", "hurdle_rate_pct"]'::jsonb, true),
        (gct, 'alternative_investment', 'real_estate',    'Real Estate Fund',
         '["property_type", "occupancy_rate_pct", "gross_asset_value", "loan_to_value_pct"]'::jsonb, true),
        (gct, 'alternative_investment', 'direct_investment','Direct Investment',
         '["committed_capital", "called_capital", "unfunded_commitment"]'::jsonb, true),
        (gct, 'alternative_investment', 'infrastructure', 'Infrastructure Fund',
         '["project_phase", "concession_expiry_year", "esg_carbon_offset_tons", "loan_to_value_pct"]'::jsonb, true)

    ON CONFLICT (tenant_id, root_object, subtype_code) DO UPDATE
        SET display_name    = EXCLUDED.display_name,
            field_allowlist = EXCLUDED.field_allowlist,
            is_active       = EXCLUDED.is_active;

    -- ---------------------------------------------------------------------------
    -- Phase C: Seed the 6 cash_flow.settlement subtypes
    -- ---------------------------------------------------------------------------
    INSERT INTO oms.subtype_registry
        (tenant_id, root_object, subtype_code, display_name, field_allowlist, is_active)
    VALUES
        (gct, 'settlement', 'dividend',            'Dividend Settlement',
         '["ex_date", "record_date", "drip_reinvest_flag", "tax_withholding_amount"]'::jsonb, true),
        (gct, 'settlement', 'coupon_fixed_income',  'Fixed Income Coupon',
         '["coupon_period_start", "accrued_interest", "payment_frequency", "call_notice_id"]'::jsonb, true),
        (gct, 'settlement', 'capital_call',         'Capital Call Notice',
         '["due_date", "management_fee_portion", "investment_portion", "mandatory_flag"]'::jsonb, true),
        (gct, 'settlement', 'lp_distribution',     'LP Distribution',
         '["due_date", "return_of_capital", "preferred_return", "carried_interest_retained"]'::jsonb, true),
        (gct, 'settlement', 'corporate_action',     'Corporate Action Settlement',
         '["action_type_code", "cash_in_lieu_amount", "mandatory_flag"]'::jsonb, true),
        (gct, 'settlement', 'expense_fee',           'Expense / Fee Settlement',
         '["fee_category", "invoice_reference_id", "vat_amount"]'::jsonb, true)

    ON CONFLICT (tenant_id, root_object, subtype_code) DO UPDATE
        SET display_name    = EXCLUDED.display_name,
            field_allowlist = EXCLUDED.field_allowlist,
            is_active       = EXCLUDED.is_active;

    -- ---------------------------------------------------------------------------
    -- Phase D: Seed the 4 master.* subtypes (customer + sales_ledger)
    -- ---------------------------------------------------------------------------
    INSERT INTO oms.subtype_registry
        (tenant_id, root_object, subtype_code, display_name, field_allowlist, is_active)
    VALUES
        -- master.customer subtypes
        (gct, 'customer', 'institutional_client', 'Institutional Client',
         '["lei_code", "kyc_status", "suitability_profile", "relationship_tier", "parent_group_id"]'::jsonb, true),
        (gct, 'customer', 'private_wealth',        'Private Wealth Client',
         '["lei_code", "kyc_status", "suitability_profile", "relationship_tier", "parent_group_id"]'::jsonb, true),
        (gct, 'customer', 'broker_dealer',         'Broker Dealer',
         '["lei_code", "kyc_status"]'::jsonb, true),
        (gct, 'customer', 'corporate_treasury',    'Corporate Treasury',
         '["lei_code", "kyc_status", "suitability_profile"]'::jsonb, true),

        -- master.sales_ledger subtypes
        (gct, 'sales_ledger', 'aum_management_fee',  'AUM Management Fee',
         '["aum_basis_amount", "effective_fee_bps", "hwm_benchmark_nav", "billing_period_end"]'::jsonb, true),
        (gct, 'sales_ledger', 'trading_commission',   'Trading Commission',
         '["billing_period_end", "invoice_status"]'::jsonb, true),
        (gct, 'sales_ledger', 'performance_fee',       'Performance Fee',
         '["hwm_benchmark_nav", "effective_fee_bps", "billing_period_end"]'::jsonb, true),
        (gct, 'sales_ledger', 'platform_subscription', 'Platform Subscription',
         '["billing_period_end", "invoice_status"]'::jsonb, true)

    ON CONFLICT (tenant_id, root_object, subtype_code) DO UPDATE
        SET display_name    = EXCLUDED.display_name,
            field_allowlist = EXCLUDED.field_allowlist,
            is_active       = EXCLUDED.is_active;

END $$;

COMMIT;
