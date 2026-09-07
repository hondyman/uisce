# Branch disposition record

Live record of every branch found in the 2026-09-07 unmerged-work sweep, one
row per branch, updated as each is reviewed and resolved. This is what makes
branch state stay discoverable — the consolidation started because it
wasn't. Update this file in the same PR that changes a branch's status.

**Status values:** `merged` / `discarded-subsumed` / `discarded-regressive` /
`re-derive` / `pending-review` / `needs-triage`

**Last verified:** 2026-09-07, against `origin/main` @ `721ad6ba2b`. Every
"subsumed" / "byte-identical" / commit-count claim below is a snapshot as of
that commit, not a standing fact — re-run before trusting if `main` or the
branch in question has moved since. Re-verification procedure, exact
commands:

```bash
git fetch origin --prune
# per-branch unique-commit count (matches the sweep table):
git log origin/main..origin/<branch> --oneline | wc -l
# full subsumption check for a branch under detailed review (matches the
# claude/wonderful-lewin-7c0c43 pass below):
MB=$(git merge-base origin/main origin/<branch>)
git log $MB..origin/<branch> --name-only --pretty=format: | sort -u > /tmp/branch_touched_files.txt
git diff --name-only origin/main origin/<branch> | sort -u > /tmp/twodot_diff_files.txt
comm -12 /tmp/branch_touched_files.txt /tmp/twodot_diff_files.txt   # still differing
comm -23 /tmp/branch_touched_files.txt /tmp/twodot_diff_files.txt   # now subsumed
```

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
| `claude/wonderful-lewin-7c0c43` | 52 | `pending-review` (partially resolved) | Real tip: `7dad212ea`, the `CatalogGraph` shared graph-visualization component (ReactFlow wrapper — does NOT import `pageStudio.ts`, confirmed by reading the file). ~96% of touched files subsumed by current main; see detailed disposition below. **CORRECTED 2026-09-07 (later same day): an earlier pass in this document attributed the cross-tenant-IDOR flagship commit to this branch. That was wrong** — this branch does carry some shared pre-fix tenant-code from a common ancestor (confirmed: `api_dispatcher.go`'s `GetTenantConnection` here is also regressive relative to current main), but the actual 23-file security commit belongs to `claude/festive-jemison-6593fd`, not this branch. |
| `claude/nifty-greider-015b86` | 52 | `re-derive` | Real tip: `01745cc5b`, dead-code removal (`CognitiveGraphStudio`/`VisualizeLens`). Shares fork point with the other two `claude/*` branches; not yet re-diffed against current main — apply the same categorization pass before trusting its deletions |
| `claude/festive-jemison-6593fd` | 52 | `pending-review`, mostly unreviewed | **Real tip: `6506c0239` — "fix: close cross-tenant IDOR in remaining X-Tenant-ID/tenant_id readers," the actual 23-file flagship security commit.** Re-run of the subsumption pass against the correct branch: 1,422 files touched, 1,349 (95%) subsumed, but **of the flagship commit's other ~22 files specifically, zero are subsumed** — only `GetTenantConnection` (already fixed via the separate `hotfix/tenant-connection-idor`, PR #21) has been directly confirmed. The other ~21 files (`bo_crud_handler.go`, `catalog_admin_handlers.go`, `common/handlers.go`, `connections_routes.go`+test, `data_quality_handlers.go`, `drift_handlers.go`, `external_compliance_handler.go`, `glossary_handler.go`, `mcp_handlers.go`, `mdm_steward_handler.go`, `query_router.go`, `rebase_handlers.go`, `report_schedule_handlers.go`, `semantic_relationships_handler.go`, `semantic_tags_rest.go`, `shadow_handler.go`, `tenant_studio_handler.go`, `security_context.go`+test, `tenant_helper.go`, `rulefabric/handler.go`) are **genuinely unreviewed** — not confirmed regressive, not confirmed subsumed, not confirmed safe. This is the real security-review backlog item from Phase 2, still open. Also shares the `pageStudio.ts` divergence (identical across all three `claude/*` branches, see below) but does not touch `CatalogGraph` (confirmed: `7dad212ea` is not in this branch's ancestry). |
| `cleanup-node-edge-deadcode` | 211 | `re-derive` | Diverged from `studio-wireup` at a shared ancestor, neither contains the other. Its dead-code analysis predates 204+ commits of change on the line it should have tracked. Do not merge — redo the dead-code pass against consolidated main and open a fresh, small PR for whatever still applies |
| `studio-wireup` | 204 | `pending-review` | Carries the verified gen-3 CRUD work via PR #18 (still open, targets this branch). This is the branch selected as the real line forward over `cleanup-node-edge-deadcode`; needs its own merge into main once PR #18's content is accounted for |
| `feat/tenant-connection-workflow` | 140 | `needs-triage` | Human decision needed first: still wanted? Not yet reviewed at all |
| `feature/studio-events-audit-expression-engine` | 117 | `needs-triage` | Same |
| `feat/northwind-abac-gold-copy-profiles` | 78 | `needs-triage` | Same |

## Correction note (2026-09-07, same day, later pass)

