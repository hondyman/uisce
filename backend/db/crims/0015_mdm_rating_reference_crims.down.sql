-- 0015_mdm_rating_reference_crims.down.sql
BEGIN;
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'rating_agency','rating_type','rating_action_type',
        'rating_outlook','rating_watch','rating_scale','rating_scale_map'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read',  t);
        EXECUTE format('ALTER TABLE IF EXISTS mdm.%I DISABLE ROW LEVEL SECURITY', t);
    END LOOP;
END
$rls$;
DROP TABLE IF EXISTS mdm.rating_scale_map CASCADE;
DROP TABLE IF EXISTS mdm.rating_agency  CASCADE;
DROP TABLE IF EXISTS mdm.rating_scale   CASCADE;
DROP TABLE IF EXISTS mdm.rating_outlook CASCADE;
DROP TABLE IF EXISTS mdm.rating_watch   CASCADE;
DROP TABLE IF EXISTS mdm.rating_type    CASCADE;
DROP TABLE IF EXISTS mdm.rating_action_type CASCADE;
COMMIT;
