# Page Designer (Page Studio) handoff

Written 2026-09-30 for the next engineer or LLM picking this up. Read this
first, then `docs/page-studio-app-model.md` (the model reference) and
`docs/adr/0001-page-designer-world-class-roadmap.md` (the roadmap, PR #231,
still open as "Proposed").

## The rule that governs everything

**Every page is built in Page Designer as configuration. No exceptions.**
A gap is closed with a generic studio building block or a domain
**operation**, never with a hand-built page or domain component. The owner
wants "like for like" with the hand-built screens and "world class pages via
configuration with little or no code".

## Where things stand

All of these are studio pages, published as **core pages in the gold copy**
(alpha), with **no domain components**:

| Page | Slug | Route | Gold-copy version |
| --- | --- | --- | --- |
| Mastering console | `mastering-console` | `/data/mastering` | v3 |
| Staging bindings | `staging-bindings` | `/data/staging-bindings` | v2 |
| Source hierarchy / Match rules / Vendor registry | `mdm-source-hierarchy`, `mdm-match-rules`, `mdm-vendors` | `/data/mdm/...` | v2 |
| Data pipelines (list) | `data-pipelines` | `/data/pipelines` | v1 |
| Data pipeline editor | `data-pipeline-editor` | `/data/pipelines/:id` | v5 |
| Schedules | `schedules` | `/automation/schedules` | v1 |

Other sessions added `mdm-source-scoring`, `validation-rules` and
`lakehouse-streaming` blueprints; those still use domain components and were
not built by this workstream.

ADR 0001 phase 1 (authoring confidence) progress:

