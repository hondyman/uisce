# ADR 0001: Make Page Designer a world-class, configuration-only page builder

- **Status:** Proposed
- **Date:** 2026-09-28
- **Deciders:** Platform / Page Studio owners
- **Related:** `docs/page-studio-app-model.md`, `docs/core-customization.md`, the Page Designer training guide

## Context

Every MDM page (mastering console, source hierarchy, match rules, vendor
registry, staging bindings, data pipelines and the pipeline editor, schedules)
is now built in Page Designer as configuration: layout containers, widgets,
variables, queries over registered operations, actions and rule-engine
conditions. No hand-built domain components remain on those pages, and each
rebuild was proven like-for-like with a parity test.

Building them exposed where the designer is weaker than the runtime:

1. **Authoring is partly JSON.** Row buttons, action forms, map specs,
   generated columns and canvas categories are edited as raw JSON; bindings
   (`{{queries.x.data.rows}}`) are typed by hand without seeing the data's shape.
2. **Mistakes reach runtime.** A binding to a missing variable, an unknown
   operation, a dialog with no way to close, or a missing translation key is
   found only when the page is used. The most common failure while building was
   "why is this empty?" (wrong rows path or a required param never set).
3. **Every page starts from nothing.** CRUD pages repeat the same
   list + drawer + edit dialog + approvals pattern by hand.
4. **Missing blocks.** No charts or stat tiles (so no dashboards); grids have
   no server paging, sorting, column filters, export, bulk actions or inline
   editing; no first-class wizard; layouts are not responsive.
5. **Lifecycle is thin.** No visual version history, diff or rollback; core
   page publishing has no review step; tenants merging core upgrades have no
   visual compare; translations are not managed per page.
6. **Little insight.** No view of which queries run when or how often, no
   usage or error analytics, and one broken binding can blank a page.

The requirement is that pages are built "via configuration with little or no
code", no exceptions, and that the result is world class.

## Decision

Invest in Page Designer along six workstreams, in this priority order. Code
stays where it belongs, in **operations** registered by domains (data,
validation, tenant isolation, maker-checker). Pages stay configuration. We will
not add hand-built domain components to close gaps; a gap becomes a studio
building block or an operation.

### Phase 1: Authoring confidence (first)

- **Binding picker:** browse `vars`, `route`, each query's live `data` (real
  response shape from the operation's declared `fields` and a sample fetch),
  and the row / form / item scope of the setting being edited.
- **Condition builder:** field, operator, value, with paths validated against
  the same scope.
- **Page checker (lint):** runs in the designer and blocks publish on errors:
  - unresolved bindings;
  - unknown operations and components;
  - required operation params never bound;
  - overlays whose Open when variable is never cleared;
  - duplicate ids;
  - missing translation keys;
  - accessibility (unlabelled inputs, switches and icon buttons).
- **Structured editors** for everything still edited as JSON.
- **Live data in design mode:** each widget shows its query's status, row count
  and error.

### Phase 2: Speed

- **Generate from an operation:** pick a query operation (plus optional
  create/update/delete operations) and generate a searchable grid, a detail
  drawer, an edit dialog and delete confirm, ready to adjust.
- **Reusable page fragments:** make the fragment pattern first-class and
  shareable, e.g. the schedule editor (`scheduleEditor`), a record drawer, an
  approvals tab. Fragments are core objects with versions.
- **Templates gallery** (console with tabs, master-detail, approval queue,
  wizard, dashboard), seeded from the MDM pages.
- **Canvas ergonomics:** undo/redo, copy/paste, multi-select, keyboard
  shortcuts.

### Phase 3: Building blocks

- **Charts and stat tiles** bound to queries (line, bar, stacked, KPI, sparkline).
- **Grid at scale:**
  - server-side paging and sorting through the operation contract;
  - column filters, show/hide and saved views;
  - CSV/Excel export;
  - bulk select with bulk actions;
  - inline cell editing that calls an update operation.
- **Wizard container:** steps with per-step validation and back/next.
- **Responsive layouts:** per-breakpoint overrides and a working mobile preview.
- **Rich text and markdown** display, attachments viewer, record history
  (audit) widget.

### Phase 4: Lifecycle and governance

- **Version history** with a visual diff and one-click rollback.
- **Review before a core publish** (maker-checker on gold-copy pages).
- **Visual compare-to-core** for extended tenant pages: accept or reject each
  change.
- **Per-page translation view:** every key used, missing languages, in-place
  editing.

### Phase 5: Insight, performance and reliability

- **Query plan view:** when each query runs, shared caches, re-run counts,
  debounce and polling.
- **Usage and error analytics** per page, action and operation.
- **Per-widget error boundaries** so one failure never blanks a page.

## Options considered

| Option | Summary | Why not chosen |
| --- | --- | --- |
| A. Keep adding building blocks only | New widgets as pages need them | Authoring pain and runtime mistakes grow with every block; does not fix "why is this empty" |
| B. Allow domain components for hard cases | Hand-built React for complex screens | Breaks the "configuration, no exceptions" rule, the core-page upgrade model and tenant customization |
| C. Adopt a third-party low-code builder | Replace Page Studio | Loses gold-copy / tenant inheritance, the rule engine, the operation registry and maker-checker; large migration of every MDM page |
| **D. Invest in designer UX, safety and blocks (chosen)** | This ADR | Builds on a runtime already proven on every MDM page |

## Consequences

**Positive**

- Faster, safer page building by non-developers; most errors caught before
  publish.
- CRUD pages in minutes from operations; dashboards possible.
- Core pages become safe to evolve (history, review, rollback, visual upgrades).

**Negative / costs**

- Significant frontend investment, with Phase 1 and Phase 3 the largest.
- The operation contract must become richer (declared fields, paging, sorting
  and CRUD pairing), so domain teams take on work too.
- The page checker adds a publish gate that existing drafts may initially fail.

**Neutral**

- The runtime model (layout, widgets, variables, queries, actions, conditions)
  is unchanged; everything here is additive.

## Success measures

- A new CRUD page over an existing operation is built and published in under
  15 minutes by someone trained with the guide.
- Zero published pages with unresolved bindings or unknown operations; the
  checker is enforced at publish.
- No hand-built domain components on any core page.
- Dashboards exist for mastering and the scheduler, built in the designer.
- Every core page has version history; a rollback takes one action.

## Open questions

- Should fragments be versioned and inherited like core pages, or copied into
  pages?
- Where does server-side paging live: the operation contract, or a generic
  paging wrapper in the runtime?
- Does the core-page review reuse the mastering maker-checker service or the
  page studio's own?
