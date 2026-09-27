-- 0012_mdm_rating_internal_override.up.sql
-- Internal-rating overrides requiring second-person approval.
-- Run against crims. Idempotent.

\set ON_ERROR_STOP on
BEGIN;

CREATE SCHEMA IF NOT EXISTS mdm;

CREATE TABLE IF NOT EXISTS mdm.rating_internal_override (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL,
    rated_party_type  varchar(30) NOT NULL,
    rated_party_key   varchar(100) NOT NULL,
    rating_value      varchar(20) NOT NULL,
    reason            text NOT NULL,
    approval_status   varchar(20) NOT NULL DEFAULT 'pending'
        CHECK (approval_status IN ('pending','approved','rejected','withdrawn')),
    requested_by      uuid,
    requested_at      timestamptz NOT NULL DEFAULT now(),
    approved_by       uuid,
    approved_at       timestamptz,
    effective_from    date NOT NULL DEFAULT CURRENT_DATE,
    effective_to      date,
    expires_at        timestamptz,
    is_active         boolean NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_rating_internal_override_window
        CHECK (effective_to IS NULL OR effective_to > effective_from)
);

COMMENT ON TABLE mdm.rating_internal_override IS
  'Internal-rating overrides requiring second-person approval. '
  'One active approved override per (tenant, rated_party_type, rated_party_key) '
  'enforced by uq_rating_internal_override_active. '
  'To close an override: UPDATE is_active = false. '
  'Do not delete rows — keep the audit trail.';

-- Index predicate immutability: approval_status and is_active are column refs.
-- Index enforces at most one active approved override per party at any moment.
-- Window shape (effective_from/effective_to) is a query-time filter, not a constraint.
CREATE UNIQUE INDEX IF NOT EXISTS uq_rating_internal_override_active
    ON mdm.rating_internal_override (tenant_id, rated_party_type, rated_party_key)
    WHERE approval_status = 'approved' AND is_active;

CREATE INDEX IF NOT EXISTS idx_rating_internal_override_pending
    ON mdm.rating_internal_override (tenant_id, rated_party_type, rated_party_key)
    WHERE is_active AND approval_status = 'pending';
CREATE INDEX IF NOT EXISTS idx_rating_internal_override_approved
    ON mdm.rating_internal_override (tenant_id, rated_party_type, rated_party_key)
    WHERE is_active AND approval_status = 'approved';

ALTER TABLE mdm.rating_internal_override ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.rating_internal_override FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rating_internal_override_tenant_read ON mdm.rating_internal_override;
CREATE POLICY rating_internal_override_tenant_read ON mdm.rating_internal_override
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS rating_internal_override_tenant_write ON mdm.rating_internal_override;
CREATE POLICY rating_internal_override_tenant_write ON mdm.rating_internal_override
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

DO $verify$
DECLARE
    t text;
    tables text[] := ARRAY['rating_internal_override'];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema='mdm' AND table_name=t
        ) THEN
            RAISE EXCEPTION 'mdm.% was not created', t;
        END IF;
    END LOOP;
    RAISE NOTICE '0012_mdm_rating_internal_override: 1 table created/verified';
END
$verify$;

COMMIT;
