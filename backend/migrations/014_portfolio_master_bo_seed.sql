-- backend/migrations/014_portfolio_master_bo_seed.sql
-- Portfolio Master — Business Object + Semantic Catalog Seed
-- Extends existing portfolio/benchmark BOs and registers 4 new BOs.
-- Seeds semantic terms and BO↔BO graph edges.

DO $$
DECLARE
    v_gold_tenant   UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_system_user   UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';

    -- BO IDs (resolved after insert)
    bo_portfolio_id          UUID;
    bo_mandate_id            UUID;
    bo_benchmark_id          UUID;
    bo_strategy_id           UUID;
    bo_compliance_rule_id    UUID;
    bo_portfolio_hier_id     UUID;

    -- Catalog node type IDs
    semantic_term_type_id    UUID;
    business_object_type_id  UUID;
    references_edge_type_id  UUID;
    applies_to_edge_type_id  UUID;
    member_of_edge_type_id   UUID;

    -- Semantic term IDs
    st_portfolio_id_id       UUID := gen_random_uuid();
    st_portfolio_code_id     UUID := gen_random_uuid();
    st_portfolio_name_id     UUID := gen_random_uuid();
    st_portfolio_type_id     UUID := gen_random_uuid();
    st_inception_date_id     UUID := gen_random_uuid();
    st_currency_code_id      UUID := gen_random_uuid();
    st_mandate_id_id         UUID := gen_random_uuid();
    st_mandate_name_id       UUID := gen_random_uuid();
    st_mandate_type_id       UUID := gen_random_uuid();
    st_benchmark_id_id       UUID := gen_random_uuid();
    st_benchmark_name_id     UUID := gen_random_uuid();
    st_benchmark_type_id     UUID := gen_random_uuid();
    st_strategy_id_id        UUID := gen_random_uuid();
    st_strategy_name_id      UUID := gen_random_uuid();
    st_compliance_rule_id_id UUID := gen_random_uuid();
    st_compliance_expr_id    UUID := gen_random_uuid();
    st_parent_portfolio_id   UUID := gen_random_uuid();
    st_child_portfolio_id    UUID := gen_random_uuid();
    st_risk_profile_id       UUID := gen_random_uuid();
    st_invest_objective_id   UUID := gen_random_uuid();
    st_legal_structure_id    UUID := gen_random_uuid();
    st_regulatory_class_id   UUID := gen_random_uuid();
    st_liquidity_profile_id  UUID := gen_random_uuid();
    st_valuation_method_id   UUID := gen_random_uuid();
    st_esg_policy_id         UUID := gen_random_uuid();
    st_invest_style_id       UUID := gen_random_uuid();
    st_geo_focus_id          UUID := gen_random_uuid();
    st_effective_date_id     UUID := gen_random_uuid();
    st_end_date_id           UUID := gen_random_uuid();
    st_rule_severity_id      UUID := gen_random_uuid();
    st_rule_frequency_id     UUID := gen_random_uuid();
    st_hier_relation_type_id UUID := gen_random_uuid();
    st_hier_weight_id        UUID := gen_random_uuid();

