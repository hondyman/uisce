-- 0016_security_master_profile.up.sql
-- A self-contained security master on the generic mastering engine (see docs/mdm-mastering-blueprint.md
-- and docs/mdm-price-mastering-design.md: prices will resolve to these golden securities). Run against
-- crims. Additive and idempotent.
--
--   * mdm.security_master.master_id - the security's stable id across its bitemporal versions (each
--     version is a new row, so the row id changes). Golden records, identifiers and the cross-reference
--     point at it. Backfilled one per security; a trigger carries it onto new versions written by other
--     loaders.
--   * staging.security_incoming - canonical records (as staging.product_incoming).
--   * ICE joins the one vendor registry (mdm.source_systems) - a vendor is one source whatever it
--     supplies.
--   * SECURITY mastering profile (gold copy), identifier-only automatic matching (a similar name only
--     raises a possible duplicate), source priority by field group, override/merge policy.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0016_security_master_profile.up.sql

\set ON_ERROR_STOP on
BEGIN;

-- 0. One vendor registry ---------------------------------------------------------------------------
-- A vendor is one source whatever it supplies. mdm.source_systems is the registry (product, benchmark,
-- price, party, counterparty, corporate actions, calendars reference it); the security and issuer tables
-- and mdm.xref still referenced the near-empty mdm.source_system. Copy its rows across with the same
-- ids (so nothing that points at them breaks), then repoint those foreign keys. mdm.source_system is kept,
-- deprecated.
INSERT INTO mdm.source_systems (id, tenant_id, code, display_name, created_at)
SELECT s.id, s.tenant_id, s.source_cd, COALESCE(s.name, s.source_cd), COALESCE(s.created_at, now())
  FROM mdm.source_system s
 WHERE NOT EXISTS (SELECT 1 FROM mdm.source_systems x WHERE x.id = s.id);

DO $fk$
DECLARE
    c record;
BEGIN
    FOR c IN
        SELECT con.conname, con.conrelid::regclass AS tbl,
               (SELECT string_agg(quote_ident(a.attname), ', ' ORDER BY k.ord)
                  FROM unnest(con.conkey) WITH ORDINALITY k(attnum, ord)
                  JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = k.attnum) AS cols,
               con.confdeltype
          FROM pg_constraint con
         WHERE con.contype = 'f' AND con.confrelid = 'mdm.source_system'::regclass
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', c.tbl, c.conname);
        EXECUTE format('ALTER TABLE %s ADD CONSTRAINT %I FOREIGN KEY (%s) REFERENCES mdm.source_systems(id)%s',
                       c.tbl, c.conname, c.cols,
                       CASE c.confdeltype WHEN 'c' THEN ' ON DELETE CASCADE' WHEN 'n' THEN ' ON DELETE SET NULL' ELSE '' END);
    END LOOP;
END
$fk$;
COMMENT ON TABLE mdm.source_system IS
    'Deprecated: mdm.source_systems is the one vendor registry (0016 repointed every foreign key to it).';

-- 1. Stable id -------------------------------------------------------------------------------------
ALTER TABLE mdm.security_master ADD COLUMN IF NOT EXISTS master_id uuid;

UPDATE mdm.security_master sm SET master_id = x.mid
  FROM (SELECT tenant_id, security_id, gen_random_uuid() AS mid
          FROM (SELECT DISTINCT tenant_id, security_id FROM mdm.security_master WHERE master_id IS NULL) d) x
 WHERE sm.master_id IS NULL AND sm.tenant_id = x.tenant_id AND sm.security_id = x.security_id;

CREATE OR REPLACE FUNCTION mdm.security_master_carry_master_id() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.master_id IS NULL THEN
        SELECT master_id INTO NEW.master_id FROM mdm.security_master
         WHERE tenant_id = NEW.tenant_id AND security_id = NEW.security_id AND master_id IS NOT NULL
         ORDER BY valid_from DESC NULLS LAST LIMIT 1;
        IF NEW.master_id IS NULL THEN
            NEW.master_id := gen_random_uuid();
        END IF;
    END IF;
    RETURN NEW;
