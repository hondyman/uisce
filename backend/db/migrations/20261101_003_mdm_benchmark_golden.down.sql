DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'benchmark_dq_rule','benchmark_feed_health','benchmark_feed_schedule',
        'benchmark_change_request','benchmark_steward','benchmark_exception',
        'benchmark_reconciliation_result','benchmark_reconciliation',
        'benchmark_golden_distribution','benchmark_golden_publication',
        'benchmark_golden_field','benchmark_golden_record',
        'benchmark_merge_log','benchmark_match_candidate','benchmark_match_rule',
        'benchmark_survivorship_log','benchmark_survivorship_rule',
        'benchmark_field_mapping','benchmark_source_priority','benchmark_type_mapping'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
