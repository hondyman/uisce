-- 20261006_001_seed_lakehouse_status_page.up.sql
-- Seeds the core Page Studio page "Lakehouse status" (slug lakehouse-status) in the
-- gold-copy tenant, and places it under the gold-copy "System" menu section that every
-- tenant inherits read-only. Mirrors 20261207_001_seed_system_lakehouse_page.up.sql.
--
-- The JSON below is GENERATED from frontend/src/pages/page-studio/app/blueprints/
-- lakehouseStatus.ts and must stay equal to it; the vitest lakehouseStatusSeedParity.test.ts
-- fails if they differ. Change the blueprint, then regenerate this block, never one
-- without the other.
--
-- The page is a sibling of "Tenant lakehouse" inside the "System" menu node. The System
-- parent node is a bridge: this migration does not create it if it already exists (see
-- 20261207_001_seed_system_lakehouse_page.up.sql), and the menu entry finds its parent
-- by node_key. The required_entitlement PLATFORM_OPERATOR keeps the entry hidden from
-- ordinary users; the API behind the page independently requires a global admin.

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000301',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Lakehouse status', 'lakehouse-status', 'Read-only: cluster health, resource groups, per-tenant wiring, and cross-check warnings. Built in Page Studio.',
    $layout${
  "root": "page_root",
  "nodes": {
    "page_root": {
      "id": "page_root",
      "type": "Column",
      "children": [
        "rg_caption",
        "rg_grid",
        "tenants_caption",
        "tenants_grid"
      ],
      "style": {
        "gap": "16px"
      }
    }
  }
}$layout$::jsonb,
    $tabs$[]$tabs$::jsonb,
    $comp${
  "hdr": {
    "id": "hdr",
    "type": "PageHeader",
    "props": {
      "icon": "monitor_heart",
      "title": "Lakehouse status",
      "subtitle": "Cluster health, per-tenant wiring, and cross-check warnings. Read-only."
    },
    "style": {
      "flex": "1 1 320px"
    }
  },
  "note_info": {
    "id": "note_info",
    "type": "AlertBanner",
    "props": {
      "severity": "info",
      "text": "{{queries.lhStart.data.notes_text}}"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.lhStart.data.notes_text",
      "operator": "is_not_empty"
    }
  },
  "cluster_caption": {
    "id": "cluster_caption",
    "type": "TextBlock",
    "props": {
      "text": "Cluster",
      "caption": "Frontends and backends from SHOW FRONTENDS / SHOW BACKENDS."
    }
  },
  "cluster_kv": {
    "id": "cluster_kv",
    "type": "KeyValue",
    "props": {
      "label": "",
      "rowsPath": "queries.lhStart.data.cluster_rows",
      "emptyText": "Cluster not configured (LAKEHOUSE_STARROCKS_DSN unset).",
      "pairs": [
        {
          "label": "{{row.label}}",
          "value": "{{row.value}}",
          "sub": "{{row.sub}}"
        }
      ]
    }
  },
  "rg_caption": {
    "id": "rg_caption",
    "type": "TextBlock",
    "props": {
      "text": "Resource groups",
      "caption": "CPU weight, memory limit, concurrency, and the classifier routing that pins each tenant user to a group."
    }
  },
  "rg_grid": {
    "id": "rg_grid",
    "type": "DataGrid",
    "props": {
      "query": "lhStart",
      "rowsPath": "resource_groups",
      "rowKey": "name",
      "emptyText": "{{queries.lhStart.data.rg_empty_text}}",
      "columns": [
        {
          "id": "name",
          "header": "Group",
          "cell": {
            "kind": "text",
            "value": "{{row.name}}",
            "bold": true
          }
        },
        {
          "id": "cpu",
          "header": "CPU",
          "cell": {
            "kind": "number",
            "value": "{{row.cpu_weight}}",
            "suffix": " bw"
          }
        },
        {
          "id": "mem",
          "header": "Memory",
          "cell": {
            "kind": "text",
            "value": "{{row.mem_limit}}"
          }
        },
        {
          "id": "conc",
          "header": "Concurrency",
          "cell": {
            "kind": "number",
            "value": "{{row.concurrency_limit}}"
          }
        },
        {
          "id": "cls",
          "header": "Classifiers",
          "cell": {
            "kind": "text",
            "value": "{{row.classifiers_text}}",
            "caption": true
          }
        }
      ]
    }
  },
  "tenants_caption": {
    "id": "tenants_caption",
    "type": "TextBlock",
    "props": {
      "text": "Tenants",
      "caption": "Per-tenant wiring (DSN, init resource group), warehouse DB size, and audit-copy posture. Warnings show the silent-failure modes the runbook gap 4 calls out."
    }
  },
  "tenants_grid": {
    "id": "tenants_grid",
    "type": "DataGrid",
    "props": {
      "query": "lhStart",
      "rowsPath": "tenants",
      "rowKey": "tenant_id",
      "emptyText": "{{queries.lhStart.data.tenants_empty_text}}",
      "columns": [
        {
          "id": "name",
          "header": "Tenant",
          "cell": {
            "kind": "text",
            "value": "{{row.name}}",
            "bold": true
          }
        },
        {
          "id": "slug",
          "header": "Slug",
          "cell": {
            "kind": "text",
            "value": "{{row.slug}}",
            "monospace": true
          }
        },
        {
          "id": "env",
          "header": "Wiring",
          "stack": [
            {
              "kind": "chip",
              "label": "{{row.dsn_label}}",
              "color": "{{row.dsn_color}}",
              "variant": "outlined"
            },
            {
              "kind": "text",
              "value": "init rg: {{row.init_rg_text}}",
              "caption": true,
              "visibleWhen": {
                "type": "condition",
                "field": "row.init_rg_text",
                "operator": "is_not_empty"
              }
            }
          ]
        },
        {
          "id": "db",
          "header": "Database",
          "stack": [
            {
              "kind": "text",
              "value": "{{row.db_size_text}}"
            },
            {
              "kind": "text",
              "value": "{{row.db_table_text}}",
              "caption": true,
              "visibleWhen": {
                "type": "condition",
                "field": "row.db_table_text",
                "operator": "is_not_empty"
              }
            }
          ]
        },
        {
          "id": "audit",
          "header": "Audit copy",
          "cell": {
            "kind": "text",
            "value": "{{row.audit_text}}",
            "caption": true
          }
        },
        {
          "id": "warn",
          "header": "Warnings",
          "stack": [
            {
              "kind": "chip",
              "label": "{{row.warnings_label}}",
              "color": "warning",
              "variant": "outlined",
              "visibleWhen": {
                "type": "condition",
                "field": "row.warnings_count",
                "operator": "greater_than",
                "value": 0
              }
            },
            {
              "kind": "chip",
              "label": "OK",
              "color": "success",
              "variant": "outlined",
              "visibleWhen": {
                "type": "condition",
                "field": "row.warnings_count",
                "operator": "less_than",
                "value": 1
              }
            }
          ]
        }
      ]
    }
  }
}$comp$::jsonb,
    $ds$[]$ds$::jsonb,
    $pe$[]$pe$::jsonb,
    $fb${
  "root": "top_root",
  "nodes": {
    "top_root": {
      "id": "top_root",
      "type": "Column",
      "children": [
        "top_header",
        "note_info",
        "cluster_caption",
        "cluster_kv"
      ],
      "style": {
        "gap": "12px"
      }
    },
    "top_header": {
      "id": "top_header",
      "type": "Row",
      "children": [
        "hdr"
      ],
      "style": {
        "alignItems": "center",
        "gap": "12px",
        "flexWrap": "wrap"
      }
    }
  }
}$fb$::jsonb,
    $app${
  "chrome": "none",
  "surface": {
    "maxWidth": 1400,
    "padding": 3
  },
  "variables": [],
  "queries": [
    {
      "id": "lhStart",
      "operation": "lakehouseStatus.platform"
    },
    {
      "id": "lhStartTenant",
      "operation": "lakehouseStatus.tenant"
    }
  ]
}$app$::jsonb,
    1, true, 'published', NOW()
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description, layout = EXCLUDED.layout,
    tabs = EXCLUDED.tabs, components = EXCLUDED.components, data_sources = EXCLUDED.data_sources,
    presentation_events = EXCLUDED.presentation_events, filter_bar = EXCLUDED.filter_bar,
    app_model = EXCLUDED.app_model, version = EXCLUDED.version, is_core = EXCLUDED.is_core,
    status = EXCLUDED.status, updated_at = NOW();

-- Menu placement under the existing "System" parent (created by 20261207_001_seed_system_lakehouse_page
-- if absent, otherwise reused by node_key match). This migration assumes the System node
-- exists or will exist; if running standalone on a fresh alpha, also apply
-- 20261207_001_seed_system_lakehouse_page.up.sql first.
INSERT INTO public.navigation_menu_nodes (
    id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement
)
SELECT '018f9d02-0001-7000-8000-000000000303', '99e99e99-99e9-49e9-89e9-99e99e99e999', p.id, 'lakehouse-status', 'Lakehouse status', 'lakehouse-status', 30, 'PLATFORM_OPERATOR'
  FROM public.navigation_menu_nodes p
 WHERE p.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND p.node_key = 'system'
ON CONFLICT (tenant_id, node_key) DO UPDATE SET
    label = EXCLUDED.label, target_page_key = EXCLUDED.target_page_key,
    display_order = EXCLUDED.display_order, required_entitlement = EXCLUDED.required_entitlement;
