# Query results centralization — phase handoff

Single-tenant, multi-consumer results surface for Query Builder, Live Query,
and Report Builder. The centralization is structural — not a copy/paste
exercise — so this doc records the **settled decisions**, **phase state**,
and the **Phase 3 completion criterion** that future sessions need.

> Read this before relitigating any choice in this workstream. If you think
> a settled decision is wrong, that's a reason to update this doc, not to
> silently revert it.

---

## The contract — three layers, no fabrication

```
┌──────────────────────────────────────────────────────────────────────┐
│  Layer 3 — PRESENTATION (one component)                               │
│  QueryResultsPanel: tabs, grid, SQL view, charts, extraTabs slot      │
│  Knows nothing about saved queries, BOs, or ABAC.                     │
│  Input contract: QueryResultSet (features/query-execution/types.ts).  │
├──────────────────────────────────────────────────────────────────────┤
│  Layer 2 — DATA CONTRACT (one canonical shape)                        │
│  QueryResultSet { columns, rows, meta } — one type, many adapters.    │
├──────────────────────────────────────────────────────────────────────┤
│  Layer 1 — DATA ACQUISITION (one hook, one API module)                │
│  features/query-execution/                                            │
│    queryExecutionApi.ts — fetchCompiledSql(), runExecute()           │
│    useQueryExecution.ts — owns run / lazy-compile / error normalize   │
│    adapters.ts — liveQueryResultToSet, savedQueryResultToSet          │
│    errorMessage.ts — friendlyQueryError (lifted from SavedQueryEditor)│
│  ZERO client-side SQL generation, ever.                              │
└──────────────────────────────────────────────────────────────────────┘
```

Every backend response funnels through an adapter that produces a
`QueryResultSet`. A new consumer (Report Builder, SSRS, future EXPLAIN
endpoints) adds a ~15-line adapter — not a fork of the panel.

---

## Settled decisions

| # | Decision | Why | Where it lives |
|---|---|---|---|
| 1 | `useQueryExecution` owns run state, lazy-compile, error normalization | One hook, not "each consumer owns its own state shape" | `features/query-execution/useQueryExecution.ts` |
| 2 | `friendlyQueryError` lifted to `features/query-execution/errorMessage.ts` | Used by hook + SavedQueryEditor + future consumers; one definition | `errorMessage.ts` |
| 3 | Two-layer ESLint guardrail | Layer 1 = naming convention ban (hard error); Layer 2 = AST structural detection (warn) | `eslint.config.cjs`, `eslint-rules/no-sql-fabrication.cjs` |
| 4 | `QueryResultsPanel` accepts `extraTabs` registry, not a single `executionPlan` slot | Real ABAC/Sentinel diagnostics, when they exist, plug in without touching the panel | `components/shared/QueryResultsPanel.tsx` |
| 5 | `initialTabId` is uncontrolled view state (defaults to `resultSet ? 'results' : 'sql'`) | Simpler consumer API; future state-restoration should add a controlled `activeTabId` prop with explicit unknown-id semantics | `QueryResultsPanel.tsx` |
| 6 | **No client-side SQL generation in the Query Builder / Reporting / Live Query migration path.** LiveQueryTab's `generatePostgresSQL` removed in `#112` / `1ef260574` (Layer 1 violation in the guardrail's own tree — first demanded deletion; see "Known violations ledger"). FilterBuilderPanel's `buildSQL`/`buildGroupSQL` removed in Phase 3. Mock-row fallbacks in any `handleRun*` removed in the same commit. Other workstreams (Data Explorer's `generateDialectSQL`, CEP's `generatePreviewSQL`) have open TBD owners and aren't gated by this workstream's guardrail — see ledger. | The whole point of this workstream (scoped) | enforced by guardrail Layer 1 + standing pre-flight |

---

## Guardrail known limitations

### Layer 2 does not see fragment assembly

`buildGroupSQL` in `FilterBuilderPanel.tsx` builds SQL fragments — e.g.
`` `${fieldRef} = ${paramRef}` ``, `` `IN (${vals})` ``, `` `WHERE ${parts.join(...)}` `` — and `buildSQL` assembles them. These are single-hole templates with **clause-keyword static text only** (no `SELECT…FROM` shape).

