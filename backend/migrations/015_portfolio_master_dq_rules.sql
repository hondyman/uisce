-- backend/migrations/015_portfolio_master_dq_rules.sql
-- Portfolio Master — Data Quality + Survivorship Rule Definitions
-- Seeds edm.survivorship_rules for gold copy engine consumption.
-- DQ rules are stored as catalog_node validation_rule entries.

DO $$
DECLARE
    v_gold_tenant        UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_system_user        UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    validation_rule_type_id UUID;
BEGIN

    -- ================================================================
    -- 1. SURVIVORSHIP RULES — edm.survivorship_rules
    --    Gold copy engine reads these at runtime.
    -- ================================================================

    -- Source priority order:
    -- AccountingSystem > OMS > Custodian > FundAdmin > Manual

    -- ---- portfolio_master field strategies ----
    INSERT INTO edm.survivorship_rules
        (id, tenant_id, entity_type, field_name, strategy, preferred_sources, priority, created_by)
    VALUES
        -- Portfolio Name: prefer AccountingSystem, fallback OMS
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'portfolio_name',
         'prefer_source', ARRAY['AccountingSystem','OMS','Custodian'], 1, v_system_user),

        -- Portfolio Code: prefer OMS (authoritative identifier)
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'portfolio_code',
         'prefer_source', ARRAY['OMS','AccountingSystem','Custodian'], 1, v_system_user),

        -- Base Currency: prefer Custodian (custodian holds cash)
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'base_currency',
         'prefer_source', ARRAY['Custodian','AccountingSystem','FundAdmin'], 1, v_system_user),

        -- Inception Date: always use earliest non-null across sources
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'inception_date',
         'earliest_non_null', ARRAY[]::TEXT[], 1, v_system_user),

        -- Benchmark: prefer OMS (mandate-driven)
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'benchmark_id',
         'prefer_source', ARRAY['OMS','AccountingSystem','Manual'], 1, v_system_user),

        -- Strategy: prefer OMS
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'strategy_id',
         'prefer_source', ARRAY['OMS','Manual'], 1, v_system_user),

        -- Risk Profile: prefer ClientOnboarding, fallback CRM
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'risk_profile',
         'prefer_source', ARRAY['ClientOnboarding','CRM','OMS'], 1, v_system_user),

        -- Investment Objective: prefer ClientOnboarding
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'investment_objective',
         'prefer_source', ARRAY['ClientOnboarding','CRM','Manual'], 1, v_system_user),

        -- Portfolio Type: prefer AccountingSystem
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'portfolio_type',
         'prefer_source', ARRAY['AccountingSystem','OMS'], 1, v_system_user),

        -- Status (termination_date): use latest_by effective date
        (gen_random_uuid(), v_gold_tenant, 'portfolio_master', 'termination_date',
         'latest_by', ARRAY[]::TEXT[], 1, v_system_user)

    ON CONFLICT (tenant_id, entity_type, field_name, priority) DO NOTHING;

    -- ---- mandate_master field strategies ----
    INSERT INTO edm.survivorship_rules
        (id, tenant_id, entity_type, field_name, strategy, preferred_sources, priority, created_by)
    VALUES
        (gen_random_uuid(), v_gold_tenant, 'mandate_master', 'investment_objective',
         'prefer_source', ARRAY['ClientOnboarding','CRM'], 1, v_system_user),

        (gen_random_uuid(), v_gold_tenant, 'mandate_master', 'risk_tolerance',
         'prefer_source', ARRAY['ClientOnboarding'], 1, v_system_user),

        (gen_random_uuid(), v_gold_tenant, 'mandate_master', 'esg_constraints',
         'prefer_source', ARRAY['ClientOnboarding','Compliance'], 1, v_system_user),

        (gen_random_uuid(), v_gold_tenant, 'mandate_master', 'tax_constraints',
         'prefer_source', ARRAY['ClientOnboarding','TaxAdvisor'], 1, v_system_user)

    ON CONFLICT (tenant_id, entity_type, field_name, priority) DO NOTHING;

    -- ---- benchmark_master field strategies ----
    INSERT INTO edm.survivorship_rules
        (id, tenant_id, entity_type, field_name, strategy, preferred_sources, priority, created_by)
    VALUES
        (gen_random_uuid(), v_gold_tenant, 'benchmark_master', 'benchmark_name',
         'prefer_source', ARRAY['MarketDataVendor','Bloomberg','MSCI'], 1, v_system_user),

        (gen_random_uuid(), v_gold_tenant, 'benchmark_master', 'currency',
         'prefer_source', ARRAY['MarketDataVendor','Bloomberg'], 1, v_system_user)

    ON CONFLICT (tenant_id, entity_type, field_name, priority) DO NOTHING;

    -- ================================================================
    -- 2. DQ VALIDATION RULES — catalog_node (type = validation_rule)
    --    Stored as JSON expressions that the rules engine interprets.
    -- ================================================================
    SELECT id INTO validation_rule_type_id
      FROM catalog_node_type WHERE catalog_type_name = 'validation_rule' LIMIT 1;

    IF validation_rule_type_id IS NOT NULL THEN
        INSERT INTO catalog_node (id, node_name, node_type_id, tenant_id, properties, qualified_path, created_at, updated_at)
        VALUES
            -- Required fields — Portfolio
            (gen_random_uuid(), 'Portfolio_RequiredPortfolioName', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'required_field',
                'field', 'portfolio_name',
                'severity', 'Hard',
                'expression', 'portfolio_name IS NOT NULL AND portfolio_name <> ''''',
                'message', 'Portfolio name is required'
             ), 'rules/dq/portfolio/RequiredPortfolioName', NOW(), NOW()),

            (gen_random_uuid(), 'Portfolio_RequiredBaseCurrency', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'required_field',
                'field', 'base_currency',
                'severity', 'Hard',
                'expression', 'base_currency IS NOT NULL',
                'message', 'Base currency is required'
             ), 'rules/dq/portfolio/RequiredBaseCurrency', NOW(), NOW()),

            (gen_random_uuid(), 'Portfolio_RequiredInceptionDate', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'required_field',
                'field', 'inception_date',
                'severity', 'Hard',
                'expression', 'inception_date IS NOT NULL',
                'message', 'Inception date is required'
             ), 'rules/dq/portfolio/RequiredInceptionDate', NOW(), NOW()),

            -- Semantic validity — Portfolio
            (gen_random_uuid(), 'Portfolio_ValidBaseCurrency', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'semantic_validity',
                'field', 'base_currency',
                'severity', 'Hard',
                'expression', 'length(base_currency) = 3',
                'reference_list', 'ISO_4217',
                'message', 'Base currency must be a valid ISO 4217 code'
             ), 'rules/dq/portfolio/ValidBaseCurrency', NOW(), NOW()),

            (gen_random_uuid(), 'Portfolio_InceptionDateNotFuture', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'semantic_validity',
                'field', 'inception_date',
                'severity', 'Hard',
                'expression', 'inception_date <= CURRENT_DATE',
                'message', 'Inception date cannot be in the future'
             ), 'rules/dq/portfolio/InceptionDateNotFuture', NOW(), NOW()),

            (gen_random_uuid(), 'Portfolio_ValidPortfolioType', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'semantic_validity',
                'field', 'portfolio_type',
                'severity', 'Hard',
                'expression', 'portfolio_type IN (''SMA'',''Fund'',''ETF'',''Model'',''Mandate'',''Composite'')',
                'message', 'Portfolio type must be one of: SMA, Fund, ETF, Model, Mandate, Composite'
             ), 'rules/dq/portfolio/ValidPortfolioType', NOW(), NOW()),

            -- Referential integrity — Portfolio
            (gen_random_uuid(), 'Portfolio_ValidMandate', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'referential_integrity',
                'field', 'mandate_id',
                'severity', 'Soft',
                'expression', 'mandate_id IS NULL OR EXISTS(SELECT 1 FROM edm.mandate_master WHERE id = mandate_id AND tenant_id = portfolio_master.tenant_id)',
                'message', 'Mandate reference must exist in mandate_master'
             ), 'rules/dq/portfolio/ValidMandate', NOW(), NOW()),

            (gen_random_uuid(), 'Portfolio_ValidBenchmark', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'referential_integrity',
                'field', 'benchmark_id',
                'severity', 'Soft',
                'expression', 'benchmark_id IS NULL OR EXISTS(SELECT 1 FROM edm.benchmark_master WHERE id = benchmark_id AND tenant_id = portfolio_master.tenant_id)',
                'message', 'Benchmark reference must exist in benchmark_master'
             ), 'rules/dq/portfolio/ValidBenchmark', NOW(), NOW()),

            (gen_random_uuid(), 'Portfolio_ValidStrategy', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_master',
                'rule_type', 'referential_integrity',
                'field', 'strategy_id',
                'severity', 'Soft',
                'expression', 'strategy_id IS NULL OR EXISTS(SELECT 1 FROM edm.strategy_master WHERE id = strategy_id AND tenant_id = portfolio_master.tenant_id)',
                'message', 'Strategy reference must exist in strategy_master'
             ), 'rules/dq/portfolio/ValidStrategy', NOW(), NOW()),

            -- Hierarchy rules
            (gen_random_uuid(), 'PortfolioHierarchy_NoSelfReference', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'portfolio_hierarchy',
                'rule_type', 'structural_integrity',
                'severity', 'Hard',
                'expression', 'parent_portfolio_id <> child_portfolio_id',
                'message', 'Portfolio cannot be parent of itself'
             ), 'rules/dq/portfolio_hierarchy/NoSelfReference', NOW(), NOW()),

            -- Compliance rule validation
            (gen_random_uuid(), 'ComplianceRule_ExpressionNotEmpty', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'compliance_rule_master',
                'rule_type', 'required_field',
                'field', 'rule_expression',
                'severity', 'Hard',
                'expression', 'rule_expression IS NOT NULL AND length(rule_expression) > 0',
                'message', 'Compliance rule expression must not be empty'
             ), 'rules/dq/compliance_rule/ExpressionNotEmpty', NOW(), NOW()),

            -- Mandate rules
            (gen_random_uuid(), 'Mandate_RequiredMandateName', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'mandate_master',
                'rule_type', 'required_field',
                'field', 'mandate_name',
                'severity', 'Hard',
                'expression', 'mandate_name IS NOT NULL AND mandate_name <> ''''',
                'message', 'Mandate name is required'
             ), 'rules/dq/mandate/RequiredMandateName', NOW(), NOW())

        ON CONFLICT DO NOTHING;
    END IF;

    RAISE NOTICE '✅ Portfolio Master DQ + Survivorship Rules seeded.';
    RAISE NOTICE '   Survivorship rules: 16 field strategies';
    RAISE NOTICE '   DQ rules: 12 validation rules';
END $$;
