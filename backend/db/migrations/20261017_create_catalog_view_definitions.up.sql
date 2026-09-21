-- Tenant-scoped, JSONB-configurable graph view definitions (ERD, semantic lineage,
-- taxonomy lineage, and tenant-authored custom views), mirroring the config-JSONB
-- idiom already used by catalog_node_types/catalog_edge_types and
-- business_objects.subtypes_config.

CREATE TABLE IF NOT EXISTS catalog_view_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    view_key VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_core BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, view_key)
);

CREATE INDEX IF NOT EXISTS idx_catalog_view_definitions_tenant ON catalog_view_definitions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_catalog_view_definitions_config ON catalog_view_definitions USING GIN(config);

COMMENT ON TABLE catalog_view_definitions IS
  'Tenant-scoped graph visualization definitions. config JSONB holds typePolicy (which node/edge types to include/exclude), grouping (cluster rules for high-cardinality fan-outs), layout (algorithm/direction), and assignedAssetTypes (which BO subtypes / node types this view applies to).';
