-- backend/migrations/017_performance_settings_rules.sql
-- Portfolio Master — Performance Settings DQ + Survivorship Rules

DO $$
DECLARE
    v_gold_tenant        UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_system_user        UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    validation_rule_type_id UUID;
BEGIN

    -- ================================================================
    -- 1. SURVIVORSHIP RULES — edm.survivorship_rules
    -- ================================================================

    INSERT INTO edm.survivorship_rules
        (id, tenant_id, entity_type, field_name, strategy, preferred_sources, priority, created_by)
    VALUES
        -- valuation_method: prefer AccountingSystem
        (gen_random_uuid(), v_gold_tenant, 'performance_settings', 'valuation_method',
         'prefer_source', ARRAY['AccountingSystem','OMS'], 1, v_system_user),

        -- fee_treatment: prefer AccountingSystem
        (gen_random_uuid(), v_gold_tenant, 'performance_settings', 'fee_treatment',
         'prefer_source', ARRAY['AccountingSystem','FundAdmin'], 1, v_system_user),

        -- cash_flow_method: prefer AccountingSystem
        (gen_random_uuid(), v_gold_tenant, 'performance_settings', 'cash_flow_method',
         'prefer_source', ARRAY['AccountingSystem','OMS'], 1, v_system_user),

        -- currency_hedging_policy: prefer OMS
        (gen_random_uuid(), v_gold_tenant, 'performance_settings', 'currency_hedging_policy',
         'prefer_source', ARRAY['OMS','AccountingSystem'], 1, v_system_user)

    ON CONFLICT (tenant_id, entity_type, field_name, priority) DO NOTHING;

    -- ================================================================
    -- 2. DQ VALIDATION RULES — catalog_node (validation_rule)
    -- ================================================================
    SELECT id INTO validation_rule_type_id
      FROM catalog_node_type WHERE catalog_type_name = 'validation_rule' LIMIT 1;

    IF validation_rule_type_id IS NOT NULL THEN
        INSERT INTO catalog_node (id, node_name, node_type_id, tenant_id, properties, qualified_path, created_at, updated_at)
        VALUES
            -- Required: Valuation Method
            (gen_random_uuid(), 'Perf_RequiredValuationMethod', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'performance_settings',
                'rule_type', 'required_field',
                'field', 'valuation_method',
                'severity', 'Hard',
                'expression', 'valuation_method IS NOT NULL',
                'message', 'Valuation method is required for performance reporting'
             ), 'rules/dq/performance/RequiredValuationMethod', NOW(), NOW()),

            -- Required: Fee Treatment
            (gen_random_uuid(), 'Perf_RequiredFeeTreatment', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'performance_settings',
                'rule_type', 'required_field',
                'field', 'fee_treatment',
                'severity', 'Hard',
                'expression', 'fee_treatment IS NOT NULL',
                'message', 'Fee treatment must be specified (Gross/Net/Both)'
             ), 'rules/dq/performance/RequiredFeeTreatment', NOW(), NOW()),

            -- Semantic: Valid Valuation Method
            (gen_random_uuid(), 'Perf_ValidValuationMethod', validation_rule_type_id, v_gold_tenant,
             jsonb_build_object(
                'scope', 'performance_settings',
                'rule_type', 'semantic_validity',
                'field', 'valuation_method',
                'severity', 'Hard',
                'expression', 'valuation_method IN (''Daily_TIB'',''Monthly_Weighted'',''Simple_Dietz'',''Modified_Dietz'')',
                'message', 'Invalid valuation method selected'
             ), 'rules/dq/performance/ValidValuationMethod', NOW(), NOW())

        ON CONFLICT DO NOTHING;
    END IF;

    RAISE NOTICE '✅ Performance Settings DQ + Survivorship Rules seeded.';
END $$;
