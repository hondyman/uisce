# PR7 / CUBE-3.3 — Legacy cube name scan notes

**Date:** 2026-10-05  
**Branch:** `feat/cube-tables-phase0`  
**Util:** `frontend/src/features/analytical-subject/legacyCubeMigrate.ts`

## Contract

- Consumers pin `cubeId` + `contractVersion` on `QuerySubject` / `DataBinding.subject`.
- Legacy `dataBindings.primary.cube` **name** strings migrate once via exact `cube_definition.name` match.
- Unresolved / ambiguous / blank names **fail closed** (no runtime name-only resolution, no long-term dual-write).

## Alpha scan (2026-10-05)

| Surface | Legacy `cube` name | Subject pin | Notes |
|---------|-------------------|-------------|-------|
| `report_definitions` | **7** | **0** | All seven seeded core/custom/shared reports |
| `page_definitions` | **0** | — | Page Studio migrate is a no-op (as designed) |
| `report_templates.layout_config` | **0** | — | SSRS seed layout is element-based, not Workday `dataBindings` |

### Legacy names present

| `report_key` | `dataBindings.primary.cube` |
|--------------|-----------------------------|
| rep-core-001 | `oms.account` |
| rep-core-002 | `altinv.alternative_investment` |
| rep-core-003 | `oms.position` |
| rep-core-004 | `oms.position` |
| rep-core-005 | `oms.account` |
| rep-custom-001 | `oms.account` |
| rep-shared-002 | `cash_flow.settlement` |

### `cube_definition` name coverage at scan time

Active cubes on alpha were smoke/designer rows (`Browser CUBE-0.4 Account Cube`, `CUBE-0.4 Account Designer Smoke`, `CUBE-2.5 Position×Account×Security Smoke`) — **none** named `oms.account` / `oms.position` / etc.

**Consequence:** migration `20261222_001_report_definition_subject_pin` rewrites **0** rows until cubes are published under those legacy names, or definitions are hand-rewritten with explicit `subject` pins. Runtime load in Report Builder surfaces a fail-closed warning and requires picking a subject via `SubjectPicker`.

## Authoring path

Live Report Builder (`SSRSReportBuilder` via `/reports/builder`, `/reports/:reportId/edit`) now:

1. Restores `metadata.subject` / layout `subject` when present.
2. Else runs `resolveReportCubeBinding` against `listCubes` for legacy `dataBindings.primary`.
3. Persists the pin on save via `buildSavePayload(..., { subject })` (no new legacy `cube` name writes).
