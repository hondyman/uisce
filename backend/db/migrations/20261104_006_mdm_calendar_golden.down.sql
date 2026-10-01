-- 20261104_006_mdm_calendar_golden.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_change_request','calendar_steward','calendar_exception',
        'calendar_reconciliation','calendar_golden_field','calendar_golden_record',
        'calendar_merge_log','calendar_match_candidate','calendar_match_rule',
        'calendar_survivorship_log','calendar_survivorship_rule',
        'calendar_field_mapping','calendar_source_priority','calendar_type_mapping',
        'calendar_source_alias'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
