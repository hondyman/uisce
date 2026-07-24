-- Unified Observability + SLO Engine Schema

-- Metrics storage (time-series-like)
CREATE TABLE IF NOT EXISTS obs_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    value FLOAT NOT NULL,
    labels JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient metric queries
CREATE INDEX IF NOT EXISTS idx_obs_metrics_tenant_name_ts ON obs_metrics(tenant_id, name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_obs_metrics_name_ts ON obs_metrics(name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_obs_metrics_timestamp ON obs_metrics(timestamp DESC);

-- Create hypertable if TimescaleDB is available (optional optimization)
-- SELECT create_hypertable('obs_metrics', 'timestamp', if_not_exists => TRUE);

-- SLO definitions
CREATE TABLE IF NOT EXISTS obs_slos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    target FLOAT NOT NULL CHECK (target >= 0 AND target <= 100),
    time_window TEXT NOT NULL, -- e.g., "7d", "30d" (renamed from 'window' to avoid reserved keyword)
    metric_query TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_obs_slos_tenant ON obs_slos(tenant_id);
CREATE INDEX IF NOT EXISTS idx_obs_slos_name ON obs_slos(tenant_id, name);

-- Alert rules associated with SLOs
CREATE TABLE IF NOT EXISTS obs_alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    slo_id UUID REFERENCES obs_slos(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    condition TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    channels JSONB DEFAULT '[]',
    enabled BOOLEAN DEFAULT TRUE,
    cooldown_minutes INT DEFAULT 60, -- Minimum time between alerts
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_obs_alert_rules_slo ON obs_alert_rules(slo_id);
CREATE INDEX IF NOT EXISTS idx_obs_alert_rules_tenant ON obs_alert_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_obs_alert_rules_enabled ON obs_alert_rules(enabled);

-- Alert instances (fired alerts)
CREATE TABLE IF NOT EXISTS obs_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_id UUID REFERENCES obs_alert_rules(id) ON DELETE SET NULL,
    slo_id UUID REFERENCES obs_slos(id) ON DELETE SET NULL,
    severity TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    message TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('firing', 'resolved', 'acknowledged')),
    value FLOAT,
    threshold FLOAT,
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by TEXT,
    resolved_at TIMESTAMPTZ,
    resolution_note TEXT
);

CREATE INDEX IF NOT EXISTS idx_obs_alerts_tenant_status ON obs_alerts(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_obs_alerts_fired ON obs_alerts(fired_at DESC);
CREATE INDEX IF NOT EXISTS idx_obs_alerts_slo ON obs_alerts(slo_id);
CREATE INDEX IF NOT EXISTS idx_obs_alerts_rule ON obs_alerts(rule_id);

-- SLO status snapshots (for historical tracking)
CREATE TABLE IF NOT EXISTS obs_slo_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slo_id UUID REFERENCES obs_slos(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    current_value FLOAT NOT NULL,
    budget_consumed FLOAT NOT NULL,
    budget_remaining FLOAT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('healthy', 'degraded', 'breached')),
    snapshot_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_obs_slo_snapshots_slo ON obs_slo_snapshots(slo_id, snapshot_at DESC);
CREATE INDEX IF NOT EXISTS idx_obs_slo_snapshots_tenant ON obs_slo_snapshots(tenant_id, snapshot_at DESC);

-- System events log
CREATE TABLE IF NOT EXISTS obs_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL, -- alert_fired, slo_breached, deployment, config_change, etc.
    severity TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_obs_events_tenant_ts ON obs_events(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_obs_events_type ON obs_events(event_type, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_obs_events_severity ON obs_events(severity, timestamp DESC);

-- Notification channels configuration
CREATE TABLE IF NOT EXISTS obs_notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    channel_type TEXT NOT NULL CHECK (channel_type IN ('slack', 'email', 'webhook', 'pagerduty')),
    config JSONB NOT NULL, -- Channel-specific configuration
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_obs_channels_tenant ON obs_notification_channels(tenant_id);
CREATE INDEX IF NOT EXISTS idx_obs_channels_type ON obs_notification_channels(channel_type);

-- Dashboard configurations
CREATE TABLE IF NOT EXISTS obs_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    config JSONB NOT NULL, -- Layout, widgets, queries
    is_default BOOLEAN DEFAULT FALSE,
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_obs_dashboards_tenant ON obs_dashboards(tenant_id);

-- Metric aggregation summaries (for faster queries on high-cardinality data)
CREATE TABLE IF NOT EXISTS obs_metric_aggregates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    granularity TEXT NOT NULL CHECK (granularity IN ('1m', '5m', '1h', '1d')),
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    count BIGINT NOT NULL,
    sum FLOAT NOT NULL,
    min FLOAT NOT NULL,
    max FLOAT NOT NULL,
    avg FLOAT NOT NULL,
    UNIQUE(tenant_id, name, granularity, window_start)
);

CREATE INDEX IF NOT EXISTS idx_obs_aggregates_query ON obs_metric_aggregates(tenant_id, name, granularity, window_start DESC);

-- Cleanup policy (scheduled job)
-- DELETE FROM obs_metrics WHERE timestamp < NOW() - INTERVAL '30 days';
-- DELETE FROM obs_events WHERE timestamp < NOW() - INTERVAL '90 days';
-- DELETE FROM obs_slo_snapshots WHERE snapshot_at < NOW() - INTERVAL '90 days';
-- DELETE FROM obs_alerts WHERE fired_at < NOW() - INTERVAL '90 days';
