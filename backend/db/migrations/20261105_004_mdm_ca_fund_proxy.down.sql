DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_proxy_vote','ca_agenda_item','ca_meeting',
        'ca_fund_liquidation','ca_fund_reorganization','ca_fund_distribution'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
