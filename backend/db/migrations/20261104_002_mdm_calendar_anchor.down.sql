-- 20261104_002_mdm_calendar_anchor.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_hierarchy_closure','calendar_hierarchy',
        'calendar_classification','calendar_identifier','calendar_master'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
