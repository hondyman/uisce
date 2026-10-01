# Core objects: how tenants use, extend and upgrade them

Core objects are authored in the gold-copy tenant and inherited by every
other tenant. A tenant never writes a core object. For each one it chooses:

| Choice | What the tenant gets | On a new core version |
| --- | --- | --- |
| **Inactive** | The object is switched off in their environment (`active = false`). Any extension is kept for when it is switched back on. | Nothing to do. |
| **Vanilla** (default) | The object as the gold copy ships it. | They get it automatically. |
| **Extended** | Their customized copy. Every customization is tracked against the core version it was made on. | They compare with the new core, keep or remove each customization, and upgrade. |
| **Cloned** | An independent tenant copy that replaces the core object. | Nothing. A clone has no upgrade path. |

Page Studio is the first object type with this lifecycle. It is built to be
reused by every core object type.

## Model

`public.core_object_adoption` (migration `20261116_001`) has one row per
(tenant, object type, core object). No row means vanilla and active.

- `active`: false means switched off.
- `mode`: `vanilla`, `extended` or `cloned`.
- For an extension: `extension` is the tenant's full copy, and
  `base_snapshot` / `base_version` is the core as it was when they last
  took it.
- For a clone: `clone_object_id` is the tenant-owned copy.

RLS scopes rows to the current tenant.

## Compare and upgrade (`backend/internal/corecustom`)

The engine is generic and knows nothing about pages. It diffs JSON
documents:

- Objects are diffed by key.
- Arrays of objects are matched by `id`, then `key`, then `name`.
- Arrays of scalars (e.g. a layout node's `children`) are diffed by
  membership, plus a separate reorder change.

This means "added a widget" shows up as a couple of changes, not as every
later array index shifting.

- **Customizations** = `Diff(base, extension)`.
- **Core updates** = `Diff(base, current core)`.
- **Conflict**: a customization and a core update touch the same element.
  Keep means the tenant's version wins; Remove means the core's version
  wins.
- **In core**: the core now ships exactly that customization.
- **Merge**: replays every customization the tenant keeps onto the current
  core. If the core deleted an element the tenant had customized, keeping
  the customization restores the tenant's whole element.
- After an upgrade the base becomes the current core. If no customizations
  remain, the object goes back to vanilla.

A **grouper** supplied by each object type turns low-level changes into the
units a person decides about. The page grouper (`pageGrouper` in
`page_studio_core.go`) groups by widget, layout container, tab, data
source, presentation rule, app-model entry, or page details. A widget's
placement in a container groups with the widget itself.

## Page Studio

- **Gold copy.** Pages the gold-copy admin creates are core by default,
  including every MDM page. `isCore: false` opts out. Saving a core page
  ships a new core version (`version` + 1).
- **Tenant admin** (`tenant_admin`, `admin`, or a global admin working in
  a regular tenant). On a core page's menu in the page list:
  - Extend or Edit customization
  - Compare with core or Review upgrade
  - Switch off or Switch on
  - Clone (no upgrades)
  - Revert to core

  Saving a core page in the designer stores the tenant's extension, never
  the core page.
- **Runtime** (`GET /page-studio/pages/slug/{slug}`):
  - The tenant's own page at that slug wins, which is how a clone replaces
    the core page.
  - Otherwise the core page is served as the tenant uses it: its extension
    if extended, or 404 if switched off.
- **Endpoints**, all under `/api/page-studio/pages/{id}` (the core page's
  id):

  | Method and path | Action |
  | --- | --- |
  | `PUT extension` | Save the tenant's extension |
  | `PUT activation` | `{active}` |
  | `POST clone` | Clone the core page |
  | `DELETE customization` | Revert to core |
  | `GET compare` | Compare with core |
  | `POST upgrade` | `{remove: [groupId]}` |

`page_definition_overlays` (the earlier add-only overlay) is gone: its rows
were copied into `core_object_adoption` (20261116_001) and the table dropped
(20261117_001).

## Next object types

For each new type:

1. Define the object's content as a JSON document.
2. Write a grouper for it.
3. Resolve the effective object per tenant on read, as `presentCore`
   does.
4. Add the same six endpoints.

The table and the engine are shared.
