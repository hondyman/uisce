-- 20261226_004_seed_rbac_roles_page (down)
-- Removes the gold-copy rbac-roles page. The hand-built RoleManagerPage it replaced is not restored.
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'rbac-roles';
