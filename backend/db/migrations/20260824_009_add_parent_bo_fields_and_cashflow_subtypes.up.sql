-- 20260824_009_add_parent_bo_fields_and_cashflow_subtypes.up.sql
-- Adds bo_fields for parent BOs (oms.account, oms.position, cash_flow.settlement)
-- so that rep-core-005 and rep-shared-002 can render.

DO $$
DECLARE
    gold_copy_tenant_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';

    account_bo_id UUID;
    position_bo_id UUID;
    cash_settlement_bo_id UUID;
    dividend_bo_id UUID;
    lp_dist_bo_id UUID;
    cap_call_bo_id UUID;
    corp_treasury_bo_id UUID;
BEGIN

    -- ============================================================================
    -- oms.account parent BO fields (for rep-core-005)
    -- ============================================================================
    SELECT id INTO account_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'oms.account';
    DELETE FROM bo_fields WHERE business_object_id = account_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'account_number', 'account_number', 'account_number', 'Account Number', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'account_name', 'account_name', 'account_name', 'Account Name', 'string', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'base_currency', 'base_currency', 'base_currency', 'Base Currency', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'status', 'status', 'status', 'Status', 'string', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'sponsor_id', 'sponsor_id', 'sponsor_id', 'Sponsor ID', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'mandate_type', 'mandate_type', 'mandate_type', 'Mandate Type', 'string', false, false, false, 7, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'erisa_flag', 'erisa_flag', 'erisa_flag', 'Erisa Flag', 'boolean', false, false, false, 8, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'fee_schedule_code', 'fee_schedule_code', 'fee_schedule_code', 'Fee Schedule Code', 'string', false, false, false, 9, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'tax_id_type', 'tax_id_type', 'tax_id_type', 'Tax Id Type', 'string', false, false, false, 10, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'citizenship', 'citizenship', 'citizenship', 'Citizenship', 'string', false, false, false, 11, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'created_at', 'created_at', 'created_at', 'Created At', 'date', false, false, false, 12, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'updated_at', 'updated_at', 'updated_at', 'Updated At', 'date', false, false, false, 13, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'valid_from', 'valid_from', 'valid_from', 'Valid From', 'date', false, false, false, 14, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, account_bo_id, 'valid_to', 'valid_to', 'valid_to', 'Valid To', 'date', false, false, false, 15, NOW(), NOW());

    -- ============================================================================
    -- oms.position parent BO fields (for rep-core-005)
    -- ============================================================================
    SELECT id INTO position_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'oms.position';
    DELETE FROM bo_fields WHERE business_object_id = position_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'account_id', 'account_id', 'account_id', 'Account ID', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'security_id', 'security_id', 'security_id', 'Security ID', 'string', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'quantity', 'quantity', 'quantity', 'Quantity', 'decimal', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'market_value', 'market_value', 'market_value', 'Market Value', 'decimal', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'currency', 'currency', 'currency', 'Currency', 'string', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'custody_account_id', 'custody_account_id', 'custody_account_id', 'Custody Account ID', 'string', false, false, false, 7, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'settled_shares', 'settled_shares', 'settled_shares', 'Settled Shares', 'decimal', false, false, false, 8, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'cost_basis_method', 'cost_basis_method', 'cost_basis_method', 'Cost Basis Method', 'string', false, false, false, 9, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'notional_amount', 'notional_amount', 'notional_amount', 'Notional Amount', 'decimal', false, false, false, 10, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'unrealized_pnl', 'unrealized_pnl', 'unrealized_pnl', 'Unrealized PnL', 'decimal', false, false, false, 11, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'created_at', 'created_at', 'created_at', 'Created At', 'date', false, false, false, 12, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'updated_at', 'updated_at', 'updated_at', 'Updated At', 'date', false, false, false, 13, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'valid_from', 'valid_from', 'valid_from', 'Valid From', 'date', false, false, false, 14, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, position_bo_id, 'valid_to', 'valid_to', 'valid_to', 'Valid To', 'date', false, false, false, 15, NOW(), NOW());

    -- ============================================================================
    -- cash_flow.settlement parent BO fields (for rep-shared-002)
    -- ============================================================================
    SELECT id INTO cash_settlement_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'cash_flow.settlement';
    DELETE FROM bo_fields WHERE business_object_id = cash_settlement_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'account_id', 'account_id', 'account_id', 'Account ID', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'amount', 'amount', 'amount', 'Amount', 'decimal', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'currency', 'currency', 'currency', 'Currency', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'settlement_date', 'settlement_date', 'settlement_date', 'Settlement Date', 'date', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'due_date', 'due_date', 'due_date', 'Due Date', 'date', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'settlement_status', 'settlement_status', 'settlement_status', 'Settlement Status', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 7, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'return_of_capital', 'return_of_capital', 'return_of_capital', 'Return Of Capital', 'decimal', false, false, false, 8, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'carried_interest_retained', 'carried_interest_retained', 'carried_interest_retained', 'Carried Interest Retained', 'decimal', false, false, false, 9, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'management_fee_portion', 'management_fee_portion', 'management_fee_portion', 'Management Fee Portion', 'decimal', false, false, false, 10, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'valid_from', 'valid_from', 'valid_from', 'Valid From', 'date', false, false, false, 11, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cash_settlement_bo_id, 'valid_to', 'valid_to', 'valid_to', 'Valid To', 'date', false, false, false, 12, NOW(), NOW());

    -- ============================================================================
    -- cash_flow.settlement child BO fields (for rep-shared-002)
    -- ============================================================================

    SELECT id INTO dividend_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'cash_flow.settlement/dividend';
    DELETE FROM bo_fields WHERE business_object_id = dividend_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, dividend_bo_id, 'account_id', 'account_id', 'account_id', 'Account ID', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, dividend_bo_id, 'amount', 'amount', 'amount', 'Amount', 'decimal', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, dividend_bo_id, 'currency', 'currency', 'currency', 'Currency', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, dividend_bo_id, 'settlement_date', 'settlement_date', 'settlement_date', 'Settlement Date', 'date', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, dividend_bo_id, 'due_date', 'due_date', 'due_date', 'Due Date', 'date', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, dividend_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, dividend_bo_id, 'settlement_status', 'settlement_status', 'settlement_status', 'Settlement Status', 'string', false, false, false, 7, NOW(), NOW());

    SELECT id INTO lp_dist_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'cash_flow.settlement/lp_distribution';
    DELETE FROM bo_fields WHERE business_object_id = lp_dist_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'account_id', 'account_id', 'account_id', 'Account ID', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'amount', 'amount', 'amount', 'Amount', 'decimal', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'currency', 'currency', 'currency', 'Currency', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'settlement_date', 'settlement_date', 'settlement_date', 'Settlement Date', 'date', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'due_date', 'due_date', 'due_date', 'Due Date', 'date', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'settlement_status', 'settlement_status', 'settlement_status', 'Settlement Status', 'string', false, false, false, 7, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'return_of_capital', 'return_of_capital', 'return_of_capital', 'Return Of Capital', 'decimal', false, false, false, 8, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, lp_dist_bo_id, 'carried_interest_retained', 'carried_interest_retained', 'carried_interest_retained', 'Carried Interest Retained', 'decimal', false, false, false, 9, NOW(), NOW());

    SELECT id INTO cap_call_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'cash_flow.settlement/capital_call';
    DELETE FROM bo_fields WHERE business_object_id = cap_call_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'account_id', 'account_id', 'account_id', 'Account ID', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'amount', 'amount', 'amount', 'Amount', 'decimal', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'currency', 'currency', 'currency', 'Currency', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'settlement_date', 'settlement_date', 'settlement_date', 'Settlement Date', 'date', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'due_date', 'due_date', 'due_date', 'Due Date', 'date', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'settlement_status', 'settlement_status', 'settlement_status', 'Settlement Status', 'string', false, false, false, 7, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, cap_call_bo_id, 'management_fee_portion', 'management_fee_portion', 'management_fee_portion', 'Management Fee Portion', 'decimal', false, false, false, 8, NOW(), NOW());

    SELECT id INTO corp_treasury_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'oms.account/corporate_treasury';
    DELETE FROM bo_fields WHERE business_object_id = corp_treasury_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, corp_treasury_bo_id, 'base_currency', 'base_currency', 'base_currency', 'Base Currency', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, corp_treasury_bo_id, 'wire_limit_daily', 'wire_limit_daily', 'wire_limit_daily', 'Wire Limit Daily', 'decimal', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, corp_treasury_bo_id, 'corporate_entity_id', 'corporate_entity_id', 'corporate_entity_id', 'Corporate Entity ID', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, corp_treasury_bo_id, 'account_number', 'account_number', 'account_number', 'Account Number', 'string', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, corp_treasury_bo_id, 'account_name', 'account_name', 'account_name', 'Account Name', 'string', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, corp_treasury_bo_id, 'fee_schedule_code', 'fee_schedule_code', 'fee_schedule_code', 'Fee Schedule Code', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, corp_treasury_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 7, NOW(), NOW());

END $$;
