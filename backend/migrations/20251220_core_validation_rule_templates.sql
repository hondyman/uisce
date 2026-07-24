-- Core validation rule templates (tenant-agnostic) + tenant instances with inheritance/lineage.

-- Core rules: versioned, platform-owned templates.
CREATE TABLE IF NOT EXISTS public.catalog_validation_rule_cores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_key TEXT NOT NULL,
    version INTEGER NOT NULL,

    rule_name TEXT NOT NULL,
    rule_type TEXT NOT NULL,
    description TEXT,

    -- Targeting (kept aligned with tenant rule schema)
    target_entity TEXT,
    target_entity_id UUID,
    target_entities TEXT[],
    target_entity_ids UUID[],

    condition_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    script_content TEXT,

    severity TEXT NOT NULL DEFAULT 'error',
    status TEXT NOT NULL DEFAULT 'active',

    -- When true, platform intends this core rule to be non-customizable.
    is_core_locked BOOLEAN NOT NULL DEFAULT false,

    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(rule_key, version),
    CONSTRAINT catalog_validation_rule_cores_status_check CHECK (status IN ('draft', 'active', 'deprecated'))
);

CREATE INDEX IF NOT EXISTS idx_catalog_validation_rule_cores_rule_key ON public.catalog_validation_rule_cores(rule_key);
CREATE INDEX IF NOT EXISTS idx_catalog_validation_rule_cores_status ON public.catalog_validation_rule_cores(status);
CREATE INDEX IF NOT EXISTS idx_catalog_validation_rule_cores_rule_type ON public.catalog_validation_rule_cores(rule_type);

COMMENT ON TABLE public.catalog_validation_rule_cores IS 'Tenant-agnostic, versioned core templates for validation rules (platform-delivered).';
COMMENT ON COLUMN public.catalog_validation_rule_cores.rule_key IS 'Stable identifier for a core rule across versions (e.g. account.max_single_security_weight).';
COMMENT ON COLUMN public.catalog_validation_rule_cores.version IS 'Monotonic version for a given rule_key; core rules are never edited in place once active.';

-- Tenant rule instances: add lineage + inheritance mode.
ALTER TABLE IF EXISTS public.catalog_validation_rules
    ADD COLUMN IF NOT EXISTS core_rule_id UUID REFERENCES public.catalog_validation_rule_cores(id),
    ADD COLUMN IF NOT EXISTS inherit_mode TEXT NOT NULL DEFAULT 'custom',
    ADD COLUMN IF NOT EXISTS created_from_core_version INTEGER,
    ADD COLUMN IF NOT EXISTS core_version_pin INTEGER,

    -- For extend mode, store only the tenant extension piece (optional).
    ADD COLUMN IF NOT EXISTS extension_script_content TEXT,
    ADD COLUMN IF NOT EXISTS extension_condition_json JSONB,

    -- Copy of lock intent for the tenant instance (UI/API enforcement), derived from core.
    ADD COLUMN IF NOT EXISTS is_core_locked BOOLEAN NOT NULL DEFAULT false;

-- Constrain inherit_mode.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'catalog_validation_rules_inherit_mode_check'
    ) THEN
        ALTER TABLE IF EXISTS public.catalog_validation_rules
            ADD CONSTRAINT catalog_validation_rules_inherit_mode_check
            CHECK (inherit_mode IN ('inherit', 'extend', 'override', 'custom'));
    END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_validation_rules') THEN
    CREATE INDEX IF NOT EXISTS idx_catalog_validation_rules_core_rule_id ON public.catalog_validation_rules(core_rule_id);
    CREATE INDEX IF NOT EXISTS idx_catalog_validation_rules_inherit_mode ON public.catalog_validation_rules(inherit_mode);
  END IF;
END$$;
