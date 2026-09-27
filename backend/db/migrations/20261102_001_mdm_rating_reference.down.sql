-- 20261102_001_mdm_rating_reference.down.sql
DO $d$
DECLARE t text;
    tables text[] := ARRAY[
        'rating_agency','rating_type','rating_action_type',
        'rating_outlook','rating_watch','rating_scale','rating_scale_map'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('ALTER TABLE mdm.%I DISABLE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END
$d$;
