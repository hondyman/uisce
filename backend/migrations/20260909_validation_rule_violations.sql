-- Persisted validation-rule violations, so "did this rule ever fire" is
-- queryable rather than living only in server logs. See
-- docs/unified-rule-engine-handoff.md, "violations visible" item.

CREATE TABLE IF NOT EXISTS validation_rule_violations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    rule_id      UUID NOT NULL,
    rule_name    TEXT NOT NULL,
    bo_key       TEXT NOT NULL,
    severity     TEXT NOT NULL,
    record_id    TEXT,
    message      TEXT NOT NULL,
    context      JSONB NOT NULL DEFAULT '{}'::jsonb,
    write_blocked BOOLEAN NOT NULL DEFAULT false,
    rule_error   BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE validation_rule_violations ADD COLUMN IF NOT EXISTS rule_error BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_validation_rule_violations_bo
    ON validation_rule_violations (tenant_id, bo_key, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_validation_rule_violations_rule
    ON validation_rule_violations (rule_id, created_at DESC);

