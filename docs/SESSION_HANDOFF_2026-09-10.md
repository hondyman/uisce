# Session Handoff — 2026-09-10

Written for a fresh session with no memory of the conversation that produced it. Read this first, then `docs/unified-rule-engine-handoff.md` (the living copy, on `feat/calc-engine-measures` — see "Two copies of that doc exist" below) for the full technical history and proof artifacts this summarizes.

Everything below was true as of a live check at the moment this was written (commands shown where it matters) — re-verify anything load-bearing before acting on it. That habit is the actual subject of most of the work described here.

## Where things stand right now

- **Current branch**: `feat/calc-engine-measures`. HEAD at time of writing: `04701228b feat(orchestrator): Phase 1 — wire ReportGenerationWorkflow on live Temporal cluster` — this branch is shared with other concurrent sessions and moves under you; always re-check `git log -1` before assuming state.
- **Three open PRs on `hondyman/uisce`, none merged yet**, in dependency order:
  1. **[PR #47](https://github.com/hondyman/uisce/pull/47)** — `docs: correct the single-AST claim about rulefabric`. Doc-only. Small, safe, read-and-merge.
  2. **[PR #48](https://github.com/hondyman/uisce/pull/48)** — `feat(rulefabric): fold compliance_rules write-hook into the unified engine`. Real backend surgery on a write-path trigger dispatcher. Worth an actual review, not a rubber stamp — see "What to check in #48" below.
  3. Neither PR is merged as of this writing (`gh pr view 47/48 --json state,mergedAt` both returned `"state":"OPEN","mergedAt":null`).
- **One new issue filed**: **[#49](https://github.com/hondyman/uisce/issues/49)** — `TriggerEngine.EvaluateTriggers` queries columns that don't exist on the live `validation_triggers` table. Real, verified bug, pre-existing, unrelated to #48's changes. **Do not confuse this with issue #4** (already closed, already fixed, a *different* struct/file — see "A false-correction that got caught" below).
- **A real stash exists on `feat/calc-engine-measures`** (`stash@{0}`), containing report-scheduling/bursting work (`report_handlers.go`, `schedule_repository.go`, `ReportScheduleBurstingTab.tsx`, etc.) from a concurrent session. Its stated base (`WIP on feat/calc-engine-measures: 0bea68282 chore: ignore keycloak-ssl directory...`) does not match an earlier verbal description of it from conversation (which cited a different commit, `660a31b30`) — a small discrepancy, not a contradiction, but proof that even a stash's own recorded message should be re-checked against `git stash show -p stash@{0}` before trusting a secondhand description of it, this one included. **Do not drop this stash carelessly** — pop it deliberately during the rebase step below, after confirming what it actually contains.

## The runbook — what to actually do, in order

This is two review/merge steps (human, "yours"), one working session (the model can execute), one more review/merge, then one proof.

### 1. Merge PR #47 (doc-only)
```bash
gh pr merge 47 --repo hondyman/uisce --merge
```
Trivial — it's a doc correction, no code.

### 2. Review and merge PR #48
Three things worth actually reading, not trusting:
- The retired `case "compliance_rules":` diff in `backend/internal/api/trigger_engine.go` — confirm nothing *else* in that action dispatcher moved (notification/temporal/rabbitmq/webhook cases must be untouched; this was a scalpel cut, not an engine replacement).
- The `evaluateAndEnforceRules` call-site change in `backend/internal/metadata/shadow_evaluation.go` — `ListByBO(ctx, tenantID, boKey, "")`, the empty-string `domain` argument. This is the line that changes real behavior the day any MDM/compliance-domain rule actually exists (all domains now enforce on the BO write path). By design, but worth being sure of.
- The PR's addendum in `docs/unified-rule-engine-handoff.md` ("Rulefabric consolidation, backend slice") — two findings worth knowing before merging: the pointer to issue #49, and the new ordering constraint (`OperatorRegistry` can't be deleted from rulefabric until `internal/services/validation_rule_engine.go`'s dependency on it is decoupled — a future step, not blocking this merge).
```bash
gh pr merge 48 --repo hondyman/uisce --merge
```
Issue #49 is not blocking — it's a ticket for whoever next picks up the pipeline/DATAPIPELINE workstream, referenced not corrected against the already-closed #4.

### 3. The rebase working session (this is what a fresh session should actually *do*)
Once #47 and #48 are merged into `main`:
1. `cd` into the repo, `git fetch origin`, and run `git stash list` — **do not assume the stash described above is still there or still means what this document says**; read `git stash show -p stash@{N}` yourself first.
2. **Reconcile the forked handoff doc by decision, not by merge tool.** `docs/unified-rule-engine-handoff.md` currently exists as two diverged copies: a ~2,400-line version on `feat/calc-engine-measures` (the *living* one — it has sessions the other copy doesn't) and a shorter version that was on `feat/unified-rule-engine` before that branch merged into `main` via PR #38 (now further extended by #47/#48's addenda once those merge). **The `feat/calc-engine-measures` copy is the one that should win** — it's the actively-maintained one. When rebasing, if this file conflicts, resolve by taking the living copy's content and folding in anything genuinely new from the merged-`main` side (the #47/#48 addenda specifically) that the living copy doesn't already have. Do this by hand, reading both sides — don't let a merge tool auto-resolve a document.
3. Rebase `feat/calc-engine-measures` onto the now-merged `main`.
4. Pop the stash (after step 1's check), resolve any conflicts.
5. Run the full backend test/build pass (`go build ./...`, `go vet ./...`, relevant test suites) to confirm the rebase didn't break anything.
6. Open PR #2 — title/scope is "the calc-engine branch, rebased onto the unified-rule-engine base" — with the current `docs/unified-rule-engine-handoff.md` (the reconciled, living copy) as the review body. This PR's own CI will likely be red in the same pre-existing, already-documented ways (see the doc's "hermeticity" sections) — judge it against that baseline, not against green.

### 4. Review and merge PR #2
Human step, same as #47/#48.

### 5. The proof that closes this whole arc
Once PR #2 is merged into `main`, the cross-branch skew that's been documented throughout this doc disappears — the rule-evaluation context loaders both branches needed (e.g. `placement_routed_sum`) are now both present. Re-run the proof script that was blocked by this:
```bash
cd backend
export DATABASE_URL="postgresql://postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslcert=$HOME/.uisce/certs/postgres-client.crt&sslkey=$HOME/.uisce/certs/postgres-client.key&sslrootcert=$HOME/.uisce/certs/ca.crt"
go run ./cmd/verify_order_validations
```
Expected: fully green (all tests PASS, exit 0) — this was previously blocked specifically by the cross-branch context-loader gap, documented in detail in PR #38's merge-record comment on GitHub and in the handoff doc. **Append the result to that same PR #38 comment thread** (`gh pr comment 38 --repo hondyman/uisce --body "..."`) — the point is closing the loop: the original FAIL is on record there, and its resolution belongs in the same place, not just in this document.

### 6. Then, and only then: the editor unification work
This is the actual next *build*, once the base above is clean:
- The unified rule editor's context-object model (`domain`/`entityScope`/`categories`/`functions`) — see `docs/unified-rule-engine-handoff.md`'s "Rulefabric consolidation" sections for the full design.
- Term autocomplete in the Monaco expression-mode completion provider (`frontend/src/rules/aslMonacoRegistry.ts` and friends) — the single highest-value UX gap identified: expression mode has function autocomplete but not BO-term autocomplete.
- Fold `frontend/src/features/bo/PolicyRuleBuilder.tsx` (raw CEL textarea) onto the same Monaco expression surface, and remove `cel-go` from `backend/go.mod` once nothing references it (cheap — zero CEL expressions were ever actually written to the database; verified).
- Close with the proof this whole design has been pointed at: **author one MDM-domain rule, one compliance-domain rule, and one BO validation rule, in the same editor, with the same term/function autocomplete, and evaluate all three through the same engine.**

## Context worth knowing before touching any of this

### A false-correction that got caught — a pattern worth repeating
Earlier in this arc, a plausible-sounding instruction was given: "the pipeline stream's `STATUS_AUDIT_BACKLOG.md` claims a trigger-lifecycle fix (✅, issue #4) — your finding about `EvaluateTriggers` being broken falsifies that, correct it." **This was checked before being acted on, and it was wrong.** Issue #4's fix (commit `07e32a451`) touched `TriggerHandler.GetValidationTriggers`/`fetchTriggers` in `trigger_handlers.go`/`trigger_handlers_chi.go` — genuinely rewritten to match the real schema, genuinely gate-integration-tested. What was actually found broken was `TriggerEngine.EvaluateTriggers` in the *different* file `trigger_engine.go` — a sibling function with the same bug *class*, never touched by #4's fix, out of scope for it. The backlog's ✅ is true. Issue #49 was filed correctly scoped against this distinction, not as a correction to #4. **Lesson for whoever reads this**: an instruction that sounds like natural continuation of an established finding still needs its own verification before being executed, every time — including instructions that read as confident and well-reasoned.

### Two copies of that doc exist, and it matters which one you edit
See runbook step 3.2. If you're ever asked to update `docs/unified-rule-engine-handoff.md` and you're not on `feat/calc-engine-measures`, check which branch you're actually on before assuming you're editing the living copy.

### Live database access
Real `alpha` Postgres (`100.84.50.65:5432`) is reachable via mTLS certs at `~/.uisce/certs/{ca.crt,postgres-client.crt,postgres-client.key}` on this Mac. Connection string pattern:
```
postgresql://postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslcert=$HOME/.uisce/certs/postgres-client.crt&sslkey=$HOME/.uisce/certs/postgres-client.key&sslrootcert=$HOME/.uisce/certs/ca.crt
```
This is real, shared, production-adjacent state — every write against it in this arc was deliberate, verified, and documented. Don't run anything against it casually.

### Browser/frontend verification sessions are short-lived
If doing frontend work, Keycloak sessions in the dev environment expire quickly (observed: sometimes within a couple of minutes of navigation). Expect to ask for re-login more than once; don't burn time debugging what looks like a broken interaction when it might just be an expired session — check for a redirect to the Keycloak login screen first.

### The migration runner (`backend/migrations/cmd`) has real, documented bugs
Comment-blind statement splitting, dollar-quote corruption (not just mis-parsing — it drops content), and a transaction-ownership conflict with files that carry their own `BEGIN;`/`COMMIT;`. Full detail and a scoped repair spec are in `docs/unified-rule-engine-handoff.md`'s migration-runner sections. Don't assume `migrate up` works against real `alpha` without reading that first — it currently can't run most of the repo's migration files.

## Quick reference — everything touched this arc, by artifact

| Artifact | What it is |
|---|---|
| [PR #38](https://github.com/hondyman/uisce/pull/38) | Merged. The original unified-rule-engine PR; its comment thread carries the full merge-day verification record (PASS/FAIL results, explanations) — append the closing `verify_order_validations` result there per step 5. |
| [PR #39](https://github.com/hondyman/uisce/pull/39) | Merged. Recovered stream_loader Debezium decimal-decode fix. |
| [PR #47](https://github.com/hondyman/uisce/pull/47) | Open. Doc correction. |
| [PR #48](https://github.com/hondyman/uisce/pull/48) | Open. Rulefabric `compliance_rules` retirement. |
| [Issue #40](https://github.com/hondyman/uisce/issues/40) | Proof-script hermeticity (scripts evaluate all active rules in shared `alpha`, not just their own). |
| [Issue #41](https://github.com/hondyman/uisce/issues/41) | Rule-state/engine-capability lockstep documentation need. |
| [Issue #42](https://github.com/hondyman/uisce/issues/42) | Migration-runner repair (four bugs, full spec). |
| [Issue #49](https://github.com/hondyman/uisce/issues/49) | `TriggerEngine.EvaluateTriggers` phantom-schema query. |
| `docs/unified-rule-engine-handoff.md` | The living technical history — read this for everything this document summarizes. |
| `backend/cmd/verify_compliance_domain/` | New permanent proof program for the rulefabric-consolidation `domain` field. |
