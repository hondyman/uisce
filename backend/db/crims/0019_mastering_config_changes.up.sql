-- 0019_mastering_config_changes.up.sql
-- Maker-checker for mastering configuration: the vendor registry
-- (mdm.source_systems), each entity's source hierarchy (mdm.<prefix>_source_priority)
-- and match rules (mdm.<prefix>_match_rule). An administrator proposes a
-- change; a second administrator (never the proposer) approves it, and only
-- then is it applied - in the approval's transaction, with the row as it was
-- (before) kept. Tenants never change the gold copy's rows: they propose
-- their own row with the same key, which mastering already prefers
-- (internal/mastering/config.go). Run against crims. Additive and idempotent.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0019_mastering_config_changes.up.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE TABLE IF NOT EXISTS mdm.mastering_config_change (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid        NOT NULL,
    -- NULL for the vendor registry, which is not per entity.
    entity_cd         varchar(30),
    kind              varchar(30) NOT NULL,
    action            varchar(10) NOT NULL,
    -- The row changed or removed; NULL for a new row (including a tenant's
    -- override of a gold-copy row, which is a new row of its own).
    target_id         uuid,
    -- Column values; the source as its registry code under "source".
    "values"          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    before            jsonb,
    reason            text,
    status            varchar(12) NOT NULL DEFAULT 'pending',
    requested_by      text        NOT NULL,
    requested_by_name text,
    requested_at      timestamptz NOT NULL DEFAULT now(),
    reviewed_by       text,
    reviewed_by_name  text,
    reviewed_at       timestamptz,
    review_comment    text,
    applied_id        uuid,
    CONSTRAINT mastering_config_change_kind_check CHECK (kind IN ('source_system', 'source_priority', 'match_rule')),
    CONSTRAINT mastering_config_change_action_check CHECK (action IN ('upsert', 'delete')),
    CONSTRAINT mastering_config_change_status_check CHECK (status IN ('pending', 'applied', 'rejected', 'withdrawn')),
    CONSTRAINT mastering_config_change_delete_target CHECK (action <> 'delete' OR target_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_mastering_config_change_tenant
    ON mdm.mastering_config_change (tenant_id, kind, status, requested_at DESC);

ALTER TABLE mdm.mastering_config_change ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.mastering_config_change FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS mastering_config_change_tenant ON mdm.mastering_config_change;
CREATE POLICY mastering_config_change_tenant ON mdm.mastering_config_change
    USING (tenant_id = (current_setting('app.current_tenant', true))::uuid)
    WITH CHECK (tenant_id = (current_setting('app.current_tenant', true))::uuid);

COMMIT;
