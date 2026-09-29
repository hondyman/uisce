-- 20260903_001_validation_rule_versions.up.sql
-- Bitemporal versioning side table for validation rules.
-- Supports valid-time (business effectiveness) and transaction-time (recorded audit history).

CREATE TABLE IF NOT EXISTS validation_rule_versions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_node_id    uuid NOT NULL,              -- FK -> catalog_node.id
    tenant_id       uuid NOT NULL,              -- tenant scoping
    version         int  NOT NULL,              -- monotonic per rule_node_id
    rule_ast        jsonb NOT NULL,             -- canonical (vm.Compact) AST snapshot
    properties      jsonb NOT NULL,             -- full ValidationRuleProperties snapshot
    checksum        text NOT NULL,              -- canonical AST checksum (porter-compatible)
    
    -- Valid time (business effectiveness)
    effective_from  timestamptz NOT NULL,
    effective_to    timestamptz,                -- NULL = currently effective
    
    -- Transaction time (recorded history)
    recorded_at     timestamptz NOT NULL DEFAULT NOW(),
    superseded_at   timestamptz,                -- NULL = current recorded version
    
    -- Governance lineage
    governance_status text NOT NULL DEFAULT 'published',
    published_by    text,
    approved_by     text,
    parent_version  int,                        -- edit lineage
    
    CONSTRAINT uq_rule_version_recorded UNIQUE (rule_node_id, version, recorded_at)
);

CREATE INDEX IF NOT EXISTS idx_vrv_asof ON validation_rule_versions
    (rule_node_id, effective_from, effective_to, recorded_at, superseded_at);

CREATE INDEX IF NOT EXISTS idx_vrv_current ON validation_rule_versions (rule_node_id)
    WHERE superseded_at IS NULL AND effective_to IS NULL;

CREATE INDEX IF NOT EXISTS idx_vrv_tenant_lookup ON validation_rule_versions
    (tenant_id, rule_node_id, version);
