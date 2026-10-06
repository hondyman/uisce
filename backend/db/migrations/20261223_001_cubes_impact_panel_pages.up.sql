-- 20261223_001_cubes_impact_panel_pages.up.sql
-- A3: refresh cubes-catalog + cube-designer gold pages with Impact drawer/tab.

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

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000401',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Cube designer', 'cube-designer', 'Compose a cube contract: overview, dimensions, metrics, grains, federation, materialization, and impact. Served at /build/cubes/new and /build/cubes/:id.',
    $layout${
  "root": "root",
  "nodes": {
    "root": {
      "id": "root",
      "type": "Column",
      "children": [
        "toolbar",
        "banners",
        "body"
      ],
      "style": {
        "gap": "0",
        "height": "calc(100vh - 64px)"
      }
    },
    "toolbar": {
      "id": "toolbar",
      "type": "Row",
      "children": [
        "back",
        "title",
        "unsaved",
        "actions"
      ],
      "style": {
        "alignItems": "center",
        "gap": "8px",
        "padding": "8px 16px",
        "borderBottom": "1px solid rgba(128,128,128,0.3)",
        "flexWrap": "wrap"
      }
    },
    "actions": {
      "id": "actions",
      "type": "Row",
      "children": [
        "validate_btn",
        "deploy_btn",
        "refresh_btn",
        "impact_btn",
        "save_create",
        "save_patch"
      ],
      "style": {
        "alignItems": "center",
        "gap": "8px",
        "marginLeft": "auto",
        "flex": "0 0 auto"
      }
    },
    "banners": {
      "id": "banners",
      "type": "Column",
      "children": [
        "error_banner",
        "info_banner"
      ],
      "style": {
        "gap": "8px",
        "padding": "0 16px"
      }
    },
    "body": {
      "id": "body",
      "type": "Column",
      "children": [
        "tabs"
      ],
      "style": {
        "flex": "1 1 0",
        "minHeight": "0",
        "overflowY": "auto",
        "padding": "16px"
      }
    },
    "tabs": {
      "id": "tabs",
      "type": "TabSet",
      "children": [
        "tab_overview",
        "tab_dimensions",
        "tab_metrics",
        "tab_grains",
        "tab_federation",
        "tab_materialization",
        "tab_versions",
        "tab_impact"
      ],
      "props": {
        "variable": "tab",
        "tabs": [
          {
            "id": "overview",
            "label": "Overview"
          },
          {
            "id": "dimensions",
            "label": "Dimensions"
          },
          {
            "id": "metrics",
            "label": "Metrics"
          },
          {
            "id": "grains",
            "label": "Grains"
          },
          {
            "id": "federation",
            "label": "Federation"
          },
          {
            "id": "materialization",
            "label": "Materialization"
          },
          {
            "id": "versions",
            "label": "Versions"
          },
          {
            "id": "impact",
            "label": "Impact"
          }
        ]
      }
    },
    "tab_overview": {
      "id": "tab_overview",
      "type": "Column",
      "children": [
        "overview_form",
        "overview_hint"
      ],
      "style": {
        "gap": "12px",
        "maxWidth": "560px"
      }
    },
    "tab_dimensions": {
      "id": "tab_dimensions",
      "type": "Column",
      "children": [
        "dims_help",
        "dims_form"
      ],
      "style": {
        "gap": "12px",
        "maxWidth": "720px"
      }
    },
    "tab_metrics": {
      "id": "tab_metrics",
      "type": "Column",
      "children": [
        "metrics_help",
        "metrics_form"
      ],
      "style": {
        "gap": "12px",
        "maxWidth": "720px"
      }
    },
    "tab_grains": {
      "id": "tab_grains",
      "type": "Column",
      "children": [
        "grains_help",
        "grains_form"
      ],
      "style": {
        "gap": "12px",
        "maxWidth": "720px"
      }
    },
    "tab_federation": {
      "id": "tab_federation",
      "type": "Column",
      "children": [
        "fed_dc"
      ],
      "style": {
        "gap": "12px"
      }
    },
    "tab_materialization": {
      "id": "tab_materialization",
      "type": "Column",
      "children": [
        "mat_form",
        "mat_engines"
      ],
      "style": {
        "gap": "12px",
        "maxWidth": "480px"
      }
    },
    "tab_versions": {
      "id": "tab_versions",
      "type": "Column",
      "children": [
        "versions_text"
      ],
      "style": {
        "gap": "12px",
        "maxWidth": "640px"
      }
    },
    "tab_impact": {
      "id": "tab_impact",
      "type": "Column",
      "children": [
        "impact_new_hint",
        "impact_dc"
      ],
      "style": {
        "gap": "12px",
        "maxWidth": "960px"
      }
    }
  }
}$layout$::jsonb,
    $tabs$[
  {
    "id": "designer",
    "label": "Designer",
    "layout": {
      "root": "root",
      "nodes": {
        "root": {
          "id": "root",
          "type": "Column",
          "children": [
            "toolbar",
            "banners",
            "body"
          ],
          "style": {
            "gap": "0",
            "height": "calc(100vh - 64px)"
          }
        },
        "toolbar": {
          "id": "toolbar",
          "type": "Row",
          "children": [
            "back",
            "title",
            "unsaved",
            "actions"
          ],
          "style": {
            "alignItems": "center",
            "gap": "8px",
            "padding": "8px 16px",
            "borderBottom": "1px solid rgba(128,128,128,0.3)",
            "flexWrap": "wrap"
          }
        },
        "actions": {
          "id": "actions",
          "type": "Row",
          "children": [
            "validate_btn",
            "deploy_btn",
            "refresh_btn",
            "impact_btn",
            "save_create",
            "save_patch"
          ],
          "style": {
            "alignItems": "center",
            "gap": "8px",
            "marginLeft": "auto",
            "flex": "0 0 auto"
          }
        },
        "banners": {
          "id": "banners",
          "type": "Column",
          "children": [
            "error_banner",
            "info_banner"
          ],
          "style": {
            "gap": "8px",
            "padding": "0 16px"
          }
        },
        "body": {
          "id": "body",
          "type": "Column",
          "children": [
            "tabs"
          ],
          "style": {
            "flex": "1 1 0",
            "minHeight": "0",
            "overflowY": "auto",
            "padding": "16px"
          }
        },
        "tabs": {
          "id": "tabs",
          "type": "TabSet",
          "children": [
            "tab_overview",
            "tab_dimensions",
            "tab_metrics",
            "tab_grains",
            "tab_federation",
            "tab_materialization",
            "tab_versions",
            "tab_impact"
          ],
          "props": {
            "variable": "tab",
            "tabs": [
              {
                "id": "overview",
                "label": "Overview"
              },
              {
                "id": "dimensions",
                "label": "Dimensions"
              },
              {
                "id": "metrics",
                "label": "Metrics"
              },
              {
                "id": "grains",
                "label": "Grains"
              },
              {
                "id": "federation",
                "label": "Federation"
              },
              {
                "id": "materialization",
                "label": "Materialization"
              },
              {
                "id": "versions",
                "label": "Versions"
              },
              {
                "id": "impact",
                "label": "Impact"
              }
            ]
          }
        },
        "tab_overview": {
          "id": "tab_overview",
          "type": "Column",
          "children": [
            "overview_form",
            "overview_hint"
          ],
          "style": {
            "gap": "12px",
            "maxWidth": "560px"
          }
        },
        "tab_dimensions": {
          "id": "tab_dimensions",
          "type": "Column",
          "children": [
            "dims_help",
            "dims_form"
          ],
          "style": {
            "gap": "12px",
            "maxWidth": "720px"
          }
        },
        "tab_metrics": {
          "id": "tab_metrics",
          "type": "Column",
          "children": [
            "metrics_help",
            "metrics_form"
          ],
          "style": {
            "gap": "12px",
            "maxWidth": "720px"
          }
        },
        "tab_grains": {
          "id": "tab_grains",
          "type": "Column",
          "children": [
            "grains_help",
            "grains_form"
          ],
          "style": {
            "gap": "12px",
            "maxWidth": "720px"
          }
        },
        "tab_federation": {
          "id": "tab_federation",
          "type": "Column",
          "children": [
            "fed_dc"
          ],
          "style": {
            "gap": "12px"
          }
        },
        "tab_materialization": {
          "id": "tab_materialization",
          "type": "Column",
          "children": [
            "mat_form",
            "mat_engines"
          ],
          "style": {
            "gap": "12px",
            "maxWidth": "480px"
          }
        },
        "tab_versions": {
          "id": "tab_versions",
          "type": "Column",
          "children": [
            "versions_text"
          ],
          "style": {
            "gap": "12px",
            "maxWidth": "640px"
          }
        },
        "tab_impact": {
          "id": "tab_impact",
          "type": "Column",
          "children": [
            "impact_new_hint",
            "impact_dc"
          ],
          "style": {
            "gap": "12px",
            "maxWidth": "960px"
          }
        }
      }
    }
  }
]$tabs$::jsonb,
    $comp${
  "back": {
    "id": "back",
    "type": "ActionButton",
    "props": {
      "label": "Cubes",
      "icon": "back",
      "variant": "text",
      "onClick": [
        {
          "kind": "navigate",
          "to": "/build/cubes"
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    }
  },
  "title": {
    "id": "title",
    "type": "TextBlock",
    "props": {
      "text": "{{queries.load.data.title}}",
      "variant": "h6"
    },
    "style": {
      "flex": "1 1 240px"
    }
  },
  "unsaved": {
    "id": "unsaved",
    "type": "ActionButton",
    "props": {
      "label": "unsaved",
      "variant": "chip",
      "onClick": []
    },
    "style": {
      "flex": "0 0 auto"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "vars.dirty",
      "operator": "is_true"
    }
  },
  "info_banner": {
    "id": "info_banner",
    "type": "AlertBanner",
    "props": {
      "severity": "info",
      "text": "{{vars.info}}"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "vars.info",
      "operator": "is_not_empty"
    }
  },
  "error_banner": {
    "id": "error_banner",
    "type": "AlertBanner",
    "props": {
      "severity": "error",
      "text": "{{vars.error}}"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "vars.error",
      "operator": "is_not_empty"
    }
  },
  "validate_btn": {
    "id": "validate_btn",
    "type": "ActionButton",
    "props": {
      "label": "Validate",
      "icon": "check",
      "variant": "text",
      "disabledWhen": {
        "type": "condition",
        "field": "queries.load.data.is_new",
        "operator": "is_true"
      },
      "tooltip": "Save the cube first, then validate (API requires an id).",
      "onClick": [
        {
          "kind": "runOperation",
          "operation": "cubes.validate",
          "params": {
            "id": "{{route.id}}",
            "draft": "{{vars.draft}}",
            "federationKeySamples": "{{vars.draft.federationKeySamples}}"
          },
          "onSuccess": [
            {
              "kind": "setVariable",
              "name": "validation",
              "value": "{{result}}"
            },
            {
              "kind": "setVariable",
              "name": "error",
              "value": null
            },
            {
              "kind": "notify",
              "severity": "success",
              "text": "Validation ok",
              "when": {
                "type": "condition",
                "field": "result.ok",
                "operator": "is_true"
              }
            },
            {
              "kind": "notify",
              "severity": "warning",
              "text": "Validation reported issues — see federation/structural flags on the result",
              "when": {
                "type": "condition",
                "field": "result.ok",
                "operator": "is_false"
              }
            }
          ]
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_false"
    }
  },
  "deploy_btn": {
    "id": "deploy_btn",
    "type": "ActionButton",
    "props": {
      "label": "Deploy",
      "icon": "play",
      "variant": "text",
      "disabledWhen": {
        "type": "group",
        "operator": "OR",
        "conditions": [
          {
            "type": "condition",
            "field": "queries.load.data.is_new",
            "operator": "is_true"
          },
          {
            "type": "condition",
            "field": "vars.dirty",
            "operator": "is_true"
          }
        ]
      },
      "onClick": [
        {
          "kind": "runOperation",
          "operation": "cubes.deploy",
          "params": {
            "id": "{{route.id}}",
            "force": false
          },
          "onSuccess": [
            {
              "kind": "notify",
              "severity": "success",
              "text": "Deploy started"
            }
          ],
          "confirm": {
            "title": "Deploy this cube?",
            "text": "Start materialization for the current contract (force=false).",
            "confirmLabel": "Deploy"
          }
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_false"
    }
  },
  "refresh_btn": {
    "id": "refresh_btn",
    "type": "ActionButton",
    "props": {
      "label": "Refresh",
      "icon": "refresh",
      "variant": "text",
      "disabledWhen": {
        "type": "group",
        "operator": "OR",
        "conditions": [
          {
            "type": "condition",
            "field": "queries.load.data.is_new",
            "operator": "is_true"
          },
          {
            "type": "condition",
            "field": "vars.dirty",
            "operator": "is_true"
          }
        ]
      },
      "onClick": [
        {
          "kind": "runOperation",
          "operation": "cubes.refresh",
          "params": {
            "id": "{{route.id}}",
            "force": false
          },
          "onSuccess": [
            {
              "kind": "notify",
              "severity": "success",
              "text": "Refresh requested"
            }
          ],
          "confirm": {
            "title": "Refresh materialization?",
            "text": "force=false may no-op when already dual-committed.",
            "confirmLabel": "Refresh"
          }
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_false"
    }
  },
  "impact_btn": {
    "id": "impact_btn",
    "type": "ActionButton",
    "props": {
      "label": "Impact",
      "icon": "insights",
      "variant": "text",
      "tooltip": "Composition, consumers, and dry-run change preview",
      "onClick": [
        {
          "kind": "setVariable",
          "name": "tab",
          "value": "impact"
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_false"
    }
  },
  "save_create": {
    "id": "save_create",
    "type": "ActionButton",
    "props": {
      "label": "Save",
      "icon": "save",
      "variant": "contained",
      "disabledWhen": {
        "type": "condition",
        "field": "vars.dirty",
        "operator": "is_false"
      },
      "onClick": [
        {
          "kind": "runOperation",
          "operation": "cubes.create",
          "params": {
            "draft": "{{vars.draft}}"
          },
          "onSuccess": [
            {
              "kind": "setVariable",
              "name": "dirty",
              "value": false
            },
            {
              "kind": "setVariable",
              "name": "draft",
              "value": "{{result.draft}}"
            },
            {
              "kind": "setVariable",
              "name": "error",
              "value": null
            },
            {
              "kind": "notify",
              "severity": "success",
              "text": "Cube created"
            },
            {
              "kind": "navigate",
              "to": "/build/cubes/{{result.id}}"
            }
          ]
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_true"
    }
  },
  "save_patch": {
    "id": "save_patch",
    "type": "ActionButton",
    "props": {
      "label": "Save",
      "icon": "save",
      "variant": "contained",
      "disabledWhen": {
        "type": "condition",
        "field": "vars.dirty",
        "operator": "is_false"
      },
      "onClick": [
        {
          "kind": "runOperation",
          "operation": "cubes.patch",
          "params": {
            "id": "{{route.id}}",
            "draft": "{{vars.draft}}"
          },
          "onSuccess": [
            {
              "kind": "setVariable",
              "name": "dirty",
              "value": false
            },
            {
              "kind": "setVariable",
              "name": "draft",
              "value": "{{result.draft}}"
            },
            {
              "kind": "setVariable",
              "name": "error",
              "value": null
            },
            {
              "kind": "notify",
              "severity": "success",
              "text": "Saved"
            }
          ]
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_false"
    }
  },
  "overview_form": {
    "id": "overview_form",
    "type": "Form",
    "props": {
      "variable": "draft",
      "fields": [
        {
          "name": "name",
          "kind": "text",
          "label": "Name",
          "required": true
        },
        {
          "name": "description",
          "kind": "multiline",
          "label": "Description"
        },
        {
          "name": "boId",
          "kind": "select",
          "label": "Business Object",
          "required": true,
          "optionsFrom": {
            "query": "bos",
            "rowsPath": "rows",
            "valueField": "key",
            "labelField": "displayName"
          }
        }
      ],
      "initFrom": "{{queries.load.data.draft}}",
      "seedKey": "{{queries.load.data.key}}",
      "onChange": [
        {
          "kind": "setVariable",
          "name": "dirty",
          "value": true
        }
      ]
    }
  },
  "overview_hint": {
    "id": "overview_hint",
    "type": "TextBlock",
    "props": {
      "text": "Logical BO key is stored on the cube. Changing BO clears authored axes in the coded designer; here clear Dimensions / Metrics / Grains tabs after a BO change.",
      "variant": "caption",
      "color": "text.secondary"
    }
  },
  "dims_help": {
    "id": "dims_help",
    "type": "TextBlock",
    "props": {
      "text": "Ordered dimension surface. Each axis must appear in at least one grain.",
      "color": "text.secondary"
    }
  },
  "dims_form": {
    "id": "dims_form",
    "type": "Form",
    "props": {
      "variable": "draft",
      "fields": [
        {
          "name": "dimensions",
          "kind": "rows",
          "label": "Dimensions",
          "addLabel": "Add dimension",
          "rowFields": [
            {
              "name": "termNodeId",
              "kind": "text",
              "label": "Term node id"
            }
          ]
        }
      ],
      "onChange": [
        {
          "kind": "setVariable",
          "name": "dirty",
          "value": true
        }
      ]
    }
  },
  "metrics_help": {
    "id": "metrics_help",
    "type": "TextBlock",
    "props": {
      "text": "Governed metrics only (metric_definition). Options follow the selected BO.",
      "color": "text.secondary"
    }
  },
  "metrics_form": {
    "id": "metrics_form",
    "type": "Form",
    "props": {
      "variable": "draft",
      "fields": [
        {
          "name": "metricIds",
          "kind": "chips",
          "label": "Metric ids",
          "optionsFrom": {
            "query": "metrics",
            "rowsPath": "rows",
            "valueField": "id",
            "labelField": "name"
          }
        }
      ],
      "onChange": [
        {
          "kind": "setVariable",
          "name": "dirty",
          "value": true
        }
      ]
    }
  },
  "grains_help": {
    "id": "grains_help",
    "type": "TextBlock",
    "props": {
      "text": "Each grain is one materialization shape: an array of dimension term ids (JSON array of string arrays).",
      "color": "text.secondary"
    }
  },
  "grains_form": {
    "id": "grains_form",
    "type": "Form",
    "props": {
      "variable": "draft",
      "fields": [
        {
          "name": "grains",
          "kind": "json",
          "label": "Grains",
          "wide": true
        }
      ],
      "onChange": [
        {
          "kind": "setVariable",
          "name": "dirty",
          "value": true
        }
      ]
    }
  },
  "fed_dc": {
    "id": "fed_dc",
    "type": "DomainComponent",
    "props": {
      "component": "cubes.FederationEditor",
      "inputs": {
        "primaryBoId": "{{vars.draft.boId}}",
        "bos": "{{queries.bos.data.bos}}",
        "federation": "{{vars.draft.federation}}",
        "keySamples": "{{vars.draft.federationKeySamples}}"
      },
      "events": {
        "onChange": [
          {
            "kind": "runOperation",
            "operation": "cubes.patchDraft",
            "params": {
              "draft": "{{vars.draft}}",
              "federation": "{{event.federation}}"
            },
            "onSuccess": [
              {
                "kind": "setVariable",
                "name": "draft",
                "value": "{{result.draft}}"
              },
              {
                "kind": "setVariable",
                "name": "dirty",
                "value": true
              }
            ]
          }
        ],
        "onKeySamplesChange": [
          {
            "kind": "runOperation",
            "operation": "cubes.patchDraft",
            "params": {
              "draft": "{{vars.draft}}",
              "federationKeySamples": "{{event.keySamples}}"
            },
            "onSuccess": [
              {
                "kind": "setVariable",
                "name": "draft",
                "value": "{{result.draft}}"
              },
              {
                "kind": "setVariable",
                "name": "dirty",
                "value": true
              }
            ]
          }
        ]
      }
    }
  },
  "mat_form": {
    "id": "mat_form",
    "type": "Form",
    "props": {
      "variable": "mat",
      "initFrom": "{{vars.draft.materialization}}",
      "seedKey": "{{queries.load.data.key}}",
      "fields": [
        {
          "name": "strategy",
          "kind": "select",
          "label": "Strategy",
          "options": [
            {
              "value": "starrocks_mv",
              "label": "starrocks_mv"
            },
            {
              "value": "aggregate_table",
              "label": "aggregate_table"
            }
          ]
        },
        {
          "name": "stalePolicy",
          "kind": "select",
          "label": "Stale policy",
          "options": [
            {
              "value": "serve_with_flag",
              "label": "serve_with_flag"
            },
            {
              "value": "force_raw_fallback",
              "label": "force_raw_fallback"
            }
          ]
        },
        {
          "name": "refreshStrategy",
          "kind": "select",
          "label": "Refresh strategy",
          "options": [
            {
              "value": "manual",
              "label": "manual"
            },
            {
              "value": "schedule",
              "label": "schedule"
            }
          ]
        }
      ],
      "onChange": [
        {
          "kind": "runOperation",
          "operation": "cubes.patchDraft",
          "params": {
            "draft": "{{vars.draft}}",
            "patch": {
              "materialization": "{{form}}"
            }
          },
          "onSuccess": [
            {
              "kind": "setVariable",
              "name": "draft",
              "value": "{{result.draft}}"
            },
            {
              "kind": "setVariable",
              "name": "mat",
              "value": "{{form}}"
            },
            {
              "kind": "setVariable",
              "name": "dirty",
              "value": true
            }
          ]
        }
      ]
    }
  },
  "mat_engines": {
    "id": "mat_engines",
    "type": "TextBlock",
    "props": {
      "text": "Hot engine: {{vars.draft.materialization.hotEngine}} · Cold engine: {{vars.draft.materialization.coldEngine}}",
      "variant": "caption",
      "color": "text.secondary"
    }
  },
  "versions_text": {
    "id": "versions_text",
    "type": "TextBlock",
    "props": {
      "text": "Current contract_version: {{queries.load.data.contractVersion}}. Additive edits use Save (PATCH). Breaking grain/dimension/metric/federation changes require POST /api/cubes/{id}/versions. Use the Impact tab to preview consumers before archive or publish."
    }
  },
  "impact_new_hint": {
    "id": "impact_new_hint",
    "type": "AlertBanner",
    "props": {
      "severity": "info",
      "text": "Save the cube first to assess consumers and preview archive / contract publish."
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_true"
    }
  },
  "impact_dc": {
    "id": "impact_dc",
    "type": "DomainComponent",
    "props": {
      "component": "cubes.ImpactPanel",
      "inputs": {
        "cubeId": "{{route.id}}",
        "draft": "{{vars.draft}}",
        "title": "Impact — {{queries.load.data.title}}"
      }
    },
    "visibleWhen": {
      "type": "condition",
      "field": "queries.load.data.is_new",
      "operator": "is_false"
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
    "padding": 0
  },
  "tabVariable": "tab",
  "variables": [
    {
      "name": "draft",
      "initFrom": {
        "query": "load",
        "path": "draft"
      },
      "description": "Working cube draft"
    },
    {
      "name": "mat",
      "description": "Materialization form slice mirrored into draft.materialization"
    },
    {
      "name": "dirty",
      "default": false,
      "description": "Unsaved changes"
    },
    {
      "name": "tab",
      "default": "overview",
      "description": "Active designer tab"
    },
    {
      "name": "validation",
      "description": "Last validate response / banner payload"
    },
    {
      "name": "info",
      "description": "Info banner"
    },
    {
      "name": "error",
      "description": "Error banner"
    }
  ],
  "queries": [
    {
      "id": "load",
      "operation": "cubes.editorStart",
      "params": {
        "id": "{{route.id}}"
      }
    },
    {
      "id": "bos",
      "operation": "cubes.businessObjects",
      "params": {}
    },
    {
      "id": "metrics",
      "operation": "cubes.metrics",
      "params": {
        "boId": "{{vars.draft.boId}}"
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
