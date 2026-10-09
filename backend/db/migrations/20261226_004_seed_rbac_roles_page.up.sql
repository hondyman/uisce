-- 20261226_004_seed_rbac_roles_page.up.sql
-- Seeds the core Page Studio page "Roles" (slug rbac-roles) in the gold-copy tenant, served at
-- /admin/rbac/roles via STUDIO_ROUTES. Replaces the hand-built RoleManagerPage.
-- JSON is GENERATED from frontend/src/pages/page-studio/app/blueprints/rbacRoles.ts;
-- vitest rbacRolesSeedParity.test.ts fails if they differ. Change the blueprint, then
-- regenerate this block, never one without the other.
--
-- Retired from the old page: its sample roles, shown whenever the API returned none; the role
-- list is now only what the API returns. The 'Assign users' placeholder now opens the user-role page.

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '519fe1bc-96d8-4115-acb6-edb80b763da0',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Roles', 'rbac-roles', 'Roles for this tenant: create and edit roles, see who holds them and what field permissions they grant. Built in Page Studio.',
    $layout${
  "root": "page_root",
  "nodes": {
    "page_root": {
      "id": "page_root",
      "type": "Column",
      "children": [
        "top",
        "filters",
        "grid",
        "role_dialog",
        "view_dialog"
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
    "filters": {
      "id": "filters",
      "type": "Row",
      "children": [
        "q",
        "level"
      ],
      "style": {
        "alignItems": "center",
        "gap": "12px",
        "flexWrap": "wrap"
      }
    },
    "role_dialog": {
      "id": "role_dialog",
      "type": "Dialog",
      "children": [
        "role_form"
      ],
      "props": {
        "title": "{{queries.roleStart.data.title}}",
        "maxWidth": "sm",
        "openWhen": {
          "type": "condition",
          "field": "vars.roleOpen",
          "operator": "is_true"
        },
        "onClose": [
          {
            "kind": "setVariable",
            "name": "roleOpen",
            "value": false
          },
          {
            "kind": "setVariable",
            "name": "roleDraft",
            "value": null
          }
        ],
        "buttons": [
          {
            "label": "Cancel",
            "onClick": [
              {
                "kind": "setVariable",
                "name": "roleOpen",
                "value": false
              },
              {
                "kind": "setVariable",
                "name": "roleDraft",
                "value": null
              }
            ]
          },
          {
            "label": "Save",
            "variant": "contained",
            "disabledWhen": {
              "type": "condition",
              "field": "vars.roleDraft.role_name",
              "operator": "is_empty"
            },
            "onClick": [
              {
                "kind": "runOperation",
                "operation": "rbac.saveRole",
                "params": {
                  "draft": "{{vars.roleDraft}}"
                },
                "onSuccess": [
                  {
                    "kind": "setVariable",
                    "name": "roleOpen",
                    "value": false
                  },
                  {
                    "kind": "setVariable",
                    "name": "roleDraft",
                    "value": null
                  }
                ],
                "successMessage": "Role saved"
              }
            ]
          }
        ]
      }
    },
    "view_dialog": {
      "id": "view_dialog",
      "type": "Dialog",
      "children": [
        "view_form",
        "view_users",
        "view_perms"
      ],
      "props": {
        "title": "Role details",
        "maxWidth": "md",
        "openWhen": {
          "type": "condition",
          "field": "vars.roleViewOpen",
          "operator": "is_true"
        },
        "onClose": [
          {
            "kind": "setVariable",
            "name": "roleViewOpen",
            "value": false
          },
          {
            "kind": "setVariable",
            "name": "roleViewId",
            "value": null
          }
        ],
        "buttons": [
          {
            "label": "Close",
            "onClick": [
              {
                "kind": "setVariable",
                "name": "roleViewOpen",
                "value": false
              },
              {
                "kind": "setVariable",
                "name": "roleViewId",
                "value": null
              }
            ]
          },
          {
            "label": "Assign users",
            "onClick": [
              {
                "kind": "navigate",
                "to": "/admin/rbac/user-roles"
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
      "icon": "gavel",
      "title": "Roles",
      "subtitle": "Roles for this tenant. Gold-copy roles are shared and read-only here."
    },
    "style": {
      "flex": "1 1 320px"
    }
  },
  "new_btn": {
    "id": "new_btn",
    "type": "ActionButton",
    "props": {
      "label": "New role",
      "icon": "add",
      "variant": "contained",
      "onClick": [
        {
          "kind": "setVariable",
          "name": "roleEditing",
          "value": ""
        },
        {
          "kind": "setVariable",
          "name": "roleOpen",
          "value": true
        }
      ]
    },
    "style": {
      "flex": "0 0 auto"
    }
  },
  "q": {
    "id": "q",
    "type": "SearchInput",
    "props": {
      "variable": "roleSearch",
      "placeholder": "Search roles",
      "maxWidth": 520
    },
    "style": {
      "flex": "1 1 260px"
    }
  },
  "level": {
    "id": "level",
    "type": "VariableSelect",
    "props": {
      "variable": "roleLevel",
      "label": "",
      "emptyLabel": "All levels",
      "minWidth": 170,
      "options": [
        {
          "value": "viewer",
          "label": "Viewer"
        },
        {
          "value": "editor",
          "label": "Editor"
        },
        {
          "value": "approver",
          "label": "Approver"
        },
        {
          "value": "admin",
          "label": "Admin"
        },
        {
          "value": "super_admin",
          "label": "Super Admin"
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
      "query": "roles",
      "rowsPath": "rows",
      "emptyText": "No roles match. Use \"New role\" to add one.",
      "columns": [
        {
          "id": "name",
          "header": "Role",
          "stack": [
            {
              "kind": "text",
              "value": "{{row.role_name}}",
              "bold": true
            },
            {
              "kind": "text",
              "value": "{{row.role_key}}",
              "caption": true
            }
          ]
        },
        {
          "id": "level",
          "header": "Level",
          "cell": {
            "kind": "chip",
            "value": "{{row.level_label}}",
            "variant": "outlined"
          }
        },
        {
          "id": "origin",
          "header": "Origin",
          "cell": {
            "kind": "chip",
            "value": "{{row.origin_label}}",
            "variant": "outlined"
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
                "label": "View",
                "icon": "preview",
                "onClick": [
                  {
                    "kind": "setVariable",
                    "name": "roleViewId",
                    "value": "{{row.id}}"
                  },
                  {
                    "kind": "setVariable",
                    "name": "roleViewOpen",
                    "value": true
                  }
                ]
              },
              {
                "label": "Edit",
                "icon": "edit",
                "visibleWhen": {
                  "type": "condition",
                  "field": "row.editable",
                  "operator": "is_true"
                },
                "onClick": [
                  {
                    "kind": "setVariable",
                    "name": "roleEditing",
                    "value": "{{row.id}}"
                  },
                  {
                    "kind": "setVariable",
                    "name": "roleOpen",
                    "value": true
                  }
                ]
              },
              {
                "label": "Remove",
                "icon": "delete",
                "visibleWhen": {
                  "type": "condition",
                  "field": "row.editable",
                  "operator": "is_true"
                },
                "onClick": [
                  {
                    "kind": "runOperation",
                    "operation": "rbac.removeRole",
                    "params": {
                      "id": "{{row.id}}"
                    },
                    "onSuccess": [],
                    "confirm": {
                      "title": "Remove this role?",
                      "text": "{{row.role_name}} will stop being listed and can no longer be assigned.",
                      "confirmLabel": "Remove"
                    },
                    "successMessage": "Role removed"
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
  "role_form": {
    "id": "role_form",
    "type": "Form",
    "props": {
      "variable": "roleDraft",
      "fields": [
        {
          "name": "role_key",
          "kind": "text",
          "label": "Role key",
          "required": true,
          "wide": true,
          "readOnlyWhen": {
            "type": "condition",
            "field": "vars.roleDraft.id",
            "operator": "is_not_empty"
          },
          "helperText": "Cannot be changed once the role exists."
        },
        {
          "name": "role_name",
          "kind": "text",
          "label": "Role name",
          "required": true,
          "wide": true
        },
        {
          "name": "description",
          "kind": "multiline",
          "label": "Description",
          "wide": true
        },
        {
          "name": "role_level",
          "kind": "select",
          "label": "Level",
          "required": true,
          "readOnlyWhen": {
            "type": "condition",
            "field": "vars.roleDraft.id",
            "operator": "is_not_empty"
          },
          "helperText": "Set when the role is created.",
          "options": [
            {
              "value": "viewer",
              "label": "Viewer"
            },
            {
              "value": "editor",
              "label": "Editor"
            },
            {
              "value": "approver",
              "label": "Approver"
            },
            {
              "value": "admin",
              "label": "Admin"
            },
            {
              "value": "super_admin",
              "label": "Super Admin"
            }
          ]
        }
      ],
      "initFrom": "{{queries.roleStart.data.draft}}",
      "seedKey": "{{queries.roleStart.data.key}}"
    }
  },
  "view_form": {
    "id": "view_form",
    "type": "Form",
    "props": {
      "variable": "roleViewDraft",
      "fields": [
        {
          "name": "role_key",
          "kind": "text",
          "label": "Role key",
          "wide": true,
          "readOnlyWhen": {
            "type": "condition",
            "field": "vars.roleViewOpen",
            "operator": "is_true"
          }
        },
        {
          "name": "role_name",
          "kind": "text",
          "label": "Role name",
          "wide": true,
          "readOnlyWhen": {
            "type": "condition",
            "field": "vars.roleViewOpen",
            "operator": "is_true"
          }
        },
        {
          "name": "description",
          "kind": "multiline",
          "label": "Description",
          "wide": true,
          "readOnlyWhen": {
            "type": "condition",
            "field": "vars.roleViewOpen",
            "operator": "is_true"
          }
        },
        {
          "name": "role_level",
          "kind": "select",
          "label": "Level",
          "options": [
            {
              "value": "viewer",
              "label": "Viewer"
            },
            {
              "value": "editor",
              "label": "Editor"
            },
            {
              "value": "approver",
              "label": "Approver"
            },
            {
              "value": "admin",
              "label": "Admin"
            },
            {
              "value": "super_admin",
              "label": "Super Admin"
            }
          ],
          "readOnlyWhen": {
            "type": "condition",
            "field": "vars.roleViewOpen",
            "operator": "is_true"
          }
        }
      ],
      "initFrom": "{{queries.roleView.data.draft}}",
      "seedKey": "{{queries.roleView.data.key}}"
    }
  },
  "view_users": {
    "id": "view_users",
    "type": "DataGrid",
    "props": {
      "query": "roleUsers",
      "rowsPath": "rows",
      "emptyText": "No users assigned to this role.",
      "columns": [
        {
          "id": "user",
          "header": "User",
          "stack": [
            {
              "kind": "text",
              "value": "{{row.name}}",
              "bold": true
            },
            {
              "kind": "text",
              "value": "{{row.email}}",
              "caption": true
            }
          ]
        },
        {
          "id": "assigned",
          "header": "Assigned",
          "cell": {
            "kind": "text",
            "value": "{{row.assigned_text}}"
          },
          "nowrap": true
        }
      ]
    }
  },
  "view_perms": {
    "id": "view_perms",
    "type": "DataGrid",
    "props": {
      "query": "roleFieldPerms",
      "rowsPath": "rows",
      "emptyText": "No field permissions configured.",
      "columns": [
        {
          "id": "field",
          "header": "Field",
          "cell": {
            "kind": "text",
            "value": "{{row.field}}",
            "bold": true
          }
        },
        {
          "id": "resource",
          "header": "Resource",
          "cell": {
            "kind": "text",
            "value": "{{row.resource_type}}"
          }
        },
        {
          "id": "level",
          "header": "Permission",
          "cell": {
            "kind": "chip",
            "value": "{{row.permission_level}}",
            "colorMap": {
              "write": "success",
              "read": "info",
              "*": "warning"
            },
            "variant": "outlined"
          }
        }
      ]
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
      "name": "roleSearch",
      "default": "",
      "description": "Role name or key filter"
    },
    {
      "name": "roleLevel",
      "default": "",
      "description": "Role level filter"
    },
    {
      "name": "roleOpen",
      "default": false,
      "description": "Whether the role editor is open"
    },
    {
      "name": "roleEditing",
      "default": "",
      "description": "The role being edited; empty = a new one"
    },
    {
      "name": "roleDraft",
      "description": "The role being edited"
    },
    {
      "name": "roleViewOpen",
      "default": false,
      "description": "Whether the role details are open"
    },
    {
      "name": "roleViewId",
      "default": "",
      "description": "The role shown in the details"
    },
    {
      "name": "roleViewDraft",
      "description": "The role shown in the details (read-only)"
    }
  ],
  "queries": [
    {
      "id": "roles",
      "operation": "rbac.listRoles",
      "params": {
        "q": "{{vars.roleSearch}}",
        "level": "{{vars.roleLevel}}"
      },
      "keepPrevious": true
    },
    {
      "id": "roleStart",
      "operation": "rbac.roleStart",
      "params": {
        "id": "{{vars.roleEditing}}",
        "open": "{{vars.roleOpen}}"
      },
      "enabledWhen": {
        "type": "condition",
        "field": "vars.roleOpen",
        "operator": "is_true"
      }
    },
    {
      "id": "roleView",
      "operation": "rbac.roleStart",
      "params": {
        "id": "{{vars.roleViewId}}",
        "open": "{{vars.roleViewOpen}}"
      },
      "enabledWhen": {
        "type": "condition",
        "field": "vars.roleViewOpen",
        "operator": "is_true"
      }
    },
    {
      "id": "roleUsers",
      "operation": "rbac.roleUsers",
      "params": {
        "roleId": "{{vars.roleViewId}}"
      },
      "enabledWhen": {
        "type": "condition",
        "field": "vars.roleViewOpen",
        "operator": "is_true"
      }
    },
    {
      "id": "roleFieldPerms",
      "operation": "rbac.roleFieldPermissions",
      "params": {
        "roleId": "{{vars.roleViewId}}"
      },
      "enabledWhen": {
        "type": "condition",
        "field": "vars.roleViewOpen",
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
