-- 20260903_002_rule_health.up.sql
-- Table storing sentinel rule health evaluations and steward alerts.

CREATE TABLE IF NOT EXISTS validation_rule_health (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL,
    rule_id             uuid NOT NULL,
    rule_key            text NOT NULL,
    bo_name             text NOT NULL,
    status              text NOT NULL, -- 'HEALTHY', 'SUSPECT_ALWAYS_FAILS', 'STALE_SCHEMA_DRIFT', 'ENGINE_DEGRADED'
    eval_count          bigint NOT NULL DEFAULT 0,
    violation_count     bigint NOT NULL DEFAULT 0,
    rule_error_count    bigint NOT NULL DEFAULT 0,
    failure_rate        double precision NOT NULL DEFAULT 0.0,
    rule_error_rate     double precision NOT NULL DEFAULT 0.0,
    details             text,
    last_evaluated_at   timestamptz NOT NULL DEFAULT NOW(),
    acknowledged        boolean NOT NULL DEFAULT false,
    acknowledged_by     text,
    acknowledged_at     timestamptz,
    created_at          timestamptz NOT NULL DEFAULT NOW(),
    updated_at          timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vr_health_tenant_rule ON validation_rule_health (tenant_id, rule_id);
CREATE INDEX IF NOT EXISTS idx_vr_health_status ON validation_rule_health (tenant_id, status) WHERE NOT acknowledged;
