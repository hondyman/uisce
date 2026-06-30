-- 20260208_create_ops_alerts.up.sql
-- Global operations alerting system for thresholds and anomaly detection

CREATE TABLE IF NOT EXISTS ops_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    scope TEXT NOT NULL, -- 'global', 'tenant', 'endpoint'
    metric TEXT NOT NULL, -- 'error_rate', 'latency_p95', 'requests'
    threshold DOUBLE PRECISION NOT NULL,
    comparison TEXT NOT NULL, -- '>' or '<'
    window_secs INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ops_alerts_scope_metric ON ops_alerts(scope, metric);
CREATE INDEX IF NOT EXISTS idx_ops_alerts_enabled ON ops_alerts(enabled);

CREATE TABLE IF NOT EXISTS ops_alert_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL REFERENCES ops_alerts(id) ON DELETE CASCADE,
    scope_id UUID, -- tenant_id for tenant scope, null for global
    endpoint TEXT, -- path for endpoint alerts
    value DOUBLE PRECISION NOT NULL,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ops_alert_events_alert_id ON ops_alert_events(alert_id);
CREATE INDEX IF NOT EXISTS idx_ops_alert_events_scope_id ON ops_alert_events(scope_id);
CREATE INDEX IF NOT EXISTS idx_ops_alert_events_triggered_at ON ops_alert_events(triggered_at DESC);

-- Seed default global alerts
INSERT INTO ops_alerts (name, scope, metric, threshold, comparison, window_secs, enabled)
VALUES
    ('High Error Rate', 'global', 'error_rate', 0.01, '>', 300, true),
    ('High Latency p95', 'global', 'latency_p95', 500, '>', 300, true),
    ('Traffic Spike', 'global', 'requests', 100000, '>', 300, true),
    ('Traffic Drop', 'global', 'requests', 1000, '<', 300, true)
ON CONFLICT DO NOTHING;
