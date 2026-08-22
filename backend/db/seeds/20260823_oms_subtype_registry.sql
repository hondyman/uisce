-- Seed initial root object subtype registrations
INSERT INTO oms.subtype_registry (tenant_id, root_object, subtype_code, display_name, field_allowlist)
VALUES
-- Account Subtypes
('00000000-0000-0000-0000-000000000001', 'account', 'institutional', 'Institutional Account', '["sponsor_id", "mandate_type", "erisa_flag", "fee_schedule_code"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'account', 'retail_wealth', 'Retail / Wealth Account', '["tax_id_type", "citizenship", "margin_agreement_flag", "accredited_investor_status"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'account', 'sma', 'Separately Managed Account (SMA)', '["sponsor_firm", "model_strategy_id", "overlay_manager_id", "rebalance_frequency"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'account', 'trust_estate', 'Trust & Estate Account', '["trust_type", "grantor_name", "trustee_signatory_id", "dissolution_date"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'account', 'qualified_retirement', 'Qualified Retirement Plan', '["plan_type", "vesting_schedule_code", "rmd_eligible_flag", "custodian_bank_id"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'account', 'corporate_treasury', 'Corporate Treasury Account', '["corporate_entity_id", "treasury_signatory_group", "wire_limit_daily"]'::jsonb),

-- Position Subtypes
('00000000-0000-0000-0000-000000000001', 'position', 'settled_long', 'Settled Long Position', '["custody_account_id", "settled_shares", "cost_basis_method"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'position', 'short_borrowed', 'Short / Borrowed Position', '["prime_broker_id", "borrow_rate_bps", "locate_id"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'position', 'derivative_exposure', 'Derivative Open Interest', '["underlying_security_id", "notional_amount", "unrealized_pnl"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'position', 'pledged_collateral', 'Pledged Collateral Position', '["pledged_to_party", "haircut_pct"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'position', 'unsettled_pipeline', 'Unsettled Pipeline Position', '["trade_date_shares", "pending_settlement_cash"]'::jsonb),

-- Security Subtypes
('00000000-0000-0000-0000-000000000001', 'security', 'equity', 'Common & Preferred Equity', '["ticker", "isin", "voting_rights_type"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'security', 'sovereign_debt', 'Sovereign & Government Debt', '["coupon_rate", "maturity_date", "day_count_convention"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'security', 'corporate_debt', 'Corporate Debt', '["credit_rating_sp", "call_date", "seniority_level"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'security', 'structured_abs_mbs', 'Securitized ABS / MBS / CLO', '["pool_number", "factor_current", "tranche_tier"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'security', 'etd_derivative', 'Exchange-Traded Derivative', '["contract_size", "strike_price", "put_call_indicator"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'security', 'otc_derivative', 'Over-The-Counter Derivative', '["isda_agreement_id", "fixed_rate", "floating_index_name"]'::jsonb),

-- Trade Subtypes
('00000000-0000-0000-0000-000000000001', 'trade_order', 'block_parent', 'Block / Parent Allocation', '["allocation_profile_id", "total_requested_quantity", "average_price"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'trade_order', 'dma_execution', 'Direct Market Access (DMA)', '["execution_algo_id", "venue_id", "liquidity_flag"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'trade_order', 'otc_bilateral', 'Bilateral OTC Trade', '["counterparty_dealer_id", "confirmation_status"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'trade_order', 'fx_spot_forward', 'FX Spot / Forward', '["base_currency", "quote_currency", "fx_rate", "value_date"]'::jsonb),
('00000000-0000-0000-0000-000000000001', 'trade_order', 'primary_auction', 'Primary Auction Syndication', '["syndicate_manager_id", "concession_amount", "allotment_shares"]'::jsonb)
ON CONFLICT (tenant_id, root_object, subtype_code) DO UPDATE
SET display_name = EXCLUDED.display_name,
    field_allowlist = EXCLUDED.field_allowlist;
