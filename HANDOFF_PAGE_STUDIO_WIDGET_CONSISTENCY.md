# Handoff: Page Studio widget consistency + Form field designer rebuild

This doc covers the Page Studio work from this session: fixing two real
backend bugs that were blocking the Order Detail page, rebuilding the Form
field designer UX, extending "no live data in design mode" to every widget
type, and bringing Style-tab parity to LineChart/KPIGroup/Slicer so they have
the same level of configurability Table already had.

## Context: how this started

Working session on the Order Detail page in Page Studio (northwind tenant)
surfaced two real production bugs, then a series of explicit UX complaints
about the Form field designer, then a final ask for cross-widget consistency.
Read the user's asks in order — each one is a separate, addressed request:

1. Order Detail's related-executions table needed to know the FK/cardinality
   between Order and Execution Business Objects. Requirement: use the
   platform's **existing centralized** relationship metadata, don't duplicate
   it.
2. Design mode should never show live data for *any* widget type (previously
   only true for Form), and Save must still work.
3. Harsh, direct feedback that the field designer was broken: couldn't
   resize fields, no visible data binding, no editable label/style.
4. Follow-ups: field deletion, "change field to one of the same type", type
   icons on data-binding UI.
5. Final ask: Table has rich Properties/Style controls; other widget types
   (LineChart, KPIGroup, Slicer) don't — make them consistent.

## What's real and working now

### Centralized BO relationship metadata (backend)

`BusinessObjectService.GetBusinessObjectRelationships`
(`backend/internal/metadata/businessobject_service.go`), exposed at
`GET /api/business-objects/{boId}/relationships`, is confirmed as the **one**
relationship endpoint already consumed by `DataBindingsPanel.tsx`,
`NewPageWizard.tsx`, and now `PropertiesPanel.tsx`'s Table master-detail UI.
Do not build a second one.

Its `join_condition`/`cardinality` fields were **empty** for real structural
`foreign_key`/`belongs_to` edges — the semantic `catalog_edge.properties`
graph only ever had text populated for descriptive `related_to` validation
edges, never for real FK edges. Fixed by resolving the real FK straight from
Postgres's `information_schema.table_constraints` /
`key_column_usage` / `constraint_column_usage` when the graph data is empty
(`resolveRealForeignKey`, same file). Verified live:
`GET /api/business-objects/{ExecutionBoId}/relationships` now returns
`"joinCondition":"execution.order_id = order.id","cardinality":"1:N"` where
it was previously blank.

Gotcha for the next person: `catalog_node.qualified_path` is a path-style id
(`/orm/order/avg_price`), not the dotted `schema.table` form
(`boMeta.DrivingTable`). There's a `qualifiedPathToSchemaTable` helper for
this now — reuse it rather than re-deriving the conversion (same root-cause
bug pattern bit `resolveBOMetadata` in an earlier session).

### tenant_id column assumption (backend)

`BOCRUDHandler`'s five handlers (`HandleGetBORecord`, `HandleUpdateBORecord`,
`HandleCreateBORecord`, `HandleListBORecords`, `HandleDeleteBORecord`,
`backend/internal/api/bo_crud_handler.go`) unconditionally added
`tenant_id = $N` to generated SQL for *any* Business Object table. `orm.order`
(and presumably other ORM-schema tables) has no `tenant_id` column at all —
every read/write against it 500'd with `column "tenant_id" does not exist`.
Fixed with a new `tableHasColumn(ctx, drivingTable, column)` helper
(plain `information_schema.columns` query — `BOCRUDHandler` has no
`BORepository` to reuse `PostgresBORepository.TableHasColumn`'s version of
this) and conditional predicate/column construction in all five handlers,
with a `WHERE TRUE` fallback in `HandleListBORecords` when no predicates
apply. This is a general fix, not northwind-specific — any BO on a
tenant_id-less table was broken before this.

### Form field designer (frontend, `FormFieldsDesigner.tsx`)

Rebuilt after direct "this is crap" feedback — treat this as the reference
implementation for field-level widget design UX going forward:

- **Resize**: the fix wasn't cosmetic — the drag handler computed the delta
  as *distance from drag start* every pointermove instead of incrementally,
  which silently clamps/freezes resize at boundary values (a full-width
  colSpan=12 field looked completely unresponsive to drag). Track a
  `lastDelta` closure variable and diff against it instead.
- **Reorder vs. click vs. resize**: don't spread `{...listeners}` (dnd-kit's
  drag activation handlers) across a whole interactive tile — it fights with
  click-to-select and the resize handle. Give reordering its own small
  `DragIndicatorIcon` element that alone gets `{...listeners}`/`{...attributes}`.
