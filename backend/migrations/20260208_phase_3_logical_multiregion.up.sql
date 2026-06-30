-- Phase 3.1: Logical Multi-Region Architecture
-- Add region as a first-class dimension across core metadata tables
-- These columns prepare the system for region-aware routing, RCA, and future CockroachDB migration

-- Add region to incidents table
ALTER TABLE IF EXISTS ops_incidents ADD COLUMN IF NOT EXISTS region VARCHAR(50);
CREATE INDEX IF NOT EXISTS idx_ops_incidents_region ON ops_incidents(region);
CREATE INDEX IF NOT EXISTS idx_ops_incidents_region_created_at ON ops_incidents(region, created_at DESC);

-- Add region to ops_events table (if not already present)
-- Note: ops_events.region may already exist, but we ensure indexes
CREATE INDEX IF NOT EXISTS idx_ops_events_region ON ops_events(region);
CREATE INDEX IF NOT EXISTS idx_ops_events_region_occurred_at ON ops_events(region, occurred_at DESC);

-- Add region to action history for region-scoped auditing
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'ops_action_history') THEN
        ALTER TABLE public.ops_action_history ADD COLUMN IF NOT EXISTS region VARCHAR(50);
        CREATE INDEX IF NOT EXISTS idx_ops_action_history_region ON ops_action_history(region);
        CREATE INDEX IF NOT EXISTS idx_ops_action_history_region_created_at ON ops_action_history(region, created_at DESC);
        COMMENT ON COLUMN ops_action_history.region IS 'Geographic region where action was executed';
    END IF;
END $$;

-- Add region to audit logs (already done in Phase 2.4c, but ensure it's there)
CREATE INDEX IF NOT EXISTS idx_ops_audit_log_region ON ops_audit_log(region);

-- Future tables that will also receive region columns:
-- - tenants: REGIONAL BY ROW (when migrating to CockroachDB)
-- - endpoints: region prefix in name
-- - semantic_objects: region for pre-aggregation placement
-- - preaggregations: region for data locality
-- - starrocks_clusters: regional deployment
-- - redpanda_topics: regional topic namespace
-- - temporal_namespaces: regional namespace isolation

COMMENT ON COLUMN ops_incidents.region IS 'Geographic region for this incident (e.g., us-east-1, eu-west-1, ap-southeast-1)';

-- Add metadata table to track region configuration
CREATE TABLE IF NOT EXISTS region_config (
    id UUID PRIMARY KEY,
    region_code VARCHAR(50) NOT NULL UNIQUE,
    region_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for region config
CREATE INDEX IF NOT EXISTS idx_region_config_active ON region_config(is_active);
CREATE INDEX IF NOT EXISTS idx_region_config_code ON region_config(region_code);

-- Seed default regions (can be extended)
INSERT INTO region_config (id, region_code, region_name, description, is_active)
VALUES
    (gen_random_uuid(), 'us-east-1', 'US East (N. Virginia)', 'Primary US eastern region', true),
    (gen_random_uuid(), 'us-west-2', 'US West (Oregon)', 'Primary US western region', true),
    (gen_random_uuid(), 'eu-west-1', 'EU (Ireland)', 'Primary EU region', true),
    (gen_random_uuid(), 'ap-southeast-1', 'AP (Singapore)', 'Primary Asia-Pacific region', true)
ON CONFLICT DO NOTHING;

-- Add routing metadata table for region-aware service discovery
CREATE TABLE IF NOT EXISTS region_routing (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    region VARCHAR(50) NOT NULL,
    starrocks_cluster VARCHAR(255),       -- StarRocks cluster in this region
    redpanda_broker VARCHAR(255),         -- Redpanda broker in this region
    temporal_namespace VARCHAR(255),      -- Temporal namespace in this region
    ops_worker_pool VARCHAR(255),         -- Worker pool for ops execution
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, region)
);

CREATE INDEX IF NOT EXISTS idx_region_routing_tenant_region ON region_routing(tenant_id, region);
CREATE INDEX IF NOT EXISTS idx_region_routing_region ON region_routing(region);

COMMENT ON TABLE region_routing IS 'Maps tenants to region-specific service endpoints for routing intelligence';
COMMENT ON COLUMN region_routing.tenant_id IS 'Tenant ID this routing applies to';
COMMENT ON COLUMN region_routing.region IS 'Geographic region';
COMMENT ON COLUMN region_routing.starrocks_cluster IS 'StarRocks cluster address in this region';
COMMENT ON COLUMN region_routing.redpanda_broker IS 'Redpanda broker address(es) in this region';
COMMENT ON COLUMN region_routing.temporal_namespace IS 'Temporal namespace for workflows in this region';
COMMENT ON COLUMN region_routing.ops_worker_pool IS 'Worker pool for executing ops actions in this region';
