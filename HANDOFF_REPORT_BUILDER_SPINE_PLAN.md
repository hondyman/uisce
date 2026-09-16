# Backlog: Report Builder as a sibling of Page Studio on a shared spine

Status: **Phase 1 done. Phase 0 done except 0.3.** Phase 0 (0.1/0.2/0.4 —
repo reads and the cube-vs-BO decision) is complete and turned up a
**major correction**: the report table this whole plan should target is
`report_templates`, not `report_definitions` — see the correction section
immediately below; it replaces what used to be here. 0.3 (PDF library
spike) is still open — it needs a running spike, not just a read. Phase 1
is now **fully done**: 1.1 (spine extraction), 1.2 (`ParamSpec`), 1.3
(widget registry discriminator), 1.4 (`suppress-repeat`), and 1.5
(regression gate) all shipped and live-verified. A real, pre-existing
Page Studio crash (unrelated to this plan's work, found while re-verifying
1.3 live) was also found and fixed along the way — see "Order Detail
crash fix" below. Everything from Phase 2 on is still backlog, now
correctly re-targeted at `report_templates`.

Current sequence: 0.3 (PDF spike, the last open item before Phase 2) →
then 2.1 as a `report_templates.layout_config`-to-typed-columns migration
with a gated fallback, not a create. Everything from Phase 3 on is
unchanged from the original ordering.

North star (verbatim from the design discussion): Oracle APEX's
object-grounded report model + Salesforce report-types' relationship
fencing, with SSRS/Jasper paginated ergonomics (bands, cascading params,
grouping, export fidelity) layered on top — built as a sibling consumer of
the same BO-bound spine Page Studio already runs on, not a separate
reporting product.

## CORRECTION (Phase 0 reads): the first "correction" below was itself wrong about the table

Ticket 0.1/0.2's reads turned up something that invalidates the section
that used to be here. There are **two entirely separate, both-live backend
reporting systems** in this repo, and the original correction (first
session) identified the wrong one as what `SSRSReportBuilder.tsx` uses.
Corrected, with file-level proof:

| | `report_templates` (the real one) | `report_definitions` (a different, live-but-disconnected system) |
|---|---|---|
| Table | `report_templates` (`template_name`, `layout_config jsonb`, `parameter_schema jsonb`, `semantic_view_ids uuid[]`, `is_active`/`is_public`/`is_personal`, `version` — no `is_core` column at all) | `backend/db/migrations/20260930_000_create_report_definitions_table.up.sql` (`definition jsonb`, `parameters_schema jsonb`, `semantic_cube_id`) |
| Go model | `backend/internal/reports/model.go` — `ReportTemplate`, `LayoutConfig map[string]interface{}` (genuinely untyped — nothing gets silently dropped) | `backend/internal/reporting/model.go` — `ReportDefinition`, `Definition *ReportLayout` (a **strict, Cube.js-shaped struct**: `Layout.Body.Sections[].Elements[]`, `DataBinding{Cube,Measures,Dimensions}`, `ChartConfig`, `ConditionalStyle{Positive,Negative,Zero}` — nothing like the SSRS free-canvas `elements` array) |
| HTTP handler | `backend/internal/api/report_handlers.go` (`CreateTemplate`/`UpdateTemplate`), routes at `/api/v1/reports*`, registered in `api.go` via `reportHandler.RegisterRoutes(r)` | `backend/internal/reporting/handler.go`, routes at `/reports/definitions` etc., registered via `semanticReportingHandler.RegisterRoutes(r)` — comment literally says *"(SSRS-style reporting)"*, which is the misleading bit; it isn't what the actual SSRS-style builder calls |
| Frontend caller | `frontend/src/api/reporting.ts` (`useCreateReportTemplate`/`useUpdateReportTemplate`/`useReportTemplate`) — confirmed this is what `SSRSReportBuilder.tsx` calls via `buildSavePayload` | **No frontend caller found.** Grepped the whole frontend tree; nothing calls `/reports/definitions`. Not proven dead (a caller could exist outside the grepped tree, or be planned-but-unbuilt), but no live UI reaches it today. |

**Why this matters for `buildSavePayload`'s round-trip, checked directly:**
`CreateTemplate` (`report_handlers.go:185`) does `json.Unmarshal(bodyBytes,
&template)` into the **untyped** `ReportTemplate.LayoutConfig
map[string]interface{}` — since the frontend payload's `layout_config` key
matches the struct tag exactly, `elements`/`sectionConfig`/
`layoutSettings`/`reportTitle`/`parameters` all survive verbatim. **No
silent field-dropping on this path** — that risk is real for
`report_definitions`' strict `ReportLayout` struct, but that path isn't in
use. One real gap found in the same handler: `is_core`/`report_key` sent by
the frontend (`handleCloneReport` sets `payload.is_core = false`) have **no
corresponding field on `ReportTemplate` at all** and are silently dropped —
`SSRSReportBuilder.tsx`'s `isCoreTemplate` check
(`loadedTemplate.is_core === true || ... || tenant_id === goldCopyId`) can
in practice only ever be satisfied by the `tenant_id` fallback, because the
backend never returns `is_core`. This is a real, narrow bug, independent of
the spine plan — worth its own ticket, not fixed here.

