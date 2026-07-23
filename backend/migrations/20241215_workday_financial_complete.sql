-- ============================================================================
-- STANDALONE WORKDAY SCHEMA + FINANCIAL BUSINESS OBJECTS + PROCESSES
-- Complete migration for wealth management platform
-- ============================================================================

-- ============================================================================
-- PART 1: BUSINESS OBJECTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS business_objects (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    technical_name TEXT,
    description TEXT,
    icon TEXT,
    is_core BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    last_modified_at TIMESTAMPTZ DEFAULT NOW(),
    last_modified_by TEXT,
    CONSTRAINT bo_unique_key UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_bo_tenant_key ON business_objects(tenant_id, key);

-- ============================================================================
-- PART 2: BUSINESS OBJECT FIELDS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS bo_fields (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    business_object_id TEXT NOT NULL REFERENCES business_objects(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    technical_name TEXT,
    type TEXT NOT NULL,
    is_core BOOLEAN DEFAULT false,
    is_required BOOLEAN DEFAULT false,
    is_readonly BOOLEAN DEFAULT false,
    is_searchable BOOLEAN DEFAULT true,
    description TEXT,
    sequence INT NOT NULL DEFAULT 0,
    section TEXT,
    default_value TEXT,
    validation_rules JSONB DEFAULT '[]',
    reference_bo TEXT,
    picklist_values TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    last_modified_at TIMESTAMPTZ DEFAULT NOW(),
    last_modified_by TEXT,
    CONSTRAINT bo_field_unique UNIQUE (business_object_id, key)
);

CREATE INDEX IF NOT EXISTS idx_bo_fields_bo ON bo_fields(business_object_id);

-- ============================================================================
-- PART 3: BUSINESS PROCESSES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS business_processes (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    category TEXT,
    status TEXT DEFAULT 'active',
    version INT DEFAULT 1,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    last_modified_at TIMESTAMPTZ DEFAULT NOW(),
    last_modified_by TEXT,
    CONSTRAINT bp_unique_key UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_bp_tenant_key ON business_processes(tenant_id, key);

-- ============================================================================
-- PART 4: PROCESS STEPS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS process_steps (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    process_id TEXT NOT NULL REFERENCES business_processes(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    step_type TEXT NOT NULL,
    sequence INT NOT NULL DEFAULT 0,
    config JSONB DEFAULT '{}',
    is_required BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    last_modified_at TIMESTAMPTZ DEFAULT NOW(),
    last_modified_by TEXT,
    CONSTRAINT ps_unique_key UNIQUE (process_id, key)
);

CREATE INDEX IF NOT EXISTS idx_ps_process ON process_steps(process_id);

-- ============================================================================
-- PART 5: PROCESS INSTANCES TABLE (Running processes)
-- ============================================================================

CREATE TABLE IF NOT EXISTS process_instances (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    process_id TEXT NOT NULL REFERENCES business_processes(id),
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    current_step_id TEXT REFERENCES process_steps(id),
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    data JSONB DEFAULT '{}',
    created_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_pi_process ON process_instances(process_id);
CREATE INDEX IF NOT EXISTS idx_pi_entity ON process_instances(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_pi_status ON process_instances(status);

-- ============================================================================
-- PART 6: STEP HISTORY TABLE (Audit trail)
-- ============================================================================

CREATE TABLE IF NOT EXISTS step_history (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    instance_id TEXT NOT NULL REFERENCES process_instances(id) ON DELETE CASCADE,
    step_id TEXT NOT NULL REFERENCES process_steps(id),
    action TEXT NOT NULL,
    actor TEXT,
    comments TEXT,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sh_instance ON step_history(instance_id);

-- ============================================================================
-- PART 7: FINANCIAL BUSINESS OBJECTS
-- ============================================================================

-- Portfolio
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'portfolio', 'Portfolio', 'Portfolio', 'portfolio', 'Investment portfolio', 'briefcase', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Security
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'security', 'Security', 'Security', 'security', 'Financial instruments', 'trending-up', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Position
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'position', 'Position', 'Position', 'position', 'Holdings within accounts', 'layers', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Tax Lot
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'taxlot', 'Tax Lot', 'Tax Lot', 'taxlot', 'Cost basis lots', 'receipt', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Benchmark
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'benchmark', 'Benchmark', 'Benchmark', 'benchmark', 'Performance benchmarks', 'target', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Performance
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'performance', 'Performance', 'Performance', 'performance', 'Portfolio performance', 'chart-line', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Allocation
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'allocation', 'Allocation', 'Asset Allocation', 'allocation', 'Asset allocation analysis', 'pie-chart', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Fee
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'fee', 'Fee', 'Fee', 'fee', 'Advisory fees', 'dollar-sign', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Client
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'client', 'Client', 'Client', 'client', 'Client information', 'user', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Account
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'account', 'Account', 'Account', 'account', 'Investment accounts', 'credit-card', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Transaction
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'transaction', 'Transaction', 'Transaction', 'transaction', 'Financial transactions', 'arrow-right-left', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- ============================================================================
-- PART 8: BUSINESS PROCESSES
-- ============================================================================

-- Client Onboarding
INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, is_system)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'client_onboarding', 'Client Onboarding', 'Client Onboarding', 'End-to-end client onboarding with KYC/AML', 'client_management', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Portfolio Rebalancing
INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, is_system)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'portfolio_rebalance', 'Portfolio Rebalancing', 'Portfolio Rebalancing', 'Systematic portfolio rebalancing', 'portfolio_management', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Performance Reporting
INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, is_system)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'performance_report', 'Performance Reporting', 'Performance Reporting', 'Generate client performance reports', 'reporting', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Fee Billing
INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, is_system)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'fee_billing', 'Fee Billing', 'Fee Billing', 'Calculate and process advisory fees', 'billing', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Trade Execution
INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, is_system)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'trade_execution', 'Trade Execution', 'Trade Execution', 'End-to-end trade lifecycle', 'trading', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Account Transfer
INSERT INTO business_processes (id, tenant_id, key, name, display_name, description, category, is_system)
VALUES (gen_random_uuid(), (SELECT id FROM tenants WHERE code = 'default-tenant' LIMIT 1), 'account_transfer', 'Account Transfer', 'Account Transfer (ACAT)', 'Transfer accounts between custodians', 'account_management', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Workday Schema + Financial BOs + Processes created successfully!';
    RAISE NOTICE '✓ Tables: business_objects, bo_fields, business_processes, process_steps, process_instances, step_history';
    RAISE NOTICE '✓ Business Objects: Portfolio, Security, Position, TaxLot, Benchmark, Performance, Allocation, Fee, Client, Account, Transaction';
    RAISE NOTICE '✓ Processes: Client Onboarding, Portfolio Rebalancing, Performance Reporting, Fee Billing, Trade Execution, Account Transfer';
END $$;
