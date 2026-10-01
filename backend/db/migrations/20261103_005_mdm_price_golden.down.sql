-- 20261103_005_mdm_price_golden.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'price_dq_rule','price_feed_health','price_feed_schedule',
        'price_change_request','price_steward','price_exception',
        'price_reconciliation_result','price_reconciliation',
        'price_golden_distribution','price_golden_publication',
        'price_golden_field','price_golden_record',
        'price_merge_log','price_match_candidate','price_match_rule',
        'price_survivorship_log','price_survivorship_rule',
        'price_field_mapping','price_source_priority','price_type_mapping'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
