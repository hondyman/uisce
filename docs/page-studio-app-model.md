# Page Studio: the page application model

**Goal.** Page Studio builds the platform's own best pages. The acceptance
test was the hand-built mastering console (`MasteringPage.tsx`, now retired):
the same console, built as a Page Studio page (`app/blueprints/masteringConsole.ts`),
renders the same header, tabs, counts and cells from the same API, cell for
cell - the hand-built output is kept as the expectations in
`src/vitest/page-studio/masteringConsoleParity.test.tsx`.

## What the console needed that the studio lacked

Reverse-engineered from the hand-built pages. Each row is a capability the
studio now has.

| Hand-built pattern | Studio construct |
| --- | --- |
| `useState` for entity, tab, open record, filters, dialog flags | `app.variables` (defaults, seed from a query, optional URL sync) |
| `useQuery(masteringApi.x(entity, filters))` | `app.queries` → a **registered operation** id + bound params |
| `useMutation` + `invalidateQueries(['mastering'])` | `runOperation` action; the domain's cache prefix refreshes |
| Icon + title + subtitle + toolbar | `PageHeader`, `VariableSelect`, `ActionButton` in a Row (`style.flex`) |
| Debounced search, status selects, "All" option | `SearchInput`, `VariableSelect` (`emptyLabel`, static or query options) |
| Tabs with warning badges; tab hidden for time series | `PageTab.badge` (binding), `PageTab.visibleWhen`, `app.tabVariable` |
| Tables with chips, captions, monospace, tabular numbers, coloured deltas, count chips, tooltips | `DataGrid` + cell kinds: text, twoLine, number, percent, delta, datetime, chip (`colorMap`, `colorBy`, `labelKey`), chips, diff, link, actions, input; stacked cells per column |
| Per-row buttons by state (mine / voted / can vote) | row buttons with `visibleWhen` on `row.state` |
| Inline comment box per row | `input` cell → `{{rowState.comment}}` |
| Merge dialog (radio keep a/b + note), approval notice | `runOperation.form` (fields, `notice.visibleWhen`) |
| Last-run alert with count chips | `AlertBanner` (severity map, chips, `onClose`) |
| Provenance drawer, source matrix, run wizard, policy dialog | **Domain components** placed on the page, inputs bound, events → actions |
| Own surface (dark shell canvas), max width 1400 | `app.surface`, `app.chrome: 'none'` |

## Building blocks for record drawers and editors

Rich pages - a record drawer with its own tabs, an editor dialog with a
form - are built from these by configuration, not placed as hand-built
domain components:

