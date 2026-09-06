# TRIAGE REPORT (2026-09-05) — read this section first

**Status: page-builder implementation paused.** What began as a census of
which BO service is "live" evolved into a triage report on the backend's
BO/entitlement layer. Short version: **six for six** hand-written SQL
statements tested by literal replay against the live `alpha` database
failed — spanning BO create/read/update/list and the entitlement layer.
This is not a page-builder problem; it's a substrate problem the page
builder depends on.

**RECOVERY NOTE (2026-09-06):** this file, its sibling `RESUME.md`, and
every other untracked artifact this effort produced in `backend/` were
deleted by an external `git clean`-class operation from a concurrent
session sharing this working tree — not by any git operation of this
effort's own. Reconstructed from conversation history and re-verified
against live `alpha` at recreation time (see each section's citations).
Escalation-message send status: **unconfirmed** — the message text was
drafted and handed to the user in the prior sitting; whether it was
actually sent could not be verified from this session. If it was not sent,
this document is currently the only surviving record of these findings and
should be sent or shared with the backend owner directly.

## What's actually wrong: three schema generations, unsynchronized

Confirmed via direct SQL replay (not code reading) against `alpha`:

| Code | Expects | Alpha has | Verdict |
|---|---|---|---|
| `internal/metadata.CreateBusinessObject` | `key`, `name`, `config`, `clones_from` (gen-1) | `bo_key`, `bo_name`, `model_id`, `classification_node_id` (gen-3) | **Code never updated** after schema migrated forward |
| `internal/metadata.GetBusinessObject` | same gen-1 shape | gen-3 | Same — always fails, silently returns "not found" |
| `internal/metadata.UpdateBusinessObject` (final stmt) | `display_name`, `icon`, `key` (gen-1) | gen-3 | Same |
| `internal/metadata.ListBusinessObjects` | gen-1 shape | gen-3 | Same, but error swallowed (`Warnf`) — route returns `200` + empty array forever |
| `boresolver.SaveBusinessObjectAtomic` | gen-3 shape + one extra column, `status` | gen-3, no `status` column | **Nearly current** — one-column drift, most likely the least-stale of the four |
| `security.FieldPermissionRepository.GetTermPermissionsForRoles` | `term_node_id`, `datasource_id` on `bp_field_permissions` (gen-N+1, forward-looking) | Neither column exists | **Schema never caught up to code** — inverse of the others |

**The migration ledger (`schema_migrations`) is empty — zero rows, both
`public` and `vend` schemas.** This database's history was never tracked by
this repo's migration runner. Falling back to column-presence facts: the
gen-3 `business_objects` migrations (e.g.
`20260812000002_sync_existing_bos_to_catalog.up.sql`) did apply — alpha has
that shape. `add_term_node_id_to_field_permissions.sql` exists in the repo,
defines exactly the column `FieldPermissionRepository` expects, and was
**never applied** — confirmed absent via `\d bp_field_permissions`.

**Conclusion: not one drift event, two different ones layered together.**
The BO-service code is stale relative to a schema that moved past it. The
entitlement code is forward-looking, waiting on a migration that was never
run. Alpha sits in the middle of a codebase whose layers point in different
directions — it agrees with no single generation consistently.

## Confirmed: `alpha` is the real target, not a stale copy

The running `uisce-backend` process (PID checked directly via `ps`/`lsof`)
has `DATABASE_URL` and `POSTGRES_DSN` pointing at the same
`100.84.50.65:5432/alpha` this investigation queried all along, with live
`ESTABLISHED` TCP connections at check time. There is no drifted-copy escape
hatch — this is genuinely the app's database, and the drift above is real,
current, and load-bearing.

## Four misattribution/environment-hazard mechanisms caught mid-investigation

1. **Context-compaction folklore.** Line numbers and even some file-content
   claims ("`CreateBusinessObject` writes `business_object_fields`") were
   carried forward across a very long conversation as settled fact well
   past the point they should have been re-checked, and turned out to not
   match a fresh re-read of the actual file.
2. **Worktree contamination.** `bo_entitlement_repository.go` and
   `bo_entitlement_filter.go` — quoted at length earlier in this
   investigation as the live, "solid" entitlement facet — exist **only**
   under `.claude/worktrees/*` (other Claude Code sessions' isolated git
   worktrees on this machine), not in the main tree. Some grep or read
   earlier in the session wandered into that directory. The real main-tree
   file is `field_permission_repository.go` — different design, different
   (and itself broken, per the table above) queries.
3. **Plaintext secret exposure** (from an earlier, separate mTLS-hardening
   effort referenced at the start of this session): an account password and
   a live `INFISICAL_TOKEN` were pasted into plaintext terminal output.
   Flagged for rotation; not this document's finding to track further.
4. **Shared-working-tree destructive-git collision.** This repo's working
   tree is shared across multiple concurrent Claude Code sessions with full
   git access. One session's `git clean`-class operation deleted every
   untracked file this effort had produced in `backend/` (this entire
   directory, four migration files) plus `frontend/src/pages/page-studio/RESPONSIVE_DESIGNER.md`
   — collateral from unrelated housekeeping work (migration-directory audits
   visible in the commit log), not malicious, but total for anything
   untracked. **Operational recommendation:** one Claude Code session per
   isolated git worktree, never multiple sessions sharing one working tree
   with destructive git commands enabled. The `.claude/worktrees/*`
   mechanism that caused hazard #2 is the same mechanism that would have
   prevented hazard #4 — per-session isolation cuts both ways.

**The methodology lesson, stated once because it applied repeatedly:**
runtime and live database/filesystem state outrank every claim carried
forward through context or through an unprotected shared workspace. The
techniques that caught these — literal SQL replay (write the exact
statement, run it in a rolled-back transaction, watch it succeed or fail),
process/`pg_stat_activity` inspection (what is actually connected to what,
right now), and `git status`/`ls` ground-truth checks before trusting an
edit "took" — are cheap enough to be the default, not a last resort.

## What's banked and survives, unconditionally

Everything below was built or verified directly against live `alpha` data,
independent of which application code is or isn't working, and independent
of this file's own survival:

- `bo_field_key_registry`, `bo_crud_capabilities`, `bo_widget_policy`,
  `bo_widget_breakpoint_fallback` — **tables exist live in `alpha` right
  now**, unaffected by the local file deletion. Schema, partial-unique-index
  fix, all originally tested against real data via a temporary smoke binary.
- The 81-row registry backfill from `business_object_fields` (real,
  FK-backed, gen-3) — **data is live in `alpha`**, source SQL being
  reconstructed from the live rows (see `misc/backfill_bo_field_key_registry.sql`).
- The drift detector query — reconstructed, re-verifiable any time by
  running it.
- The widget-policy seed (13 rows) — **data is live in `alpha`**,
  consolidating the frontend `inferWidget()` enum
  (`frontend/src/hooks/useCRUDPageConfig.ts:359`, re-verified present at
  that exact path, main tree, not a worktree) and the backend stub
  (`backend/internal/component_extensibility/forms/generator.go`).
- Design work that is entirely code-independent: the facet split
  (capability vs. entitlement), the sparse-breakpoint-fallback model, the
  responsive-layout constraints (presentation-only, two-level inheritance,
  build-time widget validation), the four update-semantics
  (add/rename-forbidden/remove-with-reference-check/type-change-gated), and
  the transactional registry-upsert invariant.

None of this needs to be redone. It transfers unchanged to whichever
service emerges from triage as the real BO write path.

## What's void

- Which BO service is "live" in the sense of *functioning* — none of the
  three tested write/read paths work against `alpha` as currently written
- The "solid facets" premise for entitlements (confirmed broken), and
  cardinality/validation (untested after this finding — not worth testing
  further until triage decides a direction)
- The `boresolver`-consolidation recommendation — directionally still the
  best-evidenced candidate (nearest to the current schema, one column off),
  but "one column fix and it works" is not yet verified end-to-end, and
  choosing it is a product decision, not something to infer unilaterally
  from code cleanliness

## The one question only a human can answer

Is `alpha` sitting between schema generations **by design** (a deliberate
checkpoint mid migration, e.g. a planned cutover that stalled) or **by
accident** (unsynced work streams that never reconciled)? And regardless of
cause: **which generation is the intended destination** — finish moving
`internal/metadata` to gen-3, or run the pending `term_node_id` migration
and align entitlements forward, or something else entirely? That answer
determines whether the next unit is "repair `boresolver`'s one column and
adopt it" or a larger consolidation this investigation hasn't scoped.

## Recommended next step

Pause page-builder implementation. Take this report to whoever owns backend
deployment/migrations. Bring: this section, the six-row evidence table, the
empty-ledger finding, and the banked-artifacts list so the conversation
starts from "here's what's confirmed and here's what's ready to resume"
rather than "something's broken, help." Also recommend adopting a migration
runner with a ledger baselined against the current live schema, **and**
adopting one-session-per-worktree discipline for any future multi-agent
work on this repo — both are infrastructure gaps this investigation hit by
accident and would recur without a deliberate fix.

**When the owner names a destination generation, resume with
re-verification, not the fix.** Both plausible fix paths (run the pending
`term_node_id` migration; repair/replace the BO write path) mutate the
environment. Everything banked in this report is verified against *today's*
`alpha`, not tomorrow's. Re-run the SQL replays and the curl checks against
the post-decision state before editing anything.

---

# Business object service census (historical — superseded by the triage report above)

## Original findings (2026-09-05, early in this investigation — see corrections above for what survived)

**1. Metadata definition/editing (frontend & backend)**
- Frontend "studio" UIs: `frontend/src/features/bo-studio/BusinessObjectStudioPage.tsx`,
  `frontend/src/features/bo/SingleScreenBOStudio.tsx`,
  `frontend/src/features/bo/BOGovernanceStudio.tsx`,
  `frontend/src/features/uisce-builder/UisceBuilder.tsx`.
- Frontend type contracts: `frontend/src/types/businessObject.ts`,
  `frontend/src/types/crud-generator-types.ts`, `frontend/src/types/cardinality.ts`,
  `frontend/src/types/pageStudio.ts`.
- Backend domain models across `internal/bo`, `internal/metadata`,
  `internal/services` (three parallel BO service packages — see census below),
  plus CUE definitions in `cue/bo_*.cue`.

**2. Postgres persistence** — no single consolidated schema file; the live
shape is the product of incremental migrations in `backend/migrations/*.sql`.
Top-level `migrations/001-002_consolidate_business_objects*.sql` and
`migrations/015_metadata_engine.sql` are legacy/prototype and confirmed dead
(015's tables `object_definitions`/`dynamic_entities` don't exist in `alpha`
at all; `view_definitions` exists but from a later, different migration).

**3. BO service census — the part later investigation revised:**
- `internal/metadata.BusinessObjectService` — wired to live HTTP routes
  (`api.go`, `NewBusinessObjectHandler` → `/business-objects`,
  `/api/v1/bo/...`). **Confirmed wired to routes, but every core method's
  SQL fails against the real schema (see triage table above) — "live" in
  the routing sense only, not functionally.**
- `internal/bo` — down to one call site, minor helper, not a competing path.
- `internal/services.BusinessObjectService` — dead code, zero call sites for
  its handler constructor anywhere, including via command dispatch.
- `pkg/meta.Service` — dead code, reachable only through
  `wealth.Bootstrap.InitializeWealthTransfer`, which has zero external
  callers (`wealth/bootstrap_integration.go` even contains its own
  unreferenced `func main()`).
- `boresolver.SaveBusinessObjectAtomic` — the closest-to-correct candidate
  once its one broken column (`status`) is fixed.

**4. Two field tables, resolved:** `business_object_fields` (11 BOs, 81
fields, real FK to `business_objects.id`) is real, gen-3, and the source of
the page-builder facet tables' backfill. `bo_fields` (46 BO IDs, 227 fields)
has **no FK constraint**, and 45 of 46 IDs match nothing else anywhere in
the database — confirmed orphaned, fed by a bug in a since-superseded
understanding of `UpdateBusinessObject` plus two offline scripts
(`cmd/populate_bo_fields/main.go`, `cmd/e2e_bo_check2/main.go`). Its one
plausible live reader, `/api/validation-rules`, turned out not to depend on
it either — `catalog_validation_rules` keys on a legacy `target_entity` name
string; the ID column that would reference `bo_fields` is empty across all
233 stored rows. `bo_fields` is dead weight, not a second catalog.

**5. BO Studio's governance screens** (`BOGovernanceStudio.tsx`,
`ValidationRuleBuilder.tsx`, `FieldSecurityConfigurator.tsx`) call
`/api/v1/bo/{boKey}/governance/...` routes that **404 against the running
server** — confirmed via direct curl against the live local instance, not
inference. These screens have never rendered real data.
