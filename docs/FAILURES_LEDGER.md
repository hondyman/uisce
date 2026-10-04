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

## Entry 2026-09-12 — CI ephemeral database (`feat/ci-ephemeral-db`, PR #77)

**Arc context**: standing up a GitHub Actions job (`backend-gated-tests.yml`) to run the `UISCE_TEST_DB` integration suite against a fresh, ephemeral Postgres container on every PR — converting a laptop-gated discipline (mTLS certs against alpha) into an enforced CI gate.

### Incidents

| # | Severity | Description | Root cause |
|---|---|---|---|
| 1 | High | **`role "app_admin_read" does not exist`** — `schema-snapshot.sql` restore fails immediately; the workflow's hand-maintained role-stub list deliberately excluded this role, reasoning its own migration would create it later. | Genuine circular dependency, not ordering: the snapshot (alpha's already-migrated state) bakes in `app_admin_read`'s grants, which must resolve *before* restore; but that role's migration grants on tables the snapshot itself creates, so it can't run *before* restore either. Two claimed root causes were proposed and both were checked against files, not assumed: "ordering bug" (survived, refined) and "the migration-log snapshot already marks it applied" (falsified — the file's last row was `20260913_002`; both `20260915_001` and `20260916_001` were absent, i.e. pending, not skipped). |
| 2 | High | **Snapshot-pair desynchronization (class, not instance)** — `schema-snapshot.sql` and `migration-log-snapshot.sql` were captured from alpha at different moments. The schema snapshot already reflects `20260915_001` (schedule_id column) and `20260916_001` (app_admin_read role+grants); the log snapshot reflects neither. Left alone, `migrate up` in CI would try to re-run both: harmless no-op for `20260915_001` (`IF NOT EXISTS` throughout — verified against the actual migration file, correcting an initial "same failure shape, one step later" prediction that assumed a crash), but a hard `CREATE ROLE ... already exists` (42710) for `20260916_001`, which has no such guard. | Two files meant to describe one consistent point in alpha's history were generated by separate, un-paired `pg_dump` invocations. |
| 3 | Medium (design intent resolved; deploy gap open) | **`tenant_product` RLS policy is fail-open on alpha**, contradicting its own test's stated contract. `tenant_product_isolation_policy`'s `USING`/`WITH CHECK` clause is `(current_setting('uisce.current_tenant', true) IS NULL) OR (tenant_id = ...)` — when the tenant GUC is unset, the clause is unconditionally true: full read *and write* access to every tenant's rows. `internal/db/tenant_tx_test.go`'s `TestWithTenantTransaction` has a subtest literally named `no_tenant_GUC_returns_zero_rows`, logging `"PASS: ... (fail-closed)"` on success — asserting the opposite of the live policy's actual behavior. | **Not ambiguous design intent** — a peer session (task_904383c1) found `backend/migrations/20260727000030_strict_tenant_rls.sql` (commit `b9d00dc2c6`, 2026-07-27, confirmed ancestor of `fd3effc71` and confirmed against its actual content) already replaces this exact policy with a strict, fail-closed one via `uisce_get_current_tenant()` (returns NULL, never a bypass value, when the GUC is unset — `tenant_id = NULL` is never true). The test's name and assertion are correct; alpha simply never had this migration applied. **But "just apply it" is not simple**: verified this migration sits only in `backend/migrations/` — one of the 8 directories `MIGRATION_DIRECTORY_DRIFT_AUDIT.md` already documented as invisible to the runner, which reads only `backend/db/migrations/*.up.sql` — and it's written in Goose's two-way format (`-- +goose Up` / `-- +goose Down` markers, 6 of them, DOWN statements physically interleaved after each UP section). The current custom runner has no concept of `+goose Down` and would execute the whole 137-line file as one blob. **Ran it against a scratch Postgres to find out what that actually does, rather than reasoning it through**: it does not run to completion and "net-undo itself" table by table (an earlier, imprecise characterization of this same finding, corrected here) — it fails hard and immediately, at the very first `CREATE POLICY` (line 36, `tenant_instance`), with `ERROR: function uisce_get_current_tenant() does not exist`. The function was already dropped one statement earlier by section 1's own `-- +goose Down` line (`DROP FUNCTION IF EXISTS uisce_get_current_tenant();`), which executes immediately after the function is created, before any table's policy gets a chance to use it. Confirmed with `psql -1` (single-transaction, matching the runner's `BeginTx`/`tx.ExecContext`/`Commit` wrapping exactly): the whole transaction aborts and rolls back — the function does not exist afterward, proving nothing survives. So the real risk is not a silently-recorded false success; it's a hard, immediate `migrate up` failure. That has its own separate consequence worth noting for the port work: `ApplyMigrations` returns on the *first* file that errors, so a malformed migration here would also block every lexically-later `.up.sql` file in that same run, not just fail its own. Needs a careful port (extract only the Up statements into a proper `.up.sql`) before it can go through the existing pipeline at all — not a one-line "apply this file" fix. Filed as `task_904383c1`, corrected twice in-thread as each characterization of the failure mode was checked against actual execution rather than left as reasoning.|

### Positive counter-entry

| # | What the countermeasures caught | How |
|---|---|---|
| A | **Dead role stubs + one missing one, both from the same hand-list.** Deriving stub roles from `schema-snapshot.sql` itself (every `GRANT ... TO` / `OWNER TO` / `ALTER DEFAULT PRIVILEGES ... TO`) rather than hand-maintaining the list showed the old list stubbed three roles the snapshot never references at all (`infisical`, `keycloak`, `nessie`) while missing the one it needed (`app_admin_read`). A hand list can silently drift in both directions at once; a derived one can't drift at all without the dump itself changing. |
| B | **The vacuous-pass mechanism for `no_tenant_GUC_returns_zero_rows` is deterministic in this CI context, not merely theoretical.** Confirmed: the ephemeral job restores a schema-only snapshot (zero data) and — before this arc's fixes — the workflow's `UISCE_TEST_DB_DSN` connects as the `postgres` superuser (bypasses RLS entirely). Either fact alone would make this subtest pass without exercising the policy at all; both are true simultaneously in the current workflow, so every CI run of this suite is guaranteed to "PASS: fail-closed" a table that is, in fact, fail-open. |
| C | **Sweep for sibling fail-open policies needed no live alpha connection.** `schema-snapshot.sql` is a verbatim `pg_dump` of alpha's live `pg_policies`, so grepping it for the same clause shape *is* the sweep. Precise result (not the raw 11-match count a naive grep first returned): exactly **2 tables** carry the genuine bug shape — `public.tenant_product` and `public.tenant_product_datasource` — both keyed off the identical `uisce.current_tenant` GUC-absence check. The other 9 matches from the naive pattern are structurally different and not bugs: explicit `current_setting('app.is_admin', true) = 'true'` overrides (three-valued NULL logic means an *unset* admin flag does not satisfy `= 'true'`, so those fail closed correctly) and one policy checking a row's own `tenant_id IS NULL` (a legitimately global, non-tenant-scoped record type, not a session-GUC check). Also found: `tenant_product_datasource` carries a second, *strict* sibling policy (`tenant_isolation_policy`, no bypass) — but Postgres combines multiple permissive policies for the same command with OR, so the strict policy is fully neutralized by its fail-open sibling. Someone appears to have tried to tighten this table's policy and the attempt does nothing. |
| D | **A peer session's claim about the Goose-file failure mode was itself checked, not relayed.** A cross-session message asserted that naively renaming `20260727000030_strict_tenant_rls.sql` into the runner's path would run to completion, silently record itself as applied, and leave the fail-open policy in place unchanged (a "gate that ran, succeeded, and recorded a lie"). Ran it against a scratch Postgres instead of repeating that characterization: it errors hard and immediately at the first `CREATE POLICY` (line 36 of 137), because the function it depends on was already dropped one statement earlier by section 1's own `-- +goose Down` line — the whole transaction aborts, nothing is recorded, nothing survives. The actual risk is a hard failure that blocks every later migration in the same `migrate up` run, not a silent lie. The correction chain now runs three deep on this single sub-finding (fifo-vs-coproc; migration-log-snapshot claim; and now this), each resolved by running the thing rather than reasoning about it one more time. |

### Structural fixes applied

| Fix | Mechanism | Status |
|---|---|---|
| **Derived stub-role list** | `backend-gated-tests.yml`'s role-creation step parses `schema-snapshot.sql` for referenced roles instead of hand-listing them | Live (commit `843f04d4a` on `fix/ci-ephemeral-db-role-scope`, not yet merged) |
| **`app_admin_read` created for real, pre-restore** | Matches its migration's real attributes (`LOGIN` + `BYPASSRLS`); `ADMIN_READ_DSN` wiring explicitly scoped out with a stated reason, not silently skipped | Live, same commit |
| **Two migration-log-snapshot.sql rows hand-patched** | `20260915_001`/`20260916_001` recorded as already-reflected, checksums verified byte-for-byte against on-disk files | Live, same commit — **interim only** |
| **`backend/db/snapshots/regenerate.sh`** | Atomic schema+log snapshot pair via `pg_export_snapshot()` + `pg_dump --snapshot=<id>` from one held-open connection, so the two files can't desync again | Live (commit `5ae279340`), needs a run against alpha to actually refresh the checked-in snapshots — not done yet (no alpha access from the environment that wrote it) |
| **`tenant_product`/`tenant_product_datasource` fail-open policy** | Design already decided on main (`20260727000030_strict_tenant_rls.sql`, fail-closed via `uisce_get_current_tenant()`) — test is correct as written, no rename needed. What's actually needed: port that migration's Up-only statements out of Goose format into a proper `backend/db/migrations/*.up.sql` file the runner can see, then apply to alpha. | **Not fixed** — deploy/porting gap, tracked as `task_904383c1`; not a live-alpha write anyone here has made |

### Verification log (this arc)

| Date | Check | Result | Tree |
|---|---|---|---|
| 2026-09-12 | Live grants causing the restore failure | 4 `GRANT ... TO app_admin_read` statements found at `schema-snapshot.sql:108426,113830,113840,113923` | `origin/feat/ci-ephemeral-db` @ `fd3effc71` |
| 2026-09-12 | Migration-log-snapshot.sql contents | Confirmed `20260915_001`/`20260916_001` both absent (last row `20260913_002`) | same tree |
| 2026-09-12 | Derived-role pipeline vs. real file | Produces exactly `{authenticated, semlayer_lookups_replica, temporal, usice_app, usice_ops}` | same tree, scratch local Postgres 16 |
| 2026-09-12 | Four previously-failing GRANTs | Succeed against the derived+real role set | scratch local Postgres 16 |
| 2026-09-12 | 42710 collision this fix avoids | Reproduced directly (`CREATE ROLE app_admin_read` against an already-created one) | scratch local Postgres 16 |
| 2026-09-12 | sha256 checksums for the two backfilled log rows | Verified twice, independently, against on-disk migration files | `origin/feat/ci-ephemeral-db` @ `fd3effc71` |
| 2026-09-12 | `regenerate.sh` first draft (fifo-based) | **Hung, killed** — not shipped | scratch local Postgres 16 |
| 2026-09-12 | `regenerate.sh` rewrite (`coproc`) | Succeeds end-to-end; seeded table and migration_log row both appear correctly in their respective outputs | scratch local Postgres 16, bash 5.3 (bash 3.2/macOS default lacks `coproc`) |
| 2026-09-12 | Branch base triple-check | Main checkout HEAD, `origin/feat/ci-ephemeral-db`, and fix branch's merge-base with origin all `fd3effc71` | — |
| 2026-09-12 | tenant_product/tenant_product_datasource sweep | 2 tables confirmed via grep against `schema-snapshot.sql`'s live `CREATE POLICY` statements; no live alpha connection needed or available from this environment | same tree |
| 2026-09-12 | `b9d00dc2c6` ancestry (peer-relayed claim) | `git merge-base --is-ancestor` confirmed true; migration content confirmed to do what was claimed | — |
| 2026-09-12 | Goose-file failure mode (peer-relayed claim, then this session's own initial characterization of it) | Neither "runs cleanly, recorded as applied" nor "table-by-table net-undo" — `psql -1` against a scratch Postgres shows a hard error at the first `CREATE POLICY` (line 36/137) and a full transaction rollback | scratch local Postgres 16 |

---

## Entry 2026-09-10 — Arc 4 (collection aggregation, Phase 3 close)

*[To be populated by the next session that produces a failure or verification worth recording.]*

---

## Prior entries

*[This ledger is persistent across arcs. Entries accumulate here as each arc closes. The lessons from each entry should be consulted before starting a new feature or a merge prep sequence.]*

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

## Entry 2026-10-03 — CI base-rate adjudication (PR #347, C1 / 9.1)

**Arc context**: the C1 metric-primitives port was blocked on CI. Two jobs were
red, and both were diagnosed as defects in the port before either was shown to be
a pre-existing condition on `main`. Merged as #347 (`353c0b0bf`).

### Incidents

| # | Severity | Description | Root cause |
|---|---|---|---|
| 1 | High | **Branch failure attributed to the branch without a base rate.** `Build Frontend` failed 3/3 on the branch while `main` was sampled once — and green. Called a flake, and "8/8 pass locally" was reported as settling it. `main` was in fact red **4 of its last 8** runs with a byte-identical signature, including main's current tip. | Comparing a branch against a single reference run instead of the reference's own history over the same window |
| 2 | High | **The reproduction run did not match CI's conditions.** CI runs `pnpm test -- --coverage` on the full suite; the local attempt ran one test file with no coverage, and was reported as evidence about CI. | Reproduction conditions chosen for convenience rather than read off the workflow step |
| 3 | Medium | **A failure-detection helper silently reported success.** The CI poll loop counted failures with `grep -cE '^\S+\s+fail'`, which cannot match a check name containing a space. It returned `fail=0` while `Build Frontend` and `Build Backend` were both red — and that loop was the thing being trusted to decide whether it was safe to merge. | A tool that returns zero because it found nothing is indistinguishable from one that returns zero because it cannot see |
| 4 | Medium | **A green run was accepted without reading its counts.** The totals are what distinguish a real pass from a suite that dropped tests: `114 passed (114)` versus `113 passed (114)`. | Test totals treated as boilerplate; only the exit code read |

### Positive counter-entry

| # | What the countermeasures caught |
|---|---|
| A | **The real defect — which no branch could have caused.** `structuredEditors.test.tsx` rendered 206 widgets in one uninterrupted block. Measured with a 25ms `setInterval` sampler: **zero** timer callbacks across 20,706ms, so the vitest worker could not answer the main thread's `onTaskUpdate` RPC. Under CI's `--coverage` that tipped into `[vitest-worker]: Timeout calling "onTaskUpdate"`, failing the suite with every assertion green — `Test Files 113 passed (114)`, `Tests 662 passed (668)` — and six tests silently unreported. CI blames whichever file ran last, which is how it first read as an assertion failure in an unrelated test. One `await setTimeout(0)` per widget fixed the mechanism: 150 callbacks served, worst block 802ms, +164ms total. |
| B | **Main's own history supplied the verdict.** Tabulating `Build Frontend` across main's last 8 runs, then reading the log of main's *current tip* failing with the same signature, turned "probably my branch" into "pre-existing, 4 of 8" — the single fact that ended the misattribution. |
| C | **A metric that measured nothing, caught by reading it.** The first drift sampler reported `0ms` for the fully-blocked case, because a `setInterval` that never fires records zero drift. Counting *ticks* rather than measuring gaps is what made the starvation visible at all. |
| D | **Golden-artifact mutation check caught a no-op mutation.** Removing an `asl:ignore` marker initially changed nothing, because the replacement comment still contained the literal string `asl:ignore`. Reading the diff instead of trusting "I made the edit" forced a correct second mutation, which reintroduced the type and failed both golden tests. |
| E | **The 8.3 corpus, for the third consecutive use**, caught a real regression (dropped ADR-026 `NumeratorID`/`DenominatorID`) rather than confirming the port. |

### Verification log

| Date | Check | Result | Tree |
|---|---|---|---|
| 2026-10-03 | `go run ./cmd/check-drift` | PASS — schema/types/monaco up to date | PR #347 @ `e77962977` |
| 2026-10-03 | `go test ./rule-engine/... ./internal/rules/vm/... ./internal/querybuilder/...` | PASS (generate-types, generate-schema, generate-monaco, generate-version, vm, querybuilder incl. 8.3 corpus) | PR #347 @ `e77962977` |
| 2026-10-03 | `asl:ignore` mutation (marker removed from one type) | Type reappears in all three artifacts; both golden tests FAIL | PR #347 @ `e77962977` |
| 2026-10-03 | Frontend full suite, pre-fix | 114/114 files, 668/668 tests, exit 0 | PR #347 @ `e77962977` |
| 2026-10-03 | CI `Build Frontend` on the fix | `114 passed (114)`, `668 passed (668)`, **no Errors line** | PR #347 @ `e77962977` |
| 2026-10-03 | CI, all checks | 22 pass, 4 skipping, 0 red; `mergeStateStatus: CLEAN` | PR #347 @ `e77962977` |

### Structural notes

| Note | File | Status |
|---|---|---|
| **Measure the metric the system optimises, not the one that is easy to compute — and a proxy is how a green-looking failure hides.** This is the fourth instance of a family, and they belong together because each presented as a different bug. (1) An event-loop sampler that could not fire when the loop was blocked, so it reported `0ms` drift for total starvation. (2) A CI poll loop whose regex could not match a job name containing a space, reporting `fail=0` on two red checks. (3) A mutation-verification run that stacked two mutations, whose failures read as a real regression. (4) A CI latency analysis that judged the critical path by **longest single job** when the pipeline's critical path is the longest **dependency chain** — the easy metric was off by 9.8–22.7m against actual wall-clock, and it *inverted* the conclusion: a full 8–11m security scan looked like a one-in-three blocker when it sat on the critical path in **every** run. The general rule: **every measurement must model the system's actual structure. The easy metric is usually a proxy, and proxies are how a green-looking result hides a red one.** | this ledger | Live |
| **Never optimize a check away on duration alone — and record both framings when the number is misleading.** The C2 PII-gate work cost two full CI cycles of attention, and `Check ASL Schema/WASM Drift` measures **0.1m** of a 13.4m job, which invites exactly the wrong conclusion. Both framings are true and both belong on the record: the drift check is cheap, *and* it is the thing that caught an ASL widening which three generators would otherwise have shipped inconsistently. Six seconds protecting semantic contract consistency across generated artifacts is the best time-per-protection ratio in the pipeline. The ledger records the 7.6m test suite as the cost and the 0.1m check as the bargain, so a future reader optimizing by wall-clock sees both. | this ledger | Live |
| **Base-rate rule** — before attributing an intermittent CI failure to a change, tabulate that job's conclusion across the default branch's recent runs. One green reference run is not a base rate. | this ledger | Live |
| **Poll-loop self-check** — any helper that counts failures must be shown to count a failure. Split on TAB and compare a field; never regex a whole line whose job names may contain spaces. Retained alongside the measurement-trap family: that entry records the same CI poll loop as instance (2), but as an *incident* — the operational remedy belongs to its own rule. | this ledger | Live |
| **CI latency: verified — the security scan now costs zero wall-clock.** `security-scan` declared `needs: [build-backend, build-frontend]` while consuming no artifact from either, so a full 8–11m scan was serialized behind 10–13m of builds and added to every PR. Removing the `needs:` (#357) was measured, not assumed, on the PR's own runs. Run 37154691858: Security Scan starts +0.1m and ends +10.3m; Build Backend ends +13.4m; the **run ends at 13.4m — exactly when the build finished, with the scan already done 3.1m earlier.** The old serialized chain predicts ~23.7m, so the saving is **10.3m, precisely the scan's own duration**. A second run (37155016045) came in at 17.0m, but its Build Backend did not start until +4.0m from runner queueing on the second push; the scan still ended at +10.3m and the run still ended when the build did. The critical path is now `Build Backend` alone, and this is why the three-tier security redesign was **declined** rather than built. | this ledger | Live |
| **Zero-found vs couldn't-look** — *any tool that reports a count must distinguish "found none" from "could not look."* This is now the fourth instance of one pattern: a `coverageManifest` counter that compared a constant to itself, an EXPLAIN-plan parser that returned an empty plan instead of an error, cache-key helpers that hashed nothing, and a CI poll loop that counted failures with a regex that could not match job names containing spaces. A helper that cannot fail loudly is worse than no helper, because its zero is indistinguishable from a real zero. | this ledger | Live |
| **A test passes vacuously if its fixture never reaches the condition under test** — verify the fixture *traverses* the predicate, not merely that the assertion holds. The C2 PII-gate equivalence case was built from an **untagged** term; an untagged field returns before the tier check is ever consulted, so the test passed even against a gate that treated "tagged" as "forbidden". It asserted its own existence rather than the behaviour it was named for. This is the fourth variant of "a test proves only what it executes", and the first about **fixture design** rather than execution. The fix was to build the case from a *tagged* term at passthrough tier and assert both directions. | this ledger | Live |
| **One mutation per run, and a marker check before reading any result** — mutation evidence is worthless if the analyst's own process contaminates it. Two mutations applied at once produced failures I misread as a real regression; a no-op mutation produced a "pass" that looked like the guard not biting. Both were caught only by grepping for the marker before interpreting output. Pair with the base-rate rule: **both are ways the analyst's own tooling manufactures the evidence it is then read against.** | this ledger | Live |
| **All-tests-passed means infrastructure** — a red check whose summary shows every test passing is a worker/RPC/timeout/OOM fault. Read the `Errors`/`Unhandled` section, not the totals. | this ledger | Live |
| **Silent result suppression is the worst failure class a test runner has** — the `onTaskUpdate` timeout failed the suite with `Test Files 113 passed (114)` and `Tests 662 passed (668)`: green-looking output hiding six tests that never ran. This is materially worse than a flaky timeout, because a missing test is invisible: there is no red to investigate, and the coverage it provided is simply absent. Any "all tests passed" conclusion should be read as "all *reported* tests passed" until the file and test totals are checked against what should have run. | this ledger | Live |
| **archguard: CI-status helper self-test** | `backend/internal/archguard` | Queued, with a caveat found while filing it: the helper that returned the false zero was a **throwaway polling loop, not committed code** — `scripts/*.sh` contain no CI-status parser. A guard that scans the repo for the bad pattern would therefore have no target and would be decorative. The constructive form is a single committed, tested status parser that future work calls instead of re-typing a regex into an ad-hoc loop. Decide which before implementing. |

---

## Entry 2026-10-03 — Metric lineage writer: schema unverifiable from the repository

Operational entry, not a CI failure. Full analysis and decision in
`docs/ARCHITECTURAL_DECISIONS.md` → *Metric catalog lineage: the writer targets
a schema that does not exist*.

**Record the answers here, not in chat.** The session's output goes in
`docs/alpha-audit-2026-10.md` — raw query output first, conclusions separately,
with every answer marked `verified-live` or `couldn't-look`. The finding is
that the dump lied about the database, so the database's real state is not
recoverable from the repository and has to be captured as a first-class
artifact. That file is written to be executable by anyone holding the mTLS
certs, with no chat history.

### The situation

`SyncMetricToCatalogGraph` has no production caller **and** its SQL references
`catalog_node.node_id`, `catalog_node.node_key` and a text `catalog_edge.edge_type`
that the authoritative schema snapshot does not contain. The port that the
wiring decision depends on cannot be completed from the repository alone. These
queries settle it. All are read-only.

### Execution order — ruling 2026-10-03

Run in this order, and stop if the first one dissolves the finding:

| Phase | Query | Purpose |
|---|---|---|
| 1 | **Q5** | Invalidation check. Can dissolve the entire finding. |
| 2 | **Q1 + Q2 together** | Settle the mismatch claim against live reality, not the dump. |
| 3 | Q3 + Q4 | Input to the *fix*, not the diagnosis. |

Q5 is the one answer that would invalidate the central claim: if lineage edges
exist with a non-zero count, some path writes them and the caller search missed
it. It is also the cheapest to interpret. Q3 and Q4 matter only once the finding
survives — they determine what a corrected writer needs (edge type IDs,
partition bounds), so they are remediation input, not diagnosis.

```sql
-- ===== PHASE 1 — run first; can invalidate the finding =====

-- Q5. Has any metric lineage edge ever actually been written?
SELECT edge_type, count(*) FROM public.catalog_edge
 WHERE upper(edge_type) IN ('METRIC_OF','USES_TERM','DERIVED_FROM')
 GROUP BY 1;

-- ===== PHASE 2 — settles the mismatch against live reality =====

-- Q1. Does catalog_node actually lack node_id / node_key on the live DB?
SELECT column_name, data_type, is_nullable, column_default
  FROM information_schema.columns
 WHERE table_schema = 'public' AND table_name = 'catalog_node'
 ORDER BY ordinal_position;

-- Q2. Same for catalog_edge. Also shows whether it is still partitioned.
SELECT column_name, data_type, is_nullable, column_default
  FROM information_schema.columns
 WHERE table_schema = 'public' AND table_name = 'catalog_edge'
 ORDER BY ordinal_position;

-- ===== PHASE 3 — input to the fix =====

-- Q3. Do METRIC_OF / USES_TERM / DERIVED_FROM exist as edge types?
--     The snapshot carries no row data, so their absence from it proved
--     nothing; this is the load-bearing unknown for the port.
SELECT id, edge_type_name, is_active
  FROM public.catalog_edge_types
 WHERE upper(edge_type_name) IN ('METRIC_OF','USES_TERM','DERIVED_FROM');

-- Q4. Which partitions of catalog_edge exist, and is there a DEFAULT one?
SELECT c.relname, pg_get_expr(c.relpartbound, c.oid) AS bounds
  FROM pg_inherits i
  JOIN pg_class c ON c.oid = i.inhrelid
  JOIN pg_class p ON p.oid = i.inhparent
 WHERE p.relname = 'catalog_edge'
 ORDER BY c.relname;
```

### What each answer implies

| Answer | Consequence |
|---|---|
| **Q5 non-zero** | Some path writes these edges today and the caller search missed it. **Stop.** Re-derive the whole finding — reachability, not schema, is then the open question. |
| Q1/Q2 return the snapshot's columns | The writer must be ported: `id`, `node_type_id`, `edge_type_id`. Proceed with the port; the remaining unknown is vocabulary. |
| Q1/Q2 return `node_id` / `node_key` | The snapshot is **stale**. Stop trusting it for these two tables, and re-run `backend/db/snapshots/regenerate.sh` — the dump and the migration log are one artifact pair. |
| Q3 returns three rows | The port is a rename. Book the three `edge_type_id` UUIDs. |
| Q3 returns nothing | The port must also **create** the vocabulary, and must decide the edge-type semantics first. This is a schema change, not a code change. |
| Q4 shows a partition covering today (or a DEFAULT) | Insertion is possible. Otherwise an insert dated in the current quarter fails with *"no partition of relation catalog_edge found for row"* — the port needs a partition-creation step first. |

### Scope amendment — the four-file sweep (ruling 2026-10-03)

Folded into the same alpha session, same root cause. Four other non-test
production files reference the pre-glossary generation. **Correction to the
original report:** two of them were understated — each writes *both* tables, not
just `catalog_node`. Citations verified against `11b114167`.

| # | File | What it uses |
|---|---|---|
| 1 | `semanticmatch/resolver.go:75,77,80` | `catalog_node (tenant_id, node_id, node_type, node_key, …)`, `ON CONFLICT (tenant_id, node_key)`, `RETURNING node_id` |
| 2 | `catalog/sti_column_scanner.go:56,70` **and `:80,83`** | `catalog_node (node_id, …)` **and** `catalog_edge (…, edge_type)` text — `COLUMN_OF` |
| 3 | `catalog/subtype_bo_builder.go:36,50` **and `:60,63`** | `catalog_node (node_id, …)` **and** `catalog_edge (…, edge_type)` text — `ATTRIBUTE_OF` |
| 4 | `bo/layout_service.go:104,108,110,112,114,116` | reads `tax.node_key`; joins on `st.node_id`, `e_bt.from_node_id`, `e_bt.edge_type IN ('DEFINED_BY','DESCRIBES')`, `e_bt.to_node_id`, `e_tax.from_node_id`, `e_tax.edge_type = 'MEMBER_OF'`, `tax.node_id` |

`resolver.go:18-20` carries its own comment — *"node_id TEXT PK (if yours is
bigserial, drop node_id from the INSERT and keep RETURNING node_id)"* and
*"UNIQUE (tenant_id, node_key) (if absent, swap to SELECT-then-INSERT)"* — so
that file is schema-adaptive debt with adaptation instructions left in place,
not a plain oversight. That should weight its disposition.

The sweep must answer **two** questions per file, not one. Schema match alone
is not enough — a file can reference a missing column and never be called, so
liveness has to be established separately rather than assumed.

Each cell of the 2×2 has a different disposition:

| | reachable | unreachable |
|---|---|---|
| **schema-mismatched** | **live defect** — fix or delete; each needs its own decision | dead code targeting a dead schema — **delete** |
| **schema-matched** | healthy — no action | dead code — delete on general principle, or note and leave |

Two of the four cells end in deletion, which is how the
`StarRocksMaterializationManager` and duplicate-suite findings resolved.
Pre-glossary code surviving a schema migration *unreached* is how dormancy debt
accumulates. `archguard` now catches recreation of a classified opener, so
deletion does not need a replacement guard for that file class.

### Same session — cube DDL validation

The alpha session is already open for the cube stream's staging DDL validation.
Fold it in rather than spending a second session. The cube DDL generator faces
the same risk class: generated SQL validated by unit tests against a schema
dump. Check its targets against Q1/Q2's answers while there.

### Structural notes

| Note | File | Status |
|---|---|---|
| **Reachability check before building** — a code path whose only callers are in `_test.go` files has never run in production. Search the whole repo, not the package, and state *looked-and-found-nothing* vs *couldn't-look* explicitly. The PII gate caught this for `CubeDDLGenerator`; the same check was owed to the #359 lineage fix and was missed. | this ledger | Live |
| **Reachability check before *extending*** — the rule above is not only for new work. Run it before adding a function to a path you did not create, and before building on one. It has now caught one false premise; that makes it a named pre-build step, not a habit. | this ledger | Live |
| **A schema dump proves shape, not data** — `schema-snapshot.sql` carries no row data, so an entity's absence from it is *couldn't-look*, never proof of absence. Pair the dump with the migration log, and regenerate both together. | this ledger | Live |
| **Main red for N consecutive runs is its own finding** — when `main` fails 12 of its last 12 runs, every PR's CI signal is noise until main is fixed. Alert on the default branch independently of any PR context; do not let it surface as a surprise on someone's branch. Observed 2026-10-03: #355's `TestEveryDatabaseOpenerIsClassified` was tripped by #358 and fixed by #360, and nothing flagged the 12-run window. | this ledger | Live |
| **Verify merged content by commit-list, never by PR diffstat** — a PR's diffstat is computed against its *original* base. After a rebase onto a moving `main`, the stat includes every upstream commit that came along with it, so a two-file docs PR reports the base PR's files too. `gh pr merge` printing `10 files changed, 755 insertions(+)` for a docs-only PR looked like a corrupted merge. The decisive check is `git log <verified-base>..origin/main`, which showed exactly three entries: two doc commits and the merge. A diffstat describes the original review surface, not the merge content. | this ledger | Live |
| **A doc comment asserting a safety property is a claim, not a guarantee** — `metric_reconciler.go:22` promises advisory locking that does not exist. Read the code, not the comment above it. | `backend/internal/querybuilder/metric_reconciler.go` | Open |
| **A concurrency test that runs one goroutine cannot test concurrency** — `TestMetricCatalogReconciler_IdempotentRun` passes against code with a cross-replica race. Naming a test `Idempotent` does not make it a race test. | `backend/internal/querybuilder/starrocks_mv_and_ridealongs_test.go` | Open |

---

## Entry 2026-10-03 — `docker-build` has never built an image; main red since the initial commit

### The finding

`.github/workflows/ci-cd.yml` `docker-build` builds
`./backend/Dockerfile.${{ matrix.service }}`. **Not one of the five matrix
entries resolves:**

| Matrix entry | Workflow looks for | Actually exists |
|---|---|---|
| `api-gateway` | `backend/Dockerfile.api-gateway` | nothing — no Dockerfile, no service directory, anywhere in the repo |
| `semantic-engine` | `backend/Dockerfile.semantic-engine` | `docker-builders/Dockerfile.semantic-engine` |
| `governance-engine` | `backend/Dockerfile.governance-engine` | `docker-builders/Dockerfile.governance` — wrong dir *and* wrong name |
| `ai-builder` | `backend/Dockerfile.ai-builder` | `docker-builders/Dockerfile.ai-builder` |
| `compliance-engine` | `backend/Dockerfile.compliance-engine` | `docker-builders/Dockerfile.compliance-engine` |

`backend/` holds 14 Dockerfiles — `audit-worker`, `catalog-sync`,
`catalog-worker`, `cdc`, `event-router`, `kafka-connect-iceberg`, `loader`,
`notifications`, `outbox-processor`, `policy`, `search`, `snapshot-worker`,
`sync-worker` — and **none of them are in the matrix**. The matrix has been
wrong since `b9fc5e129`, the initial commit.

Observed error: `failed to read dockerfile: open Dockerfile.ai-builder: no
such file or directory`.

**Only one of the five jobs reports `failure`.** The other four report
`cancelled`, because the matrix aborted on the first leg. Reading job
conclusions without distinguishing `failure` from `cancelled` yields "1 of 5
broken", which is wrong; the honest summary is *none of the five has ever
built*.

**Why nobody noticed:** the job is gated `github.event_name != 'pull_request'`,
so it only ever ran on main pushes. It never appeared in any PR's checks and
never blocked a merge — it only guaranteed main stayed red.

### Decision: quarantined, not deleted

The job is retained so the intent stays auditable, gated on
`vars.DOCKER_BUILD_ENABLED == 'true'`. Unset means skipped, and a skipped job
does not fail the run. Re-enable by setting that repository variable.

A repository variable rather than `workflow_dispatch`, because the workflow has
no `workflow_dispatch` trigger: gating on one would make the job permanently
unreachable while the comment claimed it could be run by hand, and adding the
trigger would widen the manual-run surface of the entire pipeline. The variable
is reversible from repo settings without a code change.

This job carries `push: true` and `packages: write`. Running it is a
**registry-publishing decision, not a path fix** — four services that have
never been built would begin pushing to GHCR on every main push. That is why
the paths were not simply corrected: the matrix is also factually wrong about
what exists (`api-gateway` has no Dockerfile and no service), so what *should*
be published is an open question, not a path edit.

### Correction: "main is green" was wrong, reported twice

I stated twice that the 12-of-12 red window had ended and that main was green
at `8bb7b60be`. **It was not.** I verified by reading one job —
`Build Backend: completed/success` — and generalised that to the whole run. The
run's actual conclusion was `failure`, from this job.

#360 (`8bb7b60be`) fixed a real and separate defect: `TestEveryDatabaseOpenerIsClassified`,
tripped by #358 and caught by #355's guard. That fixed the Go test. It did not
touch `docker-build`. I read the fix as the end of the window because the check
I sampled had gone green, and reported it without ever reading the run's
conclusion.

The rule this adds: **a run's health is the run's conclusion, not any sample of
its jobs.** `gh run view <id> --json jobs -q '.jobs[] | select(.conclusion=="failure") | .name'`
lists what failed; it does not tell you the run passed. Read the conclusion, and
distinguish `failure` from `cancelled` when counting matrix legs.

This is the second time the base-rate rule's family caught me in one session
(after the sampler that could not fire, the longest-job-vs-chain inversion, the
diffstat-vs-commit-list trap). The pattern is consistent: **I substitute a
cheaper proxy for the actual measurement and report it as the measurement.**

### Structural notes

| Note | File | Status |
|---|---|---|
| **A run's health is its conclusion, not a sample of its jobs** — sampling one job that passed and reporting the run green is how "main is green" was reported twice while main was red. Read the conclusion; and when counting matrix legs, `cancelled` is not `passed`. | this ledger | Live |
| **A CI job gated off PRs cannot be caught by PR CI** — `docker-build` has been broken since the initial commit and appeared in no PR's checks, because it only runs on main. A permanently-red main check is invisible to everyone reviewing pull requests. | this ledger | Live |
| **Disjoint diffs are not disjoint merge outcomes** — four PRs edited `archguard/tenant_connection_inventory_test.go` on disjoint lines, which I verified before merging them in sequence. One of them had merged `main` *back into its feature branch* (`f48b00255 Merge branch 'main' into chore/remove-dead-openers`) rather than being rebased, and the conflict resolution **resurrected an inventory line that #370 had already deleted**. #370's file deletion landed; its inventory deletion did not. `TestInventoryEntriesAreHonest` then failed for three consecutive merges (`665429514`, `7b9708b25`, `9984af09f`) and was introduced by me. **Check the merged result, not the diffs** — after merging, run the shared-state test on `main` itself, because a per-PR green was computed against a base that no longer exists. | this ledger | Live |

---


## Entry 2026-10-03 — Hybrid Stack CI removed: a weaker duplicate of a gate that already exists

Deleted `.github/workflows/hybrid_ci.yml`. It ran on both `push` and `pull_request` and did two
things:

1. `go test -short ./backend/...` — the **entire hermetic backend suite**, unsharded.
2. `Dry-run Semantic Diff` — a placeholder that runs `echo` and has its real command commented out.

### Why it could go

The test run is a **strictly weaker duplicate** of the tier in `ci-cd.yml`, which runs the same
packages with `-race`, coverage, module caching, and — as of #372 — sharded four ways:

| | `hybrid_ci.yml` | `ci-cd.yml` |
|---|---|---|
| command | `go test -short ./backend/...` | `go test -short -race -coverprofile=...` |
| concurrency | one job | 4 shards, partition verified as a bijection |
| module cache | none (hardcoded `go-version: '1.25'`) | `setup-go` `cache: true` |
| coverage | none | per-shard, merged under one codecov flag |

Nothing is lost by removing it. This is the same decision as #356, which deleted
`datasource-rename-backend.yml` for running the same Go suite as `Build Backend`: a duplicate of
a gate that already exists, kept alive by nobody relying on it.

The second step never tested anything. It has been a no-op that prints a sentence.

### Why this was hiding

**`hybrid_ci.yml` is the workflow that caught my own regression.** The stale archguard inventory
entry from #368 failed *here* first, not in `CI/CD Pipeline` — and that is how I found it while
#372's CI was red for an unrelated-looking reason. A duplicate gate is not only wasted compute:
it is a second opinion that disagrees with the first, and when the two are the same check at
different strengths, the weaker one is the one that reports.

### The measurement it was distorting

`go test ./...` over 498 packages was measured at 7.6m. Sharding `ci-cd.yml` removed one copy of
it and I reported the critical path as "~6m" — while this workflow was still paying the other
copy on every push and every pull request. The sharding was real; the claim built on it was not
yet earned. Both copies are now a single sharded matrix.

Verified on `main` at `99d5fe4d8` (run `37167722917`), the first run with shards in place:

| Job | Duration | Finished at |
|---|---|---|
| `Build Backend` (no longer runs tests) | 5.2m | +5.2m |
| `Backend Tests (3/4)` | 4.2m | +4.3m |
| `Build Frontend` | 5.7m | +6.1m |
| `Backend Tests (4/4)` | 5.1m | +6.7m |
| `Backend Tests (1/4)` | 5.4m | +7.0m |
| `Backend Tests (2/4)` | 5.4m | +7.2m |

**Critical path 13.4m → 7.2m.** The prediction was ~6m. The 1.2m gap is runner queueing for four
concurrent jobs: shard 3/4 started at +0.1m but shard 2/4 did not start until +1.8m, so the
sharding is staggered by scheduling, not by test time. The honest number is the measured one, and
it is recorded next to the prediction so neither can be quoted alone.

### Structural notes

| Note | File | Status |
|---|---|---|
| **A duplicate gate is not free** — two workflows running the same suite at different strengths double the compute and produce two answers; the weaker one reports first and can look like a different failure. Find and reconcile every copy before quoting a saving. | this ledger | Live |
| **A speedup is not a speedup until every copy is measured** — sharding one workflow while a second ran the same suite unsharded made a 7.6m saving look complete when it was half. Measure the workflow that actually costs, not the one you edited. | this ledger | Live |

---

## Entry 2026-10-03 — Path filters in `ci-cd.yml`; and how nearly three of them shipped wrong

Gates `build-backend`, `backend-tests` and `build-frontend` behind a `changes` job. A
pull request that touches only `docs/` no longer runs the four-shard backend matrix, which
finished at +7.2m on `99d5fe4d8`. Pushes to `main` emit both flags true and run everything, so
`main` stays fully verified. `security-scan` is **not** gated: since #357 removed its `needs:` it
costs ~0 wall-clock, and filtering it would trade the repo's security posture for compute that is
not the bottleneck.

### A skipped gate is a silent gate

This is the risk that made the filters conservative. A path filter that forgets an input does not
fail — it stops running, and reports green. So the backend filter carries `libs/`, `cmd/`,
`migrations/`, `go.work`, `go.work.sum` and the workflow file itself, not just `backend/`.
`go.work` at the repo root is the one that matters: it is what makes `libs/` and `cmd/` part of a
backend build, and a filter reading only `backend/` would have skipped the suite for a PR that
changed a shared library.

**A filter that reduces coverage is not an optimisation, it is a deletion with a green tick.** If
one of these paths is ever wrong, the fix is to widen the filter, never to trust the result.

### The tooling failure worth recording

This change was attempted three times with a script and the first two produced files that
**parsed as valid YAML and were still wrong**:

1. A regex over job blocks matched only the job's first line, so the `if:` conditions were applied
   but `needs: [changes]` silently was not. The workflow would have gated on an expression reading
   outputs of a job it was never given — permanently false, or empty, depending on the evaluator.
2. Block-end detection excluded comment lines from terminating a job, so blocks ran past the job
   boundary and the `if:` lines landed *after the next job's banner comment* — attached to the
   wrong job entirely.

Both were caught by parsing the result and printing each job's resolved `needs` and `if` **rather
than by reading the diff**. The third attempt used exact-string edits anchored on
`  build-backend:\n    name: Build Backend\n    runs-on: ubuntu-latest` and was verified to be a
**pure addition — 72 lines added, zero removed**, so no existing line could have been mangled.

**A partially-applied edit that still parses is the most dangerous kind**, because every cheap
check passes. `yaml.safe_load` returning a dict is not evidence the edit did what you meant; it is
only evidence the file is still YAML. Assert the specific invariant you care about — here, that
each gated job has `needs: [changes]` *and* an `if` that reads `needs.changes` — and assert the
shape of the change (pure addition) as well as its validity.

This is the same shape as the shard off-by-one: a check that reports green while measuring nothing
or the wrong thing. Twice now in one session, in two different layers.

### The filter I wrote was wrong, and reviewing it by eye did not catch it

The first version of this filter omitted `calc-engine/**`. Three files under `backend/` import it —
`internal/calc-engine/worker/init.go`, `internal/api/calc-engine_handlers.go`,
`internal/analytics/semantic_calculation_service.go` — and `build-backend` runs `go build ./...` in
`backend`. A PR touching only `calc-engine/` would have skipped the build that would have caught a
broken API, and reported green. The shards run `go test` with `working-directory: backend`, so the
suite was not affected; the *build* gate was the one that would have been silently lost.

The fix is one line. The part that matters is how the gap is now found: the check re-derives the set
of `go.work` modules that `backend/` and `cmd/` actually import, and fails if any is not covered by
the declared filter. It is not a test of the line I edited — it is a test of the *property*, so the
next module added to `go.work` is caught here rather than in a PR that quietly stops building. It
also had to be corrected once itself: the first version compared module directories to filter entries
by string equality and so reported `libs/db/queries` as uncovered by `libs/**`, inventing five gaps
that did not exist. It now asks whether a real file inside the module (`go.mod`) matches the pattern,
which is the question the filter actually answers.

**Reviewing a filter by reading it is not verification, because the failure is an absence.** Every
path you can see listed looks right; the defect is the one you did not think of. Derive the set
mechanically and intersect it with the claim.

**A merged PR deserves the same suspicion as unmerged code.** This gap was introduced by me, in this
PR, and survived a review that had already caught two failed script attempts in the same file.

---
