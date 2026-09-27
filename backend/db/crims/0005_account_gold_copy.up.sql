-- Data plane (crims): Account gold copy, field overrides, approvals, survivorship log.
-- Apply to the tenant data-plane database (crims), NOT alpha.

CREATE SCHEMA IF NOT EXISTS mdm;

CREATE TABLE IF NOT EXISTS mdm.account_survivorship_log (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL,
    account_cd   varchar(50) NOT NULL,
    decision     jsonb NOT NULL DEFAULT '{}'::jsonb,
    decided_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_acct_surv_log_cd
    ON mdm.account_survivorship_log (tenant_id, account_cd, decided_at DESC);

CREATE TABLE IF NOT EXISTS mdm.account_gold_copy (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL,
    account_master_id uuid,
    account_cd        varchar(50) NOT NULL,
    gold_version      integer NOT NULL DEFAULT 1,
    payload           jsonb NOT NULL DEFAULT '{}'::jsonb,
    source_systems    jsonb NOT NULL DEFAULT '{}'::jsonb,
    confidence_score  integer NOT NULL DEFAULT 0,
    published_at      timestamptz,
    published_by      uuid,
    change_type       varchar(30) NOT NULL DEFAULT 'created',
    change_reason     text NOT NULL DEFAULT '',
    data_hash         varchar(80),
    valid_from        timestamptz NOT NULL DEFAULT now(),
    valid_to          timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_account_gold_version UNIQUE (tenant_id, account_cd, gold_version)
);
CREATE INDEX IF NOT EXISTS idx_account_gold_current
    ON mdm.account_gold_copy (tenant_id, account_cd)
    WHERE valid_to IS NULL;

CREATE TABLE IF NOT EXISTS mdm.account_field_override (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL,
    account_cd         varchar(50) NOT NULL,
    semantic_term_id   uuid,
    field_cd           varchar(100) NOT NULL,
    override_value     text NOT NULL,
    reason             text NOT NULL,
    actor_id           uuid,
    approval_status    varchar(20) NOT NULL DEFAULT 'pending',
    approved_by        uuid,
    approved_at        timestamptz,
    expires_at         timestamptz,
    is_active          boolean NOT NULL DEFAULT true,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_acct_override_status CHECK (
        approval_status IN ('pending', 'approved', 'rejected', 'withdrawn')
    )
);
CREATE INDEX IF NOT EXISTS idx_acct_override_pending
    ON mdm.account_field_override (tenant_id, account_cd)
    WHERE is_active AND approval_status = 'pending';
CREATE INDEX IF NOT EXISTS idx_acct_override_active
    ON mdm.account_field_override (tenant_id, account_cd, field_cd)
    WHERE is_active AND approval_status = 'approved';

CREATE TABLE IF NOT EXISTS mdm.account_gold_approval (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL,
    account_cd      varchar(50) NOT NULL,
    gold_version    integer,
    status          varchar(20) NOT NULL DEFAULT 'pending',
    justification   text NOT NULL DEFAULT '',
    requested_by    uuid,
    requested_at    timestamptz NOT NULL DEFAULT now(),
    reviewed_by     uuid,
    reviewed_at     timestamptz,
    review_comment  text NOT NULL DEFAULT '',
    CONSTRAINT chk_acct_gold_approval_status CHECK (
        status IN ('pending', 'approved', 'rejected')
    )
);
CREATE INDEX IF NOT EXISTS idx_acct_gold_approval_pending
    ON mdm.account_gold_approval (tenant_id, status, requested_at DESC);

CREATE TABLE IF NOT EXISTS mdm.account_gold_lineage (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL,
    account_cd      varchar(50) NOT NULL,
    gold_version    integer NOT NULL,
    field_cd        varchar(100) NOT NULL,
    chosen_value    text,
    chosen_source   varchar(50),
    strategy        varchar(40),
    override_id     uuid,
    decided_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_acct_gold_lineage
    ON mdm.account_gold_lineage (tenant_id, account_cd, gold_version);

-- RLS
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'account_survivorship_log',
        'account_gold_copy',
        'account_field_override',
        'account_gold_approval',
        'account_gold_lineage'
    ] LOOP
        EXECUTE format('ALTER TABLE mdm.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE mdm.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant', t);
        EXECUTE format(
            'CREATE POLICY %I ON mdm.%I USING (tenant_id::text = COALESCE(current_setting(''app.current_tenant'', true), current_setting(''uisce.current_tenant'', true), '''')) WITH CHECK (tenant_id::text = COALESCE(current_setting(''app.current_tenant'', true), current_setting(''uisce.current_tenant'', true), ''''))',
            t || '_tenant', t
        );
    END LOOP;
END $$;
