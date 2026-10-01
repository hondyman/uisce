-- 20261104_005_mdm_calendar_domain.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'central_bank_calendar','fund_calendar_exception','fund_calendar_rule',
        'fund_calendar','settlement_rule_calendar','settlement_rule',
        'settlement_calendar'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
