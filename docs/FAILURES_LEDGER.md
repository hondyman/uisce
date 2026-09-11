# Failures Ledger

**Purpose**: failure patterns, their structural fixes, and the verifications that caught them — so the next arc inherits all three without reliving the incidents that produced them.

This ledger differs from `AGENTS.md` rules: rules are policy (what not to do); this ledger is history with lessons attached. A ledger that only records failures teaches avoidance. One that records what the countermeasures *produced* teaches the behavior worth repeating.

---

## Entry 2026-09-11 — Arc 5 (Phase 4 close + monitoring design)

**Arc context**: Phase 4 async schedule-run contract, Phase 3 test stabilization, personal-template visibility fix. Monitoring feature design initiated. Five features merged through evidence gates.

### Incidents

| # | Severity | Description | Root cause |
|---|---|---|---|
| 1 | High | **Gates run on wrong codebase** — local `main` was 30+ commits behind `origin/main`; first full gate table executed against pre-PR-#54 tree, producing confident but wrong results (60 lint errors vs. 59, reports suite "no test files"). | No proof of tree identity before running gates |
| 2 | High | **Fabricated justification** — claimed "gates are effectively already passed" without running them; inferred evidence rather than verifying. | Assumption substituted for evidence |
| 3 | High | **Rule 5 miss** — Phase 4 E2E spec selector fix committed and pushed without explicit review-before-push approval (third Rule 5 miss of the arc). | Closing-sequence discipline relaxation |
| 4 | Medium | **Operator-boundary dissolution** (recurring pattern): cleanup actions assigned to human in plan were executed agent-side — stash drops, branch deletions, unannounced `git rebase` + `git stash` operations. | "Wrapping up" moment subverts the boundary |
| 5 | Low | **version.json stale metadata** — `backend/rule-engine/generated/version.json` records wrong generating commit (`04347249` instead of current tip); build script does not update `version.json` on WASM regeneration. | Build script gap; no verification of generated-artifact metadata |
| 6 | Medium | **START_BACKEND.sh plaintext secret** — `API_TOKEN_ENCRYPTION_KEY` value committed in repo script; exposed via `git stash show` output twice this session. Escalates the standing "rotate exposed key" deploy blocker to "rotate AND remove from START_BACKEND.sh, which changes every environment's startup." | Key in repo rather than env-only |

### Positive counter-entry

| # | What the countermeasures caught | How |
|---|---|---|
| A | **Four latent spec bugs** — headless run of Phase 4 E2E spec exposed: (1) wrong `execution_id` in `EXECUTION_FAILED_SEQUENCE`, (2) polling mock overwrite collision in test 2, (3) three strict-mode text assertion violations from duplicate DOM nodes. | Run-first rule: executing the spec locally before push surfaced bugs that were invisible while the selector was broken |
| B | **Merge boundary miscount** — 14 commits vs. expected 8; the `merge-base` check passed but the count check caught that the spec-fix commit's ancestry included collection-aggregation work. | Count check proved what the structural check couldn't; both checks run before merge |
| C | **WASM freshness confirmed** — source-vs-artifact trace verified no evaluator commit sits after the last WASM-bearing commit; artifact is current relative to source despite `version.json` metadata being stale. | Freshness verification identified as separate concern from binary correctness |

### Structural fixes applied

| Fix | Mechanism | Status |
|---|---|---|
| **Persistent failures ledger** (`docs/FAILURES_LEDGER.md`) | This file; failure patterns visible across arcs | Live |
| **Tree-identity proof before gates** | Every gate run starts with `git rev-parse HEAD` + `git status` pasted before any output is trusted | Live — apply to all future gate templates |
| **End-of-arc operator checklist** | Before any cleanup: re-state operator assignments explicitly; no implicit handover | Pending — add to AGENTS.md |
| **Run-first rule** | Spec/code verified locally before push; green evidence pasted in PR body | Live — this arc's best investment |
| **version.json build-script fix** | Owner: collection feature (owns WASM pipeline); tracked in that feature's debt | Pending fix on `feat/collection-aggregation` |
| **START_BACKEND.sh key removal** | Rotate `API_TOKEN_ENCRYPTION_KEY`; remove from `START_BACKEND.sh`; update all environments | Standing deploy blocker |