- **Done:** page checker (#232), binding picker (#246).
- **Next:** point-and-click condition builder with validation; structured
  editors for what is still JSON (row buttons, action forms, map specs,
  generated columns, canvas categories); live query status on widgets in
  design mode.
- Then phases 2-5 in the ADR (generate from an operation, fragments, charts,
  grids at scale, version history, and so on).

## How a page works (the model)

`frontend/src/pages/page-studio/app/`:

- **Layout tree:** `Row`, `Column`, `Drawer`, `Dialog`, `TabSet` nodes plus
  page tabs and a filter bar (`containers.tsx`, `../RenderLayoutTree.tsx`
  runtime, `../LayoutCanvas.tsx` designer).
- **Widgets** (`AppWidgets.tsx`, `moreWidgets.tsx`, `canvas.tsx`, `chat.tsx`):
  PageHeader, TextBlock, AlertBanner, ActionButton, SearchInput,
  VariableSelect, DataGrid, KeyValue, Timeline, Form, Canvas, Chat,
  DomainComponent.
- **App model** (`appModel.ts`): variables (`default`, `initFrom`, `url`),
  queries (`operation`, `params`, `enabledWhen`, `keepPrevious`, `debounceMs`,
  `refetchWhile`, `onChange`), actions (`setVariable`, `runOperation` with
  `confirm` / `form` / `onSuccess` / `progressVariable`, `navigate`, `notify`;
  any action may carry `when`), conditions (the rule engine's RuleNode JSON,
  evaluated by `rule_engine.wasm`).
- **Bindings** (`bindings.ts`): `{{path}}` over `vars`, `queries`, `route`,
  and contextual names (`row`, `form`, `item`, `result`, `event`, `data`,
  `node`, `extra`, `message`...). Text specs translate i18n keys.
- **Runtime** (`AppRuntime.tsx`, `RuntimePage.tsx`): `useQueries` over
  registered operations; query key is `[op.domain, 'studio', op.id, params]`;
  a successful mutation invalidates `[domain]` (or `op.invalidates`).
- **Form fields** (`formFields.tsx`, `mapField.tsx`): text, multiline,
  number, date, time, select, radio, switch, chips, checklist, rows, json,
  map, note, button, upload; `fieldsFrom`, `seedKey`, `resetOn`.
- **Designer** (`AppWidgetInspector.tsx`, `ContainerInspector.tsx`,
  `fieldsEditor.tsx`, `editors.tsx`, `AppModelPanel.tsx`, `../PageEditor.tsx`).
- **Checker and picker:** `pageChecker.ts` + `PageCheckDialog.tsx` (publish is
  refused on errors); `bindingPicker.tsx` (wired into `BindingField` and the
  condition `LeafEditor`).
- **Blueprints** (`blueprints/*.ts`): each page as code that produces the page
  JSON; registered in `blueprints/index.ts`. `scheduleEditor()` in
  `blueprints/schedules.ts` is the reusable dialog fragment pattern.

Domain code lives in `frontend/src/features/<domain>/studio.ts(x)`
(operations registered via `studio-core/operations/registry.ts`, loaded by
`studio-core/registerDomains.ts`). Logic belongs in operations (shape rows
ready to display: labels, colours, flags); pages only bind.

Routes served by a studio page: `pages/page-studio/studioRoutes.ts`. A studio
route renders the **saved gold-copy page by slug**, not the blueprint.

## The working method (follow it)

1. **Record before you retire.** Before replacing a hand-built screen, write a
   shared scenario (`src/vitest/page-studio/<x>Scenario.tsx`) and run it on
   the hand-built component to record what it shows and sends. Then build the
   studio version, run the same scenario, and commit a parity test with the
   recording embedded and deliberate differences listed. Examples:
   `pipelineEditorParity`, `schedulesParity`, `configEditorsParity`,
   `consoleOverlaysParity`, `assistantParity`.
2. **Generic first.** If a page needs something the studio lacks, add it as a
   generic block or option with an inspector, not page-specific code.
3. **Checks before every PR:** `npx vitest run src/vitest/page-studio` (77
   passing at handoff), `npx tsc --noEmit -p .` (no new errors; a few old ones
   exist in `PageEditor.tsx`, `PagePerformanceDashboard.tsx`,
   `PropertiesPanel.tsx`), `npx eslint` on touched files. Read the result
   **before** committing; do not chain commit and PR onto the test command.
4. **`checkPage` must stay clean on every blueprint**
   (`pageChecker.test.ts`). It has already caught real bugs in merged pages.
5. **After merging a blueprint change, re-save the gold-copy page**, or the
   live page diverges from code (and breaks if a component it uses was
   removed). See below.
6. **GitNexus** (CLAUDE.md): run `impact` before editing an indexed symbol and
   `detect_changes` before committing. The index is far behind HEAD, so new
   studio files are "not found"; fall back to grep for callers and say so.

## Saving a page to the gold copy

There is no UI path used for this yet; it is done with a throwaway Go test
(never committed) that drives the real `PageStudioHandler` as the gold-copy
admin:

1. Generate the page JSON from the blueprint with esbuild + node
   (`<blueprint>()` -> `JSON.stringify`).
2. A test tagged `alphaharness` in `backend/internal/handlers/` opens
   `DATABASE_URL`, builds a chi router with `NewPageStudioHandler(db, nil, nil, nil)`,
   and calls `PUT /page-studio/pages/{id}` with `status: published` (existing
   core page) or `POST /page-studio/pages` then `PUT .../status` (new page),
   with `security.AuthInfo{IsGlobalAdmin: true}` and the gold tenant id from
   `SELECT public.uisce_gold_copy_tenant_id()`.
3. Run it, verify with a `psql` read, delete the test file.

Writing to alpha needs the owner's explicit go-ahead each time.

## Operational constraints

- **Backend:** start with `scripts/start-backends.sh [checkout] [port]`. The
  server applies **all pending alpha migrations at startup** and ignores
  SIGTERM (`kill -9`, then verify the port). Check
  `go run ./cmd/migrate status` first. Never edit an applied migration; fix
  forward.
- **Frontend:** the owner's Vite runs on 5173 from the main checkout
  (`/Users/eganpj/GitHub/uisce`). Keycloak's redirect is fixed to 5173, so a
  dev server on another port cannot sign in. The browser pane needs the owner
  to sign in; never enter credentials.
- **Main checkout:** it usually has the owner's uncommitted work. Do not touch
  it without asking; work in the worktree. When asked to rebase it, make a
  backup branch and a patch first.
- **Merging:** the auto-mode permission check may block `gh pr merge`; if so
  give the owner the command. Stacked PRs: retarget to `main` before merging.
- **Tenancy:** tenants inherit gold-copy objects read-only and never write
  back; never loosen tenant isolation, even in dev.
- **Rule engine:** `internal/rules/vm` is the only engine; conditions use its
  operators (`equals`, `not_equals`, `in`, `not_in`, `is_empty`,
  `is_not_empty`, `is_true`, `is_false`, `greater_than`, `less_than`,
  `contains`). There is no `is_not_true`.

## Open items

- **ADR #231** is open (Proposed); three questions in it await the owner.
- **Gold copy:** the saved `mdm-source-scoring` page lacks the `as_of`
  variable (declared in the blueprint by #246); the checker will refuse to
  publish it until it is re-saved or the variable is added.
- **Not verified in a browser:** the Schedules console, the pipeline editor's
  schedule dialog, the page checker and the binding picker (tests only; the
  pane needed a sign-in). The pipeline editor itself was checked live.
- **Hand-built pages still using `ScheduleEditor.tsx`:** report and query
  pages (`ReportLibrary`, `SSRSReportBuilder`, `SavedQueryEditor`,
  `QueryLibrary`). They are outside this workstream so far.
- **Weekday bug:** before PR #229, weekly schedules created west of UTC saved
  the wrong weekday; existing ones may run a day late.
- **Owner's local leftovers:** branch `backup/main-before-rebase-20260928`
  and a stash from the rebase can be deleted once confirmed.
- **Training guide:** a Page Designer training doc was written for the owner
  (a Claude doc, not in the repo); it was written from the code, not checked
  screen by screen.

## Suggested next steps

1. Condition builder: validate field paths against the same scope the binding
   picker browses (`browsableScope`), pick operators by value type, reuse
   `BindingPicker`.
2. Structured editors replacing `JsonField` for row buttons, action forms
   (`FormSpec`), `MapFieldSpec`, `dynamicColumns`, `rowDetail`, Canvas
   categories and palette groups.
3. Design-mode status: show each widget's query state (rows, loading, error)
   on the canvas.
4. Extend `checkPage`: validate operation param names (unknown params), rows
   paths against operation `fields`, contextual roots per setting.
5. ADR phase 2: generate a page from an operation; make fragments first-class.
