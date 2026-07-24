-- ============================================================================
-- FINANCIAL BUSINESS PROCESS TEMPLATES
-- Workday-style workflows for wealth management operations
-- ============================================================================

-- ============================================================================
-- 1. CLIENT ONBOARDING PROCESS
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
    EXECUTE $exec$
      INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, status, is_system, created_at)
      VALUES (gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), 'client_onboarding', 'Client Onboarding', 'Client Onboarding', 'End-to-end client onboarding with KYC/AML, document collection, and account opening', 'client_management', 'active', true, now())
      ON CONFLICT DO NOTHING;
    $exec$;

    EXECUTE $exec$
      INSERT INTO process_steps (id, tenant_id, process_id, key, name, display_name, step_type, sequence, config, is_required, created_at, created_by, last_modified_at)
      SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_processes WHERE key = 'client_onboarding'), key, label, label, step_type, seq, config::jsonb, required, now(), NULL, now()
      FROM (VALUES
        ('initiate', 'Initiate Onboarding', 'initiate', 1, '{"eventId": "client_app_submitted", "collectData": ["client.first_name", "client.last_name", "client.email", "client.phone", "client.date_of_birth"]}', true),
        ('validate_basic', 'Validate Basic Info', 'validate', 2, '{"rules": ["email_format", "phone_format", "age_check"], "onFailure": "reject"}', true),
        ('collect_kyc', 'Collect KYC Documents', 'data_entry', 3, '{"documents": ["government_id", "proof_of_address", "tax_id"], "timeout_days": 14}', true),
        ('aml_screen', 'AML Screening', 'aml', 4, '{"provider": "lexisnexis", "checkTypes": ["sanctions", "pep", "adverse_media"], "timeout": 30}', true),
        ('suitability_review', 'Suitability Assessment', 'data_entry', 5, '{"questionnaire": "suitability_form", "collectData": ["risk_tolerance", "investment_horizon", "liquidity_needs"]}', true),
        ('advisor_approval', 'Advisor Approval', 'approve', 6, '{"role": "Advisor", "escalationDays": 3, "escalateTo": "Compliance"}', true),
        ('compliance_sign_off', 'Compliance Sign-off', 'approve', 7, '{"role": "Compliance Officer", "escalationDays": 2}', true),
        ('generate_agreements', 'Generate Agreements', 'generate', 8, '{"templates": ["investment_advisory_agreement", "privacy_notice", "fee_disclosure"]}', true),
        ('client_signature', 'Client Signature', 'signature', 9, '{"method": "docusign", "documents": ["investment_advisory_agreement"]}', true),
        ('open_accounts', 'Open Accounts', 'integration', 10, '{"target": "custodian", "action": "create_account"}', true),
        ('complete', 'Complete Onboarding', 'complete', 11, '{"notifyClient": true, "notifyAdvisor": true, "createTasks": ["welcome_call", "initial_review"]}', true)
      ) AS t(key, label, step_type, seq, config, required)
      ON CONFLICT DO NOTHING;
    $exec$;
  ELSE
    RAISE NOTICE 'business_processes table not present, skipping client onboarding process';
  END IF;
END$$;

-- ============================================================================
-- 2. PORTFOLIO REBALANCING PROCESS
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
    EXECUTE $exec$
      INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, status, is_system, created_at)
      VALUES (gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), 'portfolio_rebalance', 'Portfolio Rebalancing', 'Portfolio Rebalancing', 'Systematic portfolio rebalancing with drift analysis and trade generation', 'portfolio_management', 'active', true, now())
      ON CONFLICT DO NOTHING;
    $exec$;

    EXECUTE $exec$
      INSERT INTO process_steps (id, tenant_id, process_id, key, name, display_name, step_type, sequence, config, is_required, created_at, created_by, last_modified_at)
      SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_processes WHERE key = 'portfolio_rebalance'), key, label, label, step_type, seq, config::jsonb, required, now(), NULL, now()
      FROM (VALUES
        ('analyze_drift', 'Analyze Drift', 'calculation', 1, '{"thresholds": {"asset_class": 5.0, "sector": 3.0, "security": 2.0}, "frequency": "daily"}', true),
        ('generate_proposals', 'Generate Rebalance Proposals', 'calculation', 2, '{"method": "minimize_trades", "constraints": ["tax_efficiency", "wash_sale_avoid", "minimum_trade_size"]}', true),
        ('tax_lot_selection', 'Tax Lot Selection', 'calculation', 3, '{"method": "tax_efficient", "harvestLosses": true, "avoidShortTerm": true}', false),
        ('advisor_review', 'Advisor Review', 'approve', 4, '{"role": "Advisor", "showProjectedImpact": true}', true),
        ('compliance_check', 'Compliance Check', 'validate', 5, '{"rules": ["concentration_limits", "restricted_securities", "cash_reserve"]}', true),
        ('generate_trades', 'Generate Trade Orders', 'generate', 6, '{"format": "fix", "groupBySecurity": true}', true),
        ('trader_approval', 'Trader Approval', 'approve', 7, '{"role": "Trader", "showMarketImpact": true}', false),
        ('execute_trades', 'Execute Trades', 'integration', 8, '{"target": "oms", "action": "submit_orders"}', true),
        ('reconcile', 'Reconcile Execution', 'validate', 9, '{"source": "custodian", "tolerance": {"price": 0.01, "quantity": 0}}', true),
        ('update_positions', 'Update Positions', 'update', 10, '{"updateCostBasis": true, "bookTrades": true}', true),
        ('notify_clients', 'Notify Clients', 'notify', 11, '{"template": "rebalance_complete", "channel": "email"}', false)
      ) AS t(key, label, step_type, seq, config, required)
      ON CONFLICT DO NOTHING;
    $exec$;
  ELSE
    RAISE NOTICE 'business_processes table not present, skipping portfolio rebalance process';
  END IF;
