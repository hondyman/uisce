-- =============================================================================
-- Post-migration trigger compliance check
-- =============================================================================
-- Run after: 002a + 002b + 002d + 002e + 002c + crims
-- Expected result: ZERO rows in both alpha and crims.
--
-- What this catches:
--   Triggers that write to OTHER tables, send NOTIFY, EXECUTE dynamic SQL,
--   REFRESH materialized views, or RAISE exceptions.
--
-- What this intentionally does NOT flag (accepted limitation):
--   Pure in-row NEW mutators: a trigger that only assigns to NEW fields without
--   any cross-table side effects, NOTIFY, REFRESH, EXECUTE, or RAISE.
--   Example: NEW.duration = NEW.end_time - OLD.start_time
--   This is deliberate — in-row mutation is app-logic territory. The business-object
--   semantic-terms layer is the authoritative enforcement point. The CHECK constraints
--   added in 002e provide DB-level backstop for price/type coupling.
--
-- Side-effect patterns caught:
--   INSERT INTO / DELETE FROM on other tables
--   UPDATE on a different table (UPDATE schema.table SET ...)
--   pg_notify() calls
--   EXECUTE statements
--   REFRESH MATERIALIZED VIEW
--   RAISE EXCEPTION (rejection logic — must be in CHECK constraints)
--
-- How to run:
--   psql "$ALPHA_DB_URL" -f backend/db/migrations/_reference/20260920_002_trigger_compliance_check.sql
--   psql "$CRIMS_DB_URL"  -f backend/db/migrations/_reference/20260920_002_trigger_compliance_check.sql
--   # both must return zero rows
--
-- =============================================================================

SELECT n.nspname, c.relname, t.tgname, p.proname,
       left(pg_get_functiondef(p.oid), 300) AS fn_preview
FROM pg_trigger t
JOIN pg_class c      ON t.tgrelid = c.oid
JOIN pg_namespace n   ON c.relnamespace = n.oid
JOIN pg_proc p        ON t.tgfoid = p.oid
WHERE NOT t.tgisinternal
  AND t.tgenabled = 'O'
  AND (
      -- INSERT INTO / DELETE FROM on other tables (toucher never contains this)
      pg_get_functiondef(p.oid) ~* '(?:INSERT\s+INTO|DELETE\s+FROM)\s+\S+'
        -- UPDATE on a different table (not NEW.col = val mutations)
        OR pg_get_functiondef(p.oid) ~ 'UPDATE\s+(?!NEW)\S+\.\S+\s+SET\s+'
        -- UPDATE table SET excluding the ON CONFLICT DO UPDATE same-row case
        OR (pg_get_functiondef(p.oid) ~ 'UPDATE\s+(?!NEW)\S+\s+SET\s+'
            AND pg_get_functiondef(p.oid) !~* 'ON\s+CONFLICT')
        -- NOTIFY
        OR pg_get_functiondef(p.oid) ~* 'pg_notify\s*\('
        -- Dynamic SQL
        OR pg_get_functiondef(p.oid) ~ 'EXECUTE\s+'
        -- Matview refresh
        OR pg_get_functiondef(p.oid) ~ 'REFRESH\s+MATERIALIZED\s+VIEW'
        -- Rejection logic — must be in CHECK constraints, not triggers
        OR pg_get_functiondef(p.oid) ~ 'RAISE\s+(EXCEPTION|ABORT)'
  )
ORDER BY n.nspname, c.relname, t.tgname;
