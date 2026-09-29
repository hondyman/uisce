-- preflight2_edm.sql
-- Read-only.
\set ON_ERROR_STOP on

\echo '=== 0.5.16 alpha.edm full table list (72 tables) ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'edm'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.17 alpha.edm tables grouped by prefix ==='
SELECT
    split_part(table_name, '_', 1) AS prefix,
    count(*)                       AS tables
FROM information_schema.tables
WHERE table_schema = 'edm'
  AND table_type = 'BASE TABLE'
GROUP BY prefix
ORDER BY tables DESC, prefix;
