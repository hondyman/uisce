-- 20261103_001_mdm_price_reference.down.sql
DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'curve_compounding_frequency','curve_interpolation_method','curve_category',
        'price_observation_type','fair_value_level','price_quality_tier','price_type',
        'price_source_alias','price_source'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
