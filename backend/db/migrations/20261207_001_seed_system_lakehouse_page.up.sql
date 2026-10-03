-- 20261207_001_seed_system_lakehouse_page.up.sql
-- Seeds the core Page Studio page "Tenant lakehouse" (slug system-lakehouse) in the
-- gold-copy tenant, and places it under a gold-copy "System" menu section that every
-- tenant inherits read-only (ADR-032). Mirrors 20261129_002_seed_lakehouse_streaming_page.
--
-- The JSON below is GENERATED from frontend/src/pages/page-studio/app/blueprints/
-- systemLakehouse.ts and must stay equal to it; the vitest
-- systemLakehouseSeedParity.test.ts fails if they differ. Change the blueprint, then
-- regenerate this block, never one without the other.
--
-- The System parent node is a bridge: no migration or snapshot creates one. It is
-- inserted only if absent (ON CONFLICT DO NOTHING on (tenant_id, node_key)), and the
-- page's node finds its parent by node_key, so it works whether alpha already has a
-- System node under another id or this migration creates it. Entitlement
-- PLATFORM_OPERATOR keeps the section hidden from ordinary users; the API behind the
-- page independently requires a global admin.

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000300',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Tenant lakehouse', 'system-lakehouse', 'System: each tenant''s one Iceberg warehouse and its per-tenant audit retention, for new and existing tenants. Built in Page Studio.',
    $layout${
  "root": "page_root",
  "nodes": {
    "page_root": {
      "id": "page_root",
      "type": "Column",
      "children": [
        "grid"
      ],
      "style": {
        "gap": "12px"
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
      "icon": "storage",
      "title": "Tenant lakehouse",
      "subtitle": "One Iceberg warehouse per tenant. Audit retention is set per tenant, has no default, and can only be extended."
    },
    "style": {
      "flex": "1 1 320px"
    }
  },
  "q": {
    "id": "q",
    "type": "SearchInput",
    "props": {
      "variable": "q",
      "placeholder": "Search tenants",
      "maxWidth": 420
    },
    "style": {
      "flex": "1 1 260px"
    }
  },
  "info": {
    "id": "info",
    "type": "AlertBanner",
    "props": {
      "severity": "info",
      "text": "Metadata for every tenant stays in alpha. This page configures where a tenant's own data and its immutable audit copy are kept."
    }
  },
  "grid": {
    "id": "grid",
    "type": "DataGrid",
    "props": {
      "query": "lakehouses",
      "rowsPath": "rows",
      "rowKey": "id",
      "emptyText": "{{queries.lakehouses.data.empty_text}}",
      "columns": [
        {
          "id": "tenant",
          "header": "Tenant",
          "stack": [
            {
              "kind": "text",
              "value": "{{row.name}}",
              "bold": true
            },
            {
              "kind": "text",
              "value": "{{row.code_text}}",
              "caption": true,
              "visibleWhen": {
                "type": "condition",
                "field": "row.code_text",
                "operator": "is_not_empty"
              }
            }
          ]
        },
        {
          "id": "state",
          "header": "State",
          "cell": {
            "kind": "chip",
            "value": "{{row.state}}",
            "label": "{{row.state_label}}",
            "variant": "outlined",
            "colorMap": {
              "active": "success",
              "provisioning": "warning",
              "suspended": "warning",
              "offboarding": "warning",
              "unconfigured": "default",
              "offboarded": "default",
              "*": "default"
            }
          }
        },
        {
          "id": "retention",
          "header": "Audit retention",
          "cell": {
            "kind": "text",
            "value": "{{row.retention_text}}"
          }
        },
        {
          "id": "warehouse",
          "header": "Warehouse",
          "cell": {
            "kind": "text",
            "value": "{{row.warehouse_name}}",
            "caption": true
          }
        },
        {
          "id": "act",
          "header": "",
          "cell": {
            "kind": "actions",
            "buttons": [
              {
                "label": "Set retention",
                "icon": "edit",
                "onClick": [
                  {
                    "kind": "setVariable",
                    "name": "lhTenant",
                    "value": "{{row.tenant_id}}"
                  },
                  {
                    "kind": "setVariable",
                    "name": "lhOpen",
                    "value": true
                  }
                ]
              },
              {
                "label": "Provision",
                "icon": "play",
                "visibleWhen": {
                  "type": "condition",
                  "field": "row.can_provision",
                  "operator": "is_true"
                },
                "onClick": [
                  {
                    "kind": "runOperation",
                    "operation": "systemLakehouse.provision",
                    "params": {
                      "tenant_id": "{{row.tenant_id}}"
                    },
                    "onSuccess": [],
                    "confirm": {
                      "title": "Provision lakehouse",
                      "text": "Create the bucket, key, credential and Iceberg warehouse for {{row.name}} with audit retention of {{row.retention_text}}? Compliance retention cannot be shortened once objects are locked.",
                      "confirmLabel": "Provision"
                    },
                    "successMessage": "Provisioning started"
                  }
                ]
              },
              {
                "label": "Audit trail",
                "icon": "review",
                "onClick": [
                  {
                    "kind": "setVariable",
                    "name": "auditTenant",
                    "value": "{{row.tenant_id}}"
                  },
                  {
                    "kind": "setVariable",
                    "name": "auditOpen",
                    "value": true
                  }
                ]
              }
            ]
          },
          "align": "right",
          "nowrap": true
        }
      ]
    }
  },
  "lh_note": {
    "id": "lh_note",
    "type": "AlertBanner",
    "props": {
      "severity": "warning",
      "text": "{{queries.lhStart.data.note}}"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.lhStart.data.note",
      "operator": "is_not_empty"
    }
  },
  "lh_form": {
    "id": "lh_form",
    "type": "Form",
    "props": {
      "variable": "lhDraft",
      "initFrom": "{{queries.lhStart.data.draft}}",
      "seedKey": "{{queries.lhStart.data.key}}",
      "fields": [
        {
          "name": "audit_retention_days",
          "kind": "number",
          "label": "Audit retention (days)",
          "required": true,
          "step": 1,
          "wide": true,
          "helperText": "A whole number of days. 2555 is seven years. Once set it can only be extended."
        }
      ]
    }
  },
  "au_ok": {
    "id": "au_ok",
    "type": "AlertBanner",
    "props": {
      "severity": "success",
      "text": "{{queries.lhAudit.data.chain_text}}"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.lhAudit.data.chain_intact",
      "operator": "is_true"
    }
  },
  "au_broken": {
    "id": "au_broken",
    "type": "AlertBanner",
    "props": {
      "severity": "error",
      "text": "{{queries.lhAudit.data.chain_text}}"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.lhAudit.data.chain_intact",
      "operator": "is_false"
    }
  },
  "au_grid": {
    "id": "au_grid",
    "type": "DataGrid",
    "props": {
      "query": "lhAudit",
      "rowsPath": "rows",
      "rowKey": "id",
      "emptyText": "{{queries.lhAudit.data.empty_text}}",
      "columns": [
        {
          "id": "at",
          "header": "When",
          "cell": {
            "kind": "datetime",
            "value": "{{row.at}}"
          },
          "nowrap": true
        },
        {
          "id": "who",
          "header": "Who",
          "stack": [
            {
              "kind": "text",
              "value": "{{row.actor}}"
            },
            {
              "kind": "text",
              "value": "{{row.role}}",
              "caption": true,
              "visibleWhen": {
                "type": "condition",
                "field": "row.role",
                "operator": "is_not_empty"
              }
            }
          ]
        },
        {
          "id": "change",
          "header": "Change",
          "cell": {
            "kind": "text",
            "value": "{{row.change}}"
          }
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
        "info",
        "lh_dialog",
        "au_drawer"
      ],
      "style": {
        "gap": "8px"
      }
    },
    "top_header": {
      "id": "top_header",
      "type": "Row",
      "children": [
        "hdr",
        "q"
      ],
      "style": {
        "alignItems": "center",
        "gap": "12px",
        "flexWrap": "wrap"
      }
    },
    "lh_dialog": {
      "id": "lh_dialog",
      "type": "Dialog",
      "children": [
        "lh_body"
      ],
      "props": {
        "title": "{{queries.lhStart.data.title}}",
        "maxWidth": "sm",
        "openWhen": {
          "type": "condition",
          "field": "vars.lhOpen",
          "operator": "is_true"
        },
        "onClose": [
          {
            "kind": "setVariable",
            "name": "lhOpen",
            "value": false
          },
          {
            "kind": "setVariable",
            "name": "lhTenant",
            "value": null
          },
          {
            "kind": "setVariable",
            "name": "lhDraft",
            "value": null
          }
        ],
        "buttons": [
          {
            "label": "Cancel",
            "onClick": [
              {
                "kind": "setVariable",
                "name": "lhOpen",
                "value": false
              },
              {
                "kind": "setVariable",
                "name": "lhTenant",
                "value": null
              },
              {
                "kind": "setVariable",
                "name": "lhDraft",
                "value": null
              }
            ]
          },
          {
            "label": "Save",
            "variant": "contained",
            "disabledWhen": {
              "type": "condition",
              "field": "vars.lhDraft.audit_retention_days",
              "operator": "is_empty"
            },
            "onClick": [
              {
                "kind": "runOperation",
                "operation": "systemLakehouse.setRetention",
                "params": {
                  "tenant_id": "{{vars.lhTenant}}",
                  "audit_retention_days": "{{vars.lhDraft.audit_retention_days}}"
                },
                "onSuccess": [
                  {
                    "kind": "setVariable",
                    "name": "lhOpen",
                    "value": false
                  },
                  {
                    "kind": "setVariable",
                    "name": "lhTenant",
                    "value": null
                  },
                  {
                    "kind": "setVariable",
                    "name": "lhDraft",
                    "value": null
                  }
                ],
                "successMessage": "Audit retention saved"
              }
            ]
          }
        ]
      }
    },
    "lh_body": {
      "id": "lh_body",
      "type": "Column",
      "children": [
        "lh_note",
        "lh_form"
      ],
      "style": {
        "gap": "12px"
      }
    },
    "au_drawer": {
      "id": "au_drawer",
      "type": "Drawer",
      "children": [
        "au_body"
      ],
      "props": {
        "title": "Lakehouse audit trail",
        "subtitle": "Changes to this tenant's lakehouse configuration, newest first",
        "width": 640,
        "openWhen": {
          "type": "condition",
          "field": "vars.auditOpen",
          "operator": "is_true"
        },
        "onClose": [
          {
            "kind": "setVariable",
            "name": "auditOpen",
            "value": false
          },
          {
            "kind": "setVariable",
            "name": "auditTenant",
            "value": null
          }
        ]
      }
    },
    "au_body": {
      "id": "au_body",
      "type": "Column",
      "children": [
        "au_ok",
        "au_broken",
        "au_grid"
      ],
      "style": {
        "gap": "12px"
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
  "variables": [
    {
      "name": "q",
      "default": ""
    },
    {
      "name": "lhOpen",
      "default": false
    },
    {
      "name": "lhTenant",
      "description": "The tenant whose retention is being edited"
    },
    {
      "name": "lhDraft",
      "description": "The retention form"
    },
    {
      "name": "auditOpen",
      "default": false
    },
    {
      "name": "auditTenant",
      "description": "The tenant whose audit trail is open"
    }
  ],
  "queries": [
    {
      "id": "lakehouses",
      "operation": "systemLakehouse.list",
      "params": {
        "q": "{{vars.q}}"
      },
      "keepPrevious": true,
      "debounceMs": 300
    },
    {
      "id": "lhStart",
      "operation": "systemLakehouse.editorStart",
      "params": {
        "tenant_id": "{{vars.lhTenant}}"
      },
      "enabledWhen": {
        "type": "condition",
        "field": "vars.lhOpen",
        "operator": "is_true"
      }
    },
    {
      "id": "lhAudit",
      "operation": "systemLakehouse.audit",
      "params": {
        "tenant_id": "{{vars.auditTenant}}"
      },
      "enabledWhen": {
        "type": "condition",
        "field": "vars.auditOpen",
        "operator": "is_true"
      }
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

INSERT INTO public.navigation_menu_nodes (
    id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement
) VALUES (
    'e81a3d02-0001-7000-8000-000000000301', '99e99e99-99e9-49e9-89e9-99e99e99e999', NULL, 'system', 'System', NULL, 900, 'PLATFORM_OPERATOR'
)
ON CONFLICT (tenant_id, node_key) DO NOTHING;

INSERT INTO public.navigation_menu_nodes (
    id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement
)
SELECT 'e81a3d02-0001-7000-8000-000000000300', '99e99e99-99e9-49e9-89e9-99e99e99e999', p.id, 'system-lakehouse', 'Tenant lakehouse', 'system-lakehouse', 20, 'PLATFORM_OPERATOR'
  FROM public.navigation_menu_nodes p
 WHERE p.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND p.node_key = 'system'
ON CONFLICT (tenant_id, node_key) DO UPDATE SET
    label = EXCLUDED.label, target_page_key = EXCLUDED.target_page_key,
    display_order = EXCLUDED.display_order, required_entitlement = EXCLUDED.required_entitlement;
