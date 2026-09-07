# Page builder — resume list

One page for the next session of this effort. Read this before re-reading
`BO_SERVICES.md` or the design files in full — it says what's blocked, what
isn't, and in what order, so re-entry costs one page instead of one
investigation.

**Related artifacts:** `BO_SERVICES.md` (triage report, aimed at the backend
owner), `frontend/src/types/pageStudio.ts` (`LayoutNode.responsive` schema),
`frontend/src/pages/page-studio/RESPONSIVE_DESIGNER.md` (interaction design).

**RECOVERY NOTE (2026-09-06):** this file, `BO_SERVICES.md`, the three
`pagebuilder` Go repositories, four migration SQL files, and
`RESPONSIVE_DESIGNER.md` were all deleted by a concurrent session's
`git clean`-class operation on this shared working tree (see the
shared-checkout landmine below — this is now a confirmed, not suspected,
hazard). Everything is being reconstructed on a dedicated branch
(`pagebuilder-artifacts-recovery`), committed immediately after each piece
is recreated and verified, specifically so nothing sits untracked and
exposed again. `pageStudio.ts`'s schema edit survived because it's a
tracked file that was merely modified — `git clean` doesn't touch tracked
files. Confirmed present and typechecking clean as of this recovery pass.

## 1. Gates

**Gate A — owner's destination-generation decision.** Sent as a triage
report (see `BO_SERVICES.md`). **Escalation-send status: unconfirmed** — the
message was drafted and handed off in the prior sitting; whether it was
actually sent could not be re-verified after the file loss. Confirm this
before assuming the owner has seen any of it. Two known outcomes:
- *Repair `internal/metadata`* (or replace it) to match the gen-3 schema
  already live on `alpha`. Unlocks: `CreateBusinessObject`/`GetBusinessObject`/
  `UpdateBusinessObject`/`ListBusinessObjects` become real. Larger fix (four
  broken methods in a 2,000+ line file).
