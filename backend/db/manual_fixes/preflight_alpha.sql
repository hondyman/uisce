-- preflight_alpha.sql
-- Read-only. No DDL. No DML. Run against alpha.
\set ON_ERROR_STOP on
\timing on

\echo '=== 0.1 alpha.mdm table count ==='
SELECT count(*) AS mdm_tables
FROM information_schema.tables
WHERE table_schema = 'mdm'
  AND table_type = 'BASE TABLE';

\echo '=== 0.1b alpha.mdm full table list ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'mdm'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.2 alpha.oms inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'oms'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.3 alpha.edm inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'edm'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.4 classification_scheme existence ==='
SELECT
    to_regclass('mdm.classification_scheme')       AS cs,
    to_regclass('mdm.classification_scheme_map')   AS cs_map;

\echo '=== 0.7 alpha.mdm FK inventory (all) ==='
SELECT
    c.conname                          AS constraint_name,
    c.conrelid::regclass::text         AS from_table,
    c.confrelid::regclass::text        AS to_table,
    pg_get_constraintdef(c.oid)        AS definition
FROM pg_constraint c
WHERE c.connamespace = 'mdm'::regnamespace
  AND c.contype = 'f'
ORDER BY c.conrelid::regclass::text, c.conname;

\echo '=== 0.8 cross-schema FKs from alpha.mdm ==='
SELECT
    c.conname                                  AS constraint_name,
    c.conrelid::regclass::text                 AS from_table,
    c.confrelid::regclass::text                AS to_table,
    split_part(c.confrelid::regclass::text, '.', 1) AS target_schema
FROM pg_constraint c
WHERE c.connamespace = 'mdm'::regnamespace
  AND c.contype = 'f'
  AND split_part(c.confrelid::regclass::text, '.', 1) <> 'mdm'
ORDER BY target_schema, c.conrelid::regclass::text;

\echo '=== 0.10 alpha.public inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;
