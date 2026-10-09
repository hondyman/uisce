-- 20261226_002_seed_core_domains_page.up.sql
-- Seeds the core Page Studio page "Data domains" (slug core-domains) in the gold-copy tenant, served at
-- /core/domains via STUDIO_ROUTES. Replaces the hand-built DomainsManagementPage.
-- JSON is GENERATED from frontend/src/pages/page-studio/app/blueprints/coreDomains.ts;
-- vitest coreDomainsSeedParity.test.ts fails if they differ. Change the blueprint, then
-- regenerate this block, never one without the other.
--
-- The Abbreviations tab is not on this page: it stays at /core/abbreviations (the menu links there).

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '6819d67c-1e98-4291-8d12-0a79f29e2c23',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Data domains', 'core-domains', 'The shared data-domain taxonomy: levels, parents, and descriptions. Core administrators make changes. Built in Page Studio.',
    $layout${
  "root": "page_root",
  "nodes": {
    "page_root": {
      "id": "page_root",
      "type": "Column",
      "children": [
        "top",
        "grid",
        "domain_dialog"
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
        "new_btn"
      ],
      "style": {
        "alignItems": "center",
        "gap": "12px",
        "flexWrap": "wrap"
      }
    },
    "domain_dialog": {
      "id": "domain_dialog",
      "type": "Dialog",
      "children": [
        "domain_form"
      ],
      "props": {
        "title": "{{queries.domainStart.data.title}}",
        "maxWidth": "sm",
        "openWhen": {
          "type": "condition",
          "field": "vars.domainOpen",
          "operator": "is_true"
        },
        "onClose": [
          {
            "kind": "setVariable",
            "name": "domainOpen",
            "value": false
          },
          {
            "kind": "setVariable",
            "name": "domainDraft",
            "value": null
          }
        ],
        "buttons": [
          {
            "label": "Cancel",
            "onClick": [
              {
                "kind": "setVariable",
                "name": "domainOpen",
                "value": false
              },
              {
                "kind": "setVariable",
                "name": "domainDraft",
                "value": null
              }
            ]
          },
          {
            "label": "Save",
            "variant": "contained",
            "disabledWhen": {
              "type": "condition",
              "field": "vars.domainDraft.name",
              "operator": "is_empty"
            },
            "onClick": [
              {
                "kind": "runOperation",
                "operation": "domains.save",
                "params": {
                  "draft": "{{vars.domainDraft}}"
                },
                "onSuccess": [
                  {
                    "kind": "setVariable",
                    "name": "domainOpen",
                    "value": false
                  },
                  {
                    "kind": "setVariable",
                    "name": "domainDraft",
                    "value": null
                  }
                ],
                "successMessage": "Domain saved"
              }
            ]
          }
        ]
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
      "icon": "hub",
      "title": "Data domains",
      "subtitle": "The domain taxonomy every tenant shares. Core administrators make changes."
    },
    "style": {
      "flex": "1 1 320px"
    }
  },
  "new_btn": {
    "id": "new_btn",
    "type": "ActionButton",
    "props": {
      "label": "New domain",
      "icon": "add",
      "variant": "contained",
      "onClick": [
        {
          "kind": "setVariable",
          "name": "domainEditing",
          "value": ""
        },
        {
          "kind": "setVariable",
          "name": "domainOpen",
          "value": true
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
      "query": "domains",
      "rowsPath": "rows",
      "emptyText": "No domains yet. Use \"New domain\" to add the first one.",
      "columns": [
        {
          "id": "name",
          "header": "Name",
          "stack": [
            {
              "kind": "text",
              "value": "{{row.name}}",
              "bold": true
            },
            {
              "kind": "text",
              "value": "{{row.slug}}",
              "caption": true
            }
          ]
        },
        {
          "id": "level",
          "header": "Level",
          "cell": {
            "kind": "text",
            "value": "{{row.level}}"
          },
          "align": "right"
        },
        {
          "id": "parent",
          "header": "Parent",
          "cell": {
            "kind": "text",
            "value": "{{row.parent_name}}"
          }
        },
        {
          "id": "description",
          "header": "Description",
          "cell": {
            "kind": "text",
            "value": "{{row.description}}"
          }
        },
        {
          "id": "act",
          "header": "",
          "cell": {
            "kind": "actions",
            "buttons": [
              {
                "label": "Edit",
                "icon": "edit",
                "onClick": [
                  {
                    "kind": "setVariable",
                    "name": "domainEditing",
                    "value": "{{row.id}}"
                  },
                  {
                    "kind": "setVariable",
                    "name": "domainOpen",
                    "value": true
                  }
                ]
              },
              {
                "label": "Delete",
                "icon": "delete",
                "onClick": [
                  {
                    "kind": "runOperation",
                    "operation": "domains.remove",
                    "params": {
                      "id": "{{row.id}}"
                    },
                    "onSuccess": [],
                    "confirm": {
                      "title": "Delete this domain?",
                      "text": "{{row.name}} will be removed.",
                      "confirmLabel": "Delete"
                    },
                    "successMessage": "Domain deleted"
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
  "domain_form": {
    "id": "domain_form",
    "type": "Form",
    "props": {
      "variable": "domainDraft",
      "fields": [
        {
          "name": "name",
          "kind": "text",
          "label": "Name",
          "required": true,
          "wide": true
        },
        {
          "name": "slug",
          "kind": "text",
          "label": "Slug",
          "helperText": "Leave blank to generate one.",
          "wide": true
        },
        {
          "name": "parent_id",
          "kind": "select",
          "label": "Parent domain",
          "helperText": "Leave empty for a top-level domain.",
          "optionsFrom": {
            "query": "domains",
            "rowsPath": "rows",
            "valueField": "id",
            "labelField": "name"
          }
        },
        {
          "name": "description",
          "kind": "multiline",
          "label": "Description",
          "wide": true
        }
      ],
      "initFrom": "{{queries.domainStart.data.draft}}",
      "seedKey": "{{queries.domainStart.data.key}}"
    }
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
    "maxWidth": 1200,
    "padding": 3
  },
  "variables": [
    {
      "name": "domainOpen",
      "default": false,
      "description": "Whether the domain editor is open"
    },
    {
      "name": "domainEditing",
      "default": "",
      "description": "The domain being edited; empty = a new one"
    },
    {
      "name": "domainDraft",
      "description": "The domain being edited"
    }
  ],
  "queries": [
    {
      "id": "domains",
      "operation": "domains.list",
      "params": {},
      "keepPrevious": true
    },
    {
      "id": "domainStart",
      "operation": "domains.editorStart",
      "params": {
        "id": "{{vars.domainEditing}}",
        "open": "{{vars.domainOpen}}"
      },
      "enabledWhen": {
        "type": "condition",
        "field": "vars.domainOpen",
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
