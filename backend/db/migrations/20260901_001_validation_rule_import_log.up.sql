CREATE TABLE IF NOT EXISTS validation_rule_import_log (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL,
    idempotency_key  text NOT NULL,
    bundle_checksum  text NOT NULL,
    report           jsonb NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS validation_rule_import_log_tenant_key_uniq
    ON validation_rule_import_log (tenant_id, idempotency_key);
