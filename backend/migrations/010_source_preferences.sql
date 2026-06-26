-- backend/migrations/010_source_preferences.sql
-- Phase 6: Portfolio Manager Source Preference Management

-- Source Preferences: The canonical record of preferred data sources per BO/term/region
CREATE TABLE IF NOT EXISTS edm.source_preferences (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    business_object   VARCHAR(100) NOT NULL,
    semantic_term     VARCHAR(100) NOT NULL,
    region            VARCHAR(100) NOT NULL DEFAULT 'GLOBAL',
    priority          INT NOT NULL DEFAULT 1, -- 1=first, 2=second, 3=third
    source_system     VARCHAR(100) NOT NULL,
    confidence        INT NOT NULL DEFAULT 80 CHECK (confidence BETWEEN 0 AND 100),
    status            VARCHAR(20) NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft','testing','staging','production')),
    version           INT NOT NULL DEFAULT 1,
    core_id           UUID REFERENCES edm.source_preferences(id), -- null = core, non-null = tenant override
    override_reason   TEXT,
    valid_from        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to          TIMESTAMPTZ,
    impact_analysis   JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        UUID NOT NULL,
    updated_by        UUID
);

CREATE INDEX IF NOT EXISTS idx_source_prefs_tenant  ON edm.source_preferences(tenant_id);
CREATE INDEX IF NOT EXISTS idx_source_prefs_status  ON edm.source_preferences(status);
CREATE INDEX IF NOT EXISTS idx_source_prefs_core    ON edm.source_preferences(core_id);
CREATE INDEX IF NOT EXISTS idx_source_prefs_bo      ON edm.source_preferences(business_object, semantic_term, region);

ALTER TABLE edm.source_preferences ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS source_preferences_tenant_isolation ON edm.source_preferences;
CREATE POLICY source_preferences_tenant_isolation ON edm.source_preferences
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Preference Versions: Immutable audit trail of every state transition
CREATE TABLE IF NOT EXISTS edm.preference_versions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preference_id UUID NOT NULL REFERENCES edm.source_preferences(id),
    version       INT NOT NULL,
    status        VARCHAR(20) NOT NULL CHECK (status IN ('draft','testing','staging','production')),
    reason        TEXT,
    impact_analysis JSONB NOT NULL DEFAULT '{}',
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pref_versions_pref ON edm.preference_versions(preference_id);

ALTER TABLE edm.preference_versions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS preference_versions_tenant_isolation ON edm.preference_versions;
CREATE POLICY preference_versions_tenant_isolation ON edm.preference_versions
    FOR ALL USING (
        preference_id IN (
            SELECT id FROM edm.source_preferences
            WHERE tenant_id::text = current_setting('app.current_tenant_id', true)
        )
    );

-- Source Analytics: Pre-aggregated selection statistics
CREATE TABLE IF NOT EXISTS edm.source_analytics (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    business_object          VARCHAR(100),
    semantic_term            VARCHAR(100),
    region                   VARCHAR(100),
    source_system            VARCHAR(100) NOT NULL,
    first_preference_count   INT NOT NULL DEFAULT 0,
    second_preference_count  INT NOT NULL DEFAULT 0,
    third_preference_count   INT NOT NULL DEFAULT 0,
    other_preference_count   INT NOT NULL DEFAULT 0,
    total_selections         INT NOT NULL DEFAULT 0,
    avg_confidence           FLOAT NOT NULL DEFAULT 0,
    time_window              VARCHAR(50) NOT NULL DEFAULT 'last_30_days',
    as_of_date               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confidence_trends        JSONB NOT NULL DEFAULT '{}',
    exception_stats          JSONB NOT NULL DEFAULT '{}',
    generated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_source_analytics_tenant  ON edm.source_analytics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_source_analytics_time    ON edm.source_analytics(generated_at);

ALTER TABLE edm.source_analytics ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS source_analytics_tenant_isolation ON edm.source_analytics;
CREATE POLICY source_analytics_tenant_isolation ON edm.source_analytics
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Source Exceptions: Conflict and quality exception records
CREATE TABLE IF NOT EXISTS edm.source_exceptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    business_object VARCHAR(100) NOT NULL,
    semantic_term   VARCHAR(100),
    region          VARCHAR(100),
    source_system   VARCHAR(100) NOT NULL,
    exception_type  VARCHAR(50) NOT NULL, -- SOURCE_CONFLICT, DATA_QUALITY, SYSTEM_ERROR, COMPLIANCE_VIOLATION
    description     TEXT NOT NULL,
    impact_level    INT NOT NULL DEFAULT 1 CHECK (impact_level BETWEEN 1 AND 5),
    critical_path   BOOLEAN NOT NULL DEFAULT false,
    status          VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open','in_progress','resolved')),
    metadata        JSONB NOT NULL DEFAULT '{}',
    resolution_history JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    resolved_by     UUID
);

CREATE INDEX IF NOT EXISTS idx_source_exceptions_tenant  ON edm.source_exceptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_source_exceptions_status  ON edm.source_exceptions(status);
CREATE INDEX IF NOT EXISTS idx_source_exceptions_type    ON edm.source_exceptions(exception_type);

ALTER TABLE edm.source_exceptions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS source_exceptions_tenant_isolation ON edm.source_exceptions;
CREATE POLICY source_exceptions_tenant_isolation ON edm.source_exceptions
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Exception Routes: Routing rules for intelligent exception handling
CREATE TABLE IF NOT EXISTS edm.exception_routes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    business_object VARCHAR(100),
    route_name      VARCHAR(100) NOT NULL,
    exception_type  VARCHAR(50) NOT NULL,
    priority        INT NOT NULL DEFAULT 3,
    escalation_level INT NOT NULL DEFAULT 1,
    channel         VARCHAR(50) NOT NULL DEFAULT 'default',
    conditions      JSONB NOT NULL DEFAULT '{}',
    resolution_steps JSONB NOT NULL DEFAULT '[]',
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_exception_routes_bo ON edm.exception_routes(business_object, exception_type);

ALTER TABLE edm.exception_routes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS exception_routes_tenant_isolation ON edm.exception_routes;
CREATE POLICY exception_routes_tenant_isolation ON edm.exception_routes
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Exception History: Append-only status change log
CREATE TABLE IF NOT EXISTS edm.exception_history (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exception_id UUID NOT NULL REFERENCES edm.source_exceptions(id),
    status       VARCHAR(50) NOT NULL,
    description  TEXT,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   UUID
);

CREATE INDEX IF NOT EXISTS idx_exception_history_exc ON edm.exception_history(exception_id);

ALTER TABLE edm.exception_history ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS exception_history_tenant_isolation ON edm.exception_history;
CREATE POLICY exception_history_tenant_isolation ON edm.exception_history
    FOR ALL USING (
        exception_id IN (
            SELECT id FROM edm.source_exceptions
            WHERE tenant_id::text = current_setting('app.current_tenant_id', true)
        )
    );
