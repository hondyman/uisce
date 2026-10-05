-- 20261220_001_seed_cubes_catalog_page.up.sql
-- Seeds the core Page Studio page "Cubes" (slug cubes-catalog) in the gold-copy tenant.
-- JSON is GENERATED from frontend/src/pages/page-studio/app/blueprints/;
-- vitest *SeedParity*.test.ts fails if they differ. Change the blueprint, then
-- regenerate this block, never one without the other.
--
-- PR4 seed + A3 Impact drawer. MainNavigation links /build/cubes; STUDIO_ROUTES serves this slug.

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000400',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Cubes', 'cubes-catalog', 'Aggregation contracts: dimensions, governed metrics, grains, materialization, and impact. Built in Page Studio.',
    $layout${
  "root": "page_root",
  "nodes": {
    "page_root": {
      "id": "page_root",
      "type": "Column",
      "children": [
        "top",
        "grid",
        "impact_drawer"
      ],
      "style": {
        "gap": "16px"
      }
    },
    "top": {
      "id": "top",
      "type": "Row",
      "children": [
        "hdr",
        "scope",
        "new_btn"
      ],
      "style": {
        "alignItems": "center",
        "gap": "12px",
        "flexWrap": "wrap"
      }
    },
    "impact_drawer": {
      "id": "impact_drawer",
      "type": "Drawer",
      "children": [
        "impact_dc"
      ],
      "props": {
        "title": "Cube impact",
        "subtitle": "{{vars.impactCubeName}}",
        "width": 720,
        "openWhen": {
          "type": "condition",
          "field": "vars.impactCubeId",
          "operator": "is_not_empty"
        },
        "onClose": [
          {
            "kind": "setVariable",
            "name": "impactCubeId",
            "value": null
          },
          {
            "kind": "setVariable",
            "name": "impactCubeName",
            "value": null
          }
        ]
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
      "icon": "cube",
      "title": "Cubes",
      "subtitle": "Published aggregation contracts — dimensions, governed metrics, grains, materialization, and impact."
    },
    "style": {
      "flex": "1 1 320px"
    }
  },
  "scope": {
    "id": "scope",
    "type": "VariableSelect",
    "props": {
      "variable": "scope",
      "label": "Scope",
      "minWidth": 160,
      "options": [
        {
          "value": "all",
          "label": "All"
        },
        {
          "value": "core",
          "label": "Core"
        },
        {
          "value": "custom",
          "label": "Custom"
        },
        {
          "value": "adopted",
          "label": "Adopted"
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    }
  },
  "new_btn": {
    "id": "new_btn",
    "type": "ActionButton",
    "props": {
      "label": "New cube",
      "icon": "add",
      "variant": "contained",
      "onClick": [
        {
          "kind": "navigate",
          "to": "/build/cubes/new"
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    }
  },
  "grid": {
    "id": "grid",
    "type": "DataGrid",
    "props": {
      "query": "cubes",
      "rowsPath": "rows",
      "rowKey": "id",
      "emptyText": "{{queries.cubes.data.empty_text}}",
      "onRowClick": [
        {
          "kind": "navigate",
          "to": "/build/cubes/{{row.id}}"
        }
      ],
      "columns": [
        {
          "id": "name",
          "header": "Cube",
          "stack": [
            {
              "kind": "text",
              "value": "{{row.name}}",
              "bold": true
            },
            {
              "kind": "text",
              "value": "{{row.boId}}",
              "caption": true
            }
          ]
        },
        {
          "id": "contract",
          "header": "Contract",
          "cell": {
            "kind": "number",
            "value": "{{row.contractVersion}}"
          },
          "nowrap": true
        },
        {
          "id": "core",
          "header": "Core",
          "cell": {
            "kind": "chip",
            "value": "{{row.isCore}}",
            "label": "{{row.isCore}}",
            "variant": "outlined",
            "colorMap": {
              "true": "primary",
              "false": "default",
              "*": "default"
            }
          },
          "nowrap": true
        },
        {
          "id": "status",
          "header": "Status",
          "cell": {
            "kind": "chip",
            "value": "{{row.status}}",
            "label": "{{row.status}}",
            "variant": "outlined",
            "colorMap": {
              "active": "success",
              "draft": "default",
              "*": "default"
            }
          },
          "nowrap": true
        },
        {
          "id": "updated",
          "header": "Updated",
          "cell": {
            "kind": "datetime",
            "value": "{{row.updatedAt}}"
          },
          "nowrap": true
        },
        {
          "id": "act",
          "header": "",
          "cell": {
            "kind": "actions",
            "buttons": [
              {
                "label": "Open",
                "onClick": [
                  {
                    "kind": "navigate",
                    "to": "/build/cubes/{{row.id}}"
                  }
                ]
              },
              {
                "label": "Impact",
                "icon": "insights",
                "onClick": [
                  {
                    "kind": "setVariable",
                    "name": "impactCubeId",
                    "value": "{{row.id}}"
                  },
                  {
                    "kind": "setVariable",
                    "name": "impactCubeName",
                    "value": "{{row.name}}"
                  }
                ]
              },
              {
                "label": "Deploy",
                "icon": "play",
                "onClick": [
                  {
                    "kind": "runOperation",
                    "operation": "cubes.deploy",
                    "params": {
                      "id": "{{row.id}}",
                      "force": false
                    },
                    "onSuccess": [],
                    "confirm": {
                      "title": "Deploy \"{{row.name}}\"?",
                      "text": "Start materialization for this cube contract (force=false).",
                      "confirmLabel": "Deploy"
                    },
                    "successMessage": "Deploy started"
                  }
                ]
              },
              {
                "label": "Refresh",
                "icon": "refresh",
                "onClick": [
                  {
                    "kind": "runOperation",
                    "operation": "cubes.refresh",
                    "params": {
                      "id": "{{row.id}}",
                      "force": false
                    },
                    "onSuccess": [],
                    "confirm": {
                      "title": "Refresh \"{{row.name}}\"?",
                      "text": "Refresh materialization when already dual-committed (force=false may no-op).",
                      "confirmLabel": "Refresh"
                    },
                    "successMessage": "Refresh requested"
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
  "impact_dc": {
    "id": "impact_dc",
    "type": "DomainComponent",
    "props": {
      "component": "cubes.ImpactPanel",
      "inputs": {
        "cubeId": "{{vars.impactCubeId}}",
        "title": "Impact — {{vars.impactCubeName}}"
      }
    }
  }
}$comp$::jsonb,
    $ds$[]$ds$::jsonb,
    $pe$[]$pe$::jsonb,
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
}$fb$::jsonb,
    $app${
  "chrome": "none",
  "surface": {
    "maxWidth": 1200,
    "padding": 3
  },
  "variables": [
    {
      "name": "scope",
      "default": "all",
      "url": true,
      "description": "Cube list scope filter"
    },
    {
      "name": "impactCubeId",
      "description": "Cube id open in the Impact drawer"
    },
    {
      "name": "impactCubeName",
      "description": "Cube name shown in the Impact drawer title"
    }
  ],
  "queries": [
    {
      "id": "cubes",
      "operation": "cubes.list",
      "params": {
        "scope": "{{vars.scope}}",
        "limit": 50
      },
      "keepPrevious": true
    }
  ]
}$app$::jsonb,
    2, true, 'published', NOW()
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description, layout = EXCLUDED.layout,
    tabs = EXCLUDED.tabs, components = EXCLUDED.components, data_sources = EXCLUDED.data_sources,
    presentation_events = EXCLUDED.presentation_events, filter_bar = EXCLUDED.filter_bar,
    app_model = EXCLUDED.app_model, version = EXCLUDED.version, is_core = EXCLUDED.is_core,
    status = EXCLUDED.status, updated_at = NOW();
