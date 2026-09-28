DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_ratio_history','ca_stock_component','ca_cash_component',
        'ca_term','ca_security','ca_identifier','ca_event'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
