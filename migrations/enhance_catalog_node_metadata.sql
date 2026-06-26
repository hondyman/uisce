-- Add metadata columns to catalog_node for NLQ and Governance
ALTER TABLE catalog_node ADD COLUMN IF NOT EXISTS lineage JSONB DEFAULT '{}'::jsonb;
ALTER TABLE catalog_node ADD COLUMN IF NOT EXISTS data_quality_contract JSONB DEFAULT '{}'::jsonb;
ALTER TABLE catalog_node ADD COLUMN IF NOT EXISTS sla JSONB DEFAULT '{}'::jsonb;

-- Add indexes for these JSONB columns to support efficient querying if needed
CREATE INDEX IF NOT EXISTS idx_catalog_node_lineage ON catalog_node USING GIN (lineage);
CREATE INDEX IF NOT EXISTS idx_catalog_node_dq_contract ON catalog_node USING GIN (data_quality_contract);
