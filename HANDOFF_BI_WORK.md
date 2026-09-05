# Handoff: BI capability build-out (Report/Query/Page builders)

Committed as `3465339` on `main`: "feat: BO relationship graph, multi-BO
joins, and BI widgets for Report/Query/Page builders". Read that commit's
full message for the file-level list — this doc is the *why* and *what's
next*, not a duplicate of the diff.

## Context: how this started

The user asked to make Business Objects (BOs) and their related BOs usable
across Report Builder, Query Builder, and Page Studio, "optimal and
performant." That escalated over the session into: real cross-BO SQL joins,
cardinality-aware rendering (PeopleSoft-style child "scroll levels"),
semantic-layer-driven drill-down, and BI widget parity with cross-filtering
(Charles River Workbench-style).

**Read this before touching anything else in this codebase**: this session
repeatedly discovered *parallel, overlapping, or entirely orphaned*
implementations of the same feature — sometimes 3-5 deep. Concretely found:

- **Five separate BO-relationship subsystems** before landing on the real
  one. Only `catalog_edge` with `BO_RELATES_TO_BO`/`TABLE_RELATES_TO_TABLE`
  edge types (`internal/analytics/relationship_inference_service.go`,
  `join_path_resolver.go`, `internal/api/relationship_handler.go`) is
  actually live. Do not build on `bo_relationships` (table never created,
  `JoinInference`/`BusinessObjectCachedRepository` in `boresolver` are
  unused dead code), `bo_relationship` (instance-level, different concern,
  used by `internal/ai/graph_rag.go`), or `business_object_relationships`
  (used by a separate `relationship_suggestions` admin workflow in
  `internal/api/relationships_chi.go` — real but unrelated, don't conflate).
- **Two Page Studio implementations.** The live one is `/page-studio` →
  `PageStudioPage.tsx` → `PageEditor.tsx` → `LayoutCanvas.tsx`, using
  `CorePageDefinition`/`ComponentDefinition`. There is a second, newer-
  looking, more complete-seeming one (`PageStudioCanvas.tsx`,
  `pageStudioTypes.ts`, `widgetRegistry.tsx`, `PageStudioPropertiesPanel.tsx`,
  tabs, subtypes, `relatedObject` widgets) that is **not mounted anywhere**
  — only referenced from its own test file. Verify with `grep -rn
  "<PageStudioCanvas"` before assuming it's real.
- **A third, abandoned "Page Designer"** exists only on two stale worktree
  branches (`claude/wonderful-hawking-0ae04c`,
  `claude/relaxed-antonelli-f74b57`), 31 commits behind `main`. Do not merge
  it in — it predates the RuleFabric/CEL unification and produces real
  conflicts in auth/RBAC code. If mining it for ideas, cherry-pick
  concepts, not the branch.
- **`types/pageStudio.ts` is stale relative to runtime.** `layout` is typed
  `LayoutNode[]` but every runtime call site (`LayoutCanvas.tsx`,
  `PageStudioPage.tsx`) uses `{root: string, nodes: Record<string,
  LayoutNode>}`. Same for `components` (typed as an array, used as a dict).
  This produces a long-standing, pre-existing set of `tsc` errors in that
  file — not something introduced this session, and not fixed here (too
  large a blast radius to rush). A future session should either fix the
  type declarations to match reality, or do the reverse.
- **`ABACEngine.Evaluate`** (`internal/api/trigger_engine.go:651`) is a
  stub that unconditionally returns `true`. There is **no real BO/field-
  level entitlement engine** in this codebase — only tenant isolation (RLS,
  real and enforced) exists. Every new endpoint this session added inherits
  that gap rather than fixing it; be honest about this with the user if it
  comes up again, don't imply enforcement that isn't there.

## What's real and working now

**Backend** (`backend/internal/analytics`, `internal/boresolver`,
`internal/querybuilder`, `internal/api`):

- `RelationshipInferenceService` / `JoinPathResolver` now cache the
  relationship graph (previously hit Postgres on every call) and actually
  **persist** inferred `BO_RELATES_TO_BO` edges (previously computed them
  and threw them away — a real bug, fixed).
- Subtype-scoped relationship inheritance:
  `InheritBORelationshipsFromSubtypeTable`, `SourceSubtypeID`/
  `TargetSubtypeID` on `BORelationshipEdge`.
- `JoinPath.TraversalCardinality()` — "one" vs "many" classification from
  the actual resolved join path (1:M/M:M steps anywhere in the path = "many").
  This is the single source of truth for cardinality; any client-side
  cardinality hint is cosmetic only.
- New multi-BO SQL generator: `internal/querybuilder/multi_bo_generator.go`
  (`buildMultiBOSQL`). Takes a `QueryDef` with `QueryContext.RelatedBOIDs`
  and per-field `boId`, resolves each related BO's join path server-side,
  validates every identifier via `identRe` before interpolating, and
  reuses `boresolver.CompileFilterPredicate` for filter values (never raw
  string interpolation). Wired into `QueryService.Preview` via
  `previewMultiBO`. Has real tests
  (`multi_bo_generator_test.go`) covering join shape, tenant scoping, and
  rejection of unsafe identifiers/operators.
- **Fixed a real pre-existing SQL injection**: `CompileFilterPredicate`
  (`internal/boresolver/expression_builder.go`) interpolated the caller-
  supplied filter `operator` string into SQL with no allowlist. Added
  `allowedComparisonOps` and a guard before the value-type switch. This
  function is used by the *older* single-table query path too, so the fix
  benefits both.
- Drill-down, semantic-layer-first (per explicit user direction — do not
  regress to per-widget drill config):
  - `SemanticTermView.DrillPath []string` — term node IDs for successive
    drill levels, read from `catalog_node.properties.drill_path` via a
    LEFT JOIN in `PostgresBORepository.GetBOTerms`
    (`internal/boresolver/bo_repository.go`).
  - `PUT /api/semantic-terms/{id}/drill-path` — configure it once
    (`internal/api/semantic_terms_handler.go`).
  - `RelationshipInferenceService.ResolveDrillTarget` — at click time,
    checks whether a drill-path term is bound to a field on the current BO
    or a related one (never mutates the BO's field list to make it so).
    Exposed as `GET /api/relationships/bo/{boId}/drill-target/{termId}`.
  - **Known debt**: `ResolveDrillTarget` does raw SQL against
    `bo_fields`/`business_objects` rather than going through
    `BOResolver.GetBODefinition`. It only touches BO *metadata* tables (not
    a physical driving table), so it doesn't violate "everything through
    the BO abstraction" as badly as it could, but it's not fully unified
    with the resolver interfaces either. Flagged, not fixed.
- New relationship endpoints: `GET /api/relationships/bo/{boId}/join-path/
  {toBoId}` (cardinality-annotated join path between two BOs), the
  drill-target one above.

**Frontend**:

- `ReportWidgetRenderer.tsx` (`frontend/src/components/reporting/`) — real
  table/chart(bar/line/pie)/gauge/matrix(client-side pivot)/slicer
  rendering for Report Builder canvas elements, replacing a stub where
  every one of 12 declared widget types rendered as a plain text label.
  Runs actual `QueryDef`s through `/api/query/execute` — no client-side SQL.
  Handles drill-down (calls the drill-target endpoint, falls back to
  cross-filter if not drillable) with a breadcrumb trail
  (`DrillStep[]` state model — read the comments in the file, the first
  draft of this had a genuinely buggy double-`setState` drill-stack model
  that got caught and rewritten mid-session; the shipped version is
  correct).
- `useCrossFilterStore.ts` (zustand) — shared cross-filter bus, keyed by
  `${boId}:${termNodeId}` so filters never leak across unrelated fields
  with the same name.
- `BOFormWidget.tsx` — real BO-bound form: fetches schema via
  `fetchBOSchema`, POSTs/PUTs to `/api/v1/bo/{boKey}/records`. **Caught and
  fixed a fabricated-validation bug during review**: first draft used
  `field.override` as a fake "required" proxy; `BOSchemaField` has no
  required-field metadata at all, so validation is honestly a no-op for now
  (see comment in the file) rather than pretending to validate.
  Add real required-field validation once `BOSchema` exposes it.
- Query Builder (`BusinessObjectQueryBuilder.tsx`): related-BO field
  browsing (click a related BO chip to join it into the query and browse
  its fields, tagged with `boId`), multi-BO execution via the backend
  above, `ScrollAreaResultView.tsx` for PeopleSoft-style nested "many"
  results instead of silent row fan-out.
- Page Studio: `PageComponentRenderer.tsx` wired into the **real**
  `LayoutCanvas.tsx` (see the two-Page-Studio warning above). Table/
  LineChart/KPIGroup/Form component types render against real data,
  resolved from the page's `dataSources` (added via `DataBindingsPanel.tsx`,
  which also lets a page bind to a BO's related objects). Falls back to
  the page's first bound BO when a component hasn't been pointed at a
  specific data source — there's no per-widget data-source picker UI yet.
- `SemanticTermView.drillPath` and `QueryResultColumn.{boId,cardinality}`
  added to `frontend/src/features/query-builder/types/queryDef.ts` to match
  the backend contract.

## Explicitly deferred (not started)

1. **Map widget** — needs a new mapping dependency (e.g. `react-leaflet`).
   Get user sign-off before adding a new package.
2. **Properties-panel UI** for hand-picking dimensions/measures per widget,
   in both Report Builder and Page Studio. Both currently auto-default to
   the first available dimension/measure on widget creation — real and
   functional, but not adjustable without deleting and re-adding the
   widget.
3. **Query Builder → cross-filter store**: Query Builder's result view
   doesn't publish to `useCrossFilterStore` yet (Report Builder and Page
   Studio widgets do).
4. **Field/BO-level entitlements**: `ABACEngine` is a stub (see above). Any
   drill-down/cross-BO work should eventually gate on real entitlements,
   not just tenant isolation.
5. **`types/pageStudio.ts` realignment** with actual runtime shapes
   (`layout`/`components` array-vs-dict mismatch) — pre-existing, not
   touched, produces a stable set of `tsc` errors that show up in any
   diff-based error-count check (expected, not a regression).

## Verification done this session

- `go build ./...` clean.
- `go test ./internal/querybuilder/...` passes, including new
  `multi_bo_generator_test.go`.
- Two pre-existing, unrelated `boresolver` test failures confirmed via
  `git stash` to predate this session's changes (stale assertions expecting
  unparameterized SQL literals — not something to fix as part of this
  work unless asked).
- `npx tsc --noEmit -p tsconfig.json`, diffed file-by-file against a
  pre-change baseline before every batch of frontend edits — no new
  files with errors introduced. The codebase's `tsc` baseline has ~750+
  pre-existing errors (loose tsconfig, not CI-gated); this is normal for
  this repo, not a sign of a broken build.
- Not done: no live/manual UI verification against a running dev server
  (no DB available in this environment). **The user is testing this now** —
  next session should ask what broke before re-deriving anything.

## If starting a new session on this

1. Ask the user what they found when testing before assuming anything is
   broken or done.
2. Before extending any BO-relationship, semantic-term, or Page Studio
   code, grep for existing implementations first — this session's biggest
   time sink, repeatedly, was building on/investigating orphaned parallel
   subsystems before finding the live one. The pattern that worked: check
   what's actually `import`ed/mounted from a real route before trusting
   that a promising-looking file is live.
3. The user's standing architectural rule, stated explicitly mid-session:
   **everything goes through the Business Object abstraction layer, no
   exceptions.** Don't add new raw-SQL-against-physical-tables shortcuts;
   route through `BOResolver`/`BODefinition` or the `bo_fields`/
   `business_objects` metadata layer, not driving tables directly.
