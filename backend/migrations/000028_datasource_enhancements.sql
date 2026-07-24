-- Datasource Enhancement Migration v3 (Standalone)
-- Creates base tables if they don't exist, then adds enhancements

-- ============================================================================
-- PART 0: Create tenants table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- PART 0.1: Create tenant_instance table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenant_instance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    url TEXT,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT tenant_instance_unique UNIQUE (tenant_id, instance_name)
);

-- ============================================================================
-- PART 0.2: Create alpha_product table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS alpha_product (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_name TEXT UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- PART 0.3: Create alpha_datasource table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS alpha_datasource (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datasource_code TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- PART 0.4: Create tenant_product table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenant_product (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datasource_id UUID NOT NULL REFERENCES tenant_instance(id) ON DELETE CASCADE,
    alpha_product_id UUID NOT NULL REFERENCES alpha_product(id) ON DELETE CASCADE,
    version REAL NOT NULL DEFAULT 1.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT tenant_product_uniq UNIQUE (datasource_id, alpha_product_id)
);

-- ============================================================================
-- PART 0.5: Create tenant_product_datasource table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenant_product_datasource (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_product_id UUID NOT NULL REFERENCES tenant_product(id) ON DELETE CASCADE,
    alpha_datasource_id UUID NOT NULL REFERENCES alpha_datasource(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',
    source_name TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT tenant_product_datasource_source_uniq UNIQUE (tenant_product_id, source_name)
);

-- ============================================================================
-- PART 1: Enhance existing tenant_product_datasource table
-- ============================================================================

-- Environment and classification
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS environment TEXT DEFAULT 'development';
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS read_only BOOLEAN DEFAULT false;

-- Connection pool settings (enterprise-grade)
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS pool_config JSONB DEFAULT '{
  "max_connections": 10,
  "min_connections": 2,
  "connection_timeout_ms": 30000,
  "idle_timeout_ms": 600000,
  "max_lifetime_ms": 3600000,
  "ssl_mode": "require",
  "statement_cache_size": 100
}';

-- Scheduled scanning configuration
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS scan_schedule JSONB DEFAULT '{
  "enabled": false,
  "cron": null,
  "timezone": "UTC",
  "notify_on_complete": true,
  "notify_on_failure": true
}';

-- Health monitoring (observability)
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS health_config JSONB DEFAULT '{
  "heartbeat_enabled": false,
  "heartbeat_interval_seconds": 60,
  "alert_threshold_minutes": 5,
  "auto_reconnect": true,
  "max_reconnect_attempts": 3,
  "escalation_policy": "default"
}';
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS last_heartbeat_at TIMESTAMPTZ;
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS health_status TEXT DEFAULT 'unknown';
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS health_message TEXT;

-- Data integrity controls
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS integrity_checks JSONB DEFAULT '{
  "row_count_validation": false,
  "schema_drift_detection": true,
  "checksum_verification": false,
  "referential_integrity": true,
  "null_check_columns": [],
  "tolerance_percent": 1.0
}';
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS last_integrity_check_at TIMESTAMPTZ;
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS integrity_status TEXT DEFAULT 'unknown';
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS integrity_message TEXT;

-- Connection reference (for unified connection management)
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS connection_id UUID;

-- SLA and compliance
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS sla_config JSONB DEFAULT '{
  "tier": "standard",
  "max_query_time_ms": 30000,
  "max_rows_per_query": 100000,
  "rate_limit": 100,
  "rate_limit_window_seconds": 60
}';

-- Data classification (for GDPR, CCPA, etc.)
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS data_classification JSONB DEFAULT '{
  "sensitivity_level": "internal",
  "contains_pii": false,
  "contains_financial": false,
  "retention_days": null,
  "encryption_required": false,
  "audit_enabled": true
}';

-- Audit fields
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS created_by TEXT;
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS updated_by TEXT;

-- Scan status
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS last_scan_at TIMESTAMPTZ;
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS last_scan_status TEXT DEFAULT 'pending';
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS scan_error_message TEXT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tpd_environment ON tenant_product_datasource(environment);
CREATE INDEX IF NOT EXISTS idx_tpd_health_status ON tenant_product_datasource(health_status);
CREATE INDEX IF NOT EXISTS idx_tpd_integrity_status ON tenant_product_datasource(integrity_status);
CREATE INDEX IF NOT EXISTS idx_tpd_tags ON tenant_product_datasource USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_tpd_active_env ON tenant_product_datasource(is_active, environment);

-- ============================================================================
-- PART 2: Integrity check results table
-- ============================================================================

CREATE TABLE IF NOT EXISTS datasource_integrity_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    check_type TEXT NOT NULL,
    status TEXT NOT NULL,
    
    -- Row count validation results
    postgres_row_count BIGINT,
    ignite_row_count BIGINT,
    starrocks_row_count BIGINT,
    lakehouse_row_count BIGINT,
    row_count_delta BIGINT,
    row_count_delta_percent NUMERIC(10, 4),
    
    -- Schema drift results
    schema_changes JSONB,
    baseline_snapshot_id UUID,
    
    -- Checksum results
    checksum_valid BOOLEAN,
    checksum_details JSONB,
    
    -- Referential integrity results
    orphan_records_count BIGINT,
    referential_violations JSONB,
    
    -- Metadata
    executed_by TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    error_message TEXT,
    recommendations JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dic_datasource ON datasource_integrity_checks(datasource_id);
