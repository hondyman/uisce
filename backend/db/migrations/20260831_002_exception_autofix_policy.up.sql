-- Per-tenant, per-exception-type autofix opt-in. Never a global switch:
-- every row is scoped to (tenant_id, exception_type), with an optional
-- per-user override row (user_id NOT NULL) checked before the tenant default
-- (user_id IS NULL) in ExceptionAggregator.ResolveAutofixPolicy.
CREATE TABLE IF NOT EXISTS exception_autofix_policy (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    user_id            UUID,
    exception_type     TEXT NOT NULL,
    enabled            BOOLEAN NOT NULL DEFAULT false,
    requires_approval  BOOLEAN NOT NULL DEFAULT true,
    updated_by         TEXT NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One tenant-level default row per (tenant_id, exception_type).
CREATE UNIQUE INDEX IF NOT EXISTS exception_autofix_policy_tenant_default_uq
    ON exception_autofix_policy (tenant_id, exception_type)
    WHERE user_id IS NULL;

-- One per-user override row per (tenant_id, user_id, exception_type).
CREATE UNIQUE INDEX IF NOT EXISTS exception_autofix_policy_user_override_uq
    ON exception_autofix_policy (tenant_id, user_id, exception_type)
    WHERE user_id IS NOT NULL;
