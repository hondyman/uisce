-- 20261104_007_seed_mdm_calendar_reference.down.sql
DO $down$
DECLARE gold uuid := '00000000-0000-0000-0000-000000000001'::uuid;
BEGIN
    DELETE FROM mdm.calendar_regulatory_regime WHERE tenant_id = gold;
    DELETE FROM mdm.calendar_hierarchy_type    WHERE tenant_id = gold;
    DELETE FROM mdm.business_day_definition    WHERE tenant_id = gold;
    DELETE FROM mdm.rolling_convention         WHERE tenant_id = gold;
    DELETE FROM mdm.holiday_rule_type          WHERE tenant_id = gold;
    DELETE FROM mdm.holiday_type               WHERE tenant_id = gold;
    DELETE FROM mdm.calendar_source            WHERE tenant_id = gold;
    DELETE FROM mdm.time_zone                  WHERE tenant_id = gold;
    DELETE FROM mdm.calendar_type              WHERE tenant_id = gold;
END $down$;