**Rendering is a bigger gap than "typed columns."** Confirmed via
`backend/internal/temporal/activities/report_activities.go`: the scheduled/
async render path reads `template.LayoutConfig`/`template.SemanticViewIDs`
but its own comment says *"evaluation from layout_config.sections is
deferred to the rendering phase"* — grepped for any code actually reading
`layout_config["elements"]` or `["sections"]` and found **none**. The
render pipeline for what `SSRSReportBuilder.tsx` actually authors doesn't
exist yet, stub or otherwise. `report_workflows.go` and
`ReportScheduleBurstingTab.tsx` were also checked (per the independent
review's 0.2 scope) and don't read `layout_config` directly — they're
metadata/orchestration only, not part of the hidden-reader risk.

**So, corrected: ticket 2.1 targets `report_templates.layout_config`/
`parameter_schema` (JSONB → typed columns, same gated-fallback pattern),
not `report_definitions`.** `report_definitions`/`internal/reporting` is
left alone entirely — it's a separate, apparently BI/semantic-cube-oriented
system with its own live routes and no confirmed frontend caller;
folding it into this plan or deleting it is out of scope here and would
need its own investigation into who (if anyone) was meant to use it.

Also already real and reusable (confirmed in this session, not aspirational):
- `GetBusinessObjectRelationships` (`backend/internal/metadata/businessobject_service.go`)
  is the one relationship endpoint — already used by `DataBindingsPanel.tsx`,
  `LayoutCanvas.tsx` (`ensureRelatedDataSource`, `masterFilter`), and
  `PropertiesPanel.tsx`'s Table master-detail UI. Report Studio must consume
  this, not fork it.
- `ExpressionEditorField` + `parseExpressionWasm`/`evaluateExpressionTextWasm`
  (`frontend/src/rules/wasmRuntime.ts`) is the one expression engine, now
  also the engine behind Report Builder's Calculated Fields and Expression
  Library (see this session's `SSRSReportBuilder.tsx` consolidation —
  `EventScriptsEditor`'s free-JS mechanism was removed entirely).
- `PresentationEventsPanel.tsx` / `PresentationRuntime.tsx`
  (`usePresentationOverlay`) already implement the "static style base +
  runtime events overlay" pattern for Page Studio — this is the pattern
  format-only report events (suppress-repeated-value, band-break, highlight)
  should extend, not reinvent.
- `backend/internal/mcp/` (`tool_handler.go`, `tools.go`, `mcp_server.go`)
  is the one MCP server; it already references `presentation_events`. New
  report tools are added here, not in a second server.

## Phase 0 — spike / confirm before committing to the plan

- [x] **0.1** Read `backend/internal/reports/model.go` + `repository.go` +
      `backend/internal/api/report_handlers.go` in full and diagram the
      current `layout_config` JSONB shape against what `CoreReportDefinition`
      would need. **Done — with the correction above: the target model is
      `reports.ReportTemplate`/`report_templates`, not
      `reporting.ReportDefinition`/`report_definitions`.** Gap table:

      | `layout_config` today | `CoreReportDefinition` need | Migration target (2.1 column) | Written by | Read by | Gap? |
      |---|---|---|---|---|---|
      | `elements: []` (flat array, each `{id,type,section,position,size,properties}` — SSRS free-canvas shape) | `bands: []` (structured, not free-position) | New `bands` column | `buildSavePayload` | **Nobody** — confirmed no Go code reads `layout_config["elements"]` by key; `report_activities.go`'s own comment says section evaluation is "deferred to the rendering phase" | Real gap: the authored layout has no server-side reader at all today. Phase 3's band renderer isn't replacing an existing reader, it's writing the first one. |
      | `sectionConfig`, `layoutSettings`, `reportTitle` | Superseded by band/page-settings structure | Fold into `bands` + a `pageSettings`-shaped column | `buildSavePayload` | Same as above — nobody | Same as above |
      | `parameters: []` (`ParamSpec`-shaped, per 1.2) | `parameters` typed column | New `parameters` column (supersedes `parameter_schema`) | `buildSavePayload` | `ParametersDialog.tsx`/`ReportParametersToolbar.tsx` on load (round-trips through `useReportTemplate`); **not** read by `report_activities.go` (only `SemanticViewIDs` is) | Mostly clean — this one already round-trips correctly today, just needs the typed column |
      | *(no equivalent field)* | `presentation_events` (suppress-repeat, per 1.4) | New `presentation_events` column | N/A yet | N/A yet | New capability, not a migration of existing data |
      | *(no equivalent field)* | `grouping` (subtotals, band-break) | New `grouping` column | N/A yet | N/A yet | New capability |
      | `semantic_view_ids uuid[]` (real column, not in `layout_config`) | Grounding — see 0.4 | Additive `primary_business_object_id`, per 0.4 | Not populated by `SSRSReportBuilder.tsx` today (no UI sets it) | `report_activities.go` (`QuerySemanticViewsActivity`, currently a stub — warns and produces an empty data set if unpopulated) | Confirms 0.4's premise: the cube/semantic-view grounding path exists in schema but is unused/stubbed, same as the BO-grounded path will be until Phase 3 builds it |
      | `is_core` (frontend sends it, e.g. `handleCloneReport`) | Needed for the read-only-core-template UI gate | N/A — separate bug, not a 2.1 column | `buildSavePayload`/clone flow | **Nobody** — no `IsCore` field exists on `ReportTemplate` at all | Real, narrow bug independent of this plan: `SSRSReportBuilder.tsx`'s `isCoreTemplate` check can only ever be satisfied by its `tenant_id === goldCopyId` fallback in practice. Worth its own ticket. |

      Takeaway that changes the Phase 3 estimate: `report_templates` has
      **no band/section concept at all** today (a flat widget list, not
      even a real one anyone reads back) — Phase 3 is greenfield structure,
      not a re-mapping of an existing concept, and there is no legacy
      renderer behavior to preserve around it.
- [x] **0.2** Confirm whether the report save/load path has *any* consumer
      beyond `SSRSReportBuilder.tsx` that a schema change would put at risk.
      **Done, using the corrected target (`report_templates`, not
      `report_definitions`).** Checked, per the independent review's
      widened scope (GitNexus `impact()` plus grep, since untyped
      `map[string]interface{}`/`any[]` consumers don't show up in either
      cleanly):
      - `report_handlers.go` (`CreateTemplate`/`UpdateTemplate`) — the one
        real read/write path, confirmed lossless for everything
        `buildSavePayload` sends except `is_core`/`report_key` (see 0.1's
        gap table — real bug, separate ticket).
      - `report_activities.go` (Temporal) reads `template.LayoutConfig` and
        `template.SemanticViewIDs` directly — **this is the hidden reader**
        the independent review predicted, confirmed real. It's currently a
        stub for both paths (warns and produces empty output if
        `semantic_view_ids` is unpopulated; never reads `elements`/
        `sections` from `layout_config` at all) — so today it's a
        dependency with nothing to break, but 2.1's migration must keep
        `LayoutConfig`/`SemanticViewIDs` readable in whatever form this
        activity eventually gets built out against.
      - `report_workflows.go` and `ReportScheduleBurstingTab.tsx` — checked,
        or­chestration/metadata only, don't read `layout_config` directly.
        Not in the risk set.
      - `report_definitions`/`internal/reporting` — separate system, no
        frontend caller found; not a consumer of `report_templates` at all,
        so irrelevant to this risk set (see the correction above).
- [ ] **0.3** Decide the PDF rendering approach concretely: the proposal
      recommends a Go band-layout engine over `gofpdf`/`maroto` rather than
      browser print-CSS. Confirm neither library is already a dependency
      (`go.mod`) and get a one-page spike (band → PDF, no styling) working
      before committing bands-in-schema to a print contract. **Addition (per
      independent review):** the spike must also confirm maintenance status
      — `gofpdf` (jung-kurt) is archived; the ecosystem moved to forks and
      wrappers (maroto among them). Confirm: a maintained fork, Unicode font
      support (invoices will carry € and non-ASCII customer names), and
      whether the library's layout model fights band layout — maroto's grid
      model is row/column oriented, *close* to bands but not identical, and
      a thin custom band engine over a low-level PDF lib may be less work
      than bending a grid lib to fit. One page of spike answers this.
- [x] **0.4** Decide cube-vs-BO grounding and write the decision into this
      doc (per independent review — this is a real gap the original backlog
      never closed). **Decided — corrected to the real column.** The
      dual-grounding column is `report_templates.semantic_view_ids uuid[]`
      (confirmed real, part of the actual live table — not
      `report_definitions.semantic_cube_id`, which belongs to the separate,
      uncalled system per the correction above). 0.1 confirmed
      `semantic_view_ids` is schema-real but practically unused: no
      frontend UI in `SSRSReportBuilder.tsx` sets it, and its one reader
      (`report_activities.go`'s `QuerySemanticViewsActivity`) is a stub that
      warns and produces an empty result set when it's absent. That's the
      same shape of split the platform already has elsewhere
      (`ReportWidgetRenderer`'s BO-terms path vs. `SavedQueryWidget`'s
      saved-query path) — cube-vs-BO is that pattern again, not a new class
      of problem.

      **Decision: option (a).** Add nullable `primary_business_object_id`
      to `report_templates`, additive alongside `semantic_view_ids`.
      Cube/semantic-view-grounded reports (if any ever get built against
      that stubbed path) keep working unmigrated; every report the Phase 3+
      BO-grounded builder produces sets `primary_business_object_id` and
      leaves `semantic_view_ids` null. The generator (Phase 6) emits
      BO-grounded reports only. Option (b) (map semantic views to BOs, make
      BO the only ground) is the fallback if product direction later
      decides no report should ever be view-grounded — revisit then, don't
      default to it now. This decision gates 2.1: the new column is
      additive alongside `semantic_view_ids`, not a replacement for it, and
      it's the widget-capability discriminator 1.3's `acceptsRelatedBODrop`
      flag depends on (a BO-grounded report can accept a related-BO band
      drop via `ensureRelatedDataSource`; a view-grounded one has no path
      to that today and the registry entry must say so).

## Phase 1 — spine extraction (first real ticket)

Goal: Page Studio's behavior does not change; its binding/param/component/
presentation-event logic moves to a place Report Studio can import from.

- [x] **1.1** Extract the BO/relationship binding helpers Page Studio already
      has (`ensureRelatedDataSource.ts`, the relationship-fence logic in
      `LayoutCanvas.tsx`/`DataBindingsPanel.tsx`) into a studio-agnostic
      module (proposed: `frontend/src/studio-core/binding/`). Page Studio
      re-imports from the new location; **zero behavior change** is the
      acceptance bar.
      **Done.** `boRelationships.ts` and `ensureRelatedDataSource.ts` moved
      verbatim to `frontend/src/studio-core/binding/`; every importer
      (`LayoutCanvas.tsx`, `PropertiesPanel.tsx`, `DataBindingsPanel.tsx`,
      `ObjectPalette.tsx`, `generatePageDraft.ts`, the vitest test) repointed,
      old page-studio copies deleted (no shim left behind). Scoping call:
      `buildBODataSource` stayed in `generatePageDraft.ts` — its impact graph
      came back **HIGH** risk (11 impacted symbols: `NewPageWizard`,
      `PageStudioListPage.handleGenerate`, `DataBindingsPanel`, the generator
      flow) versus LOW for the two moved files, so pulling it into
      studio-core is deferred to its own ticket rather than folded into this
      one. `ensureRelatedDataSource.ts` still imports it cross-module from
      its original location — no behavior change, just a wider import path.
      Verified: GitNexus re-index shows identical caller sets pre/post move;
      `tsc --noEmit` clean on every touched file; dev server boots with zero
      console/build errors.
- [x] **1.2** Define `ParamSpec` as a shared type (proposed:
      `frontend/src/studio-core/params/ParamSpec.ts`) covering what page
      params, report parameters (`ReportParameter` in
      `SSRSReportBuilder.tsx` today), and the API/Query builder's param
      surface all need: typed, defaults, required, multi-select, and
      FK-graph-driven cascading.
      **Done, with a scope correction against the independent review's
      proposed design.** The review's draft renamed fields
      (`id`→`key`, `name`→`key`, `prompt`→`label`, `allowBlank`→`required`,
      `allowMultiple`→`multiSelect`) and added `source`
      (`static`/`url`/`ref`-with-BO-term-fencing) and `cascadeFrom`
      (parent param + `relationshipId` + traversal direction). The
      **rename** part was not applied: grepping the actual consumers found
      a **fourth** real, untyped (`parameters?: any[]`) consumer the review
      didn't know about — `FilterBuilderPanel.tsx`
      (`handleCreateParameter`, `ValueInput`, `ConditionRow`) reads/writes
      `.name`/`.prompt` directly for filter param-binding. Because that
      prop is typed `any[]`, a field rename there compiles clean and breaks
      at runtime with zero signal — the exact silent-divergence failure
      mode this whole plan exists to avoid. `frontend/src/studio-core/params/ParamSpec.ts`
      keeps the original field names and adds `source`/`cascadeFrom`/
      `presentation` as new **optional** fields — a pure additive
      extension, zero behavior change for all four consumers
      (`SSRSReportBuilder.tsx`, `ParametersDialog.tsx`,
      `ReportParametersToolbar.tsx`, `FilterBuilderPanel.tsx`), same
      BO-term-fencing and FK-graph-cascade hooks the review wanted. The
      field-name cleanup is real follow-on work — scope it as its own
      ticket that touches `FilterBuilderPanel.tsx` on purpose, not a
      silent side effect of this one. Also noted in passing:
      `FilterBuilderPanel.tsx` already has its own ad hoc `sourceType`
      field on param objects it creates — a second, independent duplication
      of what `source` is meant to unify; worth folding in when that
      cleanup ticket happens.
      `SSRSReportBuilder.tsx`/`ParametersDialog.tsx`/`ReportParametersToolbar.tsx`
      needed no further edits (already importing `ParamSpec` as
      `ReportParameter` from 1.1's session). Verified: `tsc --noEmit` clean
      on every touched file; `FilterBuilderPanel.tsx`'s pre-existing
      (unrelated) errors confirmed unchanged via `git diff --stat` showing
      zero diff on that file.
- [x] **1.3** Extend the existing component/widget registry with report-only
      band types (`Band`, `GroupHeader`, `GroupFooter`, `PageHeader`,
      `PageFooter`, `FieldText`, `Image`, `Barcode`) as new entries, not a
      parallel registry.
      **Done.** `PageComponentRenderer.tsx` turned out not to import a
      registry at all (it just imports each widget component directly and
      switches on `component.type`) — the real live registry, confirmed by
      its single caller (`PageEditor.tsx`), is the `COMPONENT_TYPES` array
      in `pages/page-studio/ComponentPalette.tsx`. Not the orphaned
      `components/pagestudio/widgetRegistry.tsx`; no stray import from that
      tree found anywhere, guard (b) confirmed clean by construction (grep,
      not re-litigated).

      Added the discriminator shape from the independent review, in place
      (no extraction — same HIGH-risk-multi-file-move precedent as 1.1's
      `buildBODataSource` decision applies here; `types/pageStudio.ts`-style
      relocation is its own future ticket once Report Studio is a real
      second consumer):
      ```ts
      interface WidgetDefinition {
        type: string; icon: React.ReactNode; group: string;
        availableIn?: ('page' | 'report')[];   // omitted = ['page']
        acceptsRelatedBODrop?: boolean;          // capability flag, not a renderer
      }
      ```
      Added the 8 band types with `availableIn: ['report']`, `Band` also
      carrying `acceptsRelatedBODrop: true` (forward-declared for Phase 5's
      related-BO-drop-onto-a-report-body work — inert today, no drop
      handler reads it yet). Added the **palette guard**:
      `PAGE_STUDIO_COMPONENT_TYPES = COMPONENT_TYPES.filter(c =>
      (c.availableIn ?? ['page']).includes('page'))`, with the group-header
      row itself also filtered so an empty "Report Bands" heading doesn't
      render in Page Studio. Net effect: the band types exist in the
      registry for Report Studio to consume later, are structurally
      invisible in Page Studio's palette today, and — since Page Studio's
      drag payload only ever originates from a palette tile — can never
      reach `LayoutCanvas.tsx`'s `onDragEnd` or `PageComponentRenderer.tsx`
      in the live app. That's what avoids the "palette tile without a
      renderer is a dead tile" trap the independent review named: there's
      no live path for a dead tile to appear on at all.

      **Guard (2), save-time validation, deliberately not done here:**
      rejecting a report-only type on a page save needs to live in
      `page_studio_handler.go` (backend), and there is no Report Studio
      canvas yet that could actually produce such a component — a stale
      hand-edited draft is the only real threat vector right now, and it's
      already blocked by the palette guard for every normal authoring path.
      Scoping this as its own ticket once Phase 3 gives band types a real
      producer, rather than adding backend validation against a threat that
      can't currently occur, matches this doc's own "round-trip test before
      the UI feature that writes it" discipline in reverse — don't write
      the guard before there's a writer to guard against.

      Verified: `tsc --noEmit` clean on `ComponentPalette.tsx`. Live
      re-verification was blocked by an unrelated crash found on the first
      attempt (see "Order Detail crash fix" below); once that was fixed,
      Order Detail's palette confirmed live-correct — no "Report Bands"
      group visible, all existing groups unchanged.
- [x] **1.4** Confirm `presentation_events` storage/round-trip
      (`page_definitions_presentation_events` migrations +
      `page_studio_handler.go`) generalizes to a format-only report event
      vocabulary (add `suppress-repeated-value`, `band-break`) without a
      schema fork.
      **`suppress-repeat` done; `band-break` deliberately deferred** (per
      independent review's split). Real shape check first: `PresentationAction`
      is not a switch-based discriminated effect union — it's a flat bag of
      optional fields (`hidden?`, `readOnly?`, `collapsed?`, `style?`,
      `label?`) that `applyActions` (`presentationEvents.ts`) merges
      explicitly by name into an `OverlayMap`. Added `suppressRepeat?: 'band' | 'page'`
      as one more optional field on `PresentationAction`
      (`types/pageStudio.ts`) and `PresentationOverlay`
      (`presentationEvents.ts`), merged in `applyActions` alongside the
      existing fields. Page Studio's only overlay reader
      (`PageComponentRenderer.tsx`, reads `.style`/`.hidden`/`.readOnly`
      explicitly) never touches `.suppressRepeat` — silence is correct by
      construction here since the merge model is flat, not a switch, so
      there's no dispatch arm to add a "don't fix this" comment to; a
      doc-comment on the new field itself explains the intentional-ignore
      instead. `band-break` needs a `groupKey` that references a grouping
      key, and grouping is a Phase 2 typed column that doesn't exist yet —
      implementing it now would mean inventing a shape Phase 2 might not
      match. Deferred: land `band-break` with Phase 2's `grouping` column +
      Phase 4's grouped band, not before.
      GitNexus flagged `PresentationAction` HIGH risk on this edit (26
      files import `types/pageStudio.ts`) — noted per policy, but the
      change is a single new optional field on an interface, which is
      structurally non-breaking for every existing consumer (nothing reads
      or destructures a field it doesn't know about); risk here is breadth
      of the module, not fragility of the change. Verified: `tsc --noEmit`
      clean on `types/pageStudio.ts` and `presentationEvents.ts`.
- [x] **1.5** Regression gate: Order Detail page in Page Studio is
      byte-for-byte unchanged in behavior after 1.1–1.4 land. This is the
      acceptance test for the whole phase — treat it as blocking.
      **Done — live-verified end to end**, logged in as
      `Northwind Traders → Uisce One → ORM Suite → CRIMS ORM Database`:
      - Order Detail editor (`/page-studio/order-detail-tj2e`) loads fully:
        primary BO (Order) fields, the related-BO fence lists Order
        Allocation / Placement / Execution correctly, Form widget renders
        with all bound fields (1.1's `ensureRelatedDataSource`/
        `boRelationships` move is transparent to the live app).
      - Order Detail's **Events** tab (`PresentationEventsPanel`) renders
        all four rule groups (Page activate / Field change / Field edit /
        Selection change) with no crash — confirms 1.4's `suppressRepeat`
        addition to `PresentationAction`/`PresentationOverlay` didn't touch
        anything the panel or `applyActions` depends on.
      - Report Builder (`/reports/builder`) loads, its Parameters dialog
        shows the existing "Year" param (`type: number`,
        `prompt: "Enter a Year"`, `defaultValue: "2026"`) correctly, and
        opening its Edit Parameter sub-dialog round-trips Name/Type/Prompt/
        Default Value/Allow Blank/Allow Multiple exactly — confirms 1.2's
        additive `ParamSpec` consolidation preserved exact prior behavior
        for `ParametersDialog.tsx`.
      - Console showed only pre-existing/unrelated errors (tenant-context
        401/500s on `glossary` endpoints, one `invalid id` 400 from a slug
        vs. UUID route mismatch — a real but out-of-scope bug, not touched
        by this phase) across every screen checked.

## Order Detail crash fix (found while re-verifying 1.3, fixed, not part of the spine work itself)

**Symptom:** Order Detail's editor threw `TypeError: Cannot read properties
of undefined (reading 'undefined')` at `LayoutCanvas`, blocking every live
verification in this doc that needs the canvas to render.

**Triage, per the process this doc now stands by:** `git diff` on
`LayoutCanvas.tsx` showed 154 lines of pre-existing uncommitted changes.
Read in full: it's real feature work — `ContainerSlot` (drop targets that
accept both palette tiles and related-BO chips even on a non-empty
container), `handleRelatedBindExisting` (`comp:` droppable handling to
rebind an existing unbound widget), already importing from
`studio-core/binding/` (1.1's new location) — this is this thread's
placement-drop work, half-landed from an earlier session, not foreign WIP.
Confirmed by content, not assumed.

**Root cause, confirmed exactly (not just hypothesized):** `renderNode`
did `layout.nodes[nodeId]` with no guard on `layout` itself. For a
`PageLayout` object missing both `root` and `nodes` (legacy/malformed
shape — `{}` shaped, not `{root, nodes}`), `renderNode(layout.root)` calls
`renderNode(undefined)`, which then does `layout.nodes[undefined]` →
`layout.nodes` is *also* undefined → throws with the property key
literally coerced to the string `"undefined"`, matching the exact error
text. Live-confirmed which layout: **`draft.filterBar`** (the page-wide
filter-bar canvas, not the active tab) — Order Detail's tab canvas
rendered its full content (FixCommand buttons, Form with all 14 fields)
correctly the whole time; only the filter-bar strip above it was affected.
`draft.filterBar || <default>` doesn't catch this because the stored value
is a truthy-but-malformed object, not `undefined`/absent.

**Fix — boundary guard in `LayoutCanvas.tsx`, not the call sites:** `renderNode`
now guards `layout?.nodes?.[nodeId]` and `components?.[nodeId]`, and the
component's own render short-circuits with a visible notice ("This tab's
layout data is malformed…") when `!layout?.root || !layout?.nodes`, instead
of crashing the whole editor. One fix in the one component both
`<LayoutCanvas>` instances (filter bar and active tab) share, so it
protects either call site regardless of which one gets bad data next.

**What's still open — deliberately not fixed here:** *how* `draft.filterBar`
got into a malformed shape isn't root-caused. Plausible per the concurrent
`types/pageStudio.ts` change seen this session ("`PageLayout` was
previously mis-declared here as a bare `LayoutNode[]`, which never matched
any real caller" — i.e., the type was JUST corrected to `{root, nodes}`):
a page saved through an older code path may predate that correction. This
needs live JSON inspection of the actual stored `filterBar` value to
confirm, which wasn't done. **Follow-up ticket, not scoped here:**
save-time validation rejecting a malformed `PageLayout` (same discipline
as 1.3's `availableIn` palette guard) so a bad shape can't get saved again,
plus root-causing how this one did. Not added speculatively — no evidence
yet the *current* save path can produce this, only that *some* past path
did.

**Verified end to end after the fix:** Order Detail loads with no crash;
BO fence intact (Order Allocation/Placement/Execution listed); Events tab
renders; **Save Changes** round-tripped correctly (version bumped v13→v14,
confirmed by a full reload after save, guard and content both held);
`tsc --noEmit` clean on `LayoutCanvas.tsx`.

**Process rule, adopted going forward (per this session's second
find-uncommitted-WIP-in-a-live-file incident):** *a session ends either
committed or stashed with a dated note* — never leaving uncommitted,
unattributed changes sitting in a live file for the next session to do
archaeology on. This doc's own sessions should follow that from here on.

## Phase 2 — report_templates typed-column migration + round-trip handler

**Corrected target (per Phase 0's reads): `report_templates`/
`reports.ReportTemplate`, not `report_definitions`/`reporting.ReportDefinition`
— see the correction section at the top of this doc.**

- [ ] **2.1** Migration: add typed columns to `report_templates` for
      `bands`, `parameters` (superseding the `layout_config`/
      `parameter_schema` JSONB blobs — decide migrate-in-place vs. new
      column + backfill), `presentation_events`, `grouping`, plus
      `primary_business_object_id` per 0.4's decision (additive alongside
      `semantic_view_ids`, not replacing it). Also fix the `is_core`
      round-trip gap 0.1 found (no column exists for it at all today —
      either add one or confirm the `tenant_id === goldCopyId` fallback is
      the intended sole signal and drop the dead frontend field). Follow the
      `page_definitions` precedent exactly (see
      `20261016_011_page_definitions_core_status.up.sql`,
      `20261016_012_page_definitions_tabs.up.sql`,
      `20261016_016_page_definitions_presentation_events.up.sql` for the
      incremental-column pattern this repo already uses). **Migration
      strategy (per independent review, informed by 0.2's hidden-reader
      check):** new typed columns + backfill, with `layout_config` kept as
      a read-fallback through Phase 2–3 and dropped only after parity is
      proven — the same gated-transition pattern the FK backfill used
      (`resolveRealForeignKey` kept until catalog edges were proven, then
      removed, per `HANDOFF_PAGE_STUDIO_WIDGET_CONSISTENCY.md`). Bake this
      into the migration now rather than deciding mid-migration. Since 0.1
      confirmed `report_activities.go` reads `LayoutConfig`/
      `SemanticViewIDs` directly, the Go struct's field names/JSON tags for
      those two must stay stable through the transition, not just the DB
      column.
- [ ] **2.2** Backend: extend `ReportTemplate` (`internal/reports/model.go`)
      and `repository.go` to read/write the new typed columns instead of
      (or alongside, during migration) the opaque `LayoutConfig`/
      `ParameterSchema` blobs. Update `report_handlers.go`'s
      `CreateTemplate`/`UpdateTemplate` accordingly — this is the handler
      0.1 confirmed is the actual live read/write path.
- [ ] **2.3** Per-column round-trip test for every new column — write a
      value, reload, assert equality — **before** any UI feature touches
      that column. This directly targets the `filterBar`-class bug called
      out in the proposal and in `HANDOFF_PAGE_STUDIO_WIDGET_CONSISTENCY.md`,
      and the same-shaped gap 0.1 confirmed already exists for `is_core`.
- [ ] **2.4** `CoreReportDefinition` frontend type (mirrors
      `CorePageDefinition` in `frontend/src/types/pageStudio.ts`) and status/
      version/draft flow parity with Page Studio's publish pipeline.

## Phase 3 — Document report kind, end to end

- [ ] **3.1** `ReportCanvas` as a band stack (not free canvas) — reuse
      dnd-kit patterns and the incremental-resize fix already proven in
      `FormFieldsDesigner` rather than re-deriving them.
- [ ] **3.2** `ReportBandDesigner` for field placement inside a band, using
      the same `component.props.*Style` props and `ReportWidgetRenderer`
      Page Studio's Table/Text widgets already use — no forked style logic.
- [ ] **3.3** Go band-layout PDF renderer (per Phase 0.3 spike) consuming
      the same `CoreReportDefinition` the React preview renders — positions
      computed once, rendered twice ("Design = Preview = Print").
- [ ] **3.4** CSV export falls out of the tabular band structure directly;
      confirm no separate export path is needed for the Document kind's
      line-item band.
- [ ] **3.5** Acceptance case: generate an invoice for Order + Allocations
      (an existing BO relationship) end to end — canvas, preview, PDF.

## Phase 4 — Operational List report kind

- [ ] **4.1** Grouped table band with subtotals (tablix analog),
      suppress-repeated-value as a presentation event.
- [ ] **4.2** Typed params (`ParamSpec` from 1.2) with cascading resolved
      through the FK graph (`GetBusinessObjectRelationships`) — verify the
      cascade matches what the page Slicer widget already resolves for the
      same relationship, per the proposal's own verification bullet.

## Phase 5 — Master-Detail report kind + related-BO drop

- [ ] **5.1** Related-BO drop onto a report body produces a detail band with
      `masterFilter`, reusing `ensureRelatedDataSource` from 1.1 — this
      should be near-free if Phase 1 did its job.

## Phase 6 — Generator/copilot extension

- [ ] **6.1** Extend the existing page-generation contract (`generate`) with
      `reportKind`, reusing the same rules (cardinality-driven widget choice,
      no invented BOs/terms, 0–2 related BOs cap) rather than a parallel
      report-generation ruleset.

## Phase 7 — MCP additions

- [ ] **7.1** Add `list_reports`, `get_report`, `generate_report_spec`,
      `create_report_draft` to `backend/internal/mcp/tools.go` /
      `tool_handler.go` — same auth/tenancy path, same mutation refusal list
      (`save_record`, `run_sql`, etc.) already enforced for page tools.

## Phase 8 — Distribution/scheduling (explicitly last)

- [ ] **8.1** Subscriptions/email delivery. Lowest differentiation value per
      the proposal; do not pull forward.

## Observability & telemetry (cross-cutting — design now, land across Phases 2–4)

**Status: design only, nothing built.** Proposed in this session, folded in
per the same pattern as the rest of this doc. Cheapest to bake into the
execution contract (`POST /api/reports/{id}/execute`, Phase 3) at design
time than retrofit after.

**Two data planes, not one.** Debezium CDC on `report_templates`/
`page_definitions`/BO bindings captures *state changes* (Plane 1) — that's
what CDC is for. It cannot capture *activity*: a report execution's phases
(param resolution, compile, execute, render), timings, SQL, row counts —
none of that lands in a table CDC can see. Plane 2 (activity events) is
emitted by the runner itself, at the execution choke point.

**One transport for both planes — transactional outbox.** The runner writes
execution events to an `ops_event_outbox` table in Postgres (fire-and-forget,
never blocks or fails an execution on emission failure); Debezium captures
that table exactly like any other and sinks it through the same Kafka→Iceberg
path already in use. No second pipeline, no direct-to-Kafka producer to
operate. Scoping flag on Plane 1: exclude scanner/catalog churn
(`catalog_edge` re-ingestion is lineage volume, not designer activity) —
CDC-capture designer-relevant metadata tables only.

**Execution event shape** (Iceberg `ops.report_executions`, partitioned by
`tenant_id, event_date`): `execution_id` (correlation id across phases),
`tenant_id`, `report_key`, `report_version` + `definition_hash` (joins
executions back to the Plane-1 metadata history — this is what turns "p95
doubled" into "because version 8 added a 1:N band," not just "it got
slower"), `bo_key`, `binding_kind` (`'bo'|'saved-query'|'cube'` — 0.4's
decision, observed live), `trigger` (`manual|scheduled|api|mcp|export`),
`actor{user_id,functional_role,session_id}`, `status`,
`phases_ms{param_resolve,compile,execute,render}`, `total_ms`,
`rows_returned`, `bytes_returned`, `page_count`, `output_format`,
`cache_hit`, `param_shape[]` (see privacy below), `sql_template`,
`sql_fingerprint`, `statement_count`, `error_class` (a taxonomy code, never
raw error text), `event_ts`.

**Three privacy mechanics, load-bearing not cosmetic** (the whole value of
this dataset is being shareable with AI/dashboards with no privacy review —
these rules are why that's true):
1. **Parameter values: shape + count + salted hash, never the value.**
   `param_shape[]` carries `{key, type, required, multi_select, value_count,
   value_hash (tenant-salted SHA — joins without disclosing),
   range_descriptor}` — never the literal value.
2. **SQL: capture the parameterized template at compile time**, before
   literal binding — not regex-scrubbed post-execution SQL (that's how PII
   leaks). If a path interpolates instead of binding, the telemetry layer
   refuses to store it (`sql_template: null, redaction_reason:
   'interpolated_literals'`) — itself a signal, not a silent gap.
3. **Redaction at emission, not at the sink.** The outbox writer is the one
   enforcement point; nothing downstream (Kafka, Iceberg, AI) can leak what
   was never written — same "fence at the boundary" discipline as the BO
   term fence elsewhere in this platform.

**Closed loop: telemetry as system BOs, not a bolt-on dashboard.** Register
the Iceberg telemetry tables as internal, platform-owned Business Objects
with real terms. Then operational-excellence dashboards are just Report
Builder reports over a system BO — same spine, same fences, no new surface;
the AI reads telemetry through the same `get_bo_terms`/
`get_bo_relationships` MCP tools it already uses; tenant isolation falls out
of the BO binding fence + RLS automatically; the separate BI module gets the
same datasets with no special plumbing. **Do not hand-build a bespoke
analytics dashboard for this** — that would be a third reporting system,
the exact anti-pattern this whole doc exists to avoid.

**What falls out with no extra instrumentation:** performance regression per
`sql_fingerprint`; zero-result parameter combos (a UX defect signal);
unused/stale reports for rationalization; timeout clusters correlated with
param shapes; schedule collisions/idle windows; failure taxonomies;
compile-vs-execute split (slow compile = metadata problem, slow execute =
data problem); version-attributed performance regressions; adoption by
trigger type.

**Sequencing (ticket stubs — detail these out when their phase starts):**
- [ ] **T.1** Define the `ops_event_outbox` write contract + the
      `ops.report_executions` Iceberg schema above. Lands *before* Phase
      3's execution contract is finalized, so `POST /api/reports/{id}/execute`
      emits `started`/`completed` events from its first version, not
      retrofitted.
- [ ] **T.2** Add the `ops_event_outbox` table + Debezium capture config —
      same migration window as Phase 2's `report_templates` typed-column
      work (2.1), since both are schema changes to the same subsystem.
- [ ] **T.3** Wire emission into the execute endpoint, landing with Phase
      3's acceptance case (the Order invoice) — that run produces the first
      real telemetry row.
- [ ] **T.4** System-BO registration for the Iceberg tables + first
      operational dashboards, after Phase 4 (they're List-kind reports —
      build them once that kind exists).
- Deferred, not scheduled: cross-tenant operator analytics UI, alerting on
  telemetry.

**Two guardrails to hold non-negotiable, same weight as this doc's other
standing guardrails:** telemetry emission must never block or fail a report
execution (the outbox write owns its own error handling, fully decoupled);
no business data in Plane 2, ever — the redaction rules above are the
entire reason this dataset can be handed to AI without a privacy review,
not a nice-to-have.

## Standing guardrails (apply to every phase)

- One relationship API (`GetBusinessObjectRelationships`) — a report-specific
  join/relationship endpoint is a redundant build, same as the platform's
  existing refusal-list precedent for a second MCP server.
- No free-form formula/script fields (the Crystal-engine trap) — this
  session already removed `EventScriptsEditor`'s free-JS mechanism from
  Report Builder for exactly this reason; presentation events' restricted
  vocabulary is the governed replacement everywhere, reports included.
- Print via server-side band renderer, never browser print-CSS, for
  deterministic pagination.
- Every new `report_templates` column ships with its round-trip test
  before the UI feature that writes it.
- GitNexus impact analysis (`impact()`/`detect_changes()`) before editing
  and before committing, per this repo's `CLAUDE.md` — this applies with
  full force to Phase 1's extraction work since it touches symbols Page
  Studio depends on today.
- **A session ends either committed or stashed with a dated note.** Two
  separate sessions working this plan found substantial uncommitted,
  unattributed WIP already sitting in live files they were about to touch
  (`LayoutCanvas.tsx` twice — see "Order Detail crash fix" above). Left
  uncommitted, half-landed changes force every subsequent session to spend
  its first stretch on archaeology (whose lines are whose, is this safe to
  build on, is it finished) before it can start its actual task — the same
  failure mode that produced the two-Page-Studios mess this doc's earlier
  research found. Before ending a session that touched a live file: either
  commit the change, or `git stash` it with a message dated and specific
  enough that the next session (or the same one, later) can tell at a
  glance what it is and whether it's safe to pop.
  **Applied to this session:** the ~154-line uncommitted `LayoutCanvas.tsx`
  diff turned out to be one piece of a much larger already-staged
  situation — the repo was mid-`git merge`
  (`fix/strict-tenant-rls-migration-port` → `main`, recovery-hardening +
  apiFetch security sweep), blocked on one conflicted file
  (`BOBindingConfigPanel.tsx`, resolved by accepting `main`'s deletion —
  that file was already confirmed dead code by `main`'s own most recent
  commit before this merge started). Resolved and landed as merge commit
  `076aa197f`, which also carries the full pre-existing Page Studio
  rewrite and this session's Phase 1 work. The lesson generalizes past
  "commit or stash": **check `git status` for merge/rebase-in-progress
  markers before assuming a large staged diff is ordinary WIP** — a stuck
  merge looks like abandoned WIP but has different, stricter git
  mechanics (no partial commits) and usually higher stakes than a feature
  branch's leftovers.

## Verification checklist (from the original proposal, kept verbatim)

- [ ] Generate an Order invoice, Order worklist, and Order master-detail pack
      from the same graph — three different specs, same BOs, no invented
      terms.
- [ ] Drop Placement onto a report body → 1:N detail band with
      `masterFilter`; Design shows field names only; Preview shows live rows
      scoped to param-selected order; PDF paginates the band correctly.
- [ ] A param cascade (order status → order) resolved through the FK graph
      matches the page Slicer's options.
- [ ] `presentation_events` rule ("highlight Cancelled rows") round-trips
      save/reload and renders identically in preview and PDF.
- [ ] MCP `get_bo_relationships` for Order returns identical output from the
      report context, the page context, and the API/query context.
- [ ] MCP refuses `save_record` from the report tool surface.
- [ ] Page Studio regression: the spine extraction changes zero behavior on
      the Order Detail page.
