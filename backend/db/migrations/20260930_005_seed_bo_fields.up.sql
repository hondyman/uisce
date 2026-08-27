-- 20260930_005_seed_bo_fields.up.sql
-- Seeds bo_fields rows for every field in oms.subtype_registry.field_allowlist.
-- Each field is linked to its subtype BO (created by migration 004).
-- Uses delete-then-insert to avoid partial unique index / ON CONFLICT issues.

CREATE UNIQUE INDEX IF NOT EXISTS bo_fields_bo_key_idx
    ON bo_fields (business_object_id, key) WHERE business_object_id IS NOT NULL;

BEGIN;

DO $$
DECLARE
    gct UUID;
BEGIN
    SELECT id INTO gct FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gct IS NULL THEN
        gct := '00000000-0000-0000-0000-000000000001'::UUID;
    END IF;

    -- Wipe existing bo_fields for child BOs (they will be re-seeded)
    DELETE FROM bo_fields WHERE business_object_id IN (
        SELECT id FROM business_objects WHERE tenant_id = gct AND parent_id IS NOT NULL
    );

    INSERT INTO bo_fields
        (id, tenant_id, business_object_id,
         key, name, technical_name,
         display_name, type,
         is_core, is_required, is_system,
         sequence,
         created_at, last_modified_at)
    WITH subtype_bos AS (
        SELECT id AS bo_id, key AS bo_key
        FROM business_objects
        WHERE tenant_id = gct AND parent_id IS NOT NULL
    ),
    expanded_fields AS (
        SELECT
            sbo.bo_id,
            sbo.bo_key,
            sr.subtype_code,
            elem.value::text AS field_name,
            elem.ordinality
        FROM subtype_bos sbo
        JOIN oms.subtype_registry sr
          ON sr.tenant_id = gct
          AND sr.is_active = true
          AND (
              sbo.bo_key = 'oms.' || sr.root_object || '/' || sr.subtype_code
              OR sbo.bo_key = 'altinv.' || sr.root_object || '/' || sr.subtype_code
              OR sbo.bo_key = 'cash_flow.' || sr.root_object || '/' || sr.subtype_code
              OR sbo.bo_key = 'master.' || sr.root_object || '/' || sr.subtype_code
          ),
        LATERAL jsonb_array_elements_text(sr.field_allowlist) WITH ORDINALITY AS elem(value, ordinality)
    )
    SELECT
        gen_random_uuid(),
        gct,
        ef.bo_id,
        ef.field_name,
        ef.field_name,
        ef.field_name,
        initcap(replace(replace(replace(
            ef.field_name, '_', ' '), 'id', 'ID'), 'pct', 'PCT')),
        CASE
            WHEN ef.field_name LIKE '%_date' OR ef.field_name LIKE '%_at'
                OR ef.field_name IN ('ex_date','record_date','due_date','maturity_date')
                THEN 'date'
            WHEN ef.field_name LIKE '%_amount' OR ef.field_name LIKE '%_value'
                OR ef.field_name IN ('aum_basis_amount','committed_capital','called_capital')
                THEN 'decimal'
            WHEN ef.field_name LIKE '%_pct' OR ef.field_name LIKE '%_rate'
                OR ef.field_name LIKE '%_bps'
                THEN 'decimal'
            WHEN ef.field_name LIKE '%_flag'
                OR ef.field_name IN ('erisa_flag','margin_agreement_flag',
                                     'drip_reinvest_flag','mandatory_flag')
                THEN 'boolean'
            WHEN ef.field_name LIKE '%_id' AND ef.field_name NOT LIKE '%_id_type'
                THEN 'string'
            ELSE 'text'
        END,
        true,
        false,
        false,
        ef.ordinality,
        NOW(),
        NOW()
    FROM expanded_fields ef;

END $$;

COMMIT;
