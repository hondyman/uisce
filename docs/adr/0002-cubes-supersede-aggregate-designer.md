# ADR 0002: Cubes supersede the dual-mode Aggregate Designer demo

- **Status:** Accepted (CUBE-0.5)
- **Date:** 2026-10-05
- **Deciders:** Platform / cube tables program
- **Related:** ADR-011 (cubes reference metrics by ID only), `data_explorer.cube_definition`, Build → Cubes (`/build/cubes`)

## Context

`frontend/src/features/analytics/pages/AggregateDesignerPage.tsx` was a dual-mode
(StarRocks / Cube / Both) prototype that posted ad-hoc dimensions and measures to
`/api/analytics/aggregates` (and previewed via `/api/analytics/preview`). It was
never mounted in `AppRoutes` or `MainNavigation`, and those endpoints were never
a live contract on the unified API.

Phase 0 of the cube tables program delivers first-class aggregation contracts:

- logical definition in `data_explorer.cube_definition` (dimensions as term
  nodes, `metric_ids` only, grains, materialization, empty federation in v1)
- CRUD / validate / versions under `/api/cubes`
- authoring UI under Build → Cubes (`/build/cubes`, Designer at `/build/cubes/new`)

## Decision

1. Delete the Aggregate Designer page. Cubes are the only discoverable authoring
   path for aggregation contracts.
2. Serve **410 Gone** for `/api/analytics/aggregates` and `/api/analytics/preview`
   with a `Link` successor to `/api/cubes` and pointer to this ADR.
3. Keep Fabric → Preaggregations as the ops surface for materialization
   lifecycle; it is not replaced by this retirement.

## Consequences

- Callers of the demo APIs get an explicit Gone response instead of 404.
- New work authors cubes through `/api/cubes` and the Cubes Designer.
- No migration of Aggregate Designer artifacts: none were persisted through a
  governed registry.
