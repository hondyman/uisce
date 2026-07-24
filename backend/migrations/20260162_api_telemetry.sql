-- Migration: 20260162_api_telemetry.sql
-- Goal: Store API usage metrics for analytics and ASO

CREATE TABLE IF NOT EXISTS api_telemetry (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    api_id uuid NOT NULL,
    env text NOT NULL,
    tenant_id uuid,
    client_type text, -- page, external, batch
    status_code int,
    latency_ms int,
    error_message text,
    requested_at timestamp with time zone DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_telemetry_api_id ON api_telemetry(api_id);
CREATE INDEX IF NOT EXISTS idx_api_telemetry_requested_at ON api_telemetry(requested_at);
