# Responsive breakpoint designer — interaction design

Design only. No implementation in this pass — see "Wire-later" note in
`../../types/pageStudio.ts`'s `LayoutNode.responsive` doc comment for why
(save-path validation is blocked on the BO-service triage decision;
see `backend/internal/pagebuilder/BO_SERVICES.md`).

**RECOVERY NOTE (2026-09-06):** this file was deleted, pre-commit, by a
concurrent session's git-clean-class operation on a shared working tree.
Reconstructed from history — this is a design document with no database
counterpart to verify against, so unlike the SQL/Go artifacts recovered
alongside it, this one is trusted on the strength of the conversation
record alone. Re-read it with that in mind.

**Target module:** `types/pageStudio.ts` / `LayoutNode`, consumed by this
directory (`PageEditor.tsx`, `LayoutCanvas.tsx`, `PageComponentRenderer.tsx`,
`DataBindingsPanel.tsx`, `PageStudioPage.tsx`). NOT `components/pagestudio/*`
— that directory runs a separate, already-developed type system
(`pageStudioTypes.ts`'s `CanvasWidget` family) with its own `xs/md/lg`
grid-span logic already built into `PageStudioCanvas.tsx`. The two systems
are not reconciled; this design does not attempt to reconcile them. Flagged
as an open item for whenever page-builder consolidation is scoped — same
duplication pattern the backend investigation found repeatedly (five
same-named `CreateBusinessObject` methods), now visible in the frontend.

## The three states (not two)

Every node, at every breakpoint, is in exactly one of:

1. **Inherited** — no `responsive[breakpoint]` entry. Renders using the base
   `LayoutNode`'s own props/style/children, unchanged.
2. **Overridden** — a `responsive[breakpoint]` entry exists. Renders per
   that overlay (whole-overlay-replace: every property in `ResponsiveOverride`
   comes from this object, falling back to base only for fields the overlay
   itself omits — never from a sibling breakpoint).
3. **Unrenderable** — the node's widget resolves to `not_supported` for this
   breakpoint via `bo_widget_breakpoint_fallback` (a fallback row exists with
   a null `fallback_widget_key`), and the node is not hidden (`hidden: true`)
   at this breakpoint. This is the wire-later validation rule surfaced as an
   authoring-time state, not just a save-time error. It is the load-bearing
   state: an author needs to see and act on it — pick a fallback-eligible
   widget, or hide the node — before it ever reaches a save attempt.

   **PREVIEW-ONLY BOUNDARY, required, not optional:** until the save path
   exists, this state can only be computed client-side against a snapshot of
   `bo_widget_policy`/`bo_widget_breakpoint_fallback` data — it has never been
   server-validated. The badge, tooltip, and page-level summary all MUST
   carry a visible "preview only — validation pending save path" marker
   (copy TBD, but the distinction is not optional) for as long as no real
   save endpoint exists. Dropping this marker once wiring lands is fine;
   shipping the badge without it before wiring lands recreates exactly the
   author-trust ambiguity the wire-later marker in the schema exists to
   prevent — a badge with no truth-status is worse than no badge.

Designing only inherited/overridden and treating unrenderable as "the error
case save will report" is the wrong shape. It must have a visible slot in
the UI at design time, identical in prominence to the other two, even though
the check that produces it has no live endpoint yet (client-side stub against
the known `bo_widget_policy`/`bo_widget_breakpoint_fallback` data is fine to
preview the state now; the real check moves server-side at wiring time).

### Visual treatment (indicative, not final pixel spec)
- Inherited: no badge, node renders normally in the frame.
- Overridden: a small filled dot / breakpoint-icon badge on the node, with
  a tooltip naming which properties differ from base.
- Unrenderable: a warning-colored badge, node renders with a placeholder
  ("no mobile equivalent") rather than attempting the real widget, with an
  inline action to either hide the node at this breakpoint or open the
  widget picker filtered to fallback-eligible choices.

## Materialize-on-touch, dissolve-on-clear

This is the direct UX consequence of whole-overlay-replace (decision 2 from
the schema) and is the most likely thing to be built wrong by reflex.

- **Materialize:** while the canvas is in mobile or tablet frame, editing
  *any* property of a node in the properties panel creates the
  `responsive[breakpoint]` overlay object for that node if it doesn't exist
  yet, seeded with the base node's current values, then applies the edit on
  top. The overlay comes into existence on first touch, not on frame-switch —
  merely *viewing* a node in mobile frame must not create an overlay.
- **Dissolve:** the properties panel needs an explicit "remove override"
  action per breakpoint (not a reset-values-to-match-base action). Resetting
  values back to match base while leaving the overlay object in place
  produces a ghost overlay — indistinguishable from "inherited" at render
  time today, but a landmine the moment the base value changes later (the
  ghost overlay silently stops tracking base and the node diverges with no
  visible cause). Deleting the key is the only correct "back to inherited."
- The properties panel must show, unambiguously, which of the two states
  ("editing base" vs. "editing the mobile overlay") is currently active —
  this is the same information the per-node badge shows, surfaced again at
  the point of edit rather than only at the point of viewing.

## Page-level summary (batch form of the same three states)

One summary line per breakpoint, e.g.:

> Mobile: 12 nodes render as base · 3 overridden · 1 hidden by override · 2 unrenderable

Same three-state model, aggregated. This is what an author checks before
calling a mobile variant done — not a separate feature from the per-node
badge, the same computation rolled up. Clicking "2 unrenderable" filters/
scrolls the canvas to those nodes (implementation detail, not required for
the design pass, but the summary should be built expecting that affordance
so it isn't retrofitted later).

## Structural actions disabled in breakpoint frames

Per the schema constraint (`responsive` cannot add, remove, or re-parent
nodes): when the canvas is in mobile or tablet frame, "add node," "delete
node," and drag-to-reparent must be **visibly disabled** (greyed out /
tooltip explaining why), not merely unimplemented or silently no-op. An
author should never be able to attempt a structural edit in breakpoint view
and wonder why nothing happened. Only in desktop/base frame are structural
actions live; visibility/order/span/stack-direction edits are live in every
frame, base included (editing base in mobile frame view is not itself
disallowed by the schema — only structural changes are — but the properties
panel should make clear which layer, base or overlay, a non-structural edit
in breakpoint frame is landing in, per the materialize-on-touch rule above).

## Explicitly out of scope for this designer

- Entitlement/CRUD-capability gap surfacing while authoring (drift-aware
  authoring, prioritized highest for a *future* round but blocked on the
  pending `term_node_id` migration — see `BO_SERVICES.md`)
- Any AI-assisted layout suggestion
- Actual save-path wiring (see wire-later note)
- Reconciling this module with `components/pagestudio/*`
