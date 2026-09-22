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
| 6 | **No fabricated SQL anywhere.** LiveQueryTab's `generatePostgresSQL` removed in Phase 2. FilterBuilderPanel's `buildSQL`/`buildGroupSQL` removed in Phase 3. Mock-row fallbacks in any `handleRun*` removed. | The whole point of this workstream | enforced by guardrail Layer 1 |

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

- `generatePostgresSQL` function (lines 418–530, ~113 lines)
- The `generatePostgresSQL` fallback branches in `updatePreview` (lines 589, 593)
- The mock-row fallback in `handleRunQuery` (lines 740–789, ~50 lines)
- The ABAC Persona/Masking toolbar (lines 1986–2015, ~30 lines)
- The "Dynamic ABAC Masking Active" chip on Tab 0 (lines 1661–1669)
- The DAG subtree on Tab 3 (lines 1967–2141, ~175 lines) — replaced with `extraTabs` placeholder or omitted entirely
- `userRole` / `enableDynamicMasking` state (lines 230–231) — orphaned by toolbar removal
- `selectedDAGNodeId` state (line 247) — orphaned by DAG removal
- `chartType` if adopting `chartTypes` extension (line 241) — move to QueryResultsPanel internal
- `tableSearchFilter` / `tablePage` / `tableRowsPerPage` (lines 242–244) — move into QueryResultsPanel when `enableSearch`/`enablePagination` land
- All bespoke grid / SVG chart / SyntaxHighlighter JSX (lines 1637–1762, 1788–1965) — replaced with `<QueryResultsPanel>`

Net: LiveQueryTab shrinks meaningfully (target ~1200–1300 lines from ~2250), and the builder core (drag-and-drop field picker, ABAC-aware field picker if it survived) stays untouched.

### Phase 2 — out of scope, deferred past PR #112

- New props on QueryResultsPanel (`enableSearch`, `enablePagination`, `onCellDoubleClick`, `chartTypes` extension, `toolbarActions` slot, `<ResultsTable>`/`<ResultsChart>` extraction) — **separate Phase 2 PR** so this PR stays bisectable
- Virtualization (TableVirtuoso or react-window) — Phase 2.5 if pagination collapses
- SSRS migration — bundled with the second Phase 2 PR (props land first, then LiveQueryTab + SSRS adopt)

---

## Revert / baseline

Tag: `baseline/query-results-panel` at `b258de304`. The tag points at the
merge commit that contains `bab336a69` (QueryResultsPanel extraction) and
`e20d243a7` (lint cleanup). Restore any file from the baseline with
`git checkout baseline/query-results-panel -- <paths>`.

---

## History

- Phase 1 — landed in `<this PR>`. QueryResultsPanel v2 (`resultSet`/`extraTabs`),
  `useQueryExecution` hook + adapters, two-layer guardrail, three forced
  renames. SavedQueryEditor migrated; LiveQueryTab and FilterBuilderPanel
  untouched.
- Phase 2 — pending. LiveQueryTab display migration onto the new panel.
- Phase 3 — pending. FilterBuilderPanel → `previewQuery`.