END$$;

-- ============================================================================
-- 3. PERFORMANCE REPORTING PROCESS
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
    EXECUTE $exec$
      INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, status, is_system, created_at)
      VALUES (gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), 'performance_report', 'Performance Reporting', 'Performance Reporting', 'Generate and distribute client performance reports', 'reporting', 'active', true, now())
      ON CONFLICT DO NOTHING;
    $exec$;

    EXECUTE $exec$
      INSERT INTO process_steps (id, tenant_id, process_id, key, name, display_name, step_type, sequence, config, is_required, created_at, created_by, last_modified_at)
      SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_processes WHERE key = 'performance_report'), key, label, label, step_type, seq, config::jsonb, required, now(), NULL, now()
      FROM (VALUES
        ('reconcile_data', 'Reconcile Data', 'validate', 1, '{"sources": ["custodian", "trading", "pricing"], "tolerance": 0.01}', true),
        ('calculate_performance', 'Calculate Performance', 'calculation', 2, '{"methods": ["twr", "mwr"], "periods": ["mtd", "qtd", "ytd", "1y", "3y", "5y", "itd"]}', true),
        ('calculate_attribution', 'Calculate Attribution', 'calculation', 3, '{"method": "brinson", "dimensions": ["asset_class", "sector", "security"]}', false),
        ('generate_reports', 'Generate Reports', 'generate', 4, '{"templates": ["quarterly_summary", "detailed_breakdown", "tax_summary"]}', true),
        ('qa_review', 'QA Review', 'approve', 5, '{"role": "Operations", "checkRules": ["data_completeness", "outlier_detection"]}', true),
        ('advisor_preview', 'Advisor Preview', 'approve', 6, '{"role": "Advisor", "allowEdits": false, "requireComment": false}', false),
        ('client_delivery', 'Client Delivery', 'notify', 7, '{"channels": ["portal", "email"], "template": "quarterly_report_ready"}', true),
        ('archive', 'Archive Reports', 'integration', 8, '{"target": "document_management", "retention": "7_years"}', true)
      ) AS t(key, label, step_type, seq, config, required)
      ON CONFLICT DO NOTHING;
    $exec$;
  ELSE
    RAISE NOTICE 'business_processes table not present, skipping performance reporting process';
  END IF;
END$$;

-- ============================================================================
-- 4. FEE BILLING PROCESS
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
    EXECUTE $exec$
      INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, status, is_system, created_at)
      VALUES (gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), 'fee_billing', 'Fee Billing', 'Fee Billing', 'Calculate, approve, and process advisory fees', 'billing', 'active', true, now())
      ON CONFLICT DO NOTHING;
    $exec$;

    EXECUTE $exec$
      INSERT INTO process_steps (id, tenant_id, process_id, key, name, display_name, step_type, sequence, config, is_required, created_at, created_by, last_modified_at)
      SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_processes WHERE key = 'fee_billing'), key, label, label, step_type, seq, config::jsonb, required, now(), NULL, now()
      FROM (VALUES
        ('calculate_billable', 'Calculate Billable AUM', 'calculation', 1, '{"method": "average_daily_balance", "excludeAccruals": true}', true),
        ('apply_fee_schedules', 'Apply Fee Schedules', 'calculation', 2, '{"supportTiered": true, "supportBreakpoints": true}', true),
        ('apply_adjustments', 'Apply Adjustments', 'data_entry', 3, '{"types": ["waiver", "discount", "minimum_fee", "performance_fee"]}', false),
        ('generate_invoices', 'Generate Invoices', 'generate', 4, '{"template": "fee_invoice", "includeBreakdown": true}', true),
        ('finance_review', 'Finance Review', 'approve', 5, '{"role": "Finance", "threshold": 10000}', true),
        ('advisor_notification', 'Advisor Notification', 'notify', 6, '{"channel": "email", "template": "fee_billing_preview"}', false),
        ('client_notification', 'Client Notification', 'notify', 7, '{"channel": "portal", "template": "fee_notice"}', true),
        ('collect_fees', 'Collect Fees', 'integration', 8, '{"method": "custody_debit", "target": "custodian"}', true),
        ('book_revenue', 'Book Revenue', 'integration', 9, '{"target": "accounting", "gl_accounts": {"revenue": "4000", "receivable": "1200"}}', true)
      ) AS t(key, label, step_type, seq, config, required)
      ON CONFLICT DO NOTHING;
    $exec$;
  ELSE
    RAISE NOTICE 'business_processes table not present, skipping fee billing process';
  END IF;
