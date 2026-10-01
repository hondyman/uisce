-- 20261102_005_seed_mdm_rating_reference.down.sql
DELETE FROM mdm.rating_scale_map  WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.rating_scale      WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.rating_watch      WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.rating_outlook    WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.rating_action_type WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.rating_type       WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM mdm.rating_agency     WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
