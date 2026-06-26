-- Semantic ChangeSet Table for Promotions with ASO Integration

CREATE TABLE IF NOT EXISTS semantic.changeset (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    source_env text NOT NULL CHECK (source_env IN ('dev', 'staging', 'prod')),
    target_env text NOT NULL CHECK (target_env IN ('dev', 'staging', 'prod')),
    
    -- Status lifecycle
    status text NOT NULL DEFAULT 'draft' CHECK (status IN (
        'draft',
        'pending_validation',
        'validated',
        'pending_approval',
        'approved',
        'applied',
        'rejected',
        'failed'
    )),
    
    -- Changes as JSON array
    changes_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    description text NOT NULL DEFAULT '',
    
    -- ASO integration
    aso_source boolean NOT NULL DEFAULT false,
    aso_validation_result jsonb,
    
    -- Audit
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    approved_by text,
    approved_at timestamptz,
    applied_at timestamptz,
    rejected_by text,
    rejected_at timestamptz,
    rejection_reason text
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_changeset_status ON semantic.changeset(status);
CREATE INDEX IF NOT EXISTS idx_changeset_tenant ON semantic.changeset(tenant_id);
CREATE INDEX IF NOT EXISTS idx_changeset_source_env ON semantic.changeset(source_env);
CREATE INDEX IF NOT EXISTS idx_changeset_target_env ON semantic.changeset(target_env);
CREATE INDEX IF NOT EXISTS idx_changeset_created ON semantic.changeset(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_changeset_aso ON semantic.changeset(aso_source) WHERE aso_source = true;

-- Changeset audit log
CREATE TABLE IF NOT EXISTS semantic.changeset_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    changeset_id uuid NOT NULL REFERENCES semantic.changeset(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN (
        'created',
        'validated',
        'approved',
        'applied',
        'rejected',
        'failed'
    )),
    actor text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_changeset_audit_cs ON semantic.changeset_audit(changeset_id);
CREATE INDEX IF NOT EXISTS idx_changeset_audit_created ON semantic.changeset_audit(created_at DESC);
