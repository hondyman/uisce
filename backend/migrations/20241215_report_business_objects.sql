-- ============================================================================
-- REPORT BUSINESS OBJECTS SCHEMA
-- Workday-style report definitions with semantic layer integration
-- ============================================================================

-- Report Business Objects (Workday-style)
CREATE TABLE IF NOT EXISTS report_business_objects (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL,
    report_type TEXT NOT NULL DEFAULT 'standard', -- standard, composite, scheduled
    data_source JSONB NOT NULL DEFAULT '{}',
    layout JSONB DEFAULT '{}',
    parameters JSONB DEFAULT '[]',
    semantic_bindings JSONB DEFAULT '[]', -- Links to business objects
    permissions JSONB DEFAULT '{}',
    schedule JSONB,
    output_formats TEXT[] DEFAULT ARRAY['html', 'pdf', 'excel'],
    is_system BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    version INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by TEXT,
    CONSTRAINT rbo_unique_key UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_rbo_tenant_category ON report_business_objects(tenant_id, category);
CREATE INDEX IF NOT EXISTS idx_rbo_active ON report_business_objects(is_active);

-- Report Execution History
CREATE TABLE IF NOT EXISTS report_executions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    report_id TEXT NOT NULL REFERENCES report_business_objects(id),
    report_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued', -- queued, running, completed, failed
    parameters JSONB DEFAULT '{}',
    output_format TEXT NOT NULL DEFAULT 'html',
    output_url TEXT,
    output_data JSONB,
    row_count INT DEFAULT 0,
    generation_ms INT,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    requested_by TEXT,
    process_instance_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_re_report ON report_executions(report_id);
CREATE INDEX IF NOT EXISTS idx_re_status ON report_executions(status);
CREATE INDEX IF NOT EXISTS idx_re_requested_by ON report_executions(requested_by);

-- Report Subscriptions (Scheduled delivery)
CREATE TABLE IF NOT EXISTS report_subscriptions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    report_id TEXT NOT NULL REFERENCES report_business_objects(id),
    subscriber_id TEXT NOT NULL,
    subscriber_type TEXT NOT NULL DEFAULT 'user', -- user, role, email
    delivery_method TEXT NOT NULL DEFAULT 'portal', -- portal, email, sftp
    delivery_config JSONB DEFAULT '{}',
    parameters JSONB DEFAULT '{}',
    output_format TEXT NOT NULL DEFAULT 'pdf',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_rs_report ON report_subscriptions(report_id);
CREATE INDEX IF NOT EXISTS idx_rs_subscriber ON report_subscriptions(subscriber_id);

-- ============================================================================
-- SEED FINANCIAL REPORT BUSINESS OBJECTS
-- ============================================================================

-- Portfolio Summary Report
INSERT INTO report_business_objects (id, tenant_id, key, name, display_name, description, category, report_type, data_source, semantic_bindings, is_system)
VALUES (
    'rbo-portfolio-summary',
    'default-tenant',
    'portfolio_summary',
    'Portfolio Summary',
    'Portfolio Summary Report',
    'Comprehensive portfolio overview with positions, allocation, and performance',
    'portfolio',
    'standard',
    '{"type": "graphql", "query": "portfolios", "dimensions": ["positions", "allocation"], "measures": ["market_value", "cost_basis", "unrealized_gain_loss"]}',
    '[{"business_object_key": "portfolio", "fields": ["name", "market_value", "ytd_return"], "relationship": "primary"}, {"business_object_key": "position", "fields": ["ticker", "quantity", "market_value"], "relationship": "referenced"}]',
    true
) ON CONFLICT (tenant_id, key) DO NOTHING;

-- Performance Report
INSERT INTO report_business_objects (id, tenant_id, key, name, display_name, description, category, report_type, data_source, semantic_bindings, is_system)
VALUES (
    'rbo-performance',
    'default-tenant',
    'performance_report',
    'Performance Report',
    'Investment Performance Report',
    'Time-weighted returns with benchmark comparison and risk metrics',
    'performance',
    'standard',
    '{"type": "graphql", "query": "performance", "dimensions": ["period"], "measures": ["return_twr", "benchmark_return", "alpha", "volatility", "sharpe_ratio"]}',
    '[{"business_object_key": "performance", "fields": ["period", "return_twr", "benchmark_return", "alpha"], "relationship": "primary"}, {"business_object_key": "portfolio", "fields": ["name"], "relationship": "referenced"}]',
    true
) ON CONFLICT (tenant_id, key) DO NOTHING;

-- Holdings Report
INSERT INTO report_business_objects (id, tenant_id, key, name, display_name, description, category, report_type, data_source, semantic_bindings, is_system)
VALUES (
    'rbo-holdings',
    'default-tenant',
    'holdings_report',
    'Holdings Report',
    'Current Holdings Report',
    'Detailed position listing with cost basis, gains, and weights',
    'portfolio',
    'standard',
    '{"type": "graphql", "query": "positions", "dimensions": ["security", "account"], "measures": ["quantity", "market_value", "cost_basis", "unrealized_gain_loss", "weight"]}',
    '[{"business_object_key": "position", "fields": ["quantity", "market_value", "cost_basis", "weight"], "relationship": "primary"}, {"business_object_key": "security", "fields": ["ticker", "name", "asset_class", "sector"], "relationship": "referenced"}]',
    true
) ON CONFLICT (tenant_id, key) DO NOTHING;

-- Transaction History Report
INSERT INTO report_business_objects (id, tenant_id, key, name, display_name, description, category, report_type, data_source, semantic_bindings, is_system)
VALUES (
    'rbo-transactions',
    'default-tenant',
    'transaction_report',
    'Transaction History',
    'Transaction History Report',
    'Trade and activity history with settlement details',
    'activity',
    'standard',
    '{"type": "graphql", "query": "transactions", "dimensions": ["type", "security", "account"], "measures": ["quantity", "price", "amount", "fees"], "filters": [{"field": "trade_date", "operator": "gte", "value": "$start_date"}, {"field": "trade_date", "operator": "lte", "value": "$end_date"}]}',
    '[{"business_object_key": "transaction", "fields": ["type", "trade_date", "quantity", "price", "amount"], "relationship": "primary"}, {"business_object_key": "security", "fields": ["ticker", "name"], "relationship": "referenced"}]',
    true
) ON CONFLICT (tenant_id, key) DO NOTHING;

-- Fee Billing Report
INSERT INTO report_business_objects (id, tenant_id, key, name, display_name, description, category, report_type, data_source, semantic_bindings, is_system)
VALUES (
    'rbo-fee-billing',
    'default-tenant',
    'fee_billing_report',
    'Fee Billing Report',
    'Advisory Fee Billing Report',
    'Fee calculations with breakdown by account and fee type',
    'billing',
    'standard',
    '{"type": "graphql", "query": "fees", "dimensions": ["fee_type", "account"], "measures": ["billable_amount", "fee_amount"]}',
    '[{"business_object_key": "fee", "fields": ["fee_type", "billable_amount", "fee_amount", "period_start", "period_end"], "relationship": "primary"}, {"business_object_key": "account", "fields": ["account_number", "name"], "relationship": "referenced"}]',
    true
) ON CONFLICT (tenant_id, key) DO NOTHING;

-- Asset Allocation Report
INSERT INTO report_business_objects (id, tenant_id, key, name, display_name, description, category, report_type, data_source, semantic_bindings, is_system)
VALUES (
    'rbo-allocation',
    'default-tenant',
    'allocation_report',
    'Asset Allocation Report',
    'Asset Allocation Analysis',
    'Current vs target allocation with drift analysis',
    'portfolio',
    'standard',
    '{"type": "graphql", "query": "allocation", "dimensions": ["category", "dimension"], "measures": ["market_value", "weight", "target_weight", "drift"]}',
    '[{"business_object_key": "allocation", "fields": ["category", "market_value", "weight", "target_weight", "drift"], "relationship": "primary"}]',
    true
) ON CONFLICT (tenant_id, key) DO NOTHING;

-- Tax Lot Report
INSERT INTO report_business_objects (id, tenant_id, key, name, display_name, description, category, report_type, data_source, semantic_bindings, is_system)
VALUES (
    'rbo-taxlots',
    'default-tenant',
    'taxlot_report',
    'Tax Lot Report',
    'Tax Lot Detail Report',
    'Detailed tax lot listing for cost basis and gain/loss analysis',
    'tax',
    'standard',
    '{"type": "graphql", "query": "tax_lots", "dimensions": ["security", "holding_period"], "measures": ["quantity", "cost_per_share", "total_cost", "unrealized_gain_loss"]}',
    '[{"business_object_key": "taxlot", "fields": ["acquisition_date", "quantity", "cost_per_share", "holding_period"], "relationship": "primary"}, {"business_object_key": "security", "fields": ["ticker", "name"], "relationship": "referenced"}]',
    true
) ON CONFLICT (tenant_id, key) DO NOTHING;

-- ============================================================================
-- REPORT GENERATION BUSINESS PROCESS
-- ============================================================================

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
    EXECUTE $exec$
      INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, is_system)
      VALUES (gen_random_uuid()::text, 'default-tenant', 'report_execution', 'Report Execution', 'Report Generation', 'Automated report generation workflow', 'reporting', true)
      ON CONFLICT (tenant_id, key) DO NOTHING;
    $exec$;

    EXECUTE $exec$
      INSERT INTO process_steps (id, tenant_id, process_id, key, name, display_name, step_type, sequence, config, is_required, created_at)
      SELECT gen_random_uuid()::text, 'default-tenant', (SELECT id FROM business_processes WHERE key = 'report_execution'), key, name, label, step_type, seq, config::jsonb, required, now()
      FROM (VALUES
        ('validate_params', 'Validate Parameters', 'validate', 1, '{"rules": ["required_params", "date_range_valid"]}', true),
        ('fetch_data', 'Fetch Data', 'integration', 2, '{"target": "semantic_layer", "timeout": 120}', true),
        ('render_report', 'Render Report', 'generate', 3, '{"formats": ["html", "pdf", "excel"]}', true),
        ('quality_check', 'Quality Check', 'validate', 4, '{"rules": ["row_count_min", "data_freshness"]}', false),
        ('deliver', 'Deliver Report', 'notify', 5, '{"channels": ["portal", "email"]}', true),
        ('archive', 'Archive Report', 'integration', 6, '{"target": "document_store", "retention": "7_years"}', true)
      ) AS t(key, label, step_type, seq, config, required)
      ON CONFLICT DO NOTHING;
    $exec$;
  ELSE
    RAISE NOTICE 'business_processes table not present, skipping report generation process';
  END IF;
END$$;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Report Business Objects schema created successfully!';
    RAISE NOTICE '✓ Seeded: Portfolio Summary, Performance, Holdings, Transactions, Fee Billing, Allocation, Tax Lot';
    RAISE NOTICE '✓ Report Generation business process created';
END $$;
