# Branch disposition record

Live record of every branch found in the 2026-09-07 unmerged-work sweep, one
row per branch, updated as each is reviewed and resolved. This is what makes
branch state stay discoverable — the consolidation started because it
wasn't. Update this file in the same PR that changes a branch's status.

**Status values:** `merged` / `discarded-subsumed` / `discarded-regressive` /
`re-derive` / `pending-review` / `needs-triage`

| Branch | Commits ahead of main (at sweep) | Status | Evidence |
|---|---|---|---|
| `fix/runner-search-path-and-tx-control` | 3 | `merged` | PR #16 |
| `fix/quarantine-verify-scripts` | 4 | `merged` | PR #17 |
| `pagebuilder-artifacts-recovery` | 8 | `merged` | PR #15 |
| `fix/dead-handler-removal` | 2 | `merged` | PR #19 |
| `fix/backend-unblock-a11y` | 10 | `merged` | PR #20 (one conflict resolved: `baseline.spec.ts`, took main's superset) |
| `fix/audit-handler-panic` | 0 | `discarded-subsumed` | Confirmed 0 unique commits ahead of main at sweep time; deleted |
| `fix/audit-backlog-updates` | 0 | `discarded-subsumed` | Same |
| `critical-fixes-from-cleanup` | 0 | `discarded-subsumed` | Same |
| `hotfix/tenant-connection-idor` | 1 | `merged` | PR #21 — minimal extraction, not a branch merge; see below |
| `claude/wonderful-lewin-7c0c43` | 52 | `pending-review` (partially resolved) | See detailed disposition below — flagship security fix discarded-regressive, ~96% of touched files subsumed, remainder categorized |
| `claude/nifty-greider-015b86` | 52 | `re-derive` | Dead-code removal (`CognitiveGraphStudio`), shares fork point with `wonderful-lewin`; not yet re-diffed against current main — apply the same categorization pass before trusting its deletions |
| `claude/festive-jemison-6593fd` | 52 | `pending-review` | New `CatalogGraph` shared component; touches `frontend/src/types/pageStudio.ts` (the responsive-schema work from this effort) — review with that schema in front of you, not yet started |
| `cleanup-node-edge-deadcode` | 211 | `re-derive` | Diverged from `studio-wireup` at a shared ancestor, neither contains the other. Its dead-code analysis predates 204+ commits of change on the line it should have tracked. Do not merge — redo the dead-code pass against consolidated main and open a fresh, small PR for whatever still applies |
| `studio-wireup` | 204 | `pending-review` | Carries the verified gen-3 CRUD work via PR #18 (still open, targets this branch). This is the branch selected as the real line forward over `cleanup-node-edge-deadcode`; needs its own merge into main once PR #18's content is accounted for |
| `feat/tenant-connection-workflow` | 140 | `needs-triage` | Human decision needed first: still wanted? Not yet reviewed at all |
| `feature/studio-events-audit-expression-engine` | 117 | `needs-triage` | Same |
| `feat/northwind-abac-gold-copy-profiles` | 78 | `needs-triage` | Same |

## `claude/wonderful-lewin-7c0c43` — detailed disposition (2026-09-07)

Re-baselined against current `main` (not the pre-Phase-1 state) after the
`hotfix/tenant-connection-idor` PR landed. Mechanical pass: 1,421 files
touched across the branch's 52 commits; **1,361 (96%) are now byte-identical
to current `main`** — already subsumed, no action needed. 60 files still
differ. Of those:

**Confirmed `discarded-regressive`** (merging would revert already-landed,
more-correct fixes back to vulnerable/crash-prone code):
- `backend/internal/api/api_dispatcher.go` (`GetTenantConnection`) — the
  flagship "fix cross-tenant IDOR" commit itself; branch version reverts the
  now-deployed `TenantIDFromRequest`-based fix to raw, unauthenticated
  header/param trust