BEGIN
    -- ================================================================
    -- 1. RESOLVE CATALOG NODE / EDGE TYPE IDs
    -- ================================================================
    SELECT id INTO semantic_term_type_id
      FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO business_object_type_id
      FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1;
    SELECT id INTO references_edge_type_id
      FROM catalog_edge_types WHERE edge_type_name = 'references' LIMIT 1;
    SELECT id INTO applies_to_edge_type_id
      FROM catalog_edge_types WHERE edge_type_name = 'applies_to' LIMIT 1;
    SELECT id INTO member_of_edge_type_id
      FROM catalog_edge_types WHERE edge_type_name = 'member_of' LIMIT 1;

    IF semantic_term_type_id IS NULL THEN
        RAISE NOTICE 'semantic_term node type not found; skipping catalog seed.';
        RETURN;
    END IF;

    -- ================================================================
    -- 2. EXTEND EXISTING portfolio BO WITH INSTITUTIONAL MDM FIELDS
    --    (ON CONFLICT DO NOTHING keeps existing fields untouched)
    -- ================================================================
    SELECT id INTO bo_portfolio_id FROM business_objects WHERE key = 'portfolio' LIMIT 1;

    IF bo_portfolio_id IS NOT NULL THEN
        INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence)
        SELECT gen_random_uuid(), v_gold_tenant, bo_portfolio_id, key, label, label, key, type, true, required, description, seq
        FROM (VALUES
            ('portfolio_type',           'Portfolio Type',           'picklist', false, 'SMA | Fund | ETF | Model | Mandate | Composite',              50),
            ('portfolio_category',       'Portfolio Category',       'picklist', false, 'Retail | Institutional | HNW | Advisory',                      51),
            ('legal_structure',          'Legal Structure',          'text',     false, 'Trust | Fund | LP | UCITS | SICAV',                            52),
            ('regulatory_classification','Regulatory Classification','text',     false, 'UCITS | 40-Act | AIF | Other',                                 53),
            ('liquidity_profile',        'Liquidity Profile',        'picklist', false, 'Daily | Weekly | Monthly | Quarterly | Locked',                54),
            ('risk_profile',             'Risk Profile',             'picklist', false, 'Conservative | Balanced | Moderate | Aggressive | Growth',     55),
            ('investment_objective',     'Investment Objective',     'text',     false, 'Free-text investment objective',                               56),
            ('investment_guidelines',    'Investment Guidelines',    'text',     false, 'Free-text investment guidelines',                              57),
            ('is_model_portfolio',       'Is Model Portfolio',       'boolean',  false, 'Whether this is a model portfolio',                            58),
            ('is_composite_member',      'Is Composite Member',      'boolean',  false, 'Whether this portfolio is part of a composite',                59),
            ('composite_id',             'Composite',                'reference',false, 'Link to parent composite portfolio',                           60),
            ('mandate_id',               'Mandate',                  'reference',false, 'Link to Mandate Master',                                       61),
            ('strategy_id',              'Strategy',                 'reference',false, 'Link to Strategy Master',                                      62),
            ('valuation_frequency',      'Valuation Frequency',      'picklist', false, 'Daily | Weekly | Monthly | Quarterly',                         63),
            ('pricing_source',           'Pricing Source',           'text',     false, 'Primary pricing vendor',                                       64),
            ('portfolio_manager_id',     'Portfolio Manager',        'text',     false, 'Primary PM identifier',                                        65),
            ('custodian_id',             'Custodian',                'text',     false, 'Primary custodian identifier',                                 66),
            ('domicile',                 'Domicile',                 'text',     false, 'Country of registration',                                      67),
            ('termination_date',         'Termination Date',         'date',     false, 'Date the portfolio was closed',                                68)
        ) AS t(key, label, type, required, description, seq)
        ON CONFLICT DO NOTHING;
    END IF;

    -- ================================================================
    -- 3. EXTEND EXISTING benchmark BO
    -- ================================================================
    SELECT id INTO bo_benchmark_id FROM business_objects WHERE key = 'benchmark' LIMIT 1;

    IF bo_benchmark_id IS NOT NULL THEN
        INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence)
        SELECT gen_random_uuid(), v_gold_tenant, bo_benchmark_id, key, label, label, key, type, true, required, description, seq
        FROM (VALUES
            ('provider',            'Provider',               'text',     false, 'MSCI | S&P | Bloomberg | Custom',  40),
            ('rebalance_frequency', 'Rebalance Frequency',    'picklist', false, 'Daily | Monthly | Quarterly',      41),
            ('is_custom',           'Is Custom Benchmark',    'boolean',  false, 'Whether this is a custom benchmark', 42),
            ('custom_definition',   'Custom Definition',      'json',     false, 'Weights / constituents JSON',       43),
            ('composition_source',  'Composition Source',     'text',     false, 'Vendor | Internal',                 44)
        ) AS t(key, label, type, required, description, seq)
        ON CONFLICT DO NOTHING;
    END IF;

    -- ================================================================
    -- 4. REGISTER NEW BOs: mandate, strategy, compliance_rule, portfolio_hierarchy
    -- ================================================================

    -- Mandate BO
    INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, category, created_at)
    VALUES (gen_random_uuid(), v_gold_tenant, 'mandate', 'Mandate', 'Mandate', 'mandate',
            'Client investment objectives, constraints, and guidelines for one or more portfolios.',
            'file-contract', true, 'Investment', NOW())
    ON CONFLICT DO NOTHING;
    SELECT id INTO bo_mandate_id FROM business_objects WHERE key = 'mandate' AND tenant_id = v_gold_tenant LIMIT 1;

    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
    SELECT gen_random_uuid(), v_gold_tenant, bo_mandate_id, key, label, label, key, type, true, required, seq
    FROM (VALUES
        ('mandate_code',        'Mandate Code',        'text',     true,  1),
        ('mandate_name',        'Mandate Name',        'text',     true,  2),
        ('mandate_type',        'Mandate Type',        'picklist', true,  3),
        ('client_id',           'Client',              'text',     false, 4),
        ('investment_objective','Investment Objective','picklist', false, 5),
        ('risk_tolerance',      'Risk Tolerance',      'picklist', false, 6),
        ('time_horizon',        'Time Horizon',        'picklist', false, 7),
        ('liquidity_needs',     'Liquidity Needs',     'picklist', false, 8),
        ('tax_constraints',     'Tax Constraints',     'text',     false, 9),
        ('esg_constraints',     'ESG Constraints',     'text',     false, 10),
        ('custom_restrictions', 'Custom Restrictions', 'text',     false, 11),
        ('benchmark_id',        'Benchmark',           'reference',false, 12)
    ) AS t(key, label, type, required, seq)
    ON CONFLICT DO NOTHING;

    -- Strategy BO
    INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, category, created_at)
    VALUES (gen_random_uuid(), v_gold_tenant, 'strategy', 'Strategy', 'Strategy', 'strategy',
            'Defines the investment strategy applied to one or more portfolios.',
            'chess', true, 'Investment', NOW())
    ON CONFLICT DO NOTHING;
    SELECT id INTO bo_strategy_id FROM business_objects WHERE key = 'strategy' AND tenant_id = v_gold_tenant LIMIT 1;

    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
    SELECT gen_random_uuid(), v_gold_tenant, bo_strategy_id, key, label, label, key, type, true, required, seq
    FROM (VALUES
        ('strategy_code',     'Strategy Code',     'text',     true,  1),
        ('strategy_name',     'Strategy Name',     'text',     true,  2),
        ('strategy_category', 'Strategy Category', 'picklist', true,  3),
        ('investment_style',  'Investment Style',  'picklist', false, 4),
        ('geographic_focus',  'Geographic Focus',  'picklist', false, 5),
        ('sector_focus',      'Sector Focus',      'text',     false, 6),
        ('risk_budget',       'Risk Budget',       'text',     false, 7),
        ('leverage_policy',   'Leverage Policy',   'picklist', false, 8),
        ('derivatives_policy','Derivatives Policy','picklist', false, 9),
        ('esg_policy',        'ESG Policy',        'text',     false, 10)
    ) AS t(key, label, type, required, seq)
    ON CONFLICT DO NOTHING;

    -- Compliance Rule BO
    INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, category, created_at)
    VALUES (gen_random_uuid(), v_gold_tenant, 'compliance_rule', 'Compliance Rule', 'Compliance Rule', 'compliance_rule',
            'Constraint or guideline enforced on portfolios, mandates, or strategies via the rules engine.',
            'shield-check', true, 'Compliance', NOW())
    ON CONFLICT DO NOTHING;
    SELECT id INTO bo_compliance_rule_id FROM business_objects WHERE key = 'compliance_rule' AND tenant_id = v_gold_tenant LIMIT 1;

    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
    SELECT gen_random_uuid(), v_gold_tenant, bo_compliance_rule_id, key, label, label, key, type, true, required, seq
    FROM (VALUES
        ('rule_code',       'Rule Code',       'text',     true,  1),
        ('rule_name',       'Rule Name',       'text',     true,  2),
        ('portfolio_id',    'Portfolio',        'text',     false, 3),
        ('strategy_id',     'Strategy',         'reference',false, 4),
        ('mandate_id',      'Mandate',          'reference',false, 5),
        ('rule_type',       'Rule Type',        'picklist', true,  6),
        ('rule_expression', 'Rule Expression',  'text',     true,  7),
        ('severity',        'Severity',         'picklist', true,  8),
        ('frequency',       'Frequency',        'picklist', true,  9),
        ('effective_date',  'Effective Date',   'date',     true,  10),
        ('end_date',        'End Date',         'date',     false, 11)
    ) AS t(key, label, type, required, seq)
    ON CONFLICT DO NOTHING;

    -- Portfolio Hierarchy BO
    INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, category, created_at)
    VALUES (gen_random_uuid(), v_gold_tenant, 'portfolio_hierarchy', 'Portfolio Hierarchy', 'Portfolio Hierarchy', 'portfolio_hierarchy',
            'Defines parent-child relationships between portfolios: sleeves, composites, umbrella structures.',
            'sitemap', true, 'Investment', NOW())
    ON CONFLICT DO NOTHING;
    SELECT id INTO bo_portfolio_hier_id FROM business_objects WHERE key = 'portfolio_hierarchy' AND tenant_id = v_gold_tenant LIMIT 1;

    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
    SELECT gen_random_uuid(), v_gold_tenant, bo_portfolio_hier_id, key, label, label, key, type, true, required, seq
    FROM (VALUES
        ('parent_portfolio_id','Parent Portfolio', 'reference',true,  1),
        ('child_portfolio_id', 'Child Portfolio',  'reference',true,  2),
        ('relationship_type',  'Relationship Type','picklist', true,  3),
        ('weight',             'Weight',            'decimal',  false, 4),
        ('effective_date',     'Effective Date',    'date',     true,  5),
        ('end_date',           'End Date',          'date',     false, 6)
    ) AS t(key, label, type, required, seq)
    ON CONFLICT DO NOTHING;

    -- ================================================================
    -- 5. SEED SEMANTIC TERMS (catalog_node)
    --    Only inserted if the semantic_term node type exists.
    -- ================================================================
    IF semantic_term_type_id IS NOT NULL THEN
        INSERT INTO catalog_node (id, node_name, node_type_id, tenant_id, properties, qualified_path, created_at, updated_at)
        VALUES
            (st_portfolio_id_id,     'PortfolioID',              semantic_term_type_id, v_gold_tenant, '{"data_type":"uuid","domain":"portfolio"}',   'semantic/portfolio/PortfolioID',             NOW(), NOW()),
            (st_portfolio_code_id,   'PortfolioCode',            semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/PortfolioCode',           NOW(), NOW()),
            (st_portfolio_name_id,   'PortfolioName',            semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/PortfolioName',           NOW(), NOW()),
            (st_portfolio_type_id,   'PortfolioType',            semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/PortfolioType',           NOW(), NOW()),
            (st_inception_date_id,   'InceptionDate',            semantic_term_type_id, v_gold_tenant, '{"data_type":"date","domain":"portfolio"}',   'semantic/portfolio/InceptionDate',           NOW(), NOW()),
            (st_currency_code_id,    'CurrencyCode',             semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"common"}',      'semantic/common/CurrencyCode',               NOW(), NOW()),
            (st_mandate_id_id,       'MandateID',                semantic_term_type_id, v_gold_tenant, '{"data_type":"uuid","domain":"mandate"}',     'semantic/mandate/MandateID',                 NOW(), NOW()),
            (st_mandate_name_id,     'MandateName',              semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"mandate"}',     'semantic/mandate/MandateName',               NOW(), NOW()),
            (st_mandate_type_id,     'MandateType',              semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"mandate"}',     'semantic/mandate/MandateType',               NOW(), NOW()),
            (st_benchmark_id_id,     'BenchmarkID',              semantic_term_type_id, v_gold_tenant, '{"data_type":"uuid","domain":"benchmark"}',   'semantic/benchmark/BenchmarkID',             NOW(), NOW()),
            (st_benchmark_name_id,   'BenchmarkName',            semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"benchmark"}',   'semantic/benchmark/BenchmarkName',           NOW(), NOW()),
            (st_benchmark_type_id,   'BenchmarkType',            semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"benchmark"}',   'semantic/benchmark/BenchmarkType',           NOW(), NOW()),
            (st_strategy_id_id,      'StrategyID',               semantic_term_type_id, v_gold_tenant, '{"data_type":"uuid","domain":"strategy"}',    'semantic/strategy/StrategyID',               NOW(), NOW()),
            (st_strategy_name_id,    'StrategyName',             semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"strategy"}',    'semantic/strategy/StrategyName',             NOW(), NOW()),
            (st_compliance_rule_id_id,'ComplianceRuleID',        semantic_term_type_id, v_gold_tenant, '{"data_type":"uuid","domain":"compliance"}',  'semantic/compliance/ComplianceRuleID',       NOW(), NOW()),
            (st_compliance_expr_id,  'ComplianceRuleExpression', semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"compliance"}',  'semantic/compliance/RuleExpression',         NOW(), NOW()),
            (st_parent_portfolio_id, 'ParentPortfolioReference', semantic_term_type_id, v_gold_tenant, '{"data_type":"uuid","domain":"portfolio"}',   'semantic/portfolio/ParentPortfolioReference',NOW(), NOW()),
            (st_child_portfolio_id,  'ChildPortfolioReference',  semantic_term_type_id, v_gold_tenant, '{"data_type":"uuid","domain":"portfolio"}',   'semantic/portfolio/ChildPortfolioReference', NOW(), NOW()),
            (st_risk_profile_id,     'RiskProfile',              semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/RiskProfile',             NOW(), NOW()),
            (st_invest_objective_id, 'InvestmentObjective',      semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/InvestmentObjective',     NOW(), NOW()),
            (st_legal_structure_id,  'LegalStructure',           semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/LegalStructure',          NOW(), NOW()),
            (st_regulatory_class_id, 'RegulatoryClassification', semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/RegulatoryClassification',NOW(), NOW()),
            (st_liquidity_profile_id,'LiquidityProfile',         semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/LiquidityProfile',        NOW(), NOW()),
            (st_valuation_method_id, 'ValuationMethod',          semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"performance"}', 'semantic/performance/ValuationMethod',       NOW(), NOW()),
            (st_esg_policy_id,       'ESGPolicy',                semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"compliance"}',  'semantic/compliance/ESGPolicy',              NOW(), NOW()),
            (st_invest_style_id,     'InvestmentStyle',          semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"strategy"}',    'semantic/strategy/InvestmentStyle',          NOW(), NOW()),
            (st_geo_focus_id,        'GeographicFocus',          semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"strategy"}',    'semantic/strategy/GeographicFocus',          NOW(), NOW()),
            (st_effective_date_id,   'EffectiveDate',            semantic_term_type_id, v_gold_tenant, '{"data_type":"date","domain":"common"}',      'semantic/common/EffectiveDate',              NOW(), NOW()),
            (st_end_date_id,         'EndDate',                  semantic_term_type_id, v_gold_tenant, '{"data_type":"date","domain":"common"}',      'semantic/common/EndDate',                    NOW(), NOW()),
            (st_rule_severity_id,    'ComplianceRuleSeverity',   semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"compliance"}',  'semantic/compliance/RuleSeverity',           NOW(), NOW()),
            (st_rule_frequency_id,   'ComplianceRuleFrequency',  semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"compliance"}',  'semantic/compliance/RuleFrequency',          NOW(), NOW()),
            (st_hier_relation_type_id,'PortfolioRelationshipType',semantic_term_type_id,v_gold_tenant, '{"data_type":"text","domain":"portfolio"}',   'semantic/portfolio/RelationshipType',        NOW(), NOW()),
            (st_hier_weight_id,      'PortfolioHierarchyWeight', semantic_term_type_id, v_gold_tenant, '{"data_type":"decimal","domain":"portfolio"}','semantic/portfolio/HierarchyWeight',         NOW(), NOW())
        ON CONFLICT (node_name, node_type_id, tenant_id) DO NOTHING;
    END IF;

    -- ================================================================
    -- 6. SEED GRAPH EDGES — BO ↔ BO relationships (catalog_edge)
    --    Only if edge types are available.
    -- ================================================================
    SELECT id INTO bo_portfolio_id FROM business_objects WHERE key = 'portfolio' AND tenant_id = v_gold_tenant LIMIT 1;

    IF references_edge_type_id IS NOT NULL AND bo_portfolio_id IS NOT NULL AND bo_mandate_id IS NOT NULL THEN
        INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, edge_type_name, relationship_type, tenant_id, created_at, updated_at)
        VALUES
            -- Portfolio → references → Mandate
            (gen_random_uuid(), bo_portfolio_id, bo_mandate_id, references_edge_type_id, 'references', 'references', v_gold_tenant, NOW(), NOW()),
            -- Portfolio → references → Benchmark
            (gen_random_uuid(), bo_portfolio_id, bo_benchmark_id, references_edge_type_id, 'references', 'references', v_gold_tenant, NOW(), NOW()),
            -- Portfolio → references → Strategy
            (gen_random_uuid(), bo_portfolio_id, bo_strategy_id, references_edge_type_id, 'references', 'references', v_gold_tenant, NOW(), NOW()),
            -- Mandate → references → Benchmark
            (gen_random_uuid(), bo_mandate_id, bo_benchmark_id, references_edge_type_id, 'references', 'references', v_gold_tenant, NOW(), NOW())
        ON CONFLICT DO NOTHING;
    END IF;

    IF applies_to_edge_type_id IS NOT NULL AND bo_compliance_rule_id IS NOT NULL AND bo_portfolio_id IS NOT NULL THEN
        INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, edge_type_name, relationship_type, tenant_id, created_at, updated_at)
        VALUES
            -- ComplianceRule → applies_to → Portfolio
            (gen_random_uuid(), bo_compliance_rule_id, bo_portfolio_id, applies_to_edge_type_id, 'applies_to', 'applies_to', v_gold_tenant, NOW(), NOW()),
            -- ComplianceRule → applies_to → Strategy
            (gen_random_uuid(), bo_compliance_rule_id, bo_strategy_id, applies_to_edge_type_id, 'applies_to', 'applies_to', v_gold_tenant, NOW(), NOW()),
            -- ComplianceRule → applies_to → Mandate
            (gen_random_uuid(), bo_compliance_rule_id, bo_mandate_id, applies_to_edge_type_id, 'applies_to', 'applies_to', v_gold_tenant, NOW(), NOW())
        ON CONFLICT DO NOTHING;
    END IF;

    RAISE NOTICE '✅ Portfolio Master BO seed complete.';
    RAISE NOTICE '   BOs: portfolio (extended), benchmark (extended), mandate, strategy, compliance_rule, portfolio_hierarchy';
    RAISE NOTICE '   Semantic terms: % nodes inserted', 33;
END $$;
