DO $down$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN
    DELETE FROM mdm.benchmark_use_case           WHERE tenant_id = gold;
    DELETE FROM mdm.benchmark_regulatory_regime  WHERE tenant_id = gold;
    DELETE FROM mdm.benchmark_rebalance_frequency WHERE tenant_id = gold;
    DELETE FROM mdm.benchmark_weighting_method   WHERE tenant_id = gold;
    DELETE FROM mdm.benchmark_currency_variant   WHERE tenant_id = gold;
    DELETE FROM mdm.benchmark_return_variant     WHERE tenant_id = gold;
    DELETE FROM mdm.benchmark_type               WHERE tenant_id = gold;
    DELETE FROM mdm.benchmark_provider           WHERE tenant_id = gold;
END $down$;
