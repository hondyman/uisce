-- 20261102_002_mdm_rating_anchor.down.sql
DO $d$;
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['rating','rating_action','rating_default'] LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('ALTER TABLE mdm.%I DISABLE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END
$d$;
