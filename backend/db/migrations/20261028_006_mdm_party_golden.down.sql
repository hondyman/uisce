DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'party_role_assignment','party_history','party_match_rule',
        'party_match_candidate','party_merge_log','party_golden_record',
        'party_golden_field','party_exception','party_change_request',
        'party_steward'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
