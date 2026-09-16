# Page Studio gaps (adjacent backlog — not part of the Report Builder spine)

**Provenance, important:** this content originated as a document pasted
directly into a conversation with Claude (titled something like "Page
Studio: world-class gaps, AI generator, MCP"), earlier in the same session
that produced `HANDOFF_REPORT_BUILDER_SPINE_PLAN.md` — it was never a
repo file. By the time this file was created, that pasted message was no
longer visible in the session's active context (long-conversation context
management had moved past it), so **this file is a reconstruction from
what was recalled/quoted about it in later turns, not a verbatim port of
the original.** Item titles below are accurate; the level of detail under
each is whatever survived in conversation recall, which is thin in most
cases. If the original document (or its author) is available, this file
should be replaced with the real content, not just filled in from memory
a second time.

## Status of the one item that *was* fully investigated

**`filterBar` round-trip — corrected, do not use the original wording.**
The original claim was "`filterBar` is not round-tripped by
`page_studio_handler.go` (same class of bug tabs had). Slicers above tabs
will silently drop on save," listed as item #1 in the original document's
suggested sequence (ahead of page kinds, undo, everything else).

`HANDOFF_REPORT_BUILDER_SPINE_PLAN.md`'s "Order Detail crash fix" section
traced this in full and found a more precise picture:
- **Disproven for the direct-save path.** `savePage`/`updatePage` (what a
  gold-copy admin uses to edit a core page directly) round-trip `filterBar`
  cleanly — traced end to end through `pageStudioUpsertRequest`, the
  `INSERT`/`UPDATE` queries, and `getOne`'s read.
- **A real, separate bug was found and fixed**, not the one originally
  named: `normalizeUpsertDefaults` defaulted an *absent* `filterBar` to
  bare `{}` (valid JSON, missing `PageLayout`'s required `root`/`nodes`),
  which crashed `LayoutCanvas.tsx`'s renderer for any page saved with no
  filter bar at all. Fixed server-side and client-side; see that doc for
  detail.
- **A real gap survives, but it's narrower than the original claim**: the
  tenant-overlay save path (`saveOverlay`/`mergeOverlay`) never includes
  `filterBar` at all, so a tenant customizing an inherited gold-copy page
  can't add or change the page-wide filter bar — only a gold-copy admin
  editing the core page directly can. Called as a real gap (the overlay
  mechanism exists for presentation customization, and a filter bar is
  presentation), scoped as its own ticket in the spine plan doc, not fixed
  yet.

Treat the spine plan doc as authoritative on this one item; don't restate
the original claim as still-open.

## Still-open items (topic only — detail not retained)

These were named in the original document and, per this session's
tracing work, have **not** been investigated against the current repo
state. Titles below are as recalled; do not treat the absence of detail as
the absence of substance — it means the detail wasn't carried into this
file, not that the gap is small.

- **Undo/redo and copy-paste gap** — Page Studio's editor is missing
  these, or has an incomplete version of them. Scope and current state not
  verified.
- **Page kinds** — List / Detail / Master-detail / Dashboard as
  first-class authoring kinds (mirrors the report-kind concept in the
  spine plan's Phase 3+). Whether these exist today, partially, or not at
  all is not verified in this file.
- **Command-bar region** — a page-level command/action bar concept,
  presumably distinct from a widget's own actions. Not verified.
- **Publish-as-a-flow** — publishing a page (draft → published, or
  core → tenant-visible) as a guided flow rather than a single status
  toggle. Not verified against `page_studio_handler.go`'s actual
  status/version handling.
- **Widget loading/empty/error states** — whether every widget type has a
  defined loading, empty-data, and error state, or whether this is
  inconsistent across widget types (`TableDesignPlaceholder`/
  `ChartDesignPlaceholder` in `WidgetDesignPlaceholder.tsx` suggest this
  was at least partially addressed for design mode, per this session's
  file listing — not confirmed as the same thing the original document
  meant).
- **A comparable-products table** — the original document apparently
  compared Page Studio against other page/app builders (in the spirit of
  the OLTP-report-builder comparables table this session's Report Builder
  proposal used). Content not retained at all.
- **A suggested sequence** — the original document proposed an ordering
  for addressing these gaps, with `filterBar` (now corrected above) at
  item #1. The rest of the ordering is not retained.

## What to do with this file

- If the original document can be found or re-supplied, replace this
  file's content wholesale rather than merging — a reconstruction from
  recall is not a reliable base to build on top of.
- If it can't be recovered, each topic above needs to be re-scoped from
  scratch (read the current Page Studio code, form a real gap assessment)
  before it's actionable as a ticket — this file is a table of contents
  for that work, not the work itself.
- This file is deliberately **not** merged into
  `HANDOFF_REPORT_BUILDER_SPINE_PLAN.md` — Page Studio's own gaps are a
  different backlog from the Report Builder spine, sharing only the
  `studio-core` extraction work as a connection point.
