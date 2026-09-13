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
| 3 | Medium (design intent resolved; deploy gap open) | **`tenant_product` RLS policy is fail-open on alpha**, contradicting its own test's stated contract. `tenant_product_isolation_policy`'s `USING`/`WITH CHECK` clause is `(current_setting('uisce.current_tenant', true) IS NULL) OR (tenant_id = ...)` — when the tenant GUC is unset, the clause is unconditionally true: full read *and write* access to every tenant's rows. `internal/db/tenant_tx_test.go`'s `TestWithTenantTransaction` has a subtest literally named `no_tenant_GUC_returns_zero_rows`, logging `"PASS: ... (fail-closed)"` on success — asserting the opposite of the live policy's actual behavior. | **Not ambiguous design intent** — a peer session (task_904383c1) found `backend/migrations/20260727000030_strict_tenant_rls.sql` (commit `b9d00dc2c6`, 2026-07-27, confirmed ancestor of `fd3effc71` and confirmed against its actual content) already replaces this exact policy with a strict, fail-closed one via `uisce_get_current_tenant()` (returns NULL, never a bypass value, when the GUC is unset — `tenant_id = NULL` is never true). The test's name and assertion are correct; alpha simply never had this migration applied. **But "just apply it" is not simple**: verified this migration sits only in `backend/migrations/` — one of the 8 directories `MIGRATION_DIRECTORY_DRIFT_AUDIT.md` already documented as invisible to the runner, which reads only `backend/db/migrations/*.up.sql` — and it's written in Goose's two-way format (`-- +goose Up` / `-- +goose Down` markers, 6 of them, DOWN statements physically interleaved after each UP section). The current custom runner has no concept of `+goose Down` and would execute the whole 137-line file as one blob — meaning even renaming it `.up.sql` and dropping it in `db/migrations/` would enable each policy and then immediately execute that same section's DOWN statement (e.g. `ALTER TABLE tenant_instance DISABLE ROW LEVEL SECURITY` right after enabling it), net-undoing itself. This needs a careful port (extract only the Up statements into a proper `.up.sql`) before it can go through the existing pipeline at all — not a one-line "apply this file" fix. Filed as `task_904383c1`, corrected in-thread once verified.|

### Positive counter-entry

| # | What the countermeasures caught | How |
|---|---|---|
| A | **Dead role stubs + one missing one, both from the same hand-list.** Deriving stub roles from `schema-snapshot.sql` itself (every `GRANT ... TO` / `OWNER TO` / `ALTER DEFAULT PRIVILEGES ... TO`) rather than hand-maintaining the list showed the old list stubbed three roles the snapshot never references at all (`infisical`, `keycloak`, `nessie`) while missing the one it needed (`app_admin_read`). A hand list can silently drift in both directions at once; a derived one can't drift at all without the dump itself changing. |
| B | **The vacuous-pass mechanism for `no_tenant_GUC_returns_zero_rows` is deterministic in this CI context, not merely theoretical.** Confirmed: the ephemeral job restores a schema-only snapshot (zero data) and — before this arc's fixes — the workflow's `UISCE_TEST_DB_DSN` connects as the `postgres` superuser (bypasses RLS entirely). Either fact alone would make this subtest pass without exercising the policy at all; both are true simultaneously in the current workflow, so every CI run of this suite is guaranteed to "PASS: fail-closed" a table that is, in fact, fail-open. |
| C | **Sweep for sibling fail-open policies needed no live alpha connection.** `schema-snapshot.sql` is a verbatim `pg_dump` of alpha's live `pg_policies`, so grepping it for the same clause shape *is* the sweep. Precise result (not the raw 11-match count a naive grep first returned): exactly **2 tables** carry the genuine bug shape — `public.tenant_product` and `public.tenant_product_datasource` — both keyed off the identical `uisce.current_tenant` GUC-absence check. The other 9 matches from the naive pattern are structurally different and not bugs: explicit `current_setting('app.is_admin', true) = 'true'` overrides (three-valued NULL logic means an *unset* admin flag does not satisfy `= 'true'`, so those fail closed correctly) and one policy checking a row's own `tenant_id IS NULL` (a legitimately global, non-tenant-scoped record type, not a session-GUC check). Also found: `tenant_product_datasource` carries a second, *strict* sibling policy (`tenant_isolation_policy`, no bypass) — but Postgres combines multiple permissive policies for the same command with OR, so the strict policy is fully neutralized by its fail-open sibling. Someone appears to have tried to tighten this table's policy and the attempt does nothing. |

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

---

## Entry 2026-09-10 — Arc 4 (collection aggregation, Phase 3 close)

*[To be populated by the next session that produces a failure or verification worth recording.]*

---

## Prior entries

*[This ledger is persistent across arcs. Entries accumulate here as each arc closes. The lessons from each entry should be consulted before starting a new feature or a merge prep sequence.]*
