-- 0015_survivorship_selection.down.sql
\set ON_ERROR_STOP on
BEGIN;
ALTER TABLE mdm.survivorship_rule DROP CONSTRAINT IF EXISTS survivorship_rule_none_selected_ck;
ALTER TABLE mdm.survivorship_rule DROP CONSTRAINT IF EXISTS survivorship_rule_selection_mode_ck;
ALTER TABLE mdm.survivorship_rule DROP COLUMN IF EXISTS selection_min_peers,
                                  DROP COLUMN IF EXISTS on_none_selected,
                                  DROP COLUMN IF EXISTS selection_mode,
                                  DROP COLUMN IF EXISTS selection_rule_id;
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
UPDATE mdm.mastering_entity SET settings = settings - 'default_field_group' - 'field_groups'
 WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'PRODUCT';
COMMIT;
