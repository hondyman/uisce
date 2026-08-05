-- Migration: 003_secrets_and_audit.sql
-- Secrets management and audit tables with RLS

-- ============================================================================
-- Secrets Metadata Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS secret_metadata (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    secret_type TEXT DEFAULT 'kv-v2',
    description TEXT,
    ttl INTERVAL,
    max_versions INT DEFAULT 10,
    tags TEXT[],
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, path)
);

CREATE INDEX IF NOT EXISTS idx_secret_metadata_tenant ON secret_metadata(tenant_id);
CREATE INDEX IF NOT EXISTS idx_secret_metadata_path ON secret_metadata(path);
CREATE INDEX IF NOT EXISTS idx_secret_metadata_tags ON secret_metadata USING GIN(tags);

-- Audit trigger
CREATE OR REPLACE FUNCTION audit_secret_metadata()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_audit_secret_metadata') THEN
        CREATE TRIGGER trigger_audit_secret_metadata
            BEFORE UPDATE ON secret_metadata
            FOR EACH ROW EXECUTE FUNCTION audit_secret_metadata();
    END IF;
END$$;

-- ============================================================================
-- Secret Policy (link secrets to ABAC policies)
-- ============================================================================
CREATE TABLE IF NOT EXISTS secret_policy (
    secret_id UUID REFERENCES secret_metadata(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (secret_id, policy_id)
);

CREATE INDEX IF NOT EXISTS idx_secret_policy_secret ON secret_policy(secret_id);
CREATE INDEX IF NOT EXISTS idx_secret_policy_policy ON secret_policy(policy_id);

-- ============================================================================
-- Secret Version (rotation tracking)
-- ============================================================================
CREATE TABLE IF NOT EXISTS secret_version (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    secret_id UUID REFERENCES secret_metadata(id) ON DELETE CASCADE,
    version INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    created_by UUID,
    destroy_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_secret_version_secret ON secret_version(secret_id, version DESC);

-- ============================================================================
-- Secret Access Log (AI anomaly analysis)
-- ============================================================================
CREATE TABLE IF NOT EXISTS secret_access_log (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    secret_id UUID REFERENCES secret_metadata(id),
    user_id UUID,
    action TEXT NOT NULL,
    ip_address INET,
    user_agent TEXT,
    geolocation JSONB,
    requested_at TIMESTAMPTZ DEFAULT now(),
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    abac_result JSONB
);

CREATE INDEX IF NOT EXISTS idx_secret_access_log_time ON secret_access_log(requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_secret_access_log_user ON secret_access_log(user_id);
CREATE INDEX IF NOT EXISTS idx_secret_access_log_secret ON secret_access_log(secret_id);

-- ============================================================================
-- Secrets View for Rotation
-- ============================================================================
CREATE OR REPLACE VIEW secrets_needing_rotation AS
SELECT *
FROM secret_metadata
WHERE ttl IS NOT NULL
  AND (updated_at + ttl) < now()
  AND deleted_at IS NULL;

-- ============================================================================
-- Audit Logs Table (general)
-- ============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    action TEXT NOT NULL,
    resource TEXT,
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    allowed BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_time ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource, resource_id);

-- ============================================================================
-- Risk Events Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id UUID,
    event_type TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'medium',
    description TEXT,
    metadata JSONB DEFAULT '{}',
    resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_events_tenant ON risk_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_portfolio ON risk_events(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_severity ON risk_events(severity);
CREATE INDEX IF NOT EXISTS idx_risk_events_resolved ON risk_events(resolved);
CREATE INDEX IF NOT EXISTS idx_risk_events_time ON risk_events(created_at DESC);

-- Audit trigger
CREATE OR REPLACE FUNCTION audit_risk_events()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_audit_risk_events') THEN
        CREATE TRIGGER trigger_audit_risk_events
            BEFORE UPDATE ON risk_events
            FOR EACH ROW EXECUTE FUNCTION audit_risk_events();
    END IF;
END$$;

-- ============================================================================
-- Risk Mitigation Actions Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS risk_mitigation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_event_id UUID NOT NULL REFERENCES risk_events(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    assigned_to UUID,
    completed_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_mitigation_event ON risk_mitigation_actions(risk_event_id);
CREATE INDEX IF NOT EXISTS idx_risk_mitigation_status ON risk_mitigation_actions(status);

-- Audit trigger
CREATE OR REPLACE FUNCTION audit_risk_mitigation_actions()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_audit_risk_mitigation_actions') THEN
        CREATE TRIGGER trigger_audit_risk_mitigation_actions
            BEFORE UPDATE ON risk_mitigation_actions
            FOR EACH ROW EXECUTE FUNCTION audit_risk_mitigation_actions();
    END IF;
END$$;

-- ============================================================================
-- Enable Row Level Security on audit/risks tables
-- ============================================================================
ALTER TABLE secret_metadata ENABLE ROW LEVEL SECURITY;
ALTER TABLE secret_policy ENABLE ROW LEVEL SECURITY;
ALTER TABLE secret_version ENABLE ROW LEVEL SECURITY;
ALTER TABLE secret_access_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE risk_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE risk_mitigation_actions ENABLE ROW LEVEL SECURITY;

-- RLS Policies for secret_metadata
CREATE POLICY tenant_isolation_secret_metadata ON secret_metadata
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', true)::UUID);

-- RLS Policies for secret_policy (via secret_metadata)
CREATE POLICY tenant_isolation_secret_policy ON secret_policy
    FOR ALL USING (
        secret_id IN (
            SELECT id FROM secret_metadata WHERE tenant_id = current_setting('app.current_tenant', true)::UUID
        )
    );

-- RLS Policies for secret_version (via secret_metadata)
CREATE POLICY tenant_isolation_secret_version ON secret_version
    FOR ALL USING (
        secret_id IN (
            SELECT id FROM secret_metadata WHERE tenant_id = current_setting('app.current_tenant', true)::UUID
        )
    );

-- RLS Policies for secret_access_log (via secret_metadata)
CREATE POLICY tenant_isolation_secret_access_log ON secret_access_log
    FOR ALL USING (
        secret_id IN (
            SELECT id FROM secret_metadata WHERE tenant_id = current_setting('app.current_tenant', true)::UUID
        )
    );

-- RLS Policies for audit_logs
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', true)::UUID);

-- RLS Policies for risk_events
CREATE POLICY tenant_isolation_risk_events ON risk_events
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', true)::UUID);

-- RLS Policies for risk_mitigation_actions (via risk_events)
CREATE POLICY tenant_isolation_risk_mitigation_actions ON risk_mitigation_actions
    FOR ALL USING (
        risk_event_id IN (
            SELECT id FROM risk_events WHERE tenant_id = current_setting('app.current_tenant', true)::UUID
        )
    );

COMMENT ON TABLE secret_metadata IS 'Core secrets metadata for low/no-code UI - stores path, type, and config for Vault/AWS/Azure';
COMMENT ON TABLE secret_access_log IS 'Audit trail for AI-powered anomaly detection in secrets access patterns';