CREATE INDEX IF NOT EXISTS idx_dic_status ON datasource_integrity_checks(status);
CREATE INDEX IF NOT EXISTS idx_dic_created ON datasource_integrity_checks(created_at DESC);

-- ============================================================================
-- PART 3: Schema snapshot table
-- ============================================================================

CREATE TABLE IF NOT EXISTS datasource_schema_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    snapshot_data JSONB NOT NULL,
    table_count INTEGER,
    column_count INTEGER,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    captured_by TEXT,
    is_baseline BOOLEAN DEFAULT false,
    notes TEXT,
    previous_snapshot_id UUID,
    change_summary JSONB
);

CREATE INDEX IF NOT EXISTS idx_dss_datasource ON datasource_schema_snapshots(datasource_id);
CREATE INDEX IF NOT EXISTS idx_dss_baseline ON datasource_schema_snapshots(datasource_id, is_baseline) WHERE is_baseline = true;

-- ============================================================================
-- PART 4: Connections table
-- ============================================================================

CREATE TABLE IF NOT EXISTS connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database TEXT,
    schema TEXT,
    username TEXT,
    password TEXT,
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_tested_at TIMESTAMPTZ,
    last_test_status TEXT,
    last_test_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    updated_by TEXT,
    CONSTRAINT connections_unique_name UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_connections_tenant ON connections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_connections_type ON connections(type);

-- ============================================================================
-- PART 5: Health check history table
-- ============================================================================

CREATE TABLE IF NOT EXISTS datasource_health_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    response_time_ms INTEGER,
    error_message TEXT,
    connection_pool_size INTEGER,
    active_connections INTEGER,
    idle_connections INTEGER,
    diagnostics JSONB,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dhc_datasource ON datasource_health_checks(datasource_id);
CREATE INDEX IF NOT EXISTS idx_dhc_checked ON datasource_health_checks(checked_at DESC);

-- ============================================================================
-- PART 6: Data classification templates
-- ============================================================================

CREATE TABLE IF NOT EXISTS data_classification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO data_classification_templates (name, display_name, description, config, is_system)
VALUES 
    ('public', 'Public', 'Non-sensitive data', '{"sensitivity_level": "public", "contains_pii": false}', true),
    ('internal', 'Internal', 'Internal business data', '{"sensitivity_level": "internal", "audit_enabled": true}', true),
    ('confidential', 'Confidential', 'Sensitive business data', '{"sensitivity_level": "confidential", "encryption_required": true}', true),
    ('restricted_pii', 'Restricted (PII)', 'Contains PII', '{"sensitivity_level": "restricted", "contains_pii": true, "gdpr_applicable": true}', true),
    ('restricted_financial', 'Restricted (Financial)', 'Financial data', '{"sensitivity_level": "restricted", "contains_financial": true, "sox_applicable": true}', true)
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- PART 7: Triggers
-- ============================================================================

CREATE OR REPLACE FUNCTION update_datasource_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_datasource_updated ON tenant_product_datasource;
CREATE TRIGGER trg_datasource_updated
    BEFORE UPDATE ON tenant_product_datasource
    FOR EACH ROW
    EXECUTE FUNCTION update_datasource_timestamp();

-- ============================================================================
-- PART 8: Stored procedures
-- ============================================================================

CREATE OR REPLACE FUNCTION clone_datasource(
    p_source_id UUID,
    p_new_name TEXT,
    p_target_environment TEXT DEFAULT 'development'
)
RETURNS UUID AS $$
DECLARE
    v_new_id UUID;
BEGIN
    INSERT INTO tenant_product_datasource (
        tenant_product_id, alpha_datasource_id, source_name, environment,
        tags, description, read_only, pool_config, scan_schedule,
        health_config, integrity_checks, sla_config, data_classification, is_active
    )
    SELECT 
        tenant_product_id, alpha_datasource_id, p_new_name, p_target_environment,
        tags, description, read_only, pool_config, 
        jsonb_set(COALESCE(scan_schedule, '{}'::jsonb), '{enabled}', 'false'),
        health_config, integrity_checks, sla_config, data_classification, true
    FROM tenant_product_datasource
    WHERE id = p_source_id
    RETURNING id INTO v_new_id;
    
    RETURN v_new_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_tenant_datasource_health_summary(p_tenant_product_id UUID)
RETURNS TABLE (
    total_datasources BIGINT,
    healthy_count BIGINT,
    degraded_count BIGINT,
    unhealthy_count BIGINT,
    unknown_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::BIGINT,
        COUNT(*) FILTER (WHERE health_status = 'healthy')::BIGINT,
        COUNT(*) FILTER (WHERE health_status = 'degraded')::BIGINT,
        COUNT(*) FILTER (WHERE health_status = 'unhealthy')::BIGINT,
        COUNT(*) FILTER (WHERE health_status = 'unknown')::BIGINT
    FROM tenant_product_datasource
    WHERE tenant_product_id = p_tenant_product_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- DONE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Datasource enhancement migration completed successfully!';
    RAISE NOTICE '✓ Created/updated tables: tenant_product_datasource, connections, datasource_integrity_checks, datasource_schema_snapshots, datasource_health_checks, data_classification_templates';
    RAISE NOTICE '✓ Added features: environment tagging, health monitoring, integrity checks, SLA config, data classification, clone procedure';
END $$;