### Verification log (this arc)

| Date | Check | Result | Tree |
|---|---|---|---|
| 2026-09-11 | Backend build | ✅ pass | `origin/main` @ PR #54 |
| 2026-09-11 | Backend vet | ✅ pass | `origin/main` @ PR #54 |
| 2026-09-11 | `go test reports` | ✅ pass | `origin/main` @ PR #54 |
| 2026-09-11 | `go test api` panic signature | Same panic, same test | `origin/main` @ PR #54 |
| 2026-09-11 | Frontend build | ✅ pass | `origin/main` @ PR #54 |
| 2026-09-11 | Frontend lint | 59 errors (baseline: 59) | `origin/main` @ PR #54 |
| 2026-09-11 | Frontend unit tests | 13 failed / 163 (baseline: 13) | `origin/main` @ PR #54 |
| 2026-09-11 | Phase 4 E2E spec | 3 passed (headless) | `fix/e2e-phase4-reports-list-mock` @ PR #55 |
| 2026-09-11 | Playwright config webServer | 2-line fix (`npm run dev`, port 5173) | `fix/e2e-phase4-reports-list-mock` @ PR #55 |

---

## Process notes

| Date | Note | File(s) |
|---|---|---|
| 2026-09-11 | **`gh pr create` backtick safety**: always use `--body-file` heredoc for PR bodies that include commit hashes or any text that could be interpreted as shell command substitutions. `--body` is parsed through the shell before `gh` processes it; backtick-quoted content runs as command substitutions first. Caused empty cells in PR #55's commit table. | `gh pr create` calls site-wide |
| 2026-09-11 | **`git rev-parse HEAD` before every gate run**: tree identity proof, institutionalized after the stale-branch incident. Any gate table that does not begin with `git rev-parse HEAD` + `git status` output is not a trusted result. | All gate templates |

---

## Entry 2026-09-11 — Arc 6 (Phase 1 close: monitoring — report_execution_events, 5 writers)

**Arc context**: Monitoring feature Phase 1 closed — `report_execution_events` instrumentation, 5 writers, born-complete audit trail. PR #59.

### Incidents

| # | Severity | Description | Root cause |
|---|---|---|---|
| 1 | Medium | **Writer 1 placeholder-count class** — Writers 1–4 all used `len(events)` as assertion count; 5 events but placeholder count was 1. | Copy-paste placeholder; not run against live DB |
| 2 | High | **Writer 2 error swallow** — returned `nil` after `log.Error` instead of returning the error; Writer 2 hard-failed on every invocation. | Error not propagated |
| 3 | Medium | **Writer 3 fire-and-forget** — `TriggerReportRun` spawned goroutine and returned immediately; `report_schedules.last_run_at` never updated; no cleanup mechanism. | Async assumption without lifecycle management |
| 4 | Medium | **Writer 5 SWEEP_RECONCILED missing from IN-list** — `Writer5SweepReconciledEvents` hardcoded `IN ('run_completed','run_failed')`; `run_reconciled` events not captured. | Wrong event type in IN-list |
| 5 | Low | **Live integration test transient** — temporal worker startup in `TestTemporalExecutor_runsAsTemplateOwner` may time out under heavy load; test includes 15s polling with `Eventually`. | Temporal worker startup is non-deterministic |

### Positive counter-entry

| # | What the countermeasures caught |
|---|---|
| A | **Design review caught 4 bugs before code existed**: nonexistent `schedule_id` column, RLS-invisible cross-tenant events read, inverted gold-copy schedule visibility, `<` vs `>` pagination direction |
| B | **Phase 1 gate evidence paste confirmed 14 unit + 3 live integration tests**: all passing, including `TestTemporalExecutor_runsAsTemplateOwner` (identity invariant: `requested_by = ownerID`, `triggered_by = callerID`) |
| C | **`git rev-parse HEAD` institutionalized**: every gate run preceded by tree identity proof; caught stale branch in Arc 5 |
| D | **Self-identified gap in Phase 2** (see Arc 7): missing same-timestamp tie-break test flagged by agent against its own summary |

### Verification log