END
$$;
DROP TRIGGER IF EXISTS security_master_carry_master_id ON mdm.security_master;
CREATE TRIGGER security_master_carry_master_id BEFORE INSERT ON mdm.security_master
    FOR EACH ROW EXECUTE FUNCTION mdm.security_master_carry_master_id();

ALTER TABLE mdm.security_master ALTER COLUMN master_id SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_security_master_master_current
    ON mdm.security_master (tenant_id, master_id) WHERE valid_to IS NULL;
CREATE INDEX IF NOT EXISTS ix_security_master_master_id ON mdm.security_master (master_id);
COMMENT ON COLUMN mdm.security_master.master_id IS
    'The security''s stable id across bitemporal versions; golden records, identifiers and the cross-reference point at it.';

-- 2. Canonical records -------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS staging.security_incoming (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system_id    uuid NOT NULL,
    source_system_cd    varchar(30) NOT NULL,
    source_row_id       text NOT NULL,
    load_run_id         uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    tenant_id           uuid NOT NULL,
    canonical_payload   jsonb NOT NULL DEFAULT '{}'::jsonb,
    mapping_version     integer NOT NULL DEFAULT 1,
    mapped_at           timestamptz NOT NULL DEFAULT now(),
    is_valid            boolean NOT NULL DEFAULT true,
    validation_errors   jsonb NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT security_incoming_valid_ck CHECK ((is_valid AND jsonb_array_length(validation_errors) = 0) OR NOT is_valid)
);
CREATE INDEX IF NOT EXISTS ix_security_incoming_run ON staging.security_incoming (tenant_id, load_run_id, source_system_id);
CREATE INDEX IF NOT EXISTS ix_security_incoming_row ON staging.security_incoming (tenant_id, source_system_id, source_row_id, mapped_at DESC);
ALTER TABLE staging.security_incoming ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.security_incoming FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS si_tenant_read ON staging.security_incoming;
CREATE POLICY si_tenant_read ON staging.security_incoming AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                 '00000000-0000-0000-0000-000000000001'::uuid)));
DROP POLICY IF EXISTS si_tenant_write ON staging.security_incoming;
CREATE POLICY si_tenant_write ON staging.security_incoming AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- 3. Gold-copy configuration --------------------------------------------------------------------------
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);

INSERT INTO mdm.source_systems (id, tenant_id, code, display_name)
SELECT 'a4444444-4444-4444-4444-444444444444', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'ICE', 'ICE Data Services'
WHERE NOT EXISTS (SELECT 1 FROM mdm.source_systems WHERE code = 'ICE');

INSERT INTO mdm.mastering_entity
    (tenant_id, entity_cd, display_name, bo_key, table_prefix, anchor_table, anchor_code_column,
     identifier_table, incoming_table, code_prefix, settings)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'SECURITY', 'Security', 'security', 'security', 'mdm.security_master',
        'security_id', 'mdm.security_identifier_issuance', 'staging.security_incoming', 'SEC-', '{
          "bo_binding": "Security Master Binding",
          "entity_id_column": "master_id",
          "versioning": "bitemporal",
          "name_attribute": "security_name",
          "identifiers": {"key_column": "security_id", "type_column": "id_type", "value_column": "id_value",
                          "source_column": "source_system_id", "source_is_id": true,
                          "active_column": "is_valid", "active_is_flag": true},
          "derived": {"primary_identifier": ["id:ISIN", "id:FIGI", "id:CUSIP", "id:SEDOL", "@code"]},
          "required": ["security_name", "asset_class", "currency"],
          "defaults": {"status": "Active"},
          "merged_values": {"status": "Inactive"},
          "record_columns": {"asset_class_cd": "asset_class", "sec_typ_cd": "instrument_type", "sec_sub_typ_cd": "sub_asset_class"},
          "default_field_group": "IDENTITY",
          "field_groups": {
            "security_name": "NAME", "short_name": "NAME", "description": "NAME",
            "asset_class": "CLASSIFICATION", "sub_asset_class": "CLASSIFICATION", "instrument_type": "CLASSIFICATION",
            "sector": "CLASSIFICATION", "industry": "CLASSIFICATION", "regulatory_classification": "CLASSIFICATION",
            "issue_date": "TERMS", "first_trade_date": "TERMS", "maturity_date": "TERMS", "final_maturity_date": "TERMS",
            "callable_from_date": "TERMS", "puttable_from_date": "TERMS", "quotation_type": "TERMS", "settlement_currency": "TERMS",
            "listing_exchange": "LISTING", "primary_listing_venue": "LISTING", "exchange_code": "LISTING",
            "ticker": "LISTING", "local_ticker": "LISTING"
          }
        }'::jsonb)
