-- Migration: Database and Storage Tuning Optimizations
-- Created: 2025-09-08
-- Description: Adds optimized indexes, partitioning, and performance improvements

-- ===========================================
-- SCHEMA/INDEXING OPTIMIZATIONS
-- ===========================================

-- Additional indexes for existing tables (tenants, app_user, asset, role, role_member, role_claim)

-- Optimize tenant lookups
CREATE INDEX IF NOT EXISTS idx_tenants_name ON tenants(name);
CREATE INDEX IF NOT EXISTS idx_tenants_created ON tenants(created_at DESC);

-- Optimize user lookups
CREATE INDEX IF NOT EXISTS idx_app_user_email ON app_user(email);
CREATE INDEX IF NOT EXISTS idx_app_user_active ON app_user(is_active) WHERE is_active = true;

-- Optimize user-tenant relationships
CREATE INDEX IF NOT EXISTS idx_user_tenant_user ON user_tenant(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tenant_tenant ON user_tenant(tenant_id);

-- Additional asset indexes
CREATE INDEX IF NOT EXISTS idx_asset_type ON asset(asset_type);
CREATE INDEX IF NOT EXISTS idx_asset_sensitivity ON asset(sensitivity);
CREATE INDEX IF NOT EXISTS idx_asset_created ON asset(created_at DESC);

-- Role optimization indexes
CREATE INDEX IF NOT EXISTS idx_role_tenant_name ON role(tenant_id, name);

-- Role member indexes
CREATE INDEX IF NOT EXISTS idx_role_member_user ON role_member(user_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_role_member_role ON role_member(role_id);

-- Role claim indexes
CREATE INDEX IF NOT EXISTS idx_role_claim_asset ON role_claim(asset_id);
CREATE INDEX IF NOT EXISTS idx_role_claim_permission ON role_claim(permission);

-- ===========================================
-- PERFORMANCE MONITORING TABLES
-- ===========================================

-- Real-time performance metrics
CREATE TABLE IF NOT EXISTS performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(15,6) NOT NULL,
    labels JSONB,
    collected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance metrics
CREATE INDEX IF NOT EXISTS idx_performance_metrics_tenant_collected
ON performance_metrics (tenant_id, collected_at DESC);

CREATE INDEX IF NOT EXISTS idx_performance_metrics_name_collected
ON performance_metrics (metric_name, collected_at DESC);

-- ===========================================
-- CONNECTION POOL OPTIMIZATION TABLES
-- ===========================================

-- Connection pool metrics for monitoring
CREATE TABLE IF NOT EXISTS connection_pool_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_name VARCHAR(100) NOT NULL,
    total_connections INTEGER NOT NULL,
    active_connections INTEGER NOT NULL,
    idle_connections INTEGER NOT NULL,
    waiting_requests INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Prepared statement cache metrics
CREATE TABLE IF NOT EXISTS prepared_statement_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query_hash VARCHAR(64) NOT NULL,
    query_text TEXT NOT NULL,
    execution_count BIGINT DEFAULT 0,
    total_time_ms BIGINT DEFAULT 0,
    avg_time_ms DECIMAL(10,2) DEFAULT 0,
    last_executed TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(query_hash)
);
