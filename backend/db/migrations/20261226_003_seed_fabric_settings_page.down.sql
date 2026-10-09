-- 20261226_003_seed_fabric_settings_page (down)
-- Removes the gold-copy fabric-settings page. The hand-built SettingsPage it replaced is not restored.
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'fabric-settings';