Layer 2's pattern is by design:
- `TemplateLiteral` requires `≥2` interpolation holes AND SQL-shaped text
- `BinaryExpression` (`+` chain) requires at least one SQL-shaped literal AND one non-literal operand

Both miss the fragment pattern. This is **intentional** — the same relaxed
heuristic would catch UI copy like `IN ${stockSymbol}` in a tooltip.

### Option 1 endorsed: accept and re-derive the Phase 3 work list

The original premise — *"Phase 3 done when the rule passes file-wide"* — was
**vacuously true** because Layer 2 doesn't see fragments.

**Option 2** (extend Layer 2 with a fragment pattern: `≥1` hole + clause-keyword
static text, uppercase-only) was considered and **deferred**. Reasons:
- Uppercase-only clause keywords catch WHERE/HAVING/QUALIFY/BETWEEN/IN/GROUP BY
  — but they also catch UI copy like `IN ${stockSymbol}` and `WHERE to find
  help` if the hole contains text the keyword expects.
- The Phase 3 work list is **already enumerable** from the operator switch in
  `FilterBuilderPanel.tsx`. Adding false-positive surface now doesn't earn
  its keep; revisit if Phase 3 turns up a case the switch didn't catch.

### Re-derived Phase 3 completion criterion

Phase 3 is done when **all** of these hold:

1. `buildSQL` and `buildGroupSQL` deleted from
   `frontend/src/components/reporting/FilterBuilderPanel.tsx` (no rename,
   no carve-out — replaced by `previewQuery` calls through the
   `useQueryExecution` hook).
2. Filter state routes through `previewQuery` (or `useQueryExecution`) with
   full operator coverage: `equals`, `not_equals`, `greater_than`,
   `less_than`, `greater_equal`, `less_equal`, `in`, `not_in`, `contains`,
   `starts_with`, `ends_with`, `between`, `not_between`, `is_null`,
   `is_not_null`, `regexp_like`, `array_contains`, `json_extract`,
   function-wrapped fields (`fieldExpr`).
