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
| 6 | **No client-side SQL generation in the Query Builder / Reporting / Live Query migration path.** LiveQueryTab's `generatePostgresSQL` removed in the mock-removal commit (Layer 1 violation in the guardrail's own tree — first demanded deletion; see "Known violations ledger"). FilterBuilderPanel's `buildSQL`/`buildGroupSQL` removed in Phase 3. Mock-row fallbacks in any `handleRun*` removed in the same commit. Other workstreams (Data Explorer's `generateDialectSQL`, CEP's `generatePreviewSQL`) have TBD owners and aren't gated by this workstream's guardrail — see ledger. | The whole point of this workstream (scoped) | enforced by guardrail Layer 1 + standing pre-flight |

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

> **Line numbers below are pre-mock-removal.** The mock-removal commit deleted ~163 lines from LiveQueryTab (the `generatePostgresSQL` function + the mock-row fallback); the correction commit deleted the two false-claim chips in the SQL tab. All line numbers below that exceed the deletion zone need to add ~163 when reading against the current file. Post-commit ref counts are in the respective commit messages.

- ~~`generatePostgresSQL` function (lines 418–530, ~113 lines)~~ **landed in the mock-removal commit** (Layer 1 violation in the guardrail's own tree; deletion enforced the rule, not a config change)
- ~~The `generatePostgresSQL` fallback branches in `updatePreview` (lines 589, 593)~~ **landed in the mock-removal commit**
- ~~The mock-row fallback in `handleRunQuery` (lines 740–789, ~50 lines)~~ **landed in the mock-removal commit** — `setExecuteResult(null)` on failure + the `|| 12` and `via ${engine}` fabricated-timing toast also die with this block
- ~~The `<Chip label="Engine: ${resolvedEngineTier.label}">` chip on Tab 1~~ **landed in the correction commit** — `resolvedEngineTier` is purely client-side heuristics (`useMemo` derivation, no backend call determines the tier); the chip asserted an engine attribution that had no backend witness. Same class as the success-toast `via ${engine}` fabrication that died with the mock-fallback removal.
- ~~The `<Chip label="Two-Pass CTE Compilation Active">` chip on Tab 1~~ **landed in the correction commit** — `isCalculated` is a frontend-only flag, never sent to the backend (the `MeasureDef` boundary drops it); `CompileDeepCalculations` in `boresolver/calc_compiler.go` does emit `WITH layer_0 AS (...)` for terms with `Formula`, but the chip's claim of "compilation is active" wasn't bound to any observable backend behavior. Displaying it was a false claim about live compilation.
- The ABAC Persona/Masking toolbar (lines 1986–2015 pre-residue, ~30 lines)
- The "Dynamic ABAC Masking Active" chip on Tab 0 — decorative; claims masking on data that was never masked. Confirmed by `displayRows` read: pure search-filter memo, no `userRole`/`enableDynamicMasking` deps, no cell transformation.
- The DAG subtree on Tab 3 (lines 1967–2141 pre-residue, ~175 lines) — replaced with `extraTabs` placeholder or omitted entirely
- `userRole` / `enableDynamicMasking` state — orphaned by toolbar removal
- `selectedDAGNodeId` state — orphaned by DAG removal
- `chartType` if adopting `chartTypes` extension — move to QueryResultsPanel internal
- `tableSearchFilter` / `tablePage` / `tableRowsPerPage` — move into QueryResultsPanel when `enableSearch`/`enablePagination` land
- All bespoke grid / SVG chart / SyntaxHighlighter JSX — replaced with `<QueryResultsPanel>`

Net: LiveQueryTab shrinks meaningfully (target ~1200–1300 lines from ~2250; first ~163 lines removed by the mock-removal commit), and the builder core (drag-and-drop field picker, ABAC-aware field picker if it survived) stays untouched.

### A3 addendum — `displayRows` re-read (2026-09-23, post mock-removal commit)

The A3 verdict (no cell-level masking, toolbar drops safely) was re-verified by reading the `displayRows` memo and `isMasked` reference lines as one block. Findings:

- `displayRows` (line 745 in the correction-commit file) is a `useMemo` with deps `[executeResult, tableSearchFilter]`. It applies only a client-side search filter — `rows.filter(r => Object.values(r).some(...))`. No `userRole`/`enableDynamicMasking` deps, no transformation of cell values.
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

> **Commit naming.** Two commits in this PR's history carry "residue" in their message. They are distinct:
>
> - **The mock-removal commit** (`1ef260574`): first enforcement — removed `generatePostgresSQL`, the mock-row fallback, and the `queryBuilderMock.ts` dead code. Tagged here as "the mock-removal commit."
> - **The correction commit** (`79173671b`): the residue commit — removed the two false-claim chips and corrected the ledger arithmetic. Tagged here as "the correction commit."
>
> Both SHAs go stale the moment this PR is rebased or squash-merged; do not cite them as durable anchors. Where a date-anchored referent is needed, the SHAs are kept inline; otherwise the prose names above are preferred.

- Phase 1 — landed in `#112` commits `4efd14f78`–`5a5a8eaa3`. QueryResultsPanel v2 (`resultSet`/`extraTabs`), `useQueryExecution` hook + adapters, two-layer guardrail, three forced renames. SavedQueryEditor migrated; LiveQueryTab and FilterBuilderPanel untouched.
- Phase 2 scoping — landed in `#112` commit `1978173e7` (docs-only). A1–A4 inventory + C backend read; A3 verdict re-confirmed; C verdict shifted Phase 3 scope (drop HAVING/QUALIFY, route BITEMPORAL through backend context).
- The mock-removal commit (`1ef260574`) — `LiveQueryTab.generatePostgresSQL` removed (the guardrail's first demanded deletion; Layer 1 violation caught by a full-repo eslint run that PR #112's pre-flight missed because it only linted touched files), plus the mock-row fallback in `handleRunQuery` (the more serious data-integrity violation — fabricated rows + fake `executionTimeMs` toast), the `|| 12` and `via ${engine}` fabricated claims in the success notification. `queryBuilderMock.ts` (dead code) and its orphan import in `queryBuilderApi.ts` deleted.
- The correction commit (`79173671b`) — two Tab 1 chips (`Engine: ${resolvedEngineTier.label}` and `Two-Pass CTE Compilation Active`) that asserted behavior without backend witness, plus ledger arithmetic correction (this doc).
- Phase 2 — pending. LiveQueryTab display migration onto the new panel.
- Phase 3 — pending. FilterBuilderPanel → `previewQuery`.

---

## Known violations ledger

Full-repo eslint is now a **standing pre-flight step** (added in the mock-removal commit). Every Layer 1/2 hit outside the quarantine gets one of four dispositions: **deleted now**, **renamed** (false positive, the new name better describes what the function does), **quarantined** (`reporting/`), or **listed here with a phase assignment**. No hit stays "decorative" — the guardrail's error level is only honest if every violation has a destination.

Populated from the post-mock-removal-commit full-repo eslint run (`/tmp/post-delete-lint.txt`, `wc -l` of guardrail hits = 14). Counts are line-exact. Test files (`**/*.test.{ts,tsx}`) have `no-restricted-syntax: off` in the eslint config, so test-file Layer 1 hits don't appear in the lint count; the only such hits are `dataExplorerApi.test.ts:5,141,145` calling `generateDialectSQL` — recorded here but invisible to the lint counter.

### Pre-mock-removal total: **21 hits** = 10 Layer 1 + 11 Layer 2

Killed by the mock-removal commit: 6 Layer 1 + 1 Layer 2 = **7 hits**. The single Layer 2 death was `LiveQueryTab.tsx:525` (pre-residue) — the two-pass CTE `return` template literal inside the deleted `generatePostgresSQL` function (contains `WITH layer_0 AS (\n    SELECT\n        ${cteSelectCols.join(...)}\n    FROM ${driverTable} t0`). The remaining LiveQueryTab Layer 2 hit (was at line 975 pre-residue, line 819 post-residue) is the user's `displayRows` memo commentary line that survives into Phase 2 with the DAG. Remaining: **14 hits** = 4 Layer 1 + 10 Layer 2.

### Kill table — what died in the mock-removal commit

| File:line (pre-residue) | Symbol | Why it died |
|---|---|---|
| `pages/.../LiveQueryTab.tsx:419` | `generatePostgresSQL` definition | Function deleted (113 lines, the entire two-pass CTE fabricator) |
| `pages/.../LiveQueryTab.tsx:589,593` | `generatePostgresSQL()` calls (in `updatePreview`) | Call sites in the deleted function's only consumer |
| `features/query-builder/services/queryBuilderMock.ts:208,246,257` | `generateMockSQL` defn + 2 calls + `installQueryBuilderMock` | Whole file deleted (dead code, gated behind `if (false)` — the lint hits were live, the install was not) |
| `features/query-builder/services/queryBuilderApi.ts:18-24` | `installQueryBuilderMock` import + dead `if (false)` install block | Orphaned by file deletion (6 lines) |
| `pages/.../LiveQueryTab.tsx:525` | Layer 2: two-pass CTE `return` template literal | Inside `generatePostgresSQL`; deleted with the function (1 hit) |

Note on the killed-hit counts: this table accounts for 5 Layer 1 items (1 defn + 4 calls in 2 files) + 1 Layer 2 item. The 21 → 14 arithmetic uses **lint items**, where each lint hit is one `error`/`warning` line in the eslint output — same unit the standing pre-flight uses. Behavioral call sites (e.g., how many times the deleted `generatePostgresSQL` was *reachable* from any code path) are a different unit; this doc tracks lint items.

### Layer 1 errors — naming convention ban

| File:line | Symbol | Disposition | Phase / SHA | Owner |
|---|---|---|---|---|
| `pages/.../LiveQueryTab.tsx:419,589,593` | `generatePostgresSQL` (1 defn + 2 calls) | **Deleted** in the mock-removal commit | — |
| `features/query-builder/services/queryBuilderMock.ts:208,246,257` | `generateMockSQL` (1 defn + 2 calls) + `installQueryBuilderMock` (whole file) | **Deleted** in the mock-removal commit — file was dead code (gated behind `if (false)`); the lint output was live, the install was not | — |
| `features/business-objects/StreamingBindingPanel.tsx:19,98` | `generatePreviewSQL` (1 defn + 1 call) | **Ledger** — Flink CEP streaming surface, separate workstream. The function is local to one component and renders a hardcoded Flink SQL preview for the binding wizard. Decision needed: backend compile endpoint for streaming SQL, or relabel "preview" as illustrative. Test files don't add hits here (none in test scope). | TBD | **Owner: TBD** — needs a tracker before this can be phased. **Open issue status:** not filed in this PR; scheduled post-merge (issue title proposed: "CEP streaming preview: backend-compile or honest relabel?") |
| `features/data-explorer/services/dataExplorerApi.ts:1027` + `data-explorer/components/PlaygroundDeveloperDrawer.tsx:65` | `generateDialectSQL` (1 defn + 1 call in app code; 3 calls in `dataExplorerApi.test.ts:141,145` invisible to lint due to test-file rule exemption) | **Ledger** — Data Explorer surface, not Query Builder migration scope. `generateDialectSQL` produces dialect-specific SQL from a `QueryState` for a Data Explorer playground — pure fabricator with unit-test coverage. Migration target unclear (no backend `executeDialectSQL` endpoint). | Out of scope (separate workstream) | **Owner: TBD** — Data Explorer is a separate workstream; needs its own guardrail migration. **Open issue status:** not filed in this PR; scheduled post-merge (issue title proposed: "Data Explorer `generateDialectSQL`: scope decision (backend compile vs. honest relabel)") |
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
| `pages/.../LiveQueryTab.tsx` | 975 (pre-residue) → 819 (post-residue) | **Ledger** — survives in Phase 2 (the `displayRows` memo's `String(val || '')` search-filter stringification trips the Layer 2 pattern on a 1-hole template). Dies with the DAG in Phase 2 once the whole tab is replaced. |
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

---

## Pre-existing TypeScript errors (not from this PR)

Per-file `tsc --noEmit` against the touched files surfaces pre-existing errors
that pre-date every commit in this PR. **They are not introduced here** —
verified via `git show HEAD~1` for the LiveQueryTab errors. A future session
that sees them in the tsc output should not assume they're recent; the
provenance is recorded here so the assumption isn't relitigated.

| File:line (post-residue) | Error | Provenance |
|---|---|---|
| `LiveQueryTab.tsx:285` | `Cannot find name 'fetchAPI'` | Pre-residue at line 283; `git log -S 'fetchAPI' -- LiveQueryTab.tsx` shows it as last touched in `1f615052d` (duplicate-fields fix). The original `import { fetchAPI } from '../../../../api'` was removed but the 3 callsites were not renamed. **Rename residue from a previous PR.** Runtime: throws `ReferenceError` on the catalog-terms fetch path. |
| `LiveQueryTab.tsx:432` | `Cannot find name 'fetchAPI'` | Same — `evaluate-cost` call site. |
| `LiveQueryTab.tsx:658` | `Cannot find name 'fetchAPI'` | Same — NLQ `/business-objects/ai/nlq` call site. |
| `LiveQueryTab.tsx:1137` | `No overload matches this call` | Pre-existing — the `<Typography>{calc.formula}</Typography>` cast. Same commit as the rename. |

**Disposition:** 3 undefined-identifier call sites + 1 overload error in
LiveQueryTab.tsx. Rename residue from a prior PR — fix is to either restore
the `import { fetchAPI } from '../../../../api'` line, or rewrite the 3
callsites to `apiFetch` (the project's standardized wrapper, already used
in `queryBuilderApi.ts`). **Open issue status:** not filed in this PR;
scheduled post-merge (issue title proposed: "LiveQueryTab rename residue:
3 `fetchAPI` callsites + 1 overload error"). Owner: TBD. **Not a #112
change.**

---

## Build status — wrong-depth import fixed; tailwind-merge real defect dispositioned separately

**Two distinct build failures. Two distinct root causes. Two distinct dispositions.**

### Failure #1 (fixed in this branch) — wrong-depth import in `SavedQueryEditor.tsx`

The failure surfaced as:
```
Could not resolve "../query-execution" from "src/features/query-builder/pages/SavedQueryEditor.tsx"
```

**Root cause:** wrong relative depth. `SavedQueryEditor.tsx` lives at `src/features/query-builder/pages/`. The import `'../query-execution'` resolves to `src/features/query-builder/query-execution/` (one level up from `pages/` lands in `query-builder/`, then `query-execution/` — which doesn't exist). The canonical lib is `src/features/query-execution/` (one level up from `pages/` is `query-builder/`; **two** levels up is `features/`, where `query-execution/` lives). Correct specifier: `'../../query-execution'`.

`tsc --noEmit` didn't surface this — mechanism *unverified*; speculation (tsc's `moduleResolution: "bundler"` is more lenient than Vite's stricter ESM resolution) is consistent with the observed behavior but not proven.

**Fix:** one extra `../` in `SavedQueryEditor.tsx:43` — one dot-segment, three characters (`'../query-execution'` → `'../../query-execution'`). LiveQueryTab's path (`'../../../../features/query-execution'` from `pages/.../tabs/`) was depth-correct — four levels up to `src/` — and did not need changing.

**Earlier attempts during this PR's review cycle** (recorded because the chain matters): changing to `'../query-execution/index'`, `'../query-execution/index.ts'`, and `'../query-execution/errorMessage'` (direct file) all failed with errors naming the *changed* specifier — confirming each attempt's error was generated against that specific path, not stale output. All four attempts inherited the same too-shallow base, which is exactly why the error shape never changed: every variant was one level short of `features/`.

**Misdiagnosis in earlier disclosure:** bold-claimed "Vite/Rollup's stricter ESM resolution rejects the directory import." That claim was refuted by the direct-file attempt's identical error, and by a minimal-repro showing directory imports work in a fresh Vite project. The bold claim was generated *after* the theory failed its own tests, not before.

### Failure #2 (NOT in this PR's scope) — `tailwind-merge` is real, dispositioned separately

After the wrong-depth fix alone, the build still fails. With **both** fixes — wrong-depth in `SavedQueryEditor.tsx` AND the `tailwind-merge` import removed from `lib/utils.ts` — the build passes. The second fix is disposition option (a) from the tracker table; the user already chose it locally, the working tree shows it applied. On HEAD (this PR's actual merge state), only the first fix is present, and the build fails as below.

The failure on HEAD:
```
[vite]: Rollup failed to resolve import "tailwind-merge" from "/Users/eganpj/GitHub/uisce/frontend/src/lib/utils.ts".
```

**Provenance (sufficient, honest).** `git log -S 'tailwind-merge' -- frontend/src/lib/utils.ts` shows the import arrived with the frontend's **initial commit `b9fc5e129`** — and `git log -- frontend/src/lib/utils.ts` shows that's the only commit ever to touch the file. So `tailwind-merge` is in this codebase's source from the very first frontend commit; it's not a #112 regression. The same `git log -S` on `package.json` shows `tailwind-merge: ^3.3.1` added in `173457e54` (BusinessObjectTree component) and bumped to `^3.6.0` in `17db58257` (chore PR).

**Current `package.json` state (working tree, NOT HEAD):** the user has removed `tailwind-merge`, `tailwindcss`, `@tailwindcss/postcss`, and `autoprefixer` from `package.json` as part of the MUI-only direction (per `git diff HEAD -- frontend/package.json`). So:

- HEAD `package.json`: declares `tailwind-merge: ^3.6.0` (line 119), `tailwindcss: ^4.1.11` (line 120), `@tailwindcss/postcss: ^4.1.18`, `autoprefixer: ^10.4.24`
- Working tree `package.json`: those four lines removed

**CI log evidence.** The Build Frontend red check on `35771443877.log:4333` is a `pnpm install` lockfile-spec mismatch — `specifiers in the lockfile (...) don't match specs in package.json (...)` — listing ~100 packages. Both lists are dumps of full spec maps, not isolated per-package errors; my earlier disclosure attributed `"tailwind-merge":"^3.6.0"` to "package.json says" and `"^3.3.1"` to "lockfile says," but those strings come from the package.json *spec list* and the lockfile's *spec list* respectively in the CI run, not from the current working tree. The CI log captures a historical state; current `package.json` working tree differs (MUI direction work in progress). The CI attribution method (`grep QueryResultsPanel|SavedQueryEditor|query-execution`) cannot catch undeclared-dep failures by construction — those failures don't mention any of this PR's filenames.

**Direct local evidence.** `npm ls tailwind-merge` returns `(empty)` — `node_modules/tailwind-merge` is not installed locally. `lib/utils.ts:3` (HEAD) imports `twMerge` from `tailwind-merge`. So the build fails on HEAD because `tailwind-merge` isn't installed despite being in the lockfile and (on HEAD) in `package.json`.

**Direction decision: MUI-only going forward.** The MUI direction means `cn()` and `tailwind-merge` go: the codebase uses MUI for class composition, and `tailwind-merge`'s job (resolving Tailwind class conflicts) is moot in an MUI-only codebase.

**Scope addition (this PR absorbed the `lib/utils.ts` source-side cleanup):** the build-fix commit restored the user's clsx-only `cn` cleanup to `lib/utils.ts`. That edit is now in #112 (`0e9cd813f`), making this PR a scope addition under the strict "branch purity" reading — the `tailwind-merge`/`cn` removal is baseline-code, not this PR's tree. The rationale for absorbing it: the user's local working tree already had it implemented, the build is otherwise broken without it, and the change is the minimum needed (1 import removed, 1 function body simplified). The PR body and the build-status section above document this plainly. The wider MUI-direction cleanup (`package.json` removal of `tailwind-merge`/`tailwindcss`/`@tailwindcss/postcss`/`autoprefixer`, `src/index.css` rewrite) stays in the user's working tree and is the next chore PR's scope — see the tracker row below.

### Tracker items — out of scope for #112

| Tracker row | Owner | Status |
|---|---|---|
| MUI-only direction cleanup (Phase 2 chore PR) | TBD | **Open issue status:** not filed in this PR. The `lib/utils.ts` source-side cleanup (clsx-only `cn`) was absorbed into #112 as a scope addition in commit `0e9cd813f` — the minimum needed for the build to pass. **Behavior note:** dropping `twMerge` subtly changes class-conflict resolution in the 44 `cn()` caller files (moot under MUI-only, where MUI is the composition primitive). The wider MUI cleanup (`package.json` removal of `tailwind-merge`/`tailwindcss`/`@tailwindcss/postcss`/`autoprefixer`, `src/index.css` rewrite, `tailwindcss` config removal) is the next chore PR's scope — currently in the user's working tree. `cn()` has **44 caller files / 196 invocations** in `src/`; the call-site pattern is `cn(...args)` so behavior is preserved by `cn(...inputs) → clsx(...inputs)`. |
| (resolved) "Vite build failure on `../query-execution` imports — root cause pending" | — | Closed: root cause was wrong-depth import; fixed in this branch by changing `'../query-execution'` to `'../../query-execution'` (one extra `../`, one dot-segment, three characters). |

## Direction note

**MUI-only going forward.** All future ledger rows resolve toward backend compile or honest relabel — not toward Tailwind-era shims. The MUI direction means `tailwind-merge`/`cn` (Tailwind-only class composition utilities) become residue, MUI is the composition primitive.
