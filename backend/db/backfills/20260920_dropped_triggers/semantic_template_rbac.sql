-- Backfill: semantic_query_template_permissions (RBAC seeding)
--
-- The deleted `create_default_template_permissions` trigger inserted 3 RBAC
-- rows (viewer/editor/admin) on every template INSERT. Templates created
-- while the trigger was disabled (or never had a corresponding permission
-- row for any other reason) get their 3 rows here.
--
-- Run after 002c drops the trigger, before parity-check window for the
-- template_rbac handler.

BEGIN;

INSERT INTO semantic_query_template_permissions
       (template_id, role, can_run, can_edit, can_delete, can_promote)
SELECT t.id, 'viewer', true, false, false, false FROM semantic_query_templates t
WHERE  NOT EXISTS (SELECT 1 FROM semantic_query_template_permissions p WHERE p.template_id = t.id AND p.role = 'viewer')
ON CONFLICT DO NOTHING;

INSERT INTO semantic_query_template_permissions
       (template_id, role, can_run, can_edit, can_delete, can_promote)
SELECT t.id, 'editor', true, true, false, false FROM semantic_query_templates t
WHERE  NOT EXISTS (SELECT 1 FROM semantic_query_template_permissions p WHERE p.template_id = t.id AND p.role = 'editor')
ON CONFLICT DO NOTHING;

INSERT INTO semantic_query_template_permissions
       (template_id, role, can_run, can_edit, can_delete, can_promote)
SELECT t.id, 'admin', true, true, true, true FROM semantic_query_templates t
WHERE  NOT EXISTS (SELECT 1 FROM semantic_query_template_permissions p WHERE p.template_id = t.id AND p.role = 'admin')
ON CONFLICT DO NOTHING;

COMMIT;
