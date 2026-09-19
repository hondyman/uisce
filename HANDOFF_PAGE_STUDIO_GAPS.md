# Page Studio gaps (adjacent backlog — not part of the Report Builder spine)

**Provenance:** this content originated as a document pasted directly into
a conversation with Claude (titled "Page Studio: world-class gaps, AI
generator, MCP"), earlier in the same session that produced
`HANDOFF_REPORT_BUILDER_SPINE_PLAN.md`. It's a faithful condensed version
of that document's substance, not a verbatim transcript — if the full
original text is needed for any item below, it can be re-supplied on
request in a future session (it's not on disk anywhere; this file is now
the durable copy).

## Status of the one item that *was* fully investigated

**`filterBar` round-trip — corrected, do not use the original wording.**
The original claim was "`filterBar` is not round-tripped by
`page_studio_handler.go` (same class of bug tabs had). Slicers above tabs
will silently drop on save," listed as item #1 in the original document's
suggested sequence (ahead of page kinds, undo, everything else).

`HANDOFF_REPORT_BUILDER_SPINE_PLAN.md`'s "Order Detail crash fix" section
traced this in full:
- **Disproven for the direct-save path.** `savePage`/`updatePage` (what a
  gold-copy admin uses to edit a core page directly) round-trip `filterBar`
  cleanly — traced end to end through `pageStudioUpsertRequest`, the
  `INSERT`/`UPDATE` queries, and `getOne`'s read.
- **A real, separate bug was found and fixed**, not the one originally
  named: `normalizeUpsertDefaults` defaulted an *absent* `filterBar` to
  bare `{}` (valid JSON, missing `PageLayout`'s required `root`/`nodes`),
  which crashed `LayoutCanvas.tsx`'s renderer for any page saved with no
  filter bar at all. Fixed server-side and client-side.
- **A real gap survives, narrower than the original claim:** the
  tenant-overlay save path (`saveOverlay`/`mergeOverlay`) never includes
  `filterBar` at all, so a tenant customizing an inherited gold-copy page
  can't add or change the page-wide filter bar — only a gold-copy admin
  editing the core page directly can. Called as a real gap (the overlay
  mechanism exists for presentation customization, and a filter bar is
  presentation), scoped as its own ticket in the spine plan doc.

Treat the spine plan doc as authoritative on this item.

## Gaps (ranked, as recovered from the original document)

### Authoring ergonomics
- No undo/redo.
- No copy/paste or duplicate-widget, no keyboard move.
- No in-canvas copilot — "generate" is a one-shot dialog that navigates
  away rather than an iterative in-place assistant.
- AI-generated output is dashboard-only: it can't produce a Form, Panel,
  tabs, a filter bar, or reason about page kinds at all.
- Docs / Testing / Performance tabs in the editor are stubs.
- **Two Page Studios exist**: the live one (`pages/page-studio/*`) and an
  orphan (`components/pagestudio/*`, confirmed unmounted in an earlier
  session's research). Deletion/archival of the orphan is still open.

### Object-graph patterns
- No first-class page kinds (List / Detail / Master-detail / Dashboard).
  The new-page wizard infers a layout from the primary BO's related-object
  *count*, not from authoring *intent* — there's no way to say "I want a
  worklist" vs. "I want a detail form" directly.
- Related lists are placed manually; the object graph could offer them
  automatically (a related-BO chip suggested, not just accepted when
  dropped) but doesn't.
- No command-bar / page-actions region. Buttons today must invoke an
  existing BO/process event — this is a placement gap (nowhere natural to
  put page-level actions), not a governance gap; it should never grow into
  inventing new CRUD operations.
- No search/worklist page model as a distinct kind from List.

### Runtime fidelity
- Widget loading / empty / error / no-selection states are not
  consistently first-class across widget types. (`WidgetDesignPlaceholder.tsx`'s
  `TableDesignPlaceholder`/`ChartDesignPlaceholder` address this for
  *design mode* specifically — not confirmed to be the same gap the
  original document meant, which read as broader/runtime-facing.)
- No typed page parameters (e.g. `{{ url.id }}` bound into a page's data
  sources) — this is the gap ticket 1.2's `ParamSpec` was partly scoped
  against, but page-side adoption (as opposed to report-side) was
  explicitly deferred in that ticket.
- `filterBar` — see corrected status above.
- Responsive overlays (`ResponsiveOverride`, `ResponsiveBreakpoint` in
  `types/pageStudio.ts`) are designed in the type layer but not wired to
  an actual save/render path yet (per that file's own doc comments, seen
  in this session).
- Publish is a status chip (`draft`/`published`), not a flow — no preview
  URL, no entitlements step, no visibility into the gold-copy-vs-tenant
  delta before publishing.

### Governance
- No page version diff or rollback (versions increment on save, but
  there's no way to compare or revert).
- No unpublished-terms fence in the editor — nothing stops an author from
  binding a widget to a semantic term that hasn't been published yet.
- Nav-node entitlements are unchecked in Page Browser — a page can be
  reachable in navigation regardless of whether the viewer is actually
  entitled to it.

### AI generator contract (full spec, as recalled)
- `pageKind` enum and `layoutTemplate` enum drive generation.
- Cardinality-driven widget rules (matches this session's Report Builder
  spine work — the same "the graph decides the widget type, never hardcode
  it" principle, see `widgetTypeForCardinality` in
  `studio-core/binding/boRelationships.ts`).
- A hard cap of 0–2 related BOs per generated page.
- Presentation events generated by AI are format-only — same restriction
  Report Builder's spine plan holds elsewhere (no free-form scripts).
- Output is labeled with its `source: 'ai' | 'template'` so the UI can be
  honest about whether a real model ran or the deterministic fallback did
  (this part is confirmed real — see `GeneratedPageSpec.source` in
  `frontend/src/api/pageStudio.ts`, read during this session).
- The copilot loop is meant to be generate-then-patch against the current
  `CorePageDefinition` — never regenerate-and-wipe an author's in-progress
  draft. (`mergeGeneratedSpecIntoDraft` in `generatePageDraft.ts` exists
  and matches this intent, per this session's file listing — not
  independently verified as fully closing this gap.)

### MCP phases
- **Phase A** — read tools (list/get pages, BOs, terms).
- **Phase B** — generate tools (the AI generator contract above, exposed
  over MCP).
- **Phase C** — presentation-rule tools (author/inspect format-only
  presentation events over MCP).
- Refusal list: `save_record`, `delete_record`, `run_sql`,
  `create_business_object` — an MCP page-studio tool must never perform
  these, mirroring the Report Builder spine plan's own MCP refusal list
  (Phase 7 there).
- Resource URIs: `uisce://bo/{boKey}`, `uisce://page/{slug}`.

## What to do with this file

- This is now the durable copy — no need to re-locate a "gaps doc"
  elsewhere; it doesn't exist elsewhere.
- Items marked "not independently verified" above should be checked
  against current code before being treated as settled, same discipline
  as everything else in this repo's HANDOFF docs.
- Deliberately **not** merged into `HANDOFF_REPORT_BUILDER_SPINE_PLAN.md`
  — Page Studio's own gaps are a different backlog from the Report
  Builder spine, sharing only the `studio-core` extraction work as a
  connection point.
