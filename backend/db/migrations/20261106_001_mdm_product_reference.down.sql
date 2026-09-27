DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'product_lifecycle_event_type','product_target_market_type',
        'product_share_class_type','product_document_type',
        'product_registration_type','product_distribution_channel',
        'product_category','product_status','product_sub_type','product_type'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
