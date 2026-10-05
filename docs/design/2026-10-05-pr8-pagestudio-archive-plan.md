# PR8 — Orphan `components/pagestudio` archive plan

**Date:** 2026-10-05  
**Branch:** `feat/cube-tables-phase0`  
**Design:** `docs/design/2026-10-05-unify-cubes-page-designer.md` (PR8 hygiene)

## Status

| Item | State |
|------|--------|
| Product ban (do not revive) | **In force** — `Agents.md` Page Designer axiom; `HANDOFF_BI_WORK.md` |
| Live Page Designer | `/page-studio` → `frontend/src/pages/page-studio/*` + `PageStudioHandler` |
| Orphan tree | `frontend/src/components/pagestudio/` — **not mounted**; vitests only |
| Coded cube shells | **Deleted** in PR8 (`CubesCatalogPage`, `CubeDesignerPage`); hosts are STUDIO_ROUTES |

## Why not delete the orphan tree in PR8

1. Vitests under `frontend/src/vitest/components/pagestudio/` import types/helpers from the orphan package (`pageStudioTypes`, canvas ops, tab migration). Deleting the tree requires rewriting or retiring those tests in the same change.
2. Some helpers (`mapFieldToControlType`, tab ops) may still be useful as reference when mining ideas — cherry-pick concepts into `pages/page-studio`, never mount the orphan UI.
3. Design explicitly staged **documentation ban now**, **archive later**.

## Archive steps (follow-on)

1. Inventory importers:
   ```bash
   rg -n "components/pagestudio|from ['\"].*pagestudio" frontend/src
   ```
2. For each vitest: either delete (if coverage is obsolete vs live `/page-studio`) or port assertions onto live modules under `pages/page-studio`.
3. `git rm -r frontend/src/components/pagestudio` and remove `frontend/src/vitest/components/pagestudio` once empty of value.
4. Keep the Agents.md ban indefinitely so a future PR cannot re-add a parallel designer under that path.
5. Optional: move any still-valuable pure helpers into `pages/page-studio/` or `studio-core/` **before** delete — do not re-export from `components/pagestudio`.

## Related bans (already enforced in code/docs)

- No DomainComponent id `cubes.Designer` that rehosts a full coded designer (`features/cubes/studio.tsx`).
- No dual AppRoutes + STUDIO_ROUTES registration for `/build/cubes*`.
- Analytical shells (Query Builder / Report Builder) stay coded and share `QuerySubject`.
