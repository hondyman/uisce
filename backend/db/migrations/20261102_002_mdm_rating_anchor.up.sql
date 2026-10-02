-- 20261102_002_mdm_rating_anchor.up.sql
-- rating_scale gets FK to rating_agency (guarded), plus the anchor tables.

-- ── rating_scale: attach FK if agency_id column exists ─────────────────
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='mdm' AND table_name='rating_scale'
                 AND column_name='agency_id')
       AND EXISTS (SELECT 1 FROM information_schema.tables
                   WHERE table_schema='mdm' AND table_name='rating_agency') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname='fk_rs_agency' AND conrelid='mdm.rating_scale'::regclass
        ) THEN
            ALTER TABLE mdm.rating_scale
                ADD CONSTRAINT fk_rs_agency
                FOREIGN KEY (agency_id) REFERENCES mdm.rating_agency(id);
        END IF;
    END IF;
END $do$;

-- ── rating (issuer / issue / counterparty anchor) ──────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agency_id uuid NOT NULL,
    rating_type_id uuid NOT NULL,
    rating_scale_id uuid,
    rated_party_type varchar(30) NOT NULL,
    rated_party_id uuid NOT NULL,
    rated_party_key varchar(100),
    rating_value varchar(20) NOT NULL,
    rating_rank int4,
    outlook_id uuid,
    watch_id uuid,
    credit_watch_direction varchar(20),
    is_active bool DEFAULT true NOT NULL,
    is_latest bool DEFAULT true NOT NULL,
    withdrawn_reason varchar(100),
    withdrawn_at timestamptz,
    rating_date date DEFAULT CURRENT_DATE NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_pkey PRIMARY KEY (id),
    CONSTRAINT fk_r_agency    FOREIGN KEY (agency_id)      REFERENCES mdm.rating_agency(id),
    CONSTRAINT fk_r_type      FOREIGN KEY (rating_type_id)  REFERENCES mdm.rating_type(id),
    CONSTRAINT fk_r_scale     FOREIGN KEY (rating_scale_id) REFERENCES mdm.rating_scale(id),
    CONSTRAINT fk_r_outlook   FOREIGN KEY (outlook_id)      REFERENCES mdm.rating_outlook(id),
    CONSTRAINT fk_r_watch     FOREIGN KEY (watch_id)        REFERENCES mdm.rating_watch(id),
    CONSTRAINT chk_r_party CHECK (rated_party_type IN (
        'ISSUER','ISSUE','TRANCHE','COUNTERPARTY','SOVEREIGN',
        'SUB_SOVEREIGN','FINANCIAL_STRENGTH','ISSUER_GROUP','INTERNAL')),
    CONSTRAINT chk_r_watch_dir CHECK (credit_watch_direction IS NULL
        OR credit_watch_direction IN ('POSITIVE','NEGATIVE','DEVELOPING','EVOLVING'))
);
CREATE INDEX IF NOT EXISTS idx_r_party      ON mdm.rating (rated_party_type, rated_party_id) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_r_agency     ON mdm.rating (agency_id, is_latest);
CREATE INDEX IF NOT EXISTS idx_r_rank       ON mdm.rating (rating_rank) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_r_type       ON mdm.rating (rating_type_id);
CREATE INDEX IF NOT EXISTS idx_r_date       ON mdm.rating (rating_date DESC);
CREATE INDEX IF NOT EXISTS idx_r_tenant     ON mdm.rating (tenant_id);

-- ── rating_action (event history) ──────────────────────────────────────
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
CREATE INDEX IF NOT EXISTS idx_raaction_rating   ON mdm.rating_action (rating_id, action_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_raaction_type     ON mdm.rating_action (action_type_id);
CREATE INDEX IF NOT EXISTS idx_raaction_date     ON mdm.rating_action (action_date DESC);
CREATE INDEX IF NOT EXISTS idx_raaction_ce       ON mdm.rating_action (is_credit_event) WHERE is_credit_event;
CREATE INDEX IF NOT EXISTS idx_raaction_tenant   ON mdm.rating_action (tenant_id);

-- ── rating_default (default & recovery events) ─────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_default (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rating_id uuid NOT NULL,
    rated_party_type varchar(30) NOT NULL,
    rated_party_id uuid NOT NULL,
    default_type varchar(30) NOT NULL,
    default_date date NOT NULL,
    amount_at_default numeric(20,4),
    cure_date date,
    recovery_rate_pct numeric(6,3),
    is_subsequent_default bool DEFAULT false NOT NULL,
    source_document varchar(500),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_default_pkey PRIMARY KEY (id),
    CONSTRAINT fk_rd_rating FOREIGN KEY (rating_id) REFERENCES mdm.rating(id) ON DELETE CASCADE,
    CONSTRAINT chk_rd_type CHECK (default_type IN (
        'DEFAULT','RECOVERY','CURE','DISTRESSED_EXCHANGE','BANKRUPTCY',
        'STRUCTURED','FAIL_TO_PAY','REPO','WATCH_TO_DEFAULT')),
    CONSTRAINT chk_rd_party CHECK (rated_party_type IN (
        'ISSUER','ISSUE','TRANCHE','COUNTERPARTY','SOVEREIGN','ISSUER_GROUP'))
);
CREATE INDEX IF NOT EXISTS idx_rd_party  ON mdm.rating_default (rated_party_type, rated_party_id);
CREATE INDEX IF NOT EXISTS idx_rd_date   ON mdm.rating_default (default_date DESC);
CREATE INDEX IF NOT EXISTS idx_rd_rating ON mdm.rating_default (rating_id);
CREATE INDEX IF NOT EXISTS idx_rd_tenant ON mdm.rating_default (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY['rating','rating_action','rating_default'];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE mdm.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE mdm.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$,
            t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$,
            t || '_tenant_write', t);
    END LOOP;
END
$rls$;
