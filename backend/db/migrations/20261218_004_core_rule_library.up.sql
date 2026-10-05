-- 20261218_004_core_rule_library.up.sql
--
-- Core Gold-Copy Rule Library Schema Extensions:
-- 1. Effective Dating, Citations, Jurisdictions & Provenance on compliance.compliance_rule
-- 2. Tenant Unique Constraint on (tenant_id, rule_code) for idempotent upserts
-- 3. Library Status ('ACTIVE', 'PROVISIONAL', 'DEPRECATED', 'STALE_REGULATION')
-- 4. Extended Governance Audit Event Types (RULE_ACTIVATED, RULE_DEACTIVATED, REGULATORY_CHANGE_*)
-- 5. Tenant Opt-in Activation Matrix (compliance.tenant_rule_activation)
-- 6. Licensable Ruleset Membership Table (compliance.compliance_ruleset_membership)
-- 7. Idempotent Gold-Copy Master Tenant & Catalog Node Type Registration

-- 1. Effective dating + provenance + library status on compliance_rule
ALTER TABLE compliance.compliance_rule
    ADD COLUMN IF NOT EXISTS effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS effective_to   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS citation       TEXT,
    ADD COLUMN IF NOT EXISTS jurisdictions  TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS source_version TEXT,
    ADD COLUMN IF NOT EXISTS library_status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (library_status IN ('ACTIVE', 'PROVISIONAL', 'DEPRECATED', 'STALE_REGULATION'));

-- Idempotency constraint: exactly one rule code per tenant
CREATE UNIQUE INDEX IF NOT EXISTS unq_compliance_rule_tenant_code
    ON compliance.compliance_rule (tenant_id, rule_code);

-- Index for point-in-time evaluation
CREATE INDEX IF NOT EXISTS idx_compliance_rule_tenant_effective
    ON compliance.compliance_rule (tenant_id, is_active, effective_from, effective_to)
    WHERE is_active = true;

-- 2. Extend event_type check constraint on governance_audit_event
ALTER TABLE compliance.governance_audit_event 
    DROP CONSTRAINT IF EXISTS governance_audit_event_event_type_check;

ALTER TABLE compliance.governance_audit_event 
    ADD CONSTRAINT governance_audit_event_event_type_check 
    CHECK (event_type IN (
        'CORE_VERSION_PUBLISHED', 'RULE_REPINNED', 'DRIFT_RECONCILED', 'THRESHOLD_OVERRIDE',
        'RULE_ACTIVATED', 'RULE_DEACTIVATED',
        'REGULATORY_CHANGE_INTAKED', 'REGULATORY_CHANGE_TRIAGED', 'REGULATORY_CHANGE_PUBLISHED', 'REGULATORY_CHANGE_CLOSED'
    ));

-- 3. Tenant activation matrix: opt-in by default
CREATE TABLE IF NOT EXISTS compliance.tenant_rule_activation (
    tenant_id      UUID NOT NULL,
    rule_id        UUID NOT NULL REFERENCES compliance.compliance_rule(id) ON DELETE RESTRICT,
    enabled        BOOLEAN NOT NULL DEFAULT false,
    inherit_mode   TEXT NOT NULL DEFAULT 'inherit'
        CHECK (inherit_mode IN ('inherit', 'extend', 'custom')),
    activated_by   TEXT,
    activated_at   TIMESTAMPTZ,
    audit_required BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_compliance_tenant_activation_lookup
    ON compliance.tenant_rule_activation (tenant_id, enabled);

ALTER TABLE compliance.tenant_rule_activation ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.tenant_rule_activation FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_activation ON compliance.tenant_rule_activation
    FOR ALL USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 4. Ruleset membership for licensable bundles (CORE_REGULATORY, MARKET_CONDUCT, INSTITUTIONAL_CONTROLS)
CREATE TABLE IF NOT EXISTS compliance.compliance_ruleset_membership (
    ruleset_code TEXT NOT NULL,
    rule_id      UUID NOT NULL REFERENCES compliance.compliance_rule(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ruleset_code, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_compliance_ruleset_lookup
    ON compliance.compliance_ruleset_membership (ruleset_code);

-- 5. Scope Least-Privilege Grants for app_user
GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.tenant_rule_activation TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.compliance_ruleset_membership TO app_user;

-- 6. Idempotent Master Gold-Copy Tenant & Catalog Node Types Registration
DO $$
DECLARE
    v_gold_id UUID;
BEGIN
    SELECT id INTO v_gold_id FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_id IS NULL THEN
        BEGIN
            SELECT public.uisce_gold_copy_tenant_id() INTO v_gold_id;
        EXCEPTION WHEN OTHERS THEN
            v_gold_id := NULL;
        END;
    END IF;

    IF v_gold_id IS NULL THEN
        INSERT INTO public.tenants (
            id, name, display_name, description, gold_copy, is_active, status, plan, is_suspended, is_deleted
        ) VALUES (
            '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid,
            'northwind',
            'Northwind Traders (Gold Copy)',
            'Master Gold Copy Template Tenant',
            true,
            true,
            'active',
            'enterprise',
            false,
            false
        )
        ON CONFLICT (id) DO UPDATE SET gold_copy = true;

        v_gold_id := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    END IF;

    -- Also ensure standard core template tenant exists for FK safety
    INSERT INTO public.tenants (
        id, name, display_name, description, gold_copy, is_active, status, plan, is_suspended, is_deleted
    ) VALUES (
        '00000000-0000-4000-a000-000000000000'::uuid,
        'core_template',
        'Core Compliance Library Master',
        'Master Template Tenant for Core Compliance Rules',
        false,
        true,
        'active',
        'enterprise',
        false,
        false
    )
    ON CONFLICT (id) DO NOTHING;

    -- Ensure catalog_node_types are registered
    INSERT INTO public.catalog_node_types (
        id, tenant_id, catalog_type_name, description, is_active, config
    ) VALUES 
        ('e39856ec-e9e2-4151-836a-cc93b801fe6c'::uuid, v_gold_id, 'validation_rule', 'Compliance validation rule', true, '{}'::jsonb),
        ('5ed74b46-137a-424c-8aec-e84db8dcfcdf'::uuid, v_gold_id, 'rule_bundle', 'Compliance ruleset bundle', true, '{}'::jsonb)
    ON CONFLICT (id) DO NOTHING;
END $$;
