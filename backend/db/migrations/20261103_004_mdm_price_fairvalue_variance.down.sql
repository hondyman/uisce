-- 20261103_004_mdm_price_fairvalue_variance.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'price_challenge','price_stale_event','price_variance_event',
        'price_variance_threshold','valuation_sensitivity','valuation_input',
        'valuation_model','fair_value_classification'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
