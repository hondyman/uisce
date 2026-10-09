-- 20261226_003_seed_fabric_settings_page.up.sql
-- Seeds the core Page Studio page "Settings" (slug fabric-settings) in the gold-copy tenant, served at
-- /fabric/settings via STUDIO_ROUTES. Replaces the hand-built SettingsPage.
-- JSON is GENERATED from frontend/src/pages/page-studio/app/blueprints/fabricSettings.ts;
-- vitest fabricSettingsSeedParity.test.ts fails if they differ. Change the blueprint, then
-- regenerate this block, never one without the other.
--
-- Retired from the old page: the IP-validation, tenant-assignment and notification cards and the
-- access-policy list. Their values were only kept in the browser and nothing enforced them.

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '5169ee5f-77ba-4fa9-a487-437107e7e8a2',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Settings', 'fabric-settings', 'Appearance for this browser. Built in Page Studio.',
    $layout${
  "root": "page_root",
  "nodes": {
    "page_root": {
      "id": "page_root",
      "type": "Column",
      "children": [
        "hdr",
        "appearance"
      ],
      "style": {
        "gap": "16px"
      }
    }
  }
}$layout$,
    $tabs$[]$tabs$,
    $comp${
  "hdr": {
    "id": "hdr",
    "type": "PageHeader",
    "props": {
      "icon": "settings",
      "title": "Settings",
      "subtitle": "Appearance for this browser."
    },
    "style": {
      "flex": "1 1 320px"
    }
  },
  "appearance": {
    "id": "appearance",
    "type": "platformSettings.Appearance",
    "props": {}
  }
}$comp$,
    $ds$[]$ds$,
    $pe$[]$pe$,
    $fb${
  "root": "fb_root",
  "nodes": {
    "fb_root": {
      "id": "fb_root",
      "type": "Column",
      "children": [],
      "style": {
        "gap": "0px"
      }
    }
  }
}$fb$,
    $app${
  "chrome": "none",
  "surface": {
    "maxWidth": 900,
    "padding": 3
  },
  "variables": [],
  "queries": []
}$app$::jsonb,
    1, true, 'published', NOW()
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description, layout = EXCLUDED.layout,
    tabs = EXCLUDED.tabs, components = EXCLUDED.components, data_sources = EXCLUDED.data_sources,
    presentation_events = EXCLUDED.presentation_events, filter_bar = EXCLUDED.filter_bar,
    app_model = EXCLUDED.app_model, version = EXCLUDED.version, is_core = EXCLUDED.is_core,
    status = EXCLUDED.status, updated_at = NOW();
