# PR8 — Orphan `components/pagestudio` archive plan

**Date:** 2026-10-05  
**Branch:** `feat/cube-tables-phase0`  
**Design:** `docs/design/2026-10-05-unify-cubes-page-designer.md` (PR8 hygiene)

## Status

| Item | State |
|------|--------|
| Product ban (do not revive) | **In force** — `Agents.md` Page Designer axiom; `HANDOFF_BI_WORK.md` |
| Live Page Designer | `/page-studio` → `frontend/src/pages/page-studio/*` + `PageStudioHandler` |
| Orphan tree | **Deleted** (Track D) — was `frontend/src/components/pagestudio/` + `vitest/components/pagestudio/` |
| Coded cube shells | **Deleted** in PR8 (`CubesCatalogPage`, `CubeDesignerPage`); hosts are STUDIO_ROUTES |

## Why not delete the orphan tree in PR8

1. Vitests under `frontend/src/vitest/components/pagestudio/` import types/helpers from the orphan package (`pageStudioTypes`, canvas ops, tab migration). Deleting the tree requires rewriting or retiring those tests in the same change.
2. Some helpers (`mapFieldToControlType`, tab ops) may still be useful as reference when mining ideas — cherry-pick concepts into `pages/page-studio`, never mount the orphan UI.
3. Design explicitly staged **documentation ban now**, **archive later**.

## Archive steps (completed Track D)

1. Inventory: no runtime importers outside the orphan tree; only vitests + `ts-baseline` + docs references.
2. Retired 7 orphan vitests (30 `it`/`test`) — obsolete vs live `/page-studio`; helpers were not ported (live designer owns its own modules).
3. `git rm -r frontend/src/components/pagestudio` + `frontend/src/vitest/components/pagestudio`; stripped matching `frontend/docs/ts-baseline.txt` rows; coverageManifest baselines −7 files / −30 tests.
4. Agents.md ban remains indefinitely.
5. No helper cherry-pick — live `pages/page-studio` already is the product surface.

## Related bans (already enforced in code/docs)

- No DomainComponent id `cubes.Designer` that rehosts a full coded designer (`features/cubes/studio.tsx`).
- No dual AppRoutes + STUDIO_ROUTES registration for `/build/cubes*`.
- Analytical shells (Query Builder / Report Builder) stay coded and share `QuerySubject`.
