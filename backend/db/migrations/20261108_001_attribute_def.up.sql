-- Attribute definitions (control plane) live on alpha alongside catalog_node/edge.
-- Values remain on entity tables' custom_attributes JSONB (data plane).

CREATE TABLE IF NOT EXISTS public.attribute_def (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL,
    core_id          uuid REFERENCES public.attribute_def(id) ON DELETE SET NULL,
    is_shadow        boolean NOT NULL DEFAULT false,
    entity_type      varchar(50) NOT NULL,
    table_ref        varchar(200) NOT NULL,
    field_cd         varchar(100) NOT NULL,
    name             varchar(250) NOT NULL,
    description      text NOT NULL DEFAULT '',
    data_type        varchar(30) NOT NULL,
    json_path        text NOT NULL,
    is_required      boolean NOT NULL DEFAULT false,
    is_searchable    boolean NOT NULL DEFAULT true,
    is_pii           boolean NOT NULL DEFAULT false,
    validation_rules jsonb NOT NULL DEFAULT '{}'::jsonb,
    picklist_values  jsonb,
    default_value    text,
    display_order    integer NOT NULL DEFAULT 0,
    section          varchar(100) NOT NULL DEFAULT '',
    is_active        boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_attribute_def UNIQUE (tenant_id, entity_type, field_cd),
    CONSTRAINT chk_attribute_def_data_type CHECK (
        data_type IN (
            'string', 'integer', 'decimal', 'boolean', 'date',
            'enum', 'json', 'object', 'array', 'number', 'numeric', 'bool'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_attribute_def_tenant_entity
    ON public.attribute_def (tenant_id, entity_type)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS idx_attribute_def_table_ref
    ON public.attribute_def (table_ref)
    WHERE is_active;

-- Effective view: tenant CUSTOM rows + CORE (gold / shared-ref) minus shadow/override.
CREATE OR REPLACE VIEW public.attribute_def_effective AS
WITH tenant_ctx AS (
    SELECT COALESCE(
        NULLIF(current_setting('app.current_tenant', true), '')::uuid,
        '00000000-0000-0000-0000-000000000000'::uuid
    ) AS tenant_id,
    COALESCE(
        NULLIF(current_setting('app.shared_reference_tenant', true), '')::uuid,
        (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1),
        '00000000-0000-0000-0000-000000000001'::uuid
    ) AS core_tenant_id
),
shadowed AS (
    SELECT a.core_id
    FROM public.attribute_def a
    CROSS JOIN tenant_ctx t
    WHERE a.tenant_id = t.tenant_id
      AND a.is_shadow = true
      AND a.core_id IS NOT NULL
),
tenant_rows AS (
    SELECT a.*
    FROM public.attribute_def a
    CROSS JOIN tenant_ctx t
    WHERE a.tenant_id = t.tenant_id
      AND a.is_active
      AND NOT a.is_shadow
),
core_rows AS (
    SELECT a.*
    FROM public.attribute_def a
    CROSS JOIN tenant_ctx t
    WHERE a.tenant_id = t.core_tenant_id
      AND t.tenant_id IS DISTINCT FROM t.core_tenant_id
      AND a.is_active
      AND a.id NOT IN (SELECT core_id FROM shadowed)
      AND NOT EXISTS (
          SELECT 1 FROM tenant_rows tr WHERE tr.core_id = a.id
      )
)
SELECT *, 'CUSTOM'::text AS origin, 10 AS precedence_rank FROM tenant_rows
UNION ALL
SELECT *, 'CORE'::text AS origin, 100 AS precedence_rank FROM core_rows;

ALTER TABLE public.attribute_def ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.attribute_def FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS attr_def_read ON public.attribute_def;
CREATE POLICY attr_def_read ON public.attribute_def
    AS PERMISSIVE FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = COALESCE(
            NULLIF(current_setting('app.shared_reference_tenant', true), '')::uuid,
            (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1),
            '00000000-0000-0000-0000-000000000001'::uuid
        )
    );

DROP POLICY IF EXISTS attr_def_write ON public.attribute_def;
CREATE POLICY attr_def_write ON public.attribute_def
    AS PERMISSIVE FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- Node type + edge types for catalog wiring
DO $seed$
DECLARE
    _tenant uuid;
    _table_type_id uuid;
    _col_type_id uuid;
    _cf_type_id uuid;
    _bo_type_id uuid;
BEGIN
    SELECT id INTO _tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF _tenant IS NULL THEN
        RAISE NOTICE 'attribute_def seed: no gold_copy tenant; skipping catalog type seeds';
        RETURN;
    END IF;

    SELECT id INTO _table_type_id FROM public.catalog_node_types WHERE catalog_type_name = 'table' LIMIT 1;
    SELECT id INTO _col_type_id FROM public.catalog_node_types WHERE catalog_type_name = 'column' LIMIT 1;
    SELECT id INTO _bo_type_id FROM public.catalog_node_types WHERE catalog_type_name IN ('business_object', 'BUSINESS_OBJECT') LIMIT 1;

    SELECT id INTO _cf_type_id FROM public.catalog_node_types WHERE catalog_type_name = 'custom_field' LIMIT 1;
    IF _cf_type_id IS NULL THEN
        _cf_type_id := gen_random_uuid();
        INSERT INTO public.catalog_node_types (id, tenant_id, catalog_type_name, description, is_active, config)
        VALUES (
            _cf_type_id, _tenant, 'custom_field',
            'Semantic custom attribute definition bound to a JSONB custom_attributes column',
            true, '{}'::jsonb
        )
        ON CONFLICT DO NOTHING;
        SELECT id INTO _cf_type_id FROM public.catalog_node_types WHERE catalog_type_name = 'custom_field' LIMIT 1;
    END IF;

    IF _cf_type_id IS NOT NULL AND (_bo_type_id IS NOT NULL OR _table_type_id IS NOT NULL) THEN
        INSERT INTO public.catalog_edge_types (
            id, tenant_id, edge_type_name, description,
            source_node_type_id, target_node_type_id, is_directed, is_active, config
        ) VALUES (
            'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid,
            _tenant,
            'BO_HAS_ATTRIBUTE',
            'Business object or table exposes a custom attribute definition',
            COALESCE(_bo_type_id, _table_type_id),
            _cf_type_id,
            true, true, '{}'::jsonb
        )
        ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;
    END IF;

    IF _cf_type_id IS NOT NULL AND _col_type_id IS NOT NULL THEN
        INSERT INTO public.catalog_edge_types (
            id, tenant_id, edge_type_name, description,
            source_node_type_id, target_node_type_id, is_directed, is_active, config
        ) VALUES (
            'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,
            _tenant,
            'ATTRIBUTE_STORED_IN',
            'Custom attribute values are stored in a physical JSONB column',
            _cf_type_id,
            _col_type_id,
            true, true,
            '{
                "storage_kind": {"type":"string","enum":["JSONB_KEY"]},
                "json_path": {"type":"string"}
            }'::jsonb
        )
        ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;
    END IF;
END
$seed$;

-- Safe cast helpers for preview/validation (alpha or any DB that has mdm)
DO $casts$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'mdm') THEN
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_int(text)
            RETURNS int LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN $1 ~ '^-?\d+$' THEN $1::int
                    ELSE NULL
                END
            $$
        $f$;
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_numeric(text)
            RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN $1 ~ '^-?\d+(\.\d+)?([eE][-+]?\d+)?$' THEN $1::numeric
                    ELSE NULL
                END
            $$
        $f$;
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_date(text)
            RETURNS date LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN $1 ~ '^\d{4}-\d{2}-\d{2}$' THEN $1::date
                    ELSE NULL
                END
            $$
        $f$;
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_bool(text)
            RETURNS bool LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN lower($1) IN ('true','t','1','yes','y') THEN true
                    WHEN lower($1) IN ('false','f','0','no','n') THEN false
                    ELSE NULL
                END
            $$
        $f$;
    END IF;
