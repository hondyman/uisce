# Phase 1 Notes — WCAG Button-Name / A11y Infrastructure

## Frozen Baseline Provenance

| Field | Value |
|--------|-------|
| Crawl date | 2026-09-05T19:02:44.090Z |
| Tool | axe-core playwright (chromium) |
| Freeze commit | b752d3649 |
| Freeze date | 2026-09-05 |
| Provenance commit | 09c30a6623 (file landed in this commit) |
| Environment | local dev (port 5173) + remote mTLS DB (100.84.50.65) + Keycloak (https://100.84.50.65:8443/realms/uisce) |

**Ratchet-computed floor (from routes array):**
- Gated violations (excl. aria-progressbar-name): **65**
- Total reproducible violations: **69**
- Total all violations: **78**
- Denominator: **127** (151 total − 15 nonRepClean − 6 nonRepWithViolations − 3 crashed)
- Routes with reproducible violations: **49**

## Freeze History

| Date | Event | Notes |
|------|-------|-------|
| 2026-09-05 | Serial 151-route crawl run | 3 crashed, 50 routes with reproducible violations |
| 2026-09-05 | Freeze committed as b752d3649 | First freeze; baseline-frozen.json not tracked in git |
| 2026-09-05 | PR #10 merged (a336bfd63) | button-name + aria-prohibited-attr fixes |
| 2026-09-05 | A11y infrastructure committed (09c30a6623) | baseline-frozen.json tracked |

## PR #10 Fixes (Landed)

| File | Violation | Fix |
|------|-----------|-----|
| SSRSReportBuilder.tsx | aria-prohibited-attr | Removed span wrapper, added aria-label to Save/Undo/Redo IconButtons |
| AbbreviationManagerV2.tsx | aria-prohibited-attr + button-name | Table + card Edit/Delete IconButtons |
| LookupsManagementTabV2.tsx | aria-prohibited-attr + button-name | Lookup/value Edit/Delete |
| Sidebar.tsx | button-name | aria-label on toolbar buttons |
| DebugPanel.tsx | button-name | aria-label on 6 toolbar buttons |
| CopilotPanel.tsx | button-name | aria-label on 2 buttons |
| CustomNode.tsx | button-name | aria-label |
| TemplateGallery.tsx | button-name | aria-label |
| CalculationsLibraryPage.tsx | button-name | aria-label on expand/collapse |
| ReportLibrary.tsx | button-name | aria-label on ToggleButton list/grid view |

## Failure Modes Caught by Measurement Infrastructure

| # | Failure Mode | Trigger | Detection | Fix |
|---|-------------|---------|-----------|-----|
| 1 | Fake JWT | E2E_JWT=garbage | Auth guard in beforeEach | Real KC token |
| 2 | Expired token | Mid-crawl expiry | KC refresh with 60s buffer | Refresh logic |
| 3 | Unmounted root | curl sees pre-hydration HTML | Playwright waitFor networkidle | Wait strategy |
| 4 | Vite error overlay | scrollable-region-focusable on .backdrop | App must build without errors | ApiError fix |
| 5 | Unsettled routes | Non-reproducible violations | 2-run dedup; nonRepWithViolations metadata | Per-run dedup |
| 6 | Error boundary | All routes crash; 0 violations | Crashed threshold + pre-flight + passes count | Backend fix |

## Post-Freeze Expected Numbers (Pre-Stated)

button-name 0, aria-prohibited-attr 0, standing ≈ 40–53:
- aria-input-field-name ~17
- label ~9
- select-name ~5
- scrollable-region-focusable ~5
- tail ~7–13

## Known Backend Issues (Blocking Valid Crawl)

1. `public.users` view maps to `app_user` — missing `role`, `organization`, `permissions`, `is_core_admin` columns
2. john.b (`113d0169-4819-42ff-968b-778f72af79e9`) has no `iam.user_roles` entries
3. `/api/access/scopes` not registered in SetupRouter
4. `/api/tenant/scope` routing: path segment parsed as tenant ID → 403
