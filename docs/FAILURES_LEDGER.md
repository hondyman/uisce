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

## Entry 2026-09-11 — Arc 6 (Export with Watermarking & Data Classification design)

**Arc context**: design doc for 8th engagement feature; seven-section doc assembled and approved through evidence gates; two required amendments (column-scoped UPDATE grant, TTL guard + sweeper tombstone); one contradiction caught pre-commit (hard-delete vs FK NO ACTION + revoked DELETE).

### Incidents

| # | Severity | Description | Root cause |
|---|---|---|---|
| 1 | High | **Secrets transit transcripts via command lines — three instances, same failure class**: `temppass123` exposed; fresh rotation value (`nvvxS1e0NBszgfZ2NMpQoLHo5i2kujMC`) exposed when passed as a shell argument; original password before that. Pattern: any secret through a command line is transcript-exposed. | No procedural rule against shell-passed secrets; rotation deferred three times |
| 2 | Medium | **Post-rotation verification was a skip, claimed as a pass**: live test run reported `ADMIN_READ_DSN not set; skipping` — two SKIPs — and the status table carried "live tests still pass." A skip is not a pass. | No rule that a skipped gate is "not run" |
| 3 | High | **Sweeper hard-delete self-contradicted**: §5.3 sweeper rule said "hard-deletes the row" but §4 FK is `ON DELETE NO ACTION` (every export row has ≥1 event from writer 1) and §4 grant is `REVOKE DELETE FROM app_user` (sweeper runs as app_user). Would fail on first purge attempt. | Rule written without checking FK + grant constraints in same doc |
| 4 | Medium | **Stale comment leftover after model change**: after switching sweeper to tombstone model, line 502's inline comment still read "sweeper sets expires_at" — the old model's positive assertion, not a negation. The predicate's inline comment is the text that gets copied into implementation. | Model-change sweep missed positive assertions in inline predicate comments |
| 5 | Medium | **Self-SSH confusion**: agent SSH'd to 100.90.97.15 (its own Tailscale IP) and concluded the "CA Mac" was unreachable. The CA Mac was this Mac. | No self-reachability check before assuming a remote target |

### Positive counter-entry

| # | What the countermeasures caught | How |
|---|---|---|
| A | **Edit-tool diff interleaving indicated possible file corruption**: Edit tool reported old/new text interleaved in diff output. Grep counts + file reads confirmed file was actually clean — diff was a display artifact. Lesson: for untracked files, grep counts are evidence; re-typed paste-backs are claims. | Grep phrase-count verification suite (hard-delete=0, tombstone=1, header sweep) |
| B | **Column-scoped UPDATE grant caught in review**: unqualified `GRANT UPDATE ON public.report_exports TO app_user` would make all columns mutable including `storage_key` — enabling an attacker who can run SQL as app_user to rewrite row provenance and download another tenant's artifact through the authorized proxy. | Reviewer caught the gap between the COMMENT ("all other columns immutable") and the GRANT |
| C | **TTL guard gap caught in review**: Predicate C had no `expires_at` guard — after sweeper purges an artifact, the row still says `completed` and the predicate still grants. | Reviewer identified that the events vocabulary included `EXPIRED` but the predicate never consulted `expires_at` |

### Structural fixes applied

| Fix | Mechanism | Status |
|---|---|---|
| **Cert-only DSN** | Secrets class retired: `ADMIN_READ_DSN` uses client-cert auth with no password. Cert-auth conversion pending on CA-key decision. | Pending: CA-key recovery exhausted; CA rotation is remaining path |
| **Secrets class ledger entry** | "any secret through a command line is exposed" — fix is procedural (cert-only), not rotational | Live — applicable to all future secret handling |
| **Untracked-file edit verification** | Grep counts + file reads are evidence; paste-backs are claims. Especially: `grep -n "^## "` for header uniqueness; phrase counts for key terms. | Live — apply to all future untracked doc edits |
| **Rule vs FK + grant cross-check** | When a rule is written, check it against FK constraints AND grants in the same doc before committing | Live — added to design review checklist |
| **Predicate inline comment sweep on model change** | When a rule model changes, search inline comments especially in predicates — positive assertions as well as negations. Predicates are the copy-paste artifact of design docs. | Live — Phase 1 implementation inherits this discipline |

### Verification log (this arc)

| Date | Check | Result | Tree |
|---|---|---|---|
| 2026-09-11 | Design doc §4/§5.3/§7 grep verification | ✅ all phrase counts correct | `feat/reports-exports` @ `8ed9fa05e` |
| 2026-09-11 | Sweeper tombstone grep verification | ✅ hard-delete=0, tombstone=1 | `feat/reports-exports` @ `29ab5053f` |
| 2026-09-11 | Header sweep (§1–§10 each once) | ✅ clean | `feat/reports-exports` @ `29ab5053f` |
| 2026-09-11 | Stale comment grep (`sweeper sets`) | ✅ 0 matches after fix | `feat/reports-exports` |

---

## Entry 2026-09-10 — Arc 4 (collection aggregation, Phase 3 close)

*[To be populated by the next session that produces a failure or verification worth recording.]*

---

## Prior entries

*[This ledger is persistent across arcs. Entries accumulate here as each arc closes. The lessons from each entry should be consulted before starting a new feature or a merge prep sequence.]*