END
$casts$;

-- Persona B (European UCITS) starter definitions on gold-copy tenant
DO $persona$
DECLARE
    _tenant uuid;
BEGIN
    SELECT id INTO _tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF _tenant IS NULL THEN
        _tenant := '00000000-0000-0000-0000-000000000001'::uuid;
    END IF;

    INSERT INTO public.attribute_def (
        tenant_id, entity_type, table_ref, field_cd, name, description, data_type,
        json_path, is_required, is_searchable, validation_rules, picklist_values,
        display_order, section
    ) VALUES
        (_tenant, 'PRODUCT', 'mdm.product', 'sfdr_article', 'SFDR Article',
         'SFDR classification per EU Regulation 2019/2088', 'integer',
         'sfdr_article', false, true, '{}'::jsonb, '[8,9]'::jsonb, 10, 'Regulatory'),
        (_tenant, 'PRODUCT', 'mdm.product', 'tax_alignment_pct', 'Taxonomy Alignment %',
         'EU Taxonomy alignment percentage', 'decimal',
         'tax_alignment_pct', false, true, '{"min":0,"max":100}'::jsonb, NULL, 20, 'Regulatory'),
        (_tenant, 'PRODUCT', 'mdm.product', 'ucits_v_eligible', 'UCITS V Eligible',
         'Whether the product is UCITS V eligible', 'boolean',
         'ucits_v_eligible', false, true, '{}'::jsonb, NULL, 30, 'Regulatory'),
        (_tenant, 'PRODUCT', 'mdm.product', 'esg_score_provider', 'ESG Score Provider',
         'Vendor providing the ESG score', 'string',
         'esg_score_provider', false, true, '{}'::jsonb, NULL, 40, 'ESG'),
        (_tenant, 'PRODUCT', 'mdm.product', 'esg_score', 'ESG Score',
         'Composite ESG score', 'decimal',
         'esg_score', false, true, '{"min":0,"max":10}'::jsonb, NULL, 50, 'ESG'),
        (_tenant, 'PRODUCT', 'mdm.product', 'exclusion_policy', 'Exclusion Policy',
         'Structured exclusion policy object', 'json',
         'exclusion_policy', false, false, '{}'::jsonb, NULL, 60, 'ESG'),
        (_tenant, 'PRODUCT', 'mdm.product', 'distributor_restrictions', 'Distributor Restrictions',
         'Distribution restriction flags', 'json',
         'distributor_restrictions', false, false, '{}'::jsonb, NULL, 70, 'Distribution'),
        (_tenant, 'PRODUCT', 'mdm.product', 'fatca_registered', 'FATCA Registered',
         'FATCA registration status', 'boolean',
         'fatca_registered', false, true, '{}'::jsonb, NULL, 80, 'Distribution'),
        (_tenant, 'PRODUCT', 'mdm.product', 'internal_cost_center', 'Internal Cost Center',
         'Internal cost center code', 'string',
         'internal_cost_center', false, true, '{}'::jsonb, NULL, 90, 'Custom'),
        (_tenant, 'PRODUCT', 'mdm.product', 'fund_family_code', 'Fund Family Code',
         'Internal fund family grouping code', 'string',
         'fund_family_code', false, true, '{}'::jsonb, NULL, 100, 'Custom')
    ON CONFLICT (tenant_id, entity_type, field_cd) DO NOTHING;
END
$persona$;
