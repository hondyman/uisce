-- 20261104_004_mdm_calendar_days_events.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_historical_closure','calendar_special_event_impact',
        'calendar_special_event','calendar_session','calendar_day'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