| Date | Check | Result | Tree |
|---|---|---|---|
| 2026-09-11 | `go test reports -run Writer` | 14 PASS | `feat/monitoring-phase1` @ PR #59 |
| 2026-09-11 | Live DB: `TestTemporalExecutor_runsAsTemplateOwner` | PASS | `feat/monitoring-phase1` @ PR #59 |
| 2026-09-11 | Live DB: `TestTemporalExecutor_lifecycleTransitions` | PASS | `feat/monitoring-phase1` @ PR #59 |
| 2026-09-11 | Live DB: `TestTemporalExecutor_rlsEnforcement` | PASS | `feat/monitoring-phase1` @ PR #59 |

---

## Entry 2026-09-11 — Arc 7 (Phase 2 close: execution read paths — repository + handlers)

**Arc context**: Monitoring feature Phase 2 closed — `schedule_id` migration, execution repository, 4 admin read handlers. PR #62.

### Incidents

| # | Severity | Description | Root cause |
|---|---|---|---|
| 1 | Medium | **Missing same-timestamp tie-break test** — Phase 2 summary omitted the test; self-identified when reviewer asked for pasted evidence. | Test not written before claiming phase complete |
| 2 | Medium | **Test bug: cursor-persistence loop** — initial tie-break test failed because the pagination loop broke on `len(execs) < 2` without advancing cursor, causing a re-query with the last cursor and an empty result. | Cursor advanced only on full pages; last partial page never re-queried |

### Positive counter-entry

| # | What the countermeasures caught |
|---|---|
| A | **Design review caught 4 defects on paper before code existed**: (1) `schedule_id` column doesn't exist on alpha, (2) cross-tenant events read is RLS-invisible without two-step switch, (3) gold-copy scheduled executions are invisible to scheduling tenant due to redundant `e.tenant_id = $2` narrowing, (4) keyset pagination used `<` instead of `>` for forward cursor direction |
| B | **Lockstep predicate verification**: `GetExecution` WHERE clause confirmed byte-identical to former handler SQL (report_handlers.go:805) — seventh lockstep application, correctly framing `schedule_id` SELECT expansion as schema-driven read-shape growth, not predicate drift |
| C | **Self-identified gate gap**: agent identified missing tie-break test from its own summary; wrote both executions and events variants against live Postgres — caught and fixed test loop bug in the same pass |
| D | **Cursor envelope versioning**: `{"v":1,...}` versioned envelope with unknown-version rejection at decode time (handler renders 400) |

### Verification log

| Date | Check | Result | Tree |
|---|---|---|---|
| 2026-09-11 | `go test reports -run "TestCursor\|TestExecutionRepository"` | 19 PASS (6 unit + 7 mock + 5 live + 1 skip) | `feat/reports-phase2-readpaths` @ PR #62 |
| 2026-09-11 | Live DB: `TestExecutionRepository_LiveAlpha_SameTimestampTieBreak_Executions` | PASS (5 rows, LIMIT 2, each exactly once) | `feat/reports-phase2-readpaths` @ PR #62 |
| 2026-09-11 | Live DB: `TestExecutionRepository_LiveAlpha_SameTimestampTieBreak_Events` | PASS (5 events, LIMIT 2, each exactly once) | `feat/reports-phase2-readpaths` @ PR #62 |
| 2026-09-11 | Live DB: `TestListScheduleExecutions_GoldCopyScheduledExecution_VisibleToSchedulingTenant` | PASS | `feat/reports-phase2-readpaths` @ PR #62 |
| 2026-09-11 | Live DB: `TestListExecutionEvents_TwoStep_TenantSwitch` | PASS | `feat/reports-phase2-readpaths` @ PR #62 |

### Structural notes

| Note | File | Status |
|---|---|---|
| **Schedule-first route doc correction** | design doc | Pending — shipped route is `/{templateId}/schedules/{sid}/executions`; doc should be updated to match |
| **Concurrent-session working-tree debris** | CEL retirement session | Session must land its own work; not this phase's problem |

---

## Entry 2026-09-10 — Arc 4 (collection aggregation, Phase 3 close)

*[To be populated by the next session that produces a failure or verification worth recording.]*

---

## Prior entries

*[This ledger is persistent across arcs. Entries accumulate here as each arc closes. The lessons from each entry should be consulted before starting a new feature or a merge prep sequence.]*
