-- 0011_mastering_foundation.up.sql
-- Generic entity mastering: one engine, configured per entity (see docs/mdm-mastering-blueprint.md).
-- Run against crims.
--
--   mdm.mastering_entity  the per-entity profile: which BO, which mdm.<prefix>_* tables, which
--                         identifier table. A new master (security, benchmark, price) is a row here
--                         plus configuration, not new code.
--   mdm.entity_xref       source record -> golden entity, for every entity. The match stage writes it;
--                         a re-delivered source record resolves through it before any matching.
--   mdm.mastering_run     one run of canonicalize -> match -> survive -> publish over a load run.
--                         The idempotency key makes a retried trigger (Tidal, Control-M) a no-op.
--
-- Plus the Product match rules (mdm.product_match_rule was empty), seeded in the gold-copy tenant,
-- where the Product survivorship rules and source priorities already live.
--
-- Additive and idempotent. Same RLS as every mdm table: tenant read (plus the shared reference
-- tenant), tenant-only write, FORCE ROW LEVEL SECURITY.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0011_mastering_foundation.up.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE TABLE IF NOT EXISTS mdm.mastering_entity (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL,
    entity_cd           varchar(40) NOT NULL,           -- PRODUCT, SECURITY, BENCHMARK, PRICE ...
    display_name        text NOT NULL,
    bo_key              text NOT NULL,                  -- business object key in the catalog (product)
    table_prefix        text NOT NULL,                  -- mdm.<prefix>_golden_record, _match_rule, ...
    anchor_table        text NOT NULL,                  -- the mastered table (mdm.product)
    anchor_code_column  text NOT NULL,                  -- its business code column (product_cd)
    identifier_table    text,                           -- mdm.product_identifier (id_type, id_value)
    incoming_table      text NOT NULL,                  -- canonical landing table (staging.product_incoming)
    code_prefix         text NOT NULL,                  -- prefix for minted codes (PRD-)
    settings            jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_active           boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT mastering_entity_cd_uq UNIQUE (tenant_id, entity_cd),
    CONSTRAINT mastering_entity_cd_ck CHECK (entity_cd ~ '^[A-Z][A-Z0-9_]{1,39}$'),
    CONSTRAINT mastering_entity_prefix_ck CHECK (table_prefix ~ '^[a-z][a-z0-9_]{1,40}$')
);

CREATE TABLE IF NOT EXISTS mdm.entity_xref (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL,
    entity_cd           varchar(40) NOT NULL,
    source_system_id    uuid NOT NULL,
    source_key          text NOT NULL,                  -- the source's own record key
    golden_id           uuid NOT NULL,                  -- id in the anchor table
    match_method        varchar(20) NOT NULL,
    match_score         numeric(6,4),
    match_rule_cd       text,
    matched_keys        jsonb NOT NULL DEFAULT '[]'::jsonb,
    status              varchar(20) NOT NULL DEFAULT 'ACTIVE',
    first_run_id        uuid,
    last_run_id         uuid,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entity_xref_method_ck CHECK (match_method IN ('DETERMINISTIC', 'FUZZY', 'NEW', 'STEWARD')),
    CONSTRAINT entity_xref_status_ck CHECK (status IN ('ACTIVE', 'RETIRED'))
);
-- One live link per source record; a steward re-link retires the old one.
CREATE UNIQUE INDEX IF NOT EXISTS entity_xref_source_uq
    ON mdm.entity_xref (tenant_id, entity_cd, source_system_id, source_key) WHERE status = 'ACTIVE';
CREATE INDEX IF NOT EXISTS entity_xref_golden_ix ON mdm.entity_xref (tenant_id, entity_cd, golden_id);

CREATE TABLE IF NOT EXISTS mdm.mastering_run (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL,
    entity_cd           varchar(40) NOT NULL,
    load_run_id         uuid,                           -- staging._load_run being mastered
    source_system_id    uuid,
    idempotency_key     text NOT NULL,
    trigger_kind        varchar(20) NOT NULL DEFAULT 'manual',
    schedule_run_id     uuid,
    workflow_id         text,
    status              varchar(20) NOT NULL DEFAULT 'RUNNING',
    stage               varchar(20) NOT NULL DEFAULT 'CANONICALIZE',
    counts              jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_code          text,
    error_detail        text,
    started_by          text,
    started_at          timestamptz NOT NULL DEFAULT now(),
    finished_at         timestamptz,
    CONSTRAINT mastering_run_key_uq UNIQUE (tenant_id, entity_cd, idempotency_key),
    CONSTRAINT mastering_run_status_ck CHECK (status IN ('RUNNING', 'COMPLETED', 'PARTIAL', 'FAILED')),
    CONSTRAINT mastering_run_stage_ck CHECK (stage IN ('CANONICALIZE', 'MATCH', 'SURVIVE', 'PUBLISH', 'DONE')),
    CONSTRAINT mastering_run_trigger_ck CHECK (trigger_kind IN ('manual', 'schedule', 'external'))
);
CREATE INDEX IF NOT EXISTS mastering_run_recent_ix ON mdm.mastering_run (tenant_id, entity_cd, started_at DESC);

DO $rls$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['mastering_entity', 'entity_xref', 'mastering_run'] LOOP
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

-- Seeds: gold-copy tenant (shared definitions tenants inherit read-only). -----------------------------
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);

INSERT INTO mdm.mastering_entity
    (tenant_id, entity_cd, display_name, bo_key, table_prefix, anchor_table, anchor_code_column,
     identifier_table, incoming_table, code_prefix, settings)
VALUES
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRODUCT', 'Product', 'product', 'product', 'mdm.product',
     'product_cd', 'mdm.product_identifier', 'staging.product_incoming', 'PRD-',
     '{"source_table": "mdm.source_systems"}'::jsonb)
ON CONFLICT (tenant_id, entity_cd) DO NOTHING;

-- Deterministic keys match on an identifier alone; fuzzy keys score name similarity with the
-- listed attributes as agreement boosts. Scores >= auto-match link automatically, >= review raise
-- a steward candidate, below no-match mint a new golden record.
INSERT INTO mdm.product_match_rule
    (tenant_id, rule_cd, product_type_cd, rule_name, match_keys, deterministic_keys, fuzzy_keys,
     threshold_auto_match, threshold_review, threshold_no_match, priority)
VALUES
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRODUCT_ID_EXACT', 'ALL', 'Exact identifier match',
     '["ISIN", "CUSIP", "SEDOL", "LEI", "BLOOMBERG_ID", "PROVIDER_CODE"]'::jsonb,
     '["ISIN", "CUSIP", "SEDOL", "LEI", "BLOOMBERG_ID", "PROVIDER_CODE"]'::jsonb, '[]'::jsonb,
     1.0, 1.0, 0.0, 10),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRODUCT_NAME_FUZZY', 'ALL', 'Name with currency and domicile',
     '["name", "base_currency", "domicile"]'::jsonb, '[]'::jsonb,
     '[{"field": "name", "method": "trigram", "weight": 0.7},
       {"field": "base_currency", "method": "exact", "weight": 0.15},
       {"field": "domicile", "method": "exact", "weight": 0.15}]'::jsonb,
     0.95, 0.80, 0.80, 20)
ON CONFLICT (tenant_id, rule_cd) DO NOTHING;

COMMIT;