- **Delete semantics**: Form fields come 1:1 from the BO's real schema, so
  "delete" can't destroy the field. It sets
  `fieldOverrides[fieldName].hidden = true`; a "Deleted fields" chip row lets
  you restore it (`onUnhideField`). Don't add a real-delete path — there is
  nothing to delete.
- **Change field**: swaps which schema field occupies a grid slot, restricted
  to same-`type` fields (`sameTypeFields` filter in `PropertiesPanel.tsx`).
  Moves `fieldLayout` (order/colSpan) and `style` overrides onto the new
  field name and hides the old one.
- Shared helpers `labelSxFromStyle`, `iconForField`/`TYPE_ICON` are exported
  from `FormFieldsDesigner.tsx` and reused by the design-mode tile,
  `BOFormWidget.tsx` (preview), and `PropertiesPanel.tsx` (Change-field
  dropdown, Data Binding block) — keep it that way, don't fork icon/style
  logic per consumer.

Also fixed while in this file: `BOFormWidget.tsx` was keying form state by
the semantic field name (`AveragePrice`) but the fetched record by physical
column (`avg_price`) — every field rendered blank despite the API returning
correct data. Both fetch and submit now remap explicitly via
`field.physicalColumn`.

### Design mode shows no live data, for every widget type

`PageComponentRenderer.tsx`'s `mode: 'design' | 'preview'` prop now gates
Table (`TableDesignPlaceholder`), LineChart/KPIGroup/Slicer and their
saved-query equivalents (`ChartDesignPlaceholder`), and Form
(`FormFieldsDesigner`) — not just Form as before. Design-mode placeholders
show real field/dimension/measure *names* (so the canvas is still legible)
but never real row values. Save was explicitly re-verified after this change
(200 OK, `fieldLayout` persisted, page version incremented).

### Style-tab parity for LineChart / KPIGroup / Slicer

Table already had a "Table Style" section (banded rows, grid lines, header
colors) in the Style tab nobody else had an equivalent of. Added matching
sections, all following the same `component.props.<x>Style` +
`updateProps({...})` pattern:

- `chartStyle: { color?, showLegend? }` — LineChart. `showLegend` only shown
  when `chartType === 'pie'`.
- `kpiStyle: { valueColor?, valueFontSize?, label? }` — KPIGroup.
- `slicerStyle: { activeColor?, variant?: 'filled'|'outlined' }` — Slicer.

Threaded through `PageComponentRenderer.tsx` (`widgetStyle` computed from
`component.props` based on `widgetType`) into **both** rendering paths:
`ReportWidgetRenderer.tsx` (terms-based BO binding — new exported
`WidgetStyleOptions` type) and `SavedQueryWidget.tsx` (saved-query binding).
Both had to be updated in parallel since a chart/KPI/slicer can be bound
either way and they're two separate render implementations (ECharts option
building differs slightly between them — see `buildChartOption` vs. the
inline `option` in `ReportWidgetRenderer`).

**Not changed, because it was already fine**: the generic Style tab
(font/color/padding/border-radius) and the Layout tab (resize/width/height)
already applied uniformly to every component type via `component.style` —
those were never the inconsistency. The actual gap was purely the
type-specific visual controls Table had and nothing else did.

## What's next / not done

- Live browser verification of the Style-tab additions wasn't completed this
  session — the dev server binds to a fallback port (5177, not 3000, because
  5173-5176 were already in use) and the app requires a real Keycloak
  sign-in that wasn't driven through. `tsc --noEmit` is clean on every file
  touched; the wiring exactly mirrors the already-verified Table Style code
  path, but someone should still eyeball it once against a real chart/KPI/
  slicer widget.
- The drag-to-reorder gesture in `FormFieldsDesigner.tsx` is implemented but
  was only confirmed via scripted/automated drag, not a real coarse mouse
  drag — worth a manual pass.
- No chart-style equivalents exist yet for anything beyond
  color/legend/value styling (e.g. no axis label formatting, no per-series
  colors for multi-measure charts) — out of scope for "parity with Table",
  which itself only has banding/grid-lines/header color, not per-column
  styling.
- Pre-existing, unrelated `tsc` errors in `PageEditor.tsx` (3) and
  `PageStudioPropertiesPanel.tsx` (3) persisted untouched through this
  session — see the stale-types gotcha in `HANDOFF_BI_WORK.md` re:
  `types/pageStudio.ts`. Not fixed here; still not this session's to fix.
