-- Add UNIQUE constraint on catalog_node(tenant_id, qualified_path).
-- The catalog writer code (subtype_bo_builder.go, sti_column_scanner.go) uses
--   INSERT ... ON CONFLICT (tenant_id, qualified_path) DO UPDATE ...
-- which requires this constraint. It was added manually to the live DB but
-- never committed as a migration. This file reconciles that gap.
--
-- Schema-adaptive: detects whether catalog_node lives in public (pre-refactor)
-- or metadata (post-refactor, per manual_adopt/015_refactor_schemas.sql.up.sql)
-- and adds the constraint to the correct schema. This was necessary because the
-- snapshot (public) and the refactor migration (metadata) genuinely disagree for
-- environments that have run both; a future migration should resolve this
-- ambiguity permanently rather than extending the adaptive pattern.
--
-- Idempotent: the pg_constraint check prevents a redundant ADD CONSTRAINT.
-- If duplicate (tenant_id, qualified_path) rows exist, ADD CONSTRAINT fails with
-- ERROR: could not create unique index "catalog_node_tenant_path_uniq" — in that
-- case a dedup migration (repoint edges, delete duplicates) must run first.
--
-- Duplicate check was NOT pre-run: no accessible DB with catalog_node data was
-- available at time of writing (local alpha has no catalog_node; remote requires
-- mTLS certs not on developer machines). The constraint will fail loudly if
-- duplicates are present rather than silently creating a broken state.
--
-- Verification query (run before this migration in any environment that may have
-- existing data):
--   SELECT tenant_id, qualified_path, COUNT(*) AS dupes
--   FROM <schema>.catalog_node
--   GROUP BY 1, 2 HAVING COUNT(*) > 1;

DO $$
DECLARE
    target_schema TEXT;
    constraint_exists BOOL;
BEGIN
    -- Detect which schema holds catalog_node (public = pre-refactor, metadata = post-refactor)
    SELECT nspname INTO target_schema
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relname = 'catalog_node'
      AND c.relkind = 'r'
    LIMIT 1;

    IF target_schema IS NULL THEN
        RAISE NOTICE 'catalog_node table not found in any schema; skipping constraint add';
        RETURN;
    END IF;

    -- Probe pg_constraint with the resolved schema-qualified oid so we don't
    -- accidentally match a same-named constraint on a catalog_node in another
    -- schema (the table-name check above is the right scope, not a global name search).
    SELECT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = (target_schema || '.catalog_node')::regclass
          AND conname   = 'catalog_node_tenant_path_uniq'
          AND contype   = 'u'
    ) INTO constraint_exists;

    IF NOT constraint_exists THEN
        EXECUTE format(
            'ALTER TABLE %I.catalog_node ADD CONSTRAINT catalog_node_tenant_path_uniq UNIQUE (tenant_id, qualified_path)',
            target_schema
        );
        RAISE NOTICE 'Added catalog_node_tenant_path_uniq to %.catalog_node', target_schema;
    ELSE
        RAISE NOTICE 'catalog_node_tenant_path_uniq already exists on %.catalog_node; no action taken', target_schema;
    END IF;
END
$$;
