-- ops_events table: unified event log for incident timeline
CREATE TABLE IF NOT EXISTS ops_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id     UUID,                -- nullable: grouped incident
    event_type      TEXT NOT NULL,       -- "alert" | "fingerprint" | "tenant_health" | "endpoint_health" | "latency_anomaly" | ...
    scope           TEXT NOT NULL,       -- "global" | "tenant" | "endpoint" | "region"
    tenant_id       UUID,
    endpoint_path   TEXT,
    region          TEXT,
    fingerprint_id  UUID,
    alert_id        UUID,
    severity        TEXT NOT NULL,       -- "info" | "warning" | "error" | "critical"
    title           TEXT NOT NULL,
    details         JSONB NOT NULL DEFAULT '{}'::jsonb,  -- typed payload per event_type
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_ops_events_incident FOREIGN KEY (incident_id) REFERENCES ops_incidents(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_ops_events_occurred_at ON ops_events(occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_events_incident_id ON ops_events(incident_id);
CREATE INDEX IF NOT EXISTS idx_ops_events_tenant_id ON ops_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ops_events_endpoint_path ON ops_events(endpoint_path);
CREATE INDEX IF NOT EXISTS idx_ops_events_event_type ON ops_events(event_type);
CREATE INDEX IF NOT EXISTS idx_ops_events_severity ON ops_events(severity);

-- ops_incidents table: incident grouping and correlation
CREATE TABLE IF NOT EXISTS ops_incidents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status          TEXT NOT NULL DEFAULT 'open',  -- "open" | "closed"
    severity        TEXT NOT NULL,       -- "info" | "warning" | "error" | "critical"
    title           TEXT NOT NULL,
    summary         TEXT,
    root_cause      TEXT,
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ops_incidents_status ON ops_incidents(status);
CREATE INDEX IF NOT EXISTS idx_ops_incidents_started_at ON ops_incidents(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_incidents_severity ON ops_incidents(severity);
CREATE INDEX IF NOT EXISTS idx_ops_incidents_open_unresolved ON ops_incidents(status, started_at DESC) WHERE status = 'open';
