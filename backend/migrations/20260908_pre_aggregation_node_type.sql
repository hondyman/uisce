-- Registers the "pre_aggregation" catalog_node_type. PreAggregationService
-- (internal/analytics/pre_aggregation_service.go) has always modeled
-- pre-aggregations as catalog_node rows of this type - GenerateDDL,
-- ListByBO, invalidation (PreAggInvalidationListener), and the Temporal
-- refresh workflow (temporal/workflows/preagg_refresh_workflow.go) all
-- assume it exists - but the seed row was never inserted, so
-- UpsertPreAggregation always failed with "pre_aggregation node type not
-- found". This is what wiring the new /api/preaggregations handler
-- (internal/handlers/preaggregation_handler.go) surfaced.
INSERT INTO catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, created_at, updated_at)
VALUES (
    'a1b2c3d4-0000-4000-8000-000000000001',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'pre_aggregation',
    'StarRocks hot-tier rollup definition for one or more calculated semantic terms, materialized off the CDC pipeline',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
