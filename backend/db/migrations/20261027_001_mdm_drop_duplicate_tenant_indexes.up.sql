-- Removes redundant single-column btree indexes on tenant_id.
-- Every mdm table with more than one tenant-only index keeps exactly one:
-- the canonical `idx_<tablename>_tenant` if present, else the alphabetical first.
-- Idempotent — safe to re-run.

DO $cleanup$
DECLARE
    r record;
    keeper text;
    dropper text;
BEGIN
    FOR r IN
        SELECT n.nspname AS sch, c.relname AS tbl
        FROM pg_index i
        JOIN pg_class c ON c.oid = i.indrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
        WHERE n.nspname = 'mdm'
          AND c.relkind = 'r'
          AND i.indnatts = 1
          AND i.indisvalid
          AND a.attname = 'tenant_id'
        GROUP BY n.nspname, c.relname
        HAVING count(*) > 1
    LOOP
        -- Prefer the canonical name if present
        SELECT ic.relname INTO keeper
        FROM pg_index i
        JOIN pg_class ic ON ic.oid = i.indexrelid
        JOIN pg_class c ON c.oid = i.indrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
        WHERE n.nspname = r.sch AND c.relname = r.tbl
          AND i.indnatts = 1 AND a.attname = 'tenant_id'
          AND ic.relname = 'idx_' || r.tbl || '_tenant'
        LIMIT 1;

        IF keeper IS NULL THEN
            SELECT ic.relname INTO keeper
            FROM pg_index i
            JOIN pg_class ic ON ic.oid = i.indexrelid
            JOIN pg_class c ON c.oid = i.indrelid
            JOIN pg_namespace n ON n.oid = c.relnamespace
            JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
            WHERE n.nspname = r.sch AND c.relname = r.tbl
              AND i.indnatts = 1 AND a.attname = 'tenant_id'
            ORDER BY ic.relname
            LIMIT 1;
        END IF;

        FOR dropper IN
            SELECT ic.relname
            FROM pg_index i
            JOIN pg_class ic ON ic.oid = i.indexrelid
            JOIN pg_class c ON c.oid = i.indrelid
            JOIN pg_namespace n ON n.oid = c.relnamespace
            JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
            WHERE n.nspname = r.sch AND c.relname = r.tbl
              AND i.indnatts = 1 AND a.attname = 'tenant_id'
              AND ic.relname <> keeper
        LOOP
            EXECUTE format('DROP INDEX IF EXISTS %I.%I', r.sch, dropper);
            RAISE NOTICE 'dropped duplicate tenant index %.%', r.sch, dropper;
        END LOOP;
    END LOOP;
END
$cleanup$;