| Block | Kind | What it does |
| --- | --- | --- |
| **Drawer** | layout container | Side overlay holding any widgets/containers; open while `openWhen` (a rule-engine condition, usually a variable an action sets) holds; `onClose` actions; title/subtitle templates; optional footer buttons |
| **Dialog** | layout container | The same as a modal, with footer buttons (`disabledWhen`, `visibleWhen`, `onClick` actions) |
| **TabSet** | layout container | Tabs inside any region (a drawer's own tabs); tab *i* shows child *i*; badges, `visibleWhen`, optional tab variable so actions can switch tabs |
| **KeyValue** | widget | Label/value pairs of a record (`{{data.x}}`, or any cell kind), or pairs from a list/object |
| **Timeline** | widget | Versions, runs, events from a query or binding; title/subtitle/time/chip per item; `selectedWhen`, `onItemClick` |
| **Form** | widget | Fields bound to a page variable (`{{vars.draft.x}}`), seeded by `initFrom`; text, long text, number, date, select, radio, switch, chips; options static or **from a query**; per-field `visibleWhen` / `readOnlyWhen`; optional submit actions (or a dialog's buttons) |

Grids and cells for record detail (the provenance drawer's needs):

- **DataGrid `dynamicColumns`** - columns generated from data (one per
  source): each entry of `from` becomes a column whose cell sees `{{col}}`
  and `{{value}}` = `row[valuePath][col[idField]]`; `stickyFirstColumn` for
  wide matrices.
- **DataGrid `rowDetail`** - an expandable nested table under a row (the
  values that competed for an attribute, and why).
- **Cells** - `list` (several values in one cell, each rendered by an item
  cell), text `tone` (tints the value that won) / `strike` (an excluded
  value) / `tooltip`, chips whose entries carry their own colour, row
  buttons with an `icon`.
- **VariableSelect `variant: 'toggle'`**, **AlertBanner `action`**,
  **TextBlock** (headings and captions), a Drawer `width` that can be a
  binding (wider in one view than another).

Action forms (`runOperation.form`) use the same field kinds, options from a
query included (`app/formFields.tsx`). In Page Designer the containers show
as editable regions (a drawer's content is edited in place, with what opens
it read out); each block has a structured inspector plus a JSON view.

## Governance boundaries (deliberate)

- **No URLs, SQL or scripts in pages.** Data comes from operations a domain
  registers (`studio-core/operations/registry.ts`). Tenant isolation, auth
  and validation stay in the domain's endpoints.
- **One rule engine.** Every show/hide/enable/row-state decision is the
  engine's own `RuleNode` JSON, evaluated by `rule_engine.wasm`
  (`app/conditions.ts`). Bindings are path lookups and `{{template}}`s only -
  no operators - so they are plumbing, not a second evaluator. The engine
  tests (`appConditions.test.ts`) run the real wasm, not a stub.
  - Engine semantics to know: `null not_equals true` is **false**; write
    "not loaded or false" as `OR(is_empty(x), is_false(x.series))`.
- **Domains decide, pages display.** Derived row fields (`state`,
  `merge_caption`, `winning_label`), counts and result messages live in the
  operation or domain-component adapter (`features/mastering/studio.tsx`).
- **Hand-built quality is not flattened.** Rich domain UI is registered as a
  domain component with a declared contract (inputs, events) rather than
  approximated by generic widgets.

## Live

Every MDM screen is a core Page Studio page, served at its app route
(`pages/page-studio/studioRoutes.ts`) and placed on the gold copy's
**Master Data** menu section, which every tenant inherits:

| Page (slug) | Route | Built from |
| --- | --- | --- |
| `mastering-console` | `/data/mastering` | `blueprints/masteringConsole.ts` |
| `staging-bindings` | `/data/staging-bindings` | `blueprints/stagingBindings.ts` (operations + editor in `features/staging-bindings/studio.tsx`) |
| `data-pipelines` | `/data/pipelines` | `blueprints/dataPipelines.ts` (operations in `features/data-pipelines/studio.tsx`) |
| `data-pipeline-editor` | `/data/pipelines/:id` | the domain's visual editor placed as a domain component |

Tenants see each page as they use it (extended, cloned, or not at all if
switched off - `docs/core-customization.md`). The hand-built screens are
retired; an environment without a page saved says so and points to
**From blueprint**. The parity test keeps the hand-built console's recorded
output as the studio console's expectations.

Page Studio shows each page's menu entries and routes, filters by menu
section, and places a page on the menu (**Place on menu…**). Menus follow
the same inheritance as core pages: gold-copy entries are read-only for
tenants, who can add their own entries under them; an entry for a core page
a tenant switched off is hidden.

## Where things are

- Types: `pages/page-studio/app/appModel.ts`; `CorePageDefinition.app`,
  `PageTab.badge/visibleWhen`, `ComponentDefinition.visibleWhen` in `types/pageStudio.ts`.
- Runtime: `app/AppRuntime.tsx` (variables, queries, actions, form/confirm host),
  `app/AppWidgets.tsx`, `app/cells.tsx`, `app/RuntimePage.tsx` (shared by
  Preview and PageBrowser, so authors see what viewers get).
- Editor: palette group **App**; **App** tab (`app/AppModelPanel.tsx`: variables,
  queries, tab counts/conditions, chrome, operation and component catalogs);
  inspector for app widgets with a column/cell editor and a JSON view
  (`app/AppWidgetInspector.tsx`, shared editors in `app/editors.tsx`).
- Blueprints: `app/blueprints/`; **From blueprint** on the page list opens one
  as an unsaved draft (`/page-studio/new?blueprint=mastering-console`).
- Persistence: `page_definitions.app_model` (jsonb, NULL on plain BO pages),
  migration `20261114_001_page_definitions_app_model`; handler round-trip
  tests in `page_studio_app_model_test.go`.

## Adding a domain

1. `features/<domain>/studio.tsx`: `registerOperations([...])` (queries with
   `fields` for the column editor; mutations with `invalidates` if the cache
   prefix is not the domain) and `registerDomainComponents([...])`.
2. Import it from `studio-core/registerDomains.ts`.

## Not done yet (next)

- Structured editors for row buttons and action forms (JSON today).
- Studio primitives to decompose the drawers themselves (a Drawer/Dialog
  layout node, key-value and version-list widgets) so GoldenDrawer can move
  from domain component to studio-built.
- Row buttons do not disable while their mutation is in flight (the
  hand-built page does).
- Tenant customization of core pages (inactive / vanilla / extended /
  cloned, compare and upgrade) is in `docs/core-customization.md`. An
  extension is the whole page, `app` included.
- Operation catalog is registered in the frontend; a backend catalog would
  let MCP and governance list page capabilities server-side.
