-- 0017_mdm_rating_action_crims.up.sql
-- Verbatim port of mdm.rating_action from alpha
-- db/migrations/20261102_002_mdm_rating_anchor.up.sql (lines 67-91), plus
-- the model_value column added by db/crims/0013_mdm_rating_action_model_value.up.sql.
-- Idempotent.

\set ON_ERROR_STOP on
BEGIN;

CREATE SCHEMA IF NOT EXISTS mdm;

CREATE TABLE IF NOT EXISTS mdm.rating_action (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rating_id uuid NOT NULL,
    action_type_id uuid NOT NULL,
    prior_rating_id uuid,
    prior_value varchar(20),
    new_value varchar(20) NOT NULL,
    prior_rank int4,
    new_rank int4,
    action_date date DEFAULT CURRENT_DATE NOT NULL,
    action_timestamp timestamptz DEFAULT CURRENT_TIMESTAMP,
    reason text,
    outlook_id uuid,
    watch_id uuid,
    is_credit_event bool DEFAULT false NOT NULL,
    source_document varchar(500),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_action_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ra_rating  FOREIGN KEY (rating_id)       REFERENCES mdm.rating(id) ON DELETE CASCADE,
    CONSTRAINT fk_ra_atype   FOREIGN KEY (action_type_id)  REFERENCES mdm.rating_action_type(id),
    CONSTRAINT fk_ra_prior   FOREIGN KEY (prior_rating_id) REFERENCES mdm.rating(id),
    CONSTRAINT fk_ra_outlook FOREIGN KEY (outlook_id)       REFERENCES mdm.rating_outlook(id),
    CONSTRAINT fk_ra_watch   FOREIGN KEY (watch_id)         REFERENCES mdm.rating_watch(id)
);

-- model_value: populated only when model's raw output differs from BOTH
-- prior and new master values. NULL otherwise.
ALTER TABLE mdm.rating_action
    ADD COLUMN IF NOT EXISTS model_value varchar(20);

CREATE INDEX IF NOT EXISTS idx_raaction_rating   ON mdm.rating_action (rating_id, action_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_raaction_type     ON mdm.rating_action (action_type_id);
CREATE INDEX IF NOT EXISTS idx_raaction_date     ON mdm.rating_action (action_date DESC);
CREATE INDEX IF NOT EXISTS idx_raaction_ce       ON mdm.rating_action (is_credit_event) WHERE is_credit_event;
CREATE INDEX IF NOT EXISTS idx_raaction_tenant   ON mdm.rating_action (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
ALTER TABLE mdm.rating_action ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.rating_action FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rating_action_tenant_read ON mdm.rating_action;
CREATE POLICY rating_action_tenant_read ON mdm.rating_action
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS rating_action_tenant_write ON mdm.rating_action;
CREATE POLICY rating_action_tenant_write ON mdm.rating_action
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

DO $verify$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema='mdm' AND table_name='rating_action'
    ) THEN
        RAISE EXCEPTION '0017_mdm_rating_action_crims: mdm.rating_action not created';
    END IF;
    RAISE NOTICE '0017_mdm_rating_action_crims: mdm.rating_action created/verified';
END
$verify$;

COMMIT;
