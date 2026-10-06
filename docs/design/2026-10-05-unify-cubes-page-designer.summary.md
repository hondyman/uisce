# Summary: Unify Cubes with the Existing Page Designer (rev 4)

**Document:** `grok-design-doc-601d3e97.md`  
**Status:** **Approved** — implementation starts at PR0

## Recommendation

**End state — Hybrid:** One Page Designer; cube catalog + composed designer host as blueprints/`STUDIO_ROUTES` at `/build/cubes*`; small DomainComponents only (`cubes.FederationEditor` with live props including required `bos`); ban full-page designer DC.

**Phase 1 — Alternative 1 slice:** Coded admin unchanged; ship cube tiles after PR0 saved-query subject plumbing.

## Locked decisions (product + technical)

| Topic | Decision |
|-------|----------|
| Menu URL | Keep `/build/cubes`, `/new`, `/:id` via STUDIO_ROUTES |
| Create flow | `/new` (absent id) → save → `/build/cubes/{{result.id}}`; param `:id` |
| Seed | SQL migration + blueprint parity vitest |
| Publish pin | Mirror `subject` on widget props; sync `checkPage` rejects `'latest'`; runtime fail-closed if mirror ≠ saved-query subject |
| Cube saved-query identity | `source_kind='cube'`, `source_id=cubeId`, `binding_id` null; skip `BOBelongsToTenant`; subject in `query_state` + DTOs |
| Product claim | PR0–PR1b Page Designer cube-tile consume is enough; PR7 / CUBE-3.3 Report Builder later (sibling, not a gate) |

## Critical plumbing

- Today `savedQueryDef` drops Subject; create hardcodes `business_object` and requires `boId`.
- PR0 fixes identity + `Context.Subject` → `RoutePinned`.
- PR1a: mirrored pin + checker + runtime compare; PR1b always writes mirror.

## PR order

0. Saved-query subject + `source_kind='cube'`  
1a. Tile consume + mirrored pin + ZERO-PAGE  
1b. Authoring writes `savedQueryId` + `subject`  
1c. QueryBuilderModal SubjectPicker (optional)  
3. Ops + FederationEditor DC (`bos` included)  
4. Catalog blueprint + SQL seed  
5. Catalog cutover at `/build/cubes`  
6. Composed designer host; `new` + `:id`  
7. Report Builder pin + migrate  
8. Orphan pagestudio ban docs  

## Remaining open (non-blocking)

Optional QueryBuilderModal SubjectPicker (PR1c); archive timing for `components/pagestudio`.
