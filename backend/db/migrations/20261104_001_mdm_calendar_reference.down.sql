-- 20261104_001_mdm_calendar_reference.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_regulatory_regime','calendar_hierarchy_type','business_day_definition',
        'rolling_convention','holiday_rule_type','holiday_type','calendar_source',
        'time_zone','calendar_type'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
