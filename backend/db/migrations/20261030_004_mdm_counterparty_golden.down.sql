DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'counterparty_change_request','counterparty_steward',
        'counterparty_exception','counterparty_golden_field',
        'counterparty_golden_record','counterparty_merge_log',
        'counterparty_match_candidate','counterparty_match_rule'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
