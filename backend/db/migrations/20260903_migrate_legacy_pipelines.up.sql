-- Migrate rows from the legacy ReactFlow `pipelines` table (owned by the
-- retired internal/handlers.PipelineHandler) into `data_pipeline_definitions`
-- (owned by internal/datapipeline, now the single pipeline system).
--
-- This performs a best-effort structural reshape directly in SQL: each
-- legacy ReactFlow node becomes a datapipeline PipelineNode (type inferred
-- from node.type / data.filterType, falling back to "transform"), and each
-- ReactFlow edge becomes a PipelineEdge. This covers the common case of
-- simple linear/branchless canvases.
--
-- NOTE: This SQL migration cannot run the richer Go label-inference heuristic
-- in internal/datapipeline/legacy_convert.go#ConvertLegacyPipelineJSON
-- (e.g. mapping "Validation"-labelled nodes to type "validator", or
-- "External"-labelled nodes to an api_caller transform). Operators migrating
-- production data with meaningfully-labelled legacy nodes should instead run
-- the equivalent one-off conversion via that Go function (e.g. from a small
-- script/REPL importing internal/datapipeline) before/after this migration
-- to reclassify node types where the heuristic matters. This migration is
-- safe to run either way: it never drops data, and unmigrated/misclassified
-- nodes still round-trip as generic "transform" steps.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pipelines') THEN
        INSERT INTO data_pipeline_definitions (
            id, tenant_id, name, description, mode, target_entity, dag_json,
            concurrency, batch_size, error_policy, is_active, created_by, created_at, last_modified_at
        )
        SELECT
            p.id::uuid,
            p.tenant_id::uuid,
            p.name,
            COALESCE(p.description, ''),
            'business_object',
            COALESCE(p.business_object, ''),
            jsonb_build_object(
                'nodes', COALESCE((
                    SELECT jsonb_agg(jsonb_build_object(
                        'id', n->>'id',
                        'type', CASE
                            WHEN n->>'type' IN ('source', 'reader') THEN 'source'
                            WHEN n->>'type' IN ('loader', 'writer', 'sink') THEN 'loader'
                            WHEN COALESCE(n->'data'->>'filterType', '') = 'validate' THEN 'validator'
                            ELSE 'transform'
                        END,
                        'subType', COALESCE(n->'data'->>'filterType', ''),
                        'label', COALESCE(n->'data'->>'label', ''),
                        'config', COALESCE(n->'data'->'config', '{}'::jsonb) ||
                                  jsonb_build_object('legacyType', COALESCE(n->'data'->>'filterType', '')),
                        'position', COALESCE(n->'position', jsonb_build_object('x', 0, 'y', 0))
                    ))
                    FROM jsonb_array_elements(p.pipeline_json->'nodes') n
                ), '[]'::jsonb),
                'edges', COALESCE((
                    SELECT jsonb_agg(jsonb_build_object(
                        'id', COALESCE(e->>'id', (e->>'source') || '-' || (e->>'target')),
                        'source', e->>'source',
                        'target', e->>'target',
                        'label', COALESCE(e->>'label', '')
                    ))
                    FROM jsonb_array_elements(p.pipeline_json->'edges') e
                ), '[]'::jsonb)
            ),
            8,
            2000,
            'skip_and_log',
            p.is_active,
            p.created_by,
            p.created_at,
            p.last_modified_at
        FROM pipelines p
        WHERE NOT EXISTS (
            SELECT 1 FROM data_pipeline_definitions d WHERE d.id = p.id::uuid
        );
    END IF;
END $$;