The section below was originally written under the belief that
`claude/wonderful-lewin-7c0c43` was the branch containing the cross-tenant
IDOR flagship commit. It is not — confirmed by checking `git log --oneline
-1` and `git merge-base --is-ancestor` directly against all three `claude/*`
branch tips, which is what should have been done before attributing content
to a branch name in the first place. The mistake was carried from an
earlier point in the same session and repeated in this document without
re-verification — the exact failure mode ("conversation carry-forward
treated as settled fact") this whole effort exists to catch, now caught in
its own output. The file-level findings below (the specific diffs, which
functions regress, which files are byte-identical to main) are accurate
descriptions of `claude/wonderful-lewin-7c0c43` as re-confirmed; only the
framing of it as "the security branch" was wrong. The real security branch
is `claude/festive-jemison-6593fd` — see its row above for the corrected,
mostly-still-open disposition. **Lesson for whoever reads this next:**
verify a branch's identity (`git log -1`, `merge-base --is-ancestor` against
the specific commit you care about) before trusting its name, a commit
message quoted about it, or a prior session's claim about what it contains
— including this document's own prior claims.

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
`pageStudio.ts` conflict, dependency bump, runner.go question, ~15
miscellaneous files) that actually warrants human review time.

## Tenant-resolution sweep — remediation status (2026-09-07)

Full findings in `backend/docs/INCIDENT_REPORT_20260906.md`, "Platform Has
No Route-Layer Authentication Gate" entry. Remediation is a four-fix
architecture; status of each:

| Fix | Status | PR |
|---|---|---|
| 1 — canonical `ResolveTenantID` rule, `SecurityContextFromRequest` (67 sites) | **Done, merged, replay-verified** | #27 |
| 2 — route-layer auth gate (`/api/*` require-valid-JWT + public allowlist) | Not started | — |
| 3 — per-tier call-site migration (Tier 0/2/3, ~13+ endpoints) | Not started | — |
| 4 — BYPASSRLS (the class-level fix) | Not started, sequenced last per standing decision | — |

**Tier 0 (replay-confirmed live write path) still open:** `bo_crud_handler.go`'s `extractTenantUUIDFromRequest` — not yet migrated onto the canonical rule. This is the single most severe unfixed item: a confirmed-live, unauthenticated write path.

**Tier 2 (confirmed live, unauthenticated reads, ~13 endpoints) still open:** `report_schedule_handlers.go` (6), `glossary_handler.go`, `external_compliance_handler.go` (2), `shadow_handler.go`, `lookups_routes.go` (2), `catalog_admin_handlers.go`, `semantic_tags_rest.go` (2), `common/handlers.go`. Note `glossary_handler.go` also has 9 call sites into the now-fixed `SecurityContextFromRequest` — those are covered by Fix 1; only its independent raw-trust line needs separate migration.

**Tier 3 (weak fallback, lower priority) still open:** `trigger_handlers_chi.go`, `tenant_studio_handler.go`, `region/middleware.go`, `handlers/tenant_helper.go`.

**Open decision, not a code fix:** `frontend/src/components/semantic-mapper/useSemanticMapper.ts:246` sets `X-Tenant-ID` to `mapping.database_column.tenant_id` (a different record's tenant, not the caller's own). Once Fix 3 reaches any endpoint this header hits, this call site will start getting rejected unless the caller is a global admin — needs a decision on whether that's the intended UI restriction or a legitimate flow needing a global-admin path.

**Dead code, flagged for removal not hardening:** `mcp_handlers.go` (confirmed dead — a near-false-positive, see incident report), `drift_handlers.go`, `mdm_steward_handler.go`, `semantic_relationships_handler.go`, `rebase_handlers.go`, `data_quality_handlers.go`.

## `pageStudio.ts` divergence — adjudicated, mechanical resolve, no design decision needed

All three `claude/*` branches carry an identical 124-line divergence from
`main`'s `pageStudio.ts` (confirmed: the diff against each of the three
branch tips is byte-identical), inherited unchanged from their shared
fork point, which predates this effort's `LayoutNode.responsive` schema
(PR #15). Classification, per the three gathering questions:

1. **Case: forking.** The branches' `LayoutNode`/`CorePageDefinition`
   never had the `responsive` field, `ResponsiveBreakpoint`, or
   `ResponsiveOverride` types — not a rejection of the design, just
   written before it existed.
2. **What touches it: nothing that matters to the schema.** `CatalogGraph`
   (`frontend/src/components/graph/CatalogGraph.tsx`, confirmed on
   `claude/wonderful-lewin-7c0c43` only) is a ReactFlow-based graph
   visualizer for ERD/schema diagrams — read directly, it imports only
   `reactflow`, `nodeRegistry`, and layout helpers. **It does not import
   `pageStudio.ts` at all.** It neither renders nor authors `LayoutNode`s —
   the "neither" case from the original three-way classification.
3. **Frontend ownership: moot.** Since nothing in the differing branches
   actually depends on the responsive schema, the two-frontend question
   (`pages/page-studio/*` vs `components/pagestudio/*`) doesn't need
   resolving to close this item.

**Resolution:** mechanical, not a design negotiation. Any future merge
touching `pageStudio.ts` keeps `main`'s version (with `responsive` intact)
and, if still wanted, cherry-picks the branches' two genuinely additive,
unrelated changes on top: `ComponentDefinition.props` (a runtime prop bag
distinct from `defaultProps`) and a `business_object` `DataSourceDefinition`
variant with a `BusinessObjectDataSourceConfig` interface. Neither addition
touches `LayoutNode` or the responsive design; no adjudication against the
whole-overlay/closed-breakpoint-enum/no-structural-changes constraints was
needed because nothing here proposed anything that would have required it.
