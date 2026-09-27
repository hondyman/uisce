DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_change_request','ca_exception','ca_golden_field','ca_golden_record',
        'ca_merge_log','ca_match_candidate','ca_match_rule',
        'ca_survivorship_log','ca_survivorship_rule',
        'ca_field_mapping','ca_source_priority','ca_type_mapping'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
