-- 0013_mastering_overrides.up.sql
-- Steward overrides of golden values, with a per-entity policy. Run against crims.
--
--   mdm.mastering_policy        per tenant and entity: overrides need approval (N distinct approvers,
--                               more for high-risk attributes) or apply directly. A tenant's own row
--                               wins over the gold copy's; with neither, approval by one.
--   mdm.mastering_policy_audit  every policy change: who, when, before and after.
--   mdm.golden_override         a request to set (or clear) one golden attribute. Once applied, a SET
--                               stays active - it wins survivorship on every later load - until a
--                               later SET supersedes it or a CLEAR removes it.
--   mdm.golden_override_vote    each approver's decision; the proposer never votes.
--
-- Same RLS as every mdm table. Additive.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0013_mastering_overrides.up.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE TABLE IF NOT EXISTS mdm.mastering_policy (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             uuid NOT NULL,
    entity_cd             varchar(40) NOT NULL,
    override_mode         varchar(20) NOT NULL DEFAULT 'APPROVAL',
    approvals_required    int NOT NULL DEFAULT 1,
    high_risk_attributes  text[] NOT NULL DEFAULT '{}',
    high_risk_approvals   int NOT NULL DEFAULT 2,
    updated_by            text,
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT mastering_policy_uq UNIQUE (tenant_id, entity_cd),
    CONSTRAINT mastering_policy_mode_ck CHECK (override_mode IN ('APPROVAL', 'DIRECT')),
    CONSTRAINT mastering_policy_n_ck CHECK (approvals_required BETWEEN 1 AND 5 AND high_risk_approvals BETWEEN 1 AND 5)
);

CREATE TABLE IF NOT EXISTS mdm.mastering_policy_audit (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL,
    entity_cd   varchar(40) NOT NULL,
    before      jsonb,
    after       jsonb NOT NULL,
    changed_by  text NOT NULL,
    changed_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mdm.golden_override (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL,
    entity_cd           varchar(40) NOT NULL,
    golden_id           uuid NOT NULL,
    attribute           text NOT NULL,
    action              varchar(10) NOT NULL,          -- SET | CLEAR
    value               jsonb,                         -- SET: the value
    previous_value      jsonb,                         -- the golden value when proposed
    reason              text NOT NULL,
    mode                varchar(20) NOT NULL,          -- APPROVAL | DIRECT (the policy when proposed)
    approvals_required  int NOT NULL DEFAULT 0,
    status              varchar(20) NOT NULL DEFAULT 'PENDING',
    active              boolean NOT NULL DEFAULT false,
    requested_by        text NOT NULL,
    requested_by_name   text,
    requested_at        timestamptz NOT NULL DEFAULT now(),
    applied_at          timestamptz,
    applied_version     int,
    ended_by_id         uuid,                          -- the override that superseded or cleared it
    ended_at            timestamptz,
    CONSTRAINT golden_override_action_ck CHECK (action IN ('SET', 'CLEAR')),
    CONSTRAINT golden_override_status_ck CHECK (status IN ('PENDING', 'APPLIED', 'REJECTED', 'WITHDRAWN')),
    CONSTRAINT golden_override_reason_ck CHECK (length(btrim(reason)) > 0),
    CONSTRAINT golden_override_active_ck CHECK (NOT active OR (status = 'APPLIED' AND action = 'SET'))
);
-- One live override and one open request per attribute of a golden record.
CREATE UNIQUE INDEX IF NOT EXISTS golden_override_active_uq
    ON mdm.golden_override (tenant_id, entity_cd, golden_id, attribute) WHERE active;
CREATE UNIQUE INDEX IF NOT EXISTS golden_override_pending_uq
    ON mdm.golden_override (tenant_id, entity_cd, golden_id, attribute) WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS golden_override_status_ix ON mdm.golden_override (tenant_id, entity_cd, status, requested_at DESC);

CREATE TABLE IF NOT EXISTS mdm.golden_override_vote (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL,
    override_id   uuid NOT NULL REFERENCES mdm.golden_override(id) ON DELETE CASCADE,
    approver      text NOT NULL,
    approver_name text,
    decision      varchar(10) NOT NULL,
    comment       text,
    decided_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT golden_override_vote_uq UNIQUE (override_id, approver),
    CONSTRAINT golden_override_vote_ck CHECK (decision IN ('APPROVE', 'REJECT'))
);

DO $rls$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['mastering_policy', 'mastering_policy_audit', 'golden_override', 'golden_override_vote'] LOOP
        EXECUTE format('ALTER TABLE mdm.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE mdm.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$, t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$, t || '_tenant_write', t);
    END LOOP;
END
$rls$;

-- Gold copy default for Product: approval by one person (tenants inherit it until they set their own).
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
INSERT INTO mdm.mastering_policy (tenant_id, entity_cd, override_mode, approvals_required, updated_by)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRODUCT', 'APPROVAL', 1, 'migration 0013')
ON CONFLICT (tenant_id, entity_cd) DO NOTHING;

COMMIT;
