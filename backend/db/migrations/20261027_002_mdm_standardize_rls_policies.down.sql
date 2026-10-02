DO $down$
DECLARE
    r record;
    pol_read text;
    pol_write text;
BEGIN
    FOR r IN
        SELECT n.nspname AS sch, c.relname AS tbl
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'tenant_id'
        WHERE n.nspname = 'mdm'
          AND c.relkind = 'r'
          AND c.relrowsecurity
          AND a.attnum > 0 AND NOT a.attisdropped
    LOOP
        pol_read  := r.tbl || '_tenant_read';
        pol_write := r.tbl || '_tenant_write';
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I', pol_read, r.sch, r.tbl);
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I', pol_write, r.sch, r.tbl);
        EXECUTE format($p$CREATE POLICY %I ON %I.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant_id'::text))::uuid))$p$,
            'tenant_isolation', r.sch, r.tbl);
    END LOOP;
END
$down$;
