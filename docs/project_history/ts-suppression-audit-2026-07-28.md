# TypeScript / ESLint Suppression Audit

**Generated:** 2026-07-28
**Scope:** `frontend/src/`
**Purpose:** Catalog all `@ts-nocheck`, `@ts-ignore`, and `eslint-disable` suppressions as a prerequisite for phased cleanup.

---

## 1. Full-file `@ts-nocheck` suppressions (15 files)

All files have `// @ts-nocheck` on line 1.

| File | Lines | Notes |
|------|-------|-------|
| `src/components/EntityDrawerTreeView.tsx` | 1039 | |
| `src/components/EntityEditDetailModal.tsx` | 668 | |
| `src/pages/EntityConfigPageV2.tsx` | 602 | |
| `src/pages/EntityConfigPageV3.tsx` | 563 | |
| `src/pages/WorkflowTimeoutTriggersPage.tsx` | 511 | |
| `src/pages/timeouts/WorkflowTimeoutTriggersPage.tsx` | 511 | Duplicate of above? |
| `src/components/relationship/ReportBuilder.tsx` | 520 | |
| `src/pages/EntityConfigPage.tsx` | 428 | **Explicitly marked temporary** — "file being triaged (prevents noisy TS errors during batch edits)" |
| `src/components/bp-designer/TriggerBuilder.tsx` | 402 | |
| `src/components/AIRouting/AIRoutingDashboard.tsx` | 388 | |
| `src/components/pop/StewardGranularityReview.tsx` | 310 | |
| `src/components/pop/StewardUnionReview.tsx` | 297 | |
| `src/pages/admin/RelatedObjectsPage.tsx` | 193 | |
| `src/components/relationship/RelationshipDiscoveryModal.tsx` | 429 | |
| `src/components/abac/PolicyBuilder.tsx` | 278 | |

**Recommendation:** Do not remove any without first running `tsc --noEmit`. Only `EntityConfigPage.tsx` has an explicit "temporary" marker and is a candidate for triage after typecheck is available.

---

## 2. Inline `@ts-ignore` suppressions (12 files)

| File | Count | Classification |
|------|-------|----------------|
| `src/api/glossary.test.tsx` | 4 | **Test file** — acceptable |
| `src/components/UnifiedSemanticBuilder/SemanticModelOverview.tsx` | 3 | Intentional: complex CubeIDE-driven type cast (`as any`) |
| `src/features/views/pages/__tests__/ViewsCatalogPage.test.tsx` | 1 | **Test file** — acceptable |
| `src/features/tenants/components/TenantsTable.tsx` | 1 | Intentional: DataGridPro `grouping` prop not in community types |
| `src/components/validation/CrossEntityValidationBuilder.tsx` | 1 | Unknown — needs review |
| `src/components/editors/__tests__/JsonMonacoEditor.test.tsx` | 1 | **Test file** — acceptable |
| `src/components/RouteBlocker/BlockableLink.tsx` | 1 | Intentional: react-router `navigate` types are broad |
| `src/components/UnifiedSemanticBuilder/CodePanel.tsx` | 1 | Unknown — needs review |
| `src/rules/wasmRuntime.ts` | 1 | Acceptable: WASM runtime type gaps |
| `src/pages/marketplace/__tests__/Marketplace.test.tsx` | 1 | **Test file** — acceptable |

**Recommendation:** Non-test `@ts-ignore` entries in `CrossEntityValidationBuilder.tsx` and `CodePanel.tsx` should be reviewed. Test-file suppressions can be left as-is.

---

## 3. `eslint-disable` suppressions (34 files)

### 3a. File-level `/* eslint-disable ... */` (block, 6 files)

| File | Rules disabled |
|------|---------------|
| `src/ExplorerTab.tsx` | `no-unused-vars` |
| `src/QueryTemplateBrowser.tsx` | `no-unused-vars`, `@typescript-eslint/no-unused-vars` |
| `src/ResultsGrid.tsx` | `no-unused-vars`, `@typescript-eslint/no-unused-vars` |
| `src/QueryComposer.tsx` | `no-unused-vars`, `@typescript-eslint/no-unused-vars` |
| `src/SaveControls.tsx` | `no-unused-vars`, `@typescript-eslint/no-unused-vars` |
| `src/pages/TabbedModal/tabs/DataCatalogTree.tsx` | `@typescript-eslint/no-unused-vars` |

### 3b. Inline `// eslint-disable-next-line @typescript-eslint/no-unused-vars` (9 files)

| File | Count |
|------|-------|
| `src/SearchResultCard.tsx` | 3 |
| `src/hooks/useModelCatalog.ts` | 5 |
| `src/pages/SemanticLayoutBuilder.tsx` | 4 |
| `src/QueryTemplateBrowser.tsx` | 2 |
| `src/contexts/ImpersonationContext.tsx` | 2 |
| `src/ResultsGrid.tsx` | 2 |
| `src/QueryComposer.tsx` | 2 |
| `src/SaveControls.tsx` | 2 |
| `src/features/views/pages/ViewsCatalogPage.tsx` | 2 |

### 3c. Other inline `eslint-disable` (various rules, 24 files)

Includes `no-unused-expressions`, `react-hooks/exhaustive-deps`, import-ordering, etc.
Full list in `frontend/` via:
```bash
rg -c "eslint-disable" frontend/src/
```

---

## 4. `console.*` calls

143 calls across 49 files. `frontend/eslint.config.cjs` has `no-restricted-syntax` for `console.*` but many are exempted. Cannot classify without ESLint run.

---

## 5. Recommended next steps

1. **Install dependencies** (`npm ci`) to enable `tsc --noEmit`
2. **Remove `@ts-nocheck` from `EntityConfigPage.tsx`** — it is explicitly marked temporary and the file has been untouched for the longest
3. **Apply codemod** (`const x` → `const _x`) for inline `eslint-disable-next-line @typescript-eslint/no-unused-vars` files (see `codemod_preview.md`)
4. **Review** `CrossEntityValidationBuilder.tsx` and `CodePanel.tsx` inline `@ts-ignore` entries
5. **Defer** all full-file `@ts-nocheck` removals until per-file `tsc` output is clean

---

## 6. Rules-config gap analysis

`frontend/eslint.config.cjs` configures `@typescript-eslint/no-unused-vars` with:
```js
  '@typescript-eslint/no-unused-vars': [
    'warn',
    { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' }
  ],
```

This means variables prefixed with `_` are already allowed. Many inline `eslint-disable-next-line` comments are **redundant** if the unused variable were simply renamed to `_foo`. This is the basis for the Phase 4c-ii codemod.