- *Adopt `boresolver.SaveBusinessObjectAtomic`* as the write path (one column,
  `status`, doesn't exist — smallest known fix) and run the pending
  `add_term_node_id_to_field_permissions.sql` migration to align entitlements
  forward. Smaller fix, thinner surrounding code, but "smaller" was also true
  of every claim in this investigation that turned out to need re-verification
  — don't skip Gate B because this path looks cheap.

Either answer, or a third option the owner names, unlocks Gate B.

**Gate B — re-verification pass, mandatory, before any edit.** Both fix
paths mutate the environment (a code deploy, a migration run, or both).
Everything banked in this effort is verified against *the `alpha` state as
of 2026-09-05/06*, not against post-decision `alpha`. Before writing a
single line of the fix:
- Re-run the SQL-replay checks (literal statement, rolled-back transaction)
  against whichever method is now the target, on current `alpha`.
- Re-run the curl checks (`/api/business-objects`, `/api/validation-rules`,
  `/api/page-studio`) against the current running server.
- Re-check `schema_migrations` row count — if a migration runner has been
  adopted per the recommendation in `BO_SERVICES.md`, the empty-ledger
  finding may already be stale, which is good news, but confirm it rather
  than assume it.

Five minutes. Skipping it re-creates the exact failure this effort spent a
full sitting correcting: acting on a claim that used to be true.

## 2. Queue, ordered

1. **Wire the save path.** The `/api/page-studio` backend route (currently
   unimplemented — confirmed repo-wide, zero Go matches), the governance
   routes BO Studio's frontend already expects but 404s against
   (`/bo/{boKey}/governance/validation-rules`, `/governance/field-security`),
   and `ValidatePageWidgets` (already built in `pagebuilder/widget_policy_repository.go`)
   landing at its first real host — the actual save handler, not a stub.
2. **Reconcile the two page-builder frontends** —
   `types/pageStudio.ts`/`pages/page-studio/*` vs.
   `pageStudioTypes.ts`/`components/pagestudio/*`. Deliberately sequenced
   *behind* step 1, not ahead of it: which frontend wins may be partly
   answered by which one actually wires up to the real backend contract
   first. `pages/page-studio` already carries the schema this whole design
   layer targets, which is a point in its favor, not a pre-decision — let
   wiring arbitrate rather than picking now on schema-tidiness grounds alone.
3. **Drift-aware authoring** (entitlement/CRUD gaps surfaced inline while
   building a page). Named highest-priority enhancement earlier in this
   effort, but its entitlement half depends on the pending `term_node_id`
   migration (Gate A/B territory) — expect its data source to move under it.
4. **Widget-registry expansion** beyond the 13 seeded `bo_widget_policy` rows.
   Unblocked technically, premature in practice — registry growth should be
   driven by real authoring friction against a working renderer, not
   speculative pre-population.
5. **AI-assisted layout suggestions.** Last, unchanged reasoning: AI
   suggestions over a moving/unverified substrate produce confident garbage.
   Wants a stable, wired, reconciled substrate before it's worth building.

## 3. Landmines, named

Standing rules that were load-bearing more than once in this effort. Don't
relearn these the expensive way a second time:

- **Every grep/find runs with absolute paths and `.claude/` excluded.**
  `bo_entitlement_repository.go`/`bo_entitlement_filter.go` — quoted at
  length earlier as live, solid code — existed only in another session's
  `.claude/worktrees/*` checkout. Cost a full re-verification pass to catch.
- **SQL replay and runtime checks (curl, `pg_stat_activity`, process env)
  outrank any claim carried forward through conversation context**,
  including this document. If a claim here matters to what you're about to
  do, re-check it against live state first — citations here have a
  timestamp, not a permanence guarantee.
- **The two-frontend duplication is real and unresolved** — don't silently
  build against one as if the other doesn't exist; see queue item 2.
- **`schema_migrations` was empty as of this writing** — zero rows, both
  `public` and `vend` schemas, meaning "what's applied" could only be
  determined by column-presence diffing, not by trusting the ledger. If a
  migration runner gets adopted, verify it's actually being used before
  trusting it either.
- **Nothing gets unilaterally applied to the database.** The pending
  `term_node_id` migration sits right there and probably fixes the
  entitlement query — running it without the owner's decision is exactly
  the kind of environment mutation this effort was careful to avoid, and it
  would make the *next* person's forensics harder in a database with no
  migration ledger.
- **This repo's working tree is shared across multiple concurrent sessions
  with destructive git access — confirmed, not suspected.** Every untracked
  file this effort produced in `backend/` (an entire directory, four
  migration files) plus one frontend design doc was deleted by another
  session's cleanup pass. `frontend/src/types/pageStudio.ts` also reverted
  three times mid-edit before finally holding. **The mitigation that
  actually works: commit early, on a dedicated branch, immediately after
  each artifact is created and verified — untracked work in a multi-session
  repo isn't "not yet committed," it's "not yet existing."** Don't leave
  anything uncommitted across a turn boundary if it can be helped. Verify a
  file's actual on-disk content (`grep`/`cat`, not memory of having edited
  it) before assuming any edit still holds, regardless of commit status.

## What this effort actually produced, in six artifacts

1. `backend/migrations/20260905_page_builder_facets.sql` — four facet tables
2. `backend/internal/pagebuilder/*.go` — three repositories, tested against live data
3. `backend/migrations/misc/{backfill,seed,check}_*.sql` — backfill, seed, drift detector
4. `backend/internal/pagebuilder/BO_SERVICES.md` — the triage report
5. `frontend/src/types/pageStudio.ts` — the `LayoutNode.responsive` schema
6. `frontend/src/pages/page-studio/RESPONSIVE_DESIGNER.md` — interaction design

One owner question outstanding. Everything else in this queue starts the
moment that question is answered and Gate B is cleared.

**Provenance note added during the 2026-09-06 recovery pass:** items 1-4
were reconstructed with the live `alpha` database as the source of truth
(schema diffed against `information_schema`, backfill/seed content
regenerated from actual live rows), not retyped from conversation memory —
conversation carry-forward had already proven unreliable twice in this
effort (compaction folklore, worktree contamination), and rebuilding
artifacts from the same unreliable medium a third time would have repeated
the mistake. Items 5-6 (design docs, code-independent) were reconstructed
from history since there's no database to source them from, then re-verified
by compiling/typechecking rather than trusted on sight.
