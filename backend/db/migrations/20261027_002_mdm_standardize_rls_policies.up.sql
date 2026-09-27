-- Drops the redundant `tenant_isolation` ALL policy on every mdm table
-- that already has a `<table>_tenant_write` policy. The remaining read
-- and write policies are the canonical pair.
-- Idempotent.

DO $std$
DECLARE
    r record;
    dropped int := 0;
BEGIN
    FOR r IN
        SELECT p.schemaname, p.tablename
        FROM pg_policies p
        WHERE p.schemaname = 'mdm'
          AND p.policyname = 'tenant_isolation'
          AND EXISTS (
              SELECT 1 FROM pg_policies p2
              WHERE p2.schemaname = p.schemaname
                AND p2.tablename = p.tablename
                AND p2.policyname = p.tablename || '_tenant_write'
          )
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I',
                       'tenant_isolation', r.schemaname, r.tablename);
        dropped := dropped + 1;
    END LOOP;
    RAISE NOTICE 'standardized RLS: dropped % redundant tenant_isolation policies', dropped;
END
$std$;

-- Also enable RLS + FORCE on every mdm table that has a tenant_id column
-- but no RLS yet. Uses the same two-policy pattern the security tables use.

DO $rls$
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
          AND NOT c.relrowsecurity
          AND a.attnum > 0
          AND NOT a.attisdropped
    LOOP
        pol_read  := r.tbl || '_tenant_read';
        pol_write := r.tbl || '_tenant_write';

        EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', r.sch, r.tbl);
        EXECUTE format('ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY', r.sch, r.tbl);

        EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I', pol_read, r.sch, r.tbl);
        EXECUTE format($p$CREATE POLICY %I ON %I.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$,
            pol_read, r.sch, r.tbl);

        EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I', pol_write, r.sch, r.tbl);
        EXECUTE format($p$CREATE POLICY %I ON %I.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$,
            pol_write, r.sch, r.tbl);
    END LOOP;
END
$rls$;
