DO $down$
DECLARE t text;
    tables text[] := ARRAY[
        'benchmark_use_case','benchmark_regulatory_regime',
        'benchmark_rebalance_frequency','benchmark_weighting_method',
        'benchmark_currency_variant','benchmark_return_variant',
        'benchmark_type','benchmark_provider'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format('DROP TABLE IF EXISTS mdm.%I CASCADE', t);
    END LOOP;
END $down$;