END$$;

-- ============================================================================
-- 5. TRADE EXECUTION PROCESS
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
    EXECUTE $exec$
      INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, status, is_system, created_at)
      VALUES (gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), 'trade_execution', 'Trade Execution', 'Trade Execution', 'End-to-end trade lifecycle from order to settlement', 'trading', 'active', true, now())
      ON CONFLICT DO NOTHING;
    $exec$;

    EXECUTE $exec$
      INSERT INTO process_steps (id, tenant_id, process_id, key, name, display_name, step_type, sequence, config, is_required, created_at, created_by, last_modified_at)
      SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_processes WHERE key = 'trade_execution'), key, label, label, step_type, seq, config::jsonb, required, now(), NULL, now()
      FROM (VALUES
        ('validate_order', 'Validate Order', 'validate', 1, '{"rules": ["sufficient_cash", "position_check", "restricted_list", "concentration_limit"]}', true),
        ('pre_trade_compliance', 'Pre-Trade Compliance', 'validate', 2, '{"rules": ["investment_guidelines", "client_restrictions", "regulatory_limits"]}', true),
        ('route_order', 'Route Order', 'integration', 3, '{"target": "oms", "algorithms": ["vwap", "twap", "is", "arrival_price"]}', true),
        ('execute', 'Execute Trade', 'integration', 4, '{"target": "broker", "confirmRequired": true}', true),
        ('post_trade_compliance', 'Post-Trade Compliance', 'validate', 5, '{"rules": ["best_execution", "allocation_fairness"]}', true),
        ('allocate', 'Allocate Fills', 'calculation', 6, '{"method": "pro_rata", "roundLots": true}', true),
        ('confirm', 'Trade Confirmation', 'notify', 7, '{"channel": "email", "template": "trade_confirmation"}', true),
        ('settle', 'Settlement', 'integration', 8, '{"target": "custodian", "t_plus": 2}', true),
        ('reconcile', 'Reconcile', 'validate', 9, '{"sources": ["oms", "custodian"], "autoCorrect": false}', true)
      ) AS t(key, label, step_type, seq, config, required)
      ON CONFLICT DO NOTHING;
    $exec$;
  ELSE
    RAISE NOTICE 'business_processes table not present, skipping trade execution process';
  END IF;
END$$;

-- ============================================================================
-- 6. ACCOUNT TRANSFER PROCESS
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
    EXECUTE $exec$
      INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, status, is_system, created_at)
      VALUES (gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), 'account_transfer', 'Account Transfer', 'Account Transfer (ACAT)', 'Transfer accounts between custodians via ACAT', 'account_management', 'active', true, now())
      ON CONFLICT DO NOTHING;
    $exec$;

    EXECUTE $exec$
      INSERT INTO process_steps (id, tenant_id, process_id, key, name, display_name, step_type, sequence, config, is_required, created_at, created_by, last_modified_at)
      SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_processes WHERE key = 'account_transfer'), key, label, label, step_type, seq, config::jsonb, required, now(), NULL, now()
      FROM (VALUES
        ('initiate_transfer', 'Initiate Transfer Request', 'initiate', 1, '{"collectData": ["source_account", "destination_account", "transfer_type"]}', true),
        ('collect_statement', 'Collect Recent Statement', 'data_entry', 2, '{"document": "account_statement", "maxAge": 30}', true),
        ('review_positions', 'Review Positions', 'validate', 3, '{"checkNonTransferable": true, "flagOptions": true}', true),
        ('client_authorization', 'Client Authorization', 'signature', 4, '{"method": "docusign", "document": "transfer_authorization"}', true),
        ('submit_acat', 'Submit ACAT', 'integration', 5, '{"target": "nscc", "method": "acat"}', true),
        ('track_progress', 'Track Transfer Progress', 'monitor', 6, '{"checkInterval": "daily", "maxDays": 15}', true),
        ('receive_assets', 'Receive Assets', 'validate', 7, '{"reconcilePositions": true, "validateCostBasis": true}', true),
        ('book_transfer', 'Book Transfer', 'update', 8, '{"createPositions": true, "preserveCostBasis": true}', true),
        ('notify_complete', 'Notify Completion', 'notify', 9, '{"channel": "email", "template": "transfer_complete"}', true)
      ) AS t(key, label, step_type, seq, config, required)
      ON CONFLICT DO NOTHING;
    $exec$;
  ELSE
    RAISE NOTICE 'business_processes table not present, skipping account transfer process';
  END IF;
END$$;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Financial Business Processes created successfully!';
    RAISE NOTICE '✓ Processes: Client Onboarding, Portfolio Rebalancing, Performance Reporting, Fee Billing, Trade Execution, Account Transfer';
END $$;
