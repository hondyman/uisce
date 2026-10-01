DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'product_dq_rule','product_feed_health','product_feed_schedule',
        'product_change_request','product_steward','product_exception',
        'product_golden_field','product_golden_record','product_merge_log',
        'product_match_candidate','product_match_rule',
        'product_survivorship_log','product_survivorship_rule',
        'product_field_mapping','product_source_priority','product_type_mapping'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
