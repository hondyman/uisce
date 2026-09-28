-- 20261102_003_mdm_rating_specialized.down.sql
DO $d$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'rating_internal','rating_insurance','rating_bank',
        'rating_fund','rating_proprietary'
    ] LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('ALTER TABLE mdm.%I DISABLE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END
$d$;
