-- 20261103_006_seed_mdm_price_reference.down.sql
DO $down$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN
    DELETE FROM mdm.curve_compounding_frequency  WHERE tenant_id = gold;
    DELETE FROM mdm.curve_interpolation_method   WHERE tenant_id = gold;
    DELETE FROM mdm.curve_category               WHERE tenant_id = gold;
    DELETE FROM mdm.price_observation_type       WHERE tenant_id = gold;
    DELETE FROM mdm.fair_value_level             WHERE tenant_id = gold;
    DELETE FROM mdm.price_quality_tier           WHERE tenant_id = gold;
    DELETE FROM mdm.price_type                   WHERE tenant_id = gold;
    DELETE FROM mdm.price_source                 WHERE tenant_id = gold;
END $down$;
