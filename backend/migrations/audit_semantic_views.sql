DO $do$
BEGIN
  RAISE NOTICE 'Skipping Trino-only view creation: audit_semantic_views.sql — not applicable to Postgres';
END
$do$;
