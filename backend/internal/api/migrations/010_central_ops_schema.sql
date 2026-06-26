-- Central Ops Schema (Postgres)
-- This schema manages operational data across all tenants with strict isolation via RLS.

BEGIN;

-- 1. Tenants Catalog
CREATE TABLE tenants (
    tenant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status TEXT CHECK (status IN ('active', 'inactive', 'suspended')) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Exceptions Table (Partitioned by Tenant/Time in production, simplified here)
CREATE TABLE exceptions (
    exception_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    source_system TEXT NOT NULL,
    status TEXT CHECK (status IN ('open', 'investigating', 'resolved', 'escalated')) DEFAULT 'open',
    severity TEXT CHECK (severity IN ('low', 'medium', 'high', 'critical')) DEFAULT 'medium',
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    assigned_to TEXT -- Ops User ID
);

-- Index for SLA Dashboards
CREATE INDEX idx_exceptions_tenant_status_created ON exceptions (tenant_id, status, created_at);

-- 3. Workflows Table (SLA Tracking)
CREATE TABLE workflows (
    workflow_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    process_name TEXT NOT NULL,
    state TEXT NOT NULL,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_workflows_tenant_process ON workflows (tenant_id, process_name);

-- 4. Unified Audit Records (UAR) - Central Log
CREATE TABLE audit_records (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(tenant_id), -- Nullable for system-level events
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    hash_chain TEXT -- For tamper-evidence
);

-- 5. Row-Level Security (RLS) Policies

-- Enable RLS
ALTER TABLE exceptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_records ENABLE ROW LEVEL SECURITY;

-- Policy: Tenant Isolation
-- Users can only see rows where tenant_id matches their session variable 'app.tenant_id'
-- Ops Admins (with role 'ops_admin') can bypass this via a separate policy or superuser status.

CREATE POLICY tenant_isolation_exceptions ON exceptions
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_workflows ON workflows
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_audit ON audit_records
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- Seed Demo Tenants
INSERT INTO tenants (tenant_id, name) VALUES 
    ('11111111-1111-1111-1111-111111111111', 'Acme Capital'),
    ('22222222-2222-2222-2222-222222222222', 'Beta Investments');

COMMIT;