3. Completion check passes — zero clause keywords in the directory:

   ```bash
   grep -rnE '\b(WHERE|HAVING|QUALIFY|BITEMPORAL)\b' \
     frontend/src/components/reporting/ \
     --include='*.ts' --include='*.tsx'
   ```

   (case-sensitive; UI copy in mixed case can't false-positive. If a
   legitimate match somehow survives, inspect by hand — don't carve out.)

4. **QUALIFY and BITEMPORAL backend support verified** by reading
   `backend/internal/querybuilder/boresolver/` for clause emission before
   assuming lossless conversion. If backend doesn't emit them, scope Phase 3
   to WHERE/HAVING only and record the gap.

---

## Three renames the guardrail forced

The Layer 1 naming ban caught three legitimate functions whose names
matched `(generate|build|compile|format)*SQL|Sql` even though they weren't
fabricators. Each was renamed rather than exempted — the new names better
describe what the function does, and "rename-not-exempt" is the rule's
intent.

| Old name | New name | Why it was a false positive | Files |
|---|---|---|---|
| `compileSql` | `fetchCompiledSql` | It was a `previewQuery` wrapper, not a generator | `features/query-execution/queryExecutionApi.ts` (+ import sites) |
| `generateSQL` | `requestGeneratedSQL` | It was a `fetch('/api/reports/generate', …)` caller | `hooks/useReportBuilder.ts` (+ test file) |
| `formatSQL` | `prettyPrintSql` | It was a whitespace pretty-printer of existing SQL, not a formatter of input data | `pages/semantic-playground/components/SQLViewer.tsx` |

---

## Phase 1 deferred-items list

Items intentionally deferred from Phase 1 to Phase 2/3. **Don't re-add
these to the panel without a second consumer asking for them.**

| Deferred prop / capability | Why deferred | Lands in |
|---|---|---|
| `enableSearch` (client-side result filtering) | No second consumer needs it; SavedQueryEditor doesn't | Phase 2 (LiveQueryTab) |
| `enablePagination` | LiveQueryTab paginates today; virtualization may obviate | Phase 2 (after virtualization decision) |
| `onCellDoubleClick` | LiveQueryTab uses it for drill-down modal | Phase 2 |
| `toolbarActions` slot | No second consumer needs it yet; LiveQueryTab's own toolbar wraps the panel externally | Phase 2 if LiveQueryTab's surviving toolbar needs it |
| `<ResultsTable>` / `<ResultsChart>` internal sub-components | Inline today; LiveQueryTab's row counts force virtualization | Phase 2 |
| `local/no-sql-fabrication` tightened for fragment assembly | Same false-positive class as UI copy; Phase 3 work list is enumerable without it | Phase 3 only if a real fragment-level slip appears |

---

## Phase 2 scope (verified — session notes, 2026-09)

LiveQueryTab display migration. Read-through done; no code until PR #112
(this one) lands. Below is the verified A1–A4 inventory + C verdict.

### A1 — render tree map

`LiveQueryTab.tsx` (~2250 lines). `resultTab` is a number 0–4 across 5 tabs:

- **Tab 0 (Data Grid):** renders `executeResult?.rows` via `<Table>` with
  custom `tableSearchFilter` (filter-as-you-type, resets page to 0 on
  change), `tablePage`/`tableRowsPerPage` pagination, and a "Dynamic ABAC
  Masking Active" chip bound to toolbar state.
- **Tab 1 (Pushdown SQL Engine):** renders `previewSql` via
  `<SyntaxHighlighter>` with Copy SQL button. `previewSql` is the output
  of `updatePreview()` (real `previewQuery` → `setPreviewSql(res.sql)` on
  success, `generatePostgresSQL()` fallback on failure/empty).
- **Tab 2 (Visual Chart):** inline SVG charts (NOT `buildChartOption`).
  `chartType` state: `'bar' | 'line' | 'area' | 'pie'`.
- **Tab 3 (Explain & ABAC Sentinel DAG):** fabricated subtree; `selectedDAGNodeId`
  state. **Whole tab deletes in Phase 2.**
- **Tab 4 (JSON Result):** `JSON.stringify(executeResult?.rows || [], null, 2)`.
  Debug view; **stays as LiveQueryTab's own tab outside the shared panel**.

`updatePreview` and `handleRunQuery` carry the whole data-acquisition story
today; Phase 2 routes them through `useQueryExecution`.

### A2 — deferred-prop adoption list

| Deferred prop | LiveQueryTab need | Evidence line |
|---|---|---|
| `enableSearch` | **Yes, adopt.** | `tableSearchFilter` TextField (line ~1643); filter-as-you-type semantics, resets `tablePage=0` on change |
| `enablePagination` | **Maybe drop in favor of virtualization.** | `tablePage`/`tableRowsPerPage` + `TablePagination` (default 10/page). If `<ResultsTable>` adopts virtualization, the page model collapses to "show first N" + scroll. Decision pending Phase 2 code-read. |
| `onCellDoubleClick` | **Yes, adopt.** | Lines 1716–1720: `setDrillField`/`setDrillFilterContext`/`setDrillModalOpen` → `DrillDownGridModal`. Independent of ABAC toolbar; **real drill affordance, must survive.** |
| `toolbarActions` slot | **Yes, adopt.** | Copy SQL button (line 1622, survives), `executionTimeMs` chip (line 1625, survives). The "Dynamic ABAC Masking Active" chip (line 1661) **drops with the toolbar**. |
| `chartTypes` extension (`'area'`) | **Yes, extend the type union.** | `chartType` state at line 241 includes `'area'`. Either add to QueryResultsPanel's chart-type prop, OR LiveQueryTab keeps its bespoke SVG chart tab. **Recommend extending the type** to consolidate chart pipelines — `buildChartOption` already handles `area` as a smooth filled line. |

### A3 — masking verdict

**Toolbar-theater only. No real functionality lost by dropping the toolbar.**

`userRole` and `enableDynamicMasking` are referenced only in:
- `generatePostgresSQL`'s fake SQL comment (deleted when the function is removed)
- `handleRunQuery`'s mock-fallback execution time (deleted when the mock is removed)
- The DAG subtree's hardcoded description (deleted with the DAG)
- Lock icons next to dimension chips in the field selector (line 1441 — decorative, doesn't mask)
- Lock icons next to sensitive columns in the Data Grid header (line 1689 — decorative, doesn't mask)
- The "Dynamic ABAC Masking Active" chip on Tab 0 (line 1661 — drops with the toolbar)
- The toolbar itself (lines 1986–2013)

**No cell-level masking.** Cells at line 1723 render raw
`String(row[colName] ?? '')`. Drop the toolbar; nothing real rides on it.

### A4 — SSRS bundling

`SSRSReportBuilder.tsx` is the only SSRS file using the legacy
`/api/semantic/query` endpoint. `ReportWidgetRenderer.tsx` already calls the
real `executeQuery`. **Phase 2 includes SSRS:** replace `runPreviewQuery`'s
body with a `useQueryExecution`-driven call (a saved-query-style preview for
the report designer), drop the `/api/semantic/query` fetch and the synthetic
mock-row fallback. Same migration shape as LiveQueryTab.

### C — QUALIFY/BITEMPORAL backend verdict (scope change for Phase 3)

Read of `backend/internal/boresolver/` and `backend/internal/querybuilder/`:

| Clause | Backend support | Phase 3 implication |
|---|---|---|
| `WHERE` | Full — main `BOSQLGenerator` path | Route through `previewQuery`. Lossless. |
| `HAVING` | Minimal — only a test helper at `bo_sql_generator.go:1011`, not in the main generation path | **Phase 3 drops the HAVING category** from `buildSQL`. UI marks it disabled; backend doesn't compile it. |
| `QUALIFY` | **Zero** — keyword doesn't appear anywhere in `boresolver` or `querybuilder` | **Phase 3 drops QUALIFY entirely.** Decorative in `buildSQL` today. |
| `BITEMPORAL` | Real, but **not as a SQL clause** — `BitemporalRangeCompiler` + `InjectBitemporalScoping` inject `system_valid_from`/`system_valid_to` predicates into WHERE | **Phase 3 maps the BITEMPORAL category to `KnowledgeDate` context**, not a SQL keyword. If `KnowledgeDate` isn't already wired through `previewQuery`'s `QueryDef`, that's the Phase 3 backend change. |

This **changes Phase 3's scope from "delete `buildSQL`" to "delete `buildSQL`, drop HAVING and QUALIFY, route BITEMPORAL through a backend context field."** Document in Phase 3 planning, not in this PR.

### Phase 2 deletion list

Concrete items to remove from `LiveQueryTab.tsx`:

> **Line numbers below are pre-`1ef260574`.** The residue commit deleted ~163 lines from LiveQueryTab (the `generatePostgresSQL` function + the mock-row fallback + the two false-claim chips); all line numbers below that exceed the deletion zone need to add ~163 when reading against the current file. Post-residue ref counts are in the `1ef260574` commit message.

- ~~`generatePostgresSQL` function (lines 418–530, ~113 lines)~~ **landed in `#112` / `1ef260574`** (Layer 1 violation in the guardrail's own tree; deletion enforced the rule, not a config change)
- ~~The `generatePostgresSQL` fallback branches in `updatePreview` (lines 589, 593)~~ **landed in `#112` / `1ef260574`**
- ~~The mock-row fallback in `handleRunQuery` (lines 740–789, ~50 lines)~~ **landed in `#112` / `1ef260574`** — `setExecuteResult(null)` on failure + the `|| 12` and `via ${engine}` fabricated-timing toast also die with this block
- ~~The `<Chip label="Engine: ${resolvedEngineTier.label}">` chip on Tab 1~~ **landed in `#112` / `1ef260574`** — `resolvedEngineTier` is purely client-side heuristics (`useMemo` derivation, no backend call determines the tier); the chip asserted an engine attribution that had no backend witness. Same class as the success-toast `via ${engine}` fabrication that died with the mock-fallback removal.
- ~~The `<Chip label="Two-Pass CTE Compilation Active">` chip on Tab 1~~ **landed in `#112` / `1ef260574`** — `isCalculated` is a frontend-only flag, never sent to the backend (the `MeasureDef` boundary drops it); `CompileDeepCalculations` in `boresolver/calc_compiler.go` does emit `WITH layer_0 AS (...)` for terms with `Formula`, but the chip's claim of "compilation is active" wasn't bound to any observable backend behavior. Displaying it was a false claim about live compilation.
- The ABAC Persona/Masking toolbar (lines 1986–2015 pre-residue, ~30 lines)
- The "Dynamic ABAC Masking Active" chip on Tab 0 — decorative; claims masking on data that was never masked. Confirmed by `displayRows` read: pure search-filter memo, no `userRole`/`enableDynamicMasking` deps, no cell transformation.
- The DAG subtree on Tab 3 (lines 1967–2141 pre-residue, ~175 lines) — replaced with `extraTabs` placeholder or omitted entirely
- `userRole` / `enableDynamicMasking` state — orphaned by toolbar removal
- `selectedDAGNodeId` state — orphaned by DAG removal
- `chartType` if adopting `chartTypes` extension — move to QueryResultsPanel internal
- `tableSearchFilter` / `tablePage` / `tableRowsPerPage` — move into QueryResultsPanel when `enableSearch`/`enablePagination` land
- All bespoke grid / SVG chart / SyntaxHighlighter JSX — replaced with `<QueryResultsPanel>`

Net: LiveQueryTab shrinks meaningfully (target ~1200–1300 lines from ~2250; first ~163 lines removed by the `#112` / `1ef260574` commit), and the builder core (drag-and-drop field picker, ABAC-aware field picker if it survived) stays untouched.

### A3 addendum — `displayRows` re-read (2026-09-23, post `#112` / `1ef260574`)

The A3 verdict (no cell-level masking, toolbar drops safely) was re-verified by reading the `displayRows` memo and `isMasked` reference lines as one block. Findings:

- `displayRows` (line 745 in the residue-commit file) is a `useMemo` with deps `[executeResult, tableSearchFilter]`. It applies only a client-side search filter — `rows.filter(r => Object.values(r).some(...))`. No `userRole`/`enableDynamicMasking` deps, no transformation of cell values.
- The `isMasked` reference at line 762 lives inside the `explainPlanDAGNodes` memo. It feeds only the DAG's hardcoded display strings ("Masking Policy", "Masked Columns"), which die with the DAG in Phase 2.
- The "Dynamic ABAC Masking Active" chip on Tab 0 is decorative — bound to `userRole === 'analyst' && enableDynamicMasking` but does not mask any cell value displayed in the table.

**Verdict confirmed: toolbar drops safely; no real functionality lost.** The chip is a remaining "false claim" surface (it claims masking that never existed). Removed in Phase 2 with the toolbar.

### Phase 2 — out of scope, deferred past PR #112

- New props on QueryResultsPanel (`enableSearch`, `enablePagination`, `onCellDoubleClick`, `chartTypes` extension, `toolbarActions` slot, `<ResultsTable>`/`<ResultsChart>` extraction) — **separate Phase 2 PR** so this PR stays bisectable
- Virtualization (TableVirtuoso or react-window) — Phase 2.5 if pagination collapses
- SSRS migration — bundled with the second Phase 2 PR (props land first, then LiveQueryTab + SSRS adopt)
- **CSV export + column type formatting** in `<ResultsTable>` — scope change: re-homed from Phase 1 to the Phase 2 props PR. They're `<ResultsTable>` concerns; `<ResultsTable>` extraction is already Phase 2's first commit; bolting them onto the fabrication-removal commit would have muddied scope.
- **RuleTester fixture** for `no-sql-fabrication` (covers: 2-hole → warn, UI copy → clean, 1-hole → clean, function-declaration → error, arrow-const → error). Deferred to Phase 2 props PR with **owner: TBD** — needs a human to commit to maintaining it. Until then, the guardrail's correctness is verified manually via the smoke probes the commits already document.

---

## Revert / baseline

Tag: `baseline/query-results-panel` at `b258de304`. The tag points at the
merge commit that contains `bab336a69` (QueryResultsPanel extraction) and
`e20d243a7` (lint cleanup). Restore any file from the baseline with
`git checkout baseline/query-results-panel -- <paths>`.

---

## History

- Phase 1 — landed in `#112` commits `4efd14f78`–`5a5a8eaa3` (PR opened against `main`). QueryResultsPanel v2 (`resultSet`/`extraTabs`), `useQueryExecution` hook + adapters, two-layer guardrail, three forced renames. SavedQueryEditor migrated; LiveQueryTab and FilterBuilderPanel untouched.
- Phase 2 scoping — landed in `#112` commit `1978173e7` (docs-only). A1–A4 inventory + C backend read; A3 verdict re-confirmed; C verdict shifted Phase 3 scope (drop HAVING/QUALIFY, route BITEMPORAL through backend context).
- Residue / first enforcement — landed in `#112` commit `1ef260574` (the SHA referenced throughout this doc). `LiveQueryTab.generatePostgresSQL` removed (the guardrail's first demanded deletion; Layer 1 violation caught by a full-repo eslint run that PR #112's pre-flight missed because it only linted touched files), plus the mock-row fallback in `handleRunQuery` (the more serious data-integrity violation — fabricated rows + fake `executionTimeMs` toast), the `|| 12` and `via ${engine}` fabricated claims in the success notification, and the two Tab 1 chips (`Engine: ${resolvedEngineTier.label}` and `Two-Pass CTE Compilation Active`) that asserted behavior without backend witness. `queryBuilderMock.ts` (dead code) and its orphan import in `queryBuilderApi.ts` deleted; cleared 4 stale Layer 1 hits.
- Phase 2 — pending. LiveQueryTab display migration onto the new panel.
- Phase 3 — pending. FilterBuilderPanel → `previewQuery`.

---

## Known violations ledger

Full-repo eslint is now a **standing pre-flight step** (added in `#112` commit `1ef260574`). Every Layer 1/2 hit outside the quarantine gets one of four dispositions: **deleted now**, **renamed** (false positive, the new name better describes what the function does), **quarantined** (`reporting/`), or **listed here with a phase assignment**. No hit stays "decorative" — the guardrail's error level is only honest if every violation has a destination.

Populated from the post-`1ef260574` full-repo eslint run (`/tmp/post-delete-lint.txt`, `wc -l` of guardrail hits = 14). Counts are line-exact. Test files (`**/*.test.{ts,tsx}`) have `no-restricted-syntax: off` in the eslint config, so test-file Layer 1 hits don't appear in the lint count; the only such hits are `dataExplorerApi.test.ts:5,141,145` calling `generateDialectSQL` — recorded here but invisible to the lint counter.

### Pre-residue (`1ef260574^`) total: **21 hits** = 10 Layer 1 + 11 Layer 2

Killed by the residue commit: 6 Layer 1 + 1 Layer 2 = 7 hits. Remaining: **14 hits** = 4 Layer 1 + 10 Layer 2.

### Layer 1 errors — naming convention ban

| File:line | Symbol | Disposition | Phase / SHA | Owner |
|---|---|---|---|---|
| `pages/.../LiveQueryTab.tsx:419,589,593` | `generatePostgresSQL` (1 defn + 2 calls) | **Deleted** | `#112` / `1ef260574` | — |
| `features/query-builder/services/queryBuilderMock.ts:208,246,257` | `generateMockSQL` (1 defn + 2 calls) + `installQueryBuilderMock` (whole file) | **Deleted** — file was dead code (gated behind `if (false)`); the lint output was live, the install was not | `#112` / `1ef260574` | — |
| `features/business-objects/StreamingBindingPanel.tsx:19,98` | `generatePreviewSQL` (1 defn + 1 call) | **Ledger** — Flink CEP streaming surface, separate workstream. The function is local to one component and renders a hardcoded Flink SQL preview for the binding wizard. Decision needed: backend compile endpoint for streaming SQL, or relabel "preview" as illustrative. Test files don't add hits here (none in test scope). | TBD | **Owner: TBD** — needs a tracker before this can be phased. Open issue: "CEP streaming preview: backend-compile or honest relabel?" |
| `features/data-explorer/services/dataExplorerApi.ts:1027` + `data-explorer/components/PlaygroundDeveloperDrawer.tsx:65` | `generateDialectSQL` (1 defn + 1 call in app code; 3 calls in `dataExplorerApi.test.ts:141,145` invisible to lint due to test-file rule exemption) | **Ledger** — Data Explorer surface, not Query Builder migration scope. `generateDialectSQL` produces dialect-specific SQL from a `QueryState` for a Data Explorer playground — pure fabricator with unit-test coverage. Migration target unclear (no backend `executeDialectSQL` endpoint). | Out of scope (separate workstream) | **Owner: TBD** — Data Explorer is a separate workstream; needs its own guardrail migration. Open issue: "Data Explorer `generateDialectSQL`: scope decision (backend compile vs. honest relabel)" |
| `dataExplorerApi.test.ts:5,141,145` | `generateDialectSQL` imports + 2 calls | **Ledger** — test-file Layer 1 hits invisible to lint; recorded here for completeness | (same as above) | (same as above) |

### Layer 2 warnings — AST structural detection

Mostly false positives or out-of-scope. None block the workstream. Listed here so a future session knows they're expected, not regressions.

| File | Line | Disposition |
|---|---|---|
| `components/BusinessObjectManager/PreAggregationWizard.tsx` | 168 | **Ledger** — likely UI copy / mock; not a real fabricator. Verify in-place if Phase 2 expands scope. |
| `components/reporting/SSRSReportBuilder.tsx` | 752 | **Ledger** — Phase 3 work list (SSRS preview migration). |
| `components/semantic-mapper/__tests__/ReportingServerUI.tsx` | 384 | **Ledger** — test file, gated. |
| `features/api-builder/pages/APIBuilderPage.tsx` | 299, 332 | **Ledger** — API Builder surface, separate workstream. **Owner: TBD** — needs tracker. |
| `features/tenants/components/BYOBIConfigTab.tsx` | 59 | **Ledger** — likely UI string, not a fabricator. |
| `hooks/useUnifiedSemanticBuilder.tsx` | 491 | **Ledger** — Semantic Builder surface, separate workstream. **Owner: TBD** — needs tracker. |
| `pages/.../LiveQueryTab.tsx` | 525, 975 (pre-residue) → 1 line post-residue | **Ledger** — the remaining one dies with the DAG in Phase 2. |
| `pages/DataExplorerPage.tsx` | 103 | **Ledger** — Data Explorer surface, separate workstream. **Owner: TBD** — same workstream as `generateDialectSQL`. |
| `pages/ModelGeneratorPage.tsx` | 250 | **Ledger** — Model Generator surface, separate workstream. **Owner: TBD** — needs tracker. |

The "Owner: TBD" rows are the honest state: nobody has committed to closing them. The decision is whether to open tracker issues now or amend the workstream goal statement to scope narrower than "no fabricated SQL anywhere" — current PR body language ("no client-side SQL generation, enforced by ESLint Layer 1/2") is **scoped to the Query Builder / Reporting migration path**, not "every fabricator in the repo." A future session that reads this doc and finds a `TBD` owner knows what to do.

### Renames — false-positive Layer 1 hits, renamed rather than exempted

The naming ban fires on these despite them being legitimate backend-call
wrappers or pretty-printers, because their names matched the convention.
Each was renamed rather than exempted (the new name better describes what
the function does):

| Old name | New name | File |
|---|---|---|
| `compileSql` | `fetchCompiledSql` | `features/query-execution/queryExecutionApi.ts` |
| `generateSQL` | `requestGeneratedSQL` | `hooks/useReportBuilder.ts` (+ test file) |
| `formatSQL` | `prettyPrintSql` | `pages/semantic-playground/components/SQLViewer.tsx` |