- `backend/internal/api/connections_routes.go` — reverts `main`'s
  fail-closed `getTenantIDFromRequest` (PR #19) back to a raw
  `X-Tenant-ID` header fallback
- `backend/internal/rulefabric/handler.go` — reverts `main`'s
  `security.AuthInfoFromContext`-based resolution to
  `jwtmiddleware.GetClaimsFromContext` + unguarded header fallback
- `backend/internal/api/helpers.go` — branch predates `TenantIDFromRequest`
  entirely; doesn't have it
- `backend/internal/api/node_types_routes.go` — branch calls
  `jwtmiddleware.GetClaimsFromContext(r).TenantID` directly with no nil
  check (a **crash bug**, not just a security regression, if claims is nil)
  instead of `main`'s fail-closed `TenantIDFromRequest`

**Confirmed `discarded-subsumed`, needs closer look to be certain:**
- `backend/internal/tenant/context.go` — `main`'s version contains an
  RLS-session-variable-name investigation/fix (`app.current_tenant_id` vs.
  the actually-checked `uisce.current_tenant`) that reads like it's already
  ahead of the branch's understanding. Likely subsumed by the BYPASSRLS
  design work referenced in `fix/backend-unblock-a11y` (PR #20), but not
  independently confirmed line-by-line — flag before assuming
- `backend/internal/middleware/auth_context.go` — same
  `allowClientTenantHeaderFallback` machinery visible on both sides but
  restructured; needs the same closer look as above before calling it
  fully resolved

**Genuinely `re-derive`, not `discarded`:**
- `backend/internal/semanticmatch/*` (11 files) — the branch deletes this
  package as dead code; `main` still has it. Unlike
  `cleanup-node-edge-deadcode`'s 204-commits-stale analysis, this is a
  small, more recent, focused deletion — plausibly still valid, but not
  independently re-verified against current `main` in this pass. Re-run a
  dead-code check before trusting the branch's removal.

**`pending-review` — novel, needs a real look, not yet done:**
- `backend/internal/api/validation_rules_routes.go` +
  `validation_rules_cel_e2e_test.go` — a CEL-based rule-evaluation feature
  that looks unrelated to the tenant-security question and absent from
  `main`; genuinely new capability, not a security fix
- `frontend/src/components/graph/*` + `CatalogGraph.test.tsx` +
  `ImprovedErdDiagram.tsx`/test — the shared `CatalogGraph` component feature
  (same one duplicated in `claude/festive-jemison-6593fd` — check whether
  these two branches' versions agree or conflict before reviewing either
  alone)
- `frontend/src/types/pageStudio.ts` — conflicts with this effort's
  responsive-schema work; review side by side with
  `frontend/src/pages/page-studio/RESPONSIVE_DESIGNER.md`
- `backend/go.mod` / `go.sum` — dependency changes, unreviewed
- `backend/internal/migrations/runner.go` — differs from `main`'s
  already-fixed runner (PR #16/#17); check whether this is a third
  independent runner change or stale relative to those fixes before
  assuming either direction
- Remaining ungrouped files (frontend misc components — `AppRoutes.tsx`,
  `MainNavigation.tsx`, `NotificationBell.tsx`, `AbbreviationManagerV2.tsx`,
  reporting/tenants components, `apiClient.ts`; backend handlers —
  `audit_handler.go`, `pre_aggregation_handler.go`,
  `validation_rules_list.go`, `nba/websocket.go`, `boresolver/*`,
  `analytics/*`, `calcengine/unified_engine.go`; `cmd/demo_xirr_execution`,
  `cmd/semantic-rules-api`, `cmd/semanticmatch`; assorted test files) — not
  yet individually reviewed. Full list: `/tmp/branch_touched_and_differing.txt`
  from this session (regenerate via the merge-base + touched-files method
  documented in this effort's transcript if that file is gone).

**Net effect of this pass:** 52 unreviewed commits converted into 5
confirmed-regressive files (discard), ~1,361 subsumed files (no action), 2
files needing one more confirmation pass, 11 files needing a dead-code
re-check, and a genuinely small novel set (CEL validation feature,
CatalogGraph component, `pageStudio.ts` conflict, dependency bump, runner.go
question, ~15 miscellaneous files) that actually warrants human review time.
