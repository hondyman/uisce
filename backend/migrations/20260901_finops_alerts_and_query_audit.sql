-- 20260901_finops_alerts_and_query_audit.sql
-- FinOps Budget Webhooks, Alert History & Non-Leaking Analytical Query Audit Telemetry

CREATE SCHEMA IF NOT EXISTS finops;
CREATE SCHEMA IF NOT EXISTS audit;

-- 1. Tenant Budget Webhook & Alert Channel Configuration (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS finops.budget_alert_configurations (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    slack_webhook_url TEXT,
    email_notification_recipients JSONB DEFAULT '[]'::jsonb, -- ["admin@institution.com"]
    warning_threshold_pct NUMERIC(5, 2) DEFAULT 80.00,       -- 80% Warning
    critical_threshold_pct NUMERIC(5, 2) DEFAULT 95.00,      -- 95% Urgent
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_budget_alert_config UNIQUE (tenant_id)
);

-- 2. Dispatched Budget Alert History
CREATE TABLE IF NOT EXISTS finops.budget_alert_history (
    alert_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    billing_period VARCHAR(7) NOT NULL,                      -- YYYY-MM
    threshold_tier VARCHAR(20) NOT NULL,                     -- WARNING_80, CRITICAL_95, EXCEEDED_100
    spend_percentage NUMERIC(5, 2) NOT NULL,
    current_spend_usd NUMERIC(12, 4) NOT NULL,
    budget_limit_usd NUMERIC(12, 2) NOT NULL,
    channel_type VARCHAR(50) NOT NULL,                       -- SLACK_WEBHOOK, EMAIL
    delivery_status VARCHAR(30) NOT NULL,                    -- SENT, FAILED
    error_message TEXT,
    dispatched_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_budget_alert_history 
ON finops.budget_alert_history (tenant_id, billing_period, threshold_tier);

-- 3. Non-Leaking Analytical Query Audit Log (Zero Client Result Data Retention)
CREATE TABLE IF NOT EXISTS audit.analytical_query_execution_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    request_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    server_host VARCHAR(100) NOT NULL,                       -- e.g. "100.84.50.65" or "macbook-local"
    execution_type VARCHAR(50) NOT NULL,                     -- SEMANTIC_QUERY, SSRS_REPORT_RENDER, VECTOR_CALC, API_EXPORT
    query_fingerprint VARCHAR(64) NOT NULL,                  -- SHA-256 of normalized query AST
    normalized_query_text TEXT NOT NULL,                     -- SQL / DSL template with redacted constants
    execution_plan_json JSONB,                               -- Stored explain-plan DAG JSON
    row_count_returned BIGINT NOT NULL DEFAULT 0,            -- Result metric ONLY (no row data stored)
    scanned_bytes BIGINT NOT NULL DEFAULT 0,
    cpu_duration_ms INT NOT NULL DEFAULT 0,
    total_latency_ms INT NOT NULL DEFAULT 0,
    engine_type VARCHAR(50) NOT NULL,                        -- STARROCKS, POSTGRES_ALPHA, ICEBERG, WASM_VECTOR
    attributed_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    status VARCHAR(30) NOT NULL,                             -- COMPLETED, FAILED, BLOCKED_CIRCUIT_BREAKER
    error_summary TEXT,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analytical_audit_tenant_time 
ON audit.analytical_query_execution_logs (tenant_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_analytical_audit_fingerprint 
ON audit.analytical_query_execution_logs (tenant_id, query_fingerprint);
