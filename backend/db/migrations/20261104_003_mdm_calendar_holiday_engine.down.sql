-- 20261104_003_mdm_calendar_holiday_engine.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_rule','holiday_definition_calendar','holiday_manual_date','holiday_definition'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
