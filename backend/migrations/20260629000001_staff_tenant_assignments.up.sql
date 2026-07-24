-- =============================================================================
-- Migration: Staff tenant assignment lease table
-- Purpose:
--   Enforce a second, independent boundary gate for helpdesk and
--   professional_services operators. Even when a user holds one of those roles
--   in Keycloak, they may only impersonate a tenant while an explicit,
--   time-bounded staff_tenant_assignments lease is active.
-- =============================================================================

CREATE TABLE IF NOT EXISTS staff_tenant_assignments (
    assignment_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_user_id   TEXT NOT NULL,
    target_tenant_id   UUID NOT NULL,
    granted_by         TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at         TIMESTAMPTZ NOT NULL
);

COMMENT ON TABLE staff_tenant_assignments IS
    'Time-bounded tenant access grants for helpdesk and professional_services operators.';

CREATE INDEX IF NOT EXISTS idx_staff_tenant_assignments_active
    ON staff_tenant_assignments (operator_user_id, target_tenant_id, expires_at);