ON CONFLICT (tenant_id, entity_cd) DO NOTHING;

-- Identifiers match automatically; a similar name never does (auto threshold above 1): it only raises
-- a possible duplicate for a steward (share classes, series and tranches share near-identical names).
INSERT INTO mdm.security_match_rule
    (tenant_id, rule_cd, asset_class_cd, rule_name, match_keys, deterministic_keys, fuzzy_keys,
     threshold_auto_match, threshold_review, threshold_no_match, priority)
VALUES
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'SEC_ID_EXACT', '*', 'Exact identifier match',
     '["ISIN", "CUSIP", "SEDOL", "FIGI", "BLOOMBERG_ID"]', '["ISIN", "CUSIP", "SEDOL", "FIGI", "BLOOMBERG_ID"]', '[]',
     1.0, 1.0, 0.0, 10),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'SEC_NAME_REVIEW', '*', 'Similar name, currency and asset class (review only)',
     '["security_name", "currency", "asset_class"]', '[]',
     '[{"field": "security_name", "method": "trigram", "weight": 0.6},
       {"field": "currency", "method": "exact", "weight": 0.2},
       {"field": "asset_class", "method": "exact", "weight": 0.2}]',
     1.01, 0.85, 0.85, 20)
ON CONFLICT (tenant_id, rule_cd) DO NOTHING;

-- Source hierarchy by field group, for every asset class ('*').
INSERT INTO mdm.security_source_priority (tenant_id, asset_class_cd, field_group, source_system_id, priority)
SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999', '*', g.grp, s.id, g.prio
  FROM (VALUES
        ('IDENTITY', 'BLOOMBERG', 10), ('IDENTITY', 'REFINITIV', 20), ('IDENTITY', 'ICE', 30),
        ('NAME', 'BLOOMBERG', 10), ('NAME', 'REFINITIV', 20), ('NAME', 'ICE', 30),
        ('CLASSIFICATION', 'BLOOMBERG', 10), ('CLASSIFICATION', 'REFINITIV', 20), ('CLASSIFICATION', 'ICE', 30),
        ('TERMS', 'ICE', 10), ('TERMS', 'BLOOMBERG', 20), ('TERMS', 'REFINITIV', 30),
        ('LISTING', 'BLOOMBERG', 10), ('LISTING', 'REFINITIV', 20), ('LISTING', 'ICE', 30)
       ) AS g(grp, code, prio)
  JOIN mdm.source_systems s ON s.code = g.code
 WHERE NOT EXISTS (SELECT 1 FROM mdm.security_source_priority x
                    WHERE x.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND x.asset_class_cd = '*'
                      AND x.field_group = g.grp AND x.source_system_id = s.id);

INSERT INTO mdm.mastering_policy (tenant_id, entity_cd, override_mode, approvals_required, updated_by)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'SECURITY', 'APPROVAL', 1, 'migration 0016')
ON CONFLICT (tenant_id, entity_cd) DO NOTHING;

COMMIT;
