-- Control-plane source hierarchy (Market EDM / GoldenSource / Asset Control style).
-- Rules are keyed by semantic_term_id so mastering and validation share the same contract.
-- Lives on alpha; never write masters here.

CREATE TABLE IF NOT EXISTS public.mdm_source_systems (
    code          varchar(50) PRIMARY KEY,
    display_name  varchar(100) NOT NULL,
    default_rank  integer NOT NULL DEFAULT 100,
    description   text NOT NULL DEFAULT '',
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE public.mdm_source_systems IS
    'Platform registry of MDM source systems for survivorship priority pickers (GOLDENSOURCE, MARKET_EDM, …)';

INSERT INTO public.mdm_source_systems (code, display_name, default_rank, description) VALUES
    ('GOLDENSOURCE',  'GoldenSource',  10, 'Enterprise MDM vendor — often authoritative for legal entity / account identifiers'),
    ('MARKET_EDM',    'Market EDM',    20, 'Markit / Market EDM — market and reference data hierarchy'),
    ('ASSET_CONTROL', 'Asset Control', 30, 'Asset Control — security and pricing-oriented MDM'),
    ('BLOOMBERG',     'Bloomberg',     40, 'Bloomberg vendor feed'),
    ('REFINITIV',     'Refinitiv',     50, 'Refinitiv / LSEG vendor feed'),
    ('CRIMS',         'CRIMS',         60, 'Charles River / internal OMS account source'),
    ('INTERNAL',      'Internal',      90, 'Firm-internal or manual steward source')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS public.semantic_survivorship_rules (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL,
    entity_type        varchar(50) NOT NULL,
    semantic_term_id   uuid NOT NULL,
    strategy           varchar(40) NOT NULL DEFAULT 'SOURCE_PRIORITY',
    priority_order     text[] NOT NULL DEFAULT '{}',
    max_stale_seconds  integer NOT NULL DEFAULT 0,
    is_active          boolean NOT NULL DEFAULT true,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_semantic_survivorship_term
        UNIQUE (tenant_id, entity_type, semantic_term_id),
    CONSTRAINT chk_semantic_survivorship_strategy CHECK (
        strategy IN (
            'SOURCE_PRIORITY',
            'MOST_RECENT',
            'CONSERVATIVE_MIN',
            'CONSERVATIVE_MAX',
            'WEIGHTED_CONFIDENCE'
        )
    ),
    CONSTRAINT chk_semantic_survivorship_stale CHECK (max_stale_seconds >= 0)
);

CREATE INDEX IF NOT EXISTS idx_semantic_surv_tenant_entity
    ON public.semantic_survivorship_rules (tenant_id, entity_type)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS idx_semantic_surv_term
    ON public.semantic_survivorship_rules (semantic_term_id)
    WHERE is_active;

COMMENT ON TABLE public.semantic_survivorship_rules IS
    'Per-tenant survivorship rules keyed by catalog semantic_term_id; physical fields resolve via attribute_def / BO term bindings';

COMMENT ON COLUMN public.semantic_survivorship_rules.semantic_term_id IS
    'catalog_node.id of type semantic_term — required; orphan field_cd rules are refused by the API';

COMMENT ON COLUMN public.semantic_survivorship_rules.priority_order IS
    'Ordered source system codes (first wins for SOURCE_PRIORITY), e.g. {GOLDENSOURCE,MARKET_EDM,ASSET_CONTROL,INTERNAL}';

ALTER TABLE public.semantic_survivorship_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.semantic_survivorship_rules FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS semantic_surv_read ON public.semantic_survivorship_rules;
CREATE POLICY semantic_surv_read ON public.semantic_survivorship_rules
    AS PERMISSIVE FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = COALESCE(
            NULLIF(current_setting('app.shared_reference_tenant', true), '')::uuid,
            (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1),
            '00000000-0000-0000-0000-000000000001'::uuid
        )
        OR tenant_id = '00000000-0000-0000-0000-000000000000'::uuid
    );

DROP POLICY IF EXISTS semantic_surv_write ON public.semantic_survivorship_rules;
CREATE POLICY semantic_surv_write ON public.semantic_survivorship_rules
    AS PERMISSIVE FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- Seed Account semantic terms + default SOURCE_PRIORITY hierarchy for the demo tenant.
DO $seed$
DECLARE
    _gold uuid;
    _demo uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _term_type_id uuid;
    _priority text[] := ARRAY['GOLDENSOURCE', 'MARKET_EDM', 'ASSET_CONTROL', 'INTERNAL'];
    _fields text[] := ARRAY[
        'account_cd', 'account_name', 'account_type_cd', 'status_cd',
        'base_currency', 'domicile', 'custodian_id', 'manager_id',
        'holder_party_id', 'client_group_id', 'opened_date', 'closed_date'
    ];
    _f text;
    _term_id uuid;
    _path text;
BEGIN
    SELECT id INTO _gold FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF _gold IS NULL THEN
        _gold := _demo;
    END IF;

    SELECT id INTO _term_type_id FROM public.catalog_node_types
      WHERE catalog_type_name IN ('semantic_term', 'SEMANTIC_TERM') LIMIT 1;
    IF _term_type_id IS NULL THEN
        RAISE NOTICE 'semantic_survivorship seed: no semantic_term node type — skipping term/rule seed';
        RETURN;
    END IF;

    FOREACH _f IN ARRAY _fields LOOP
        _path := 'semantic/account/' || _f;
        SELECT id INTO _term_id
        FROM public.catalog_node
        WHERE tenant_id = _gold
          AND node_type_id = _term_type_id
          AND qualified_path = _path
        LIMIT 1;

        IF _term_id IS NULL THEN
            INSERT INTO public.catalog_node (
                id, tenant_id, node_type_id, node_name, description,
                qualified_path, is_active, properties
            ) VALUES (
                gen_random_uuid(),
                _gold,
                _term_type_id,
                _f,
                'Account master field: ' || _f,
                _path,
                true,
                jsonb_build_object('entity_type', 'ACCOUNT', 'field_cd', _f, 'source', 'survivorship_seed')
            )
            RETURNING id INTO _term_id;
        END IF;

        INSERT INTO public.semantic_survivorship_rules (
            tenant_id, entity_type, semantic_term_id, strategy, priority_order, max_stale_seconds, is_active
        ) VALUES (
            _demo, 'ACCOUNT', _term_id, 'SOURCE_PRIORITY', _priority, 0, true
        )
        ON CONFLICT (tenant_id, entity_type, semantic_term_id) DO UPDATE
            SET strategy = EXCLUDED.strategy,
                priority_order = EXCLUDED.priority_order,
                is_active = true,
                updated_at = now();

        -- Bind matching attribute_def rows (custom attrs) when field_cd matches.
        UPDATE public.attribute_def
           SET semantic_term_id = _term_id,
               updated_at = now()
         WHERE entity_type = 'ACCOUNT'
           AND field_cd = _f
           AND semantic_term_id IS NULL;
    END LOOP;
END
$seed$;
