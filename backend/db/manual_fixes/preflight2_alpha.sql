-- preflight2_alpha.sql
-- Read-only. Closes the four unknowns surfaced by Step 0.
\set ON_ERROR_STOP on
\timing on

\echo '=== 0.5.1 alpha.public table count ==='
SELECT count(*) AS public_tables
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE';

\echo '=== 0.5.2 alpha.public tables grouped by prefix (before first _) ==='
SELECT
    split_part(table_name, '_', 1) AS prefix,
    count(*)                       AS tables
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE'
GROUP BY prefix
ORDER BY tables DESC, prefix;

\echo '=== 0.5.3 alpha.public fabric-relevant tables ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE'
  AND (
        table_name LIKE 'business_object%'
     OR table_name LIKE 'catalog_%'
     OR table_name LIKE 'ref_%'
     OR table_name LIKE 'survivorship_rule%'
     OR table_name LIKE 'match_rule%'
     OR table_name LIKE 'dq_rule%'
     OR table_name LIKE 'field_mapping%'
     OR table_name LIKE 'source_priority%'
     OR table_name LIKE 'type_mapping%'
     OR table_name LIKE 'feed_schedule%'
     OR table_name LIKE 'okf_%'
     OR table_name LIKE 'semantic_%'
     OR table_name LIKE 'tenant_registry%'
     OR table_name LIKE 'fabric_version%'
     OR table_name LIKE 'fabric_changeset%'
     OR table_name LIKE 'fabric_publication%'
     OR table_name LIKE 'core_delta_precedence%'
     OR table_name LIKE 'referential_integrity_rule%'
     OR table_name = 'tenants'
  )
ORDER BY table_name;

\echo '=== 0.5.4 alpha.mdm every FK to public.tenants (confirm count) ==='
SELECT
    c.conname                   AS constraint_name,
    c.conrelid::regclass::text  AS from_table,
    c.confrelid::regclass::text AS to_table
FROM pg_constraint c
WHERE c.connamespace = 'mdm'::regnamespace
  AND c.contype = 'f'
  AND c.confrelid = 'public.tenants'::regclass
ORDER BY c.conrelid::regclass::text;

\echo '=== 0.5.5 count of that FK set ==='
SELECT count(*) AS fks_to_public_tenants
FROM pg_constraint c
WHERE c.connamespace = 'mdm'::regnamespace
  AND c.contype = 'f'
  AND c.confrelid = 'public.tenants'::regclass;

\echo '=== 0.5.6 alpha.mdm FK targets by schema (dedup) ==='
SELECT
    split_part(c.confrelid::regclass::text, '.', 1) AS target_schema,
    count(*)                                        AS fk_count
FROM pg_constraint c
WHERE c.connamespace = 'mdm'::regnamespace
  AND c.contype = 'f'
GROUP BY target_schema
ORDER BY fk_count DESC;
