# CI Hygiene Merge Plan — 7 Open PRs

**Objective:** Merge 7 PRs from the glossary/CI hygiene session in correct dependency order, verifying signal at each step. CI is currently broken (red baseline) — sequencing matters because some PRs *fix* CI, others *depend* on CI being fixed to show clean signal.

**After all merges:** one deploy cycle from merged main.

---

## Pre-Flight: PR Freshness Check

Before starting the sequence, verify all 7 PRs are still mergeable and contain their expected files:

```bash
gh pr list --state open --json number,title,mergeable,baseRefName
```

Any `mergeable: false` (conflicts) or `mergeable: null` (unknown) → resolve that PR first. Do not discover rebase conflicts mid-sequence.

---

## Global Rules

- All merges use `--admin` (GitHub UI "Merge without waiting for requirements" or `gh pr merge --admin --repo`) given the red baseline
- Before each merge: `git fetch origin && git rebase origin/main` onto the post-merge main (not onto the previous PR's branch — branches drift from each other)
- **Pre-merge:** PR checks green on the rebased branch (runs against branch, pre-merge)
- **Post-merge:** watch the workflow run on `main` for that merge commit before proceeding to the next PR — this is the run that proves the change works in context. PR runs on the branch are pre-merge signal only.
- **Stop rule:** if the post-merge main run for any PR is still red, stop — do not merge subsequent PRs into a broken main. This is especially critical for #92 (frontend signal proof) and #101 (workflow-edit proof).
- After the full batch: one deploy cycle (build → scp → START_BACKEND.sh → /health)

---

## Per-PR Recipes

### 1. PR #92 — Bootstrap warning + orphan test + peer-deps
**What it fixes:**
- `START_BACKEND.sh`: `|| true` → loud red warning on Infisical bootstrap failure
- `compliance_test.go`: removes orphan `TestParseFIXNewOrderSingle` (function deleted, test left behind — caused `go vet` to fail)
- 6 frontend workflow files: `--legacy-peer-deps` added to all `npm ci` calls (react-json-view peer dep conflict)

**Depends on:** nothing — first in chain

**Merge method:** rebase on post-`main`, `--admin` merge

**What CI must show (pre-merge):**
- `ci.yml` Frontend Typecheck & Build: green (was red since Jun 26)
- `lint-react-hooks.yml` (both jobs): green
- `frontend-e2e.yml`: green
- `frontend-temporal.yml`: green
- `a11y-ratchet.yml`: green
- `workstation-ci.yml`: green

**What to watch for:**
- `npm ci` output should NOT show `WARN deprecated` for react-json-view (expected, since --legacy-peer-deps suppresses the conflict)
- If any frontend workflow still fails peer-dep issue, the `--legacy-peer-deps` flag may need to be in a different position in the npm command

**Post-merge action:** watch `main` run — frontend workflows must be green on main after this merge. If still red, stop.

---

### 2. PR #93 — RBAC: getTenantIDFromRequest falls back to AuthInfoFromContext
**What it fixes:**
- `getTenantIDFromRequest` panicked with nil-pointer when JWT claims absent but `security.AuthInfo` was in context
- 3 targeted tests added: JWT-priority, empty-tenant fail-closed, missing-datasource 400
- Key fix: context key must use `jwtmiddleware.ClaimsContextKey` (typed) not a plain string literal (Go context uses reflect.DeepEqual — type matters)

**Depends on:** #92 (clean CI signal needed to verify the JWT-priority test is meaningful)

**Merge method:** rebase on post-#92 main, `--admin` merge

**What CI must show (pre-merge):**
- `Backend Gated Tests`: green (was failing due to orphan test in #92's scope)
- `backend/internal/api/middleware/`: 7/7 tests green

**What to watch for:**
- JWT-priority test (`TestRequirePermission_JWTClaimsTakePriorityOverAuthInfo`): confirm it's in the PR diff and passes — this was the review condition
- The test uses `jwtmiddleware.ClaimsContextKey` (typed const) not `"jwt_claims"` string — verify in the diff

**Post-merge action:** confirm backend gated tests are green on main

---

### 3. PR #94 — Migration: remove explicit BEGIN/COMMIT from 20260914_013
**What it fixes:**
- Migration runner wraps each `.up.sql` in a transaction automatically
- The migration file had explicit `BEGIN;` and `COMMIT;` inside, causing runner to reject it with "contains transaction-control statements"

**Depends on:** nothing (independent)

**Merge method:** rebase on post-#93 main, `--admin` merge

**What CI must show (pre-merge):**
- `Backend Gated Tests`: still green (this fix is for the migration runner, not a code change)
- No specific CI job for migration — verify by running `go test ./backend/internal/migrations/...` if present

**What to watch for:**
- On dev server after deploy: migration runner will report `"content has changed since it was applied; skipping"` for `20260914_013` — this is **expected**, not a failure. The runner is correctly detecting the file-on-disk differs from the recorded-applied hash.
- The migration is idempotent; a future clean apply (fresh DB or reset) will apply the fixed version without the transaction statements

**Post-merge action:** note the expected dev-server warning; no action required

---

### 4. PR #96 — Minio: pin image to specific version tag
**What it fixes:**
- `docker-compose.yml`: `minio/minio:latest` → `minio/minio:RELEASE.2025-10-15T17-29-55Z`
- Docker Hub rate-limits unauthenticated pulls of `minio/minio:latest` on GitHub Actions

**Depends on:** nothing (independent)

**Merge method:** rebase on post-#94 main, `--admin` merge

**What CI must show (pre-merge):**
- `Health check`: green (was rate-limited pulling latest)

**What to watch for:**
- Verify the image tag is valid: `docker pull minio/minio:RELEASE.2025-10-15T17-29-55Z` locally if uncertain
- The health check workflow uses `docker compose up -d --build` at repo root — verify no other service depends on `latest` being mutable

**Post-merge action:** watch the health check workflow run on this PR as verification

---

### 5. PR #98 — E2E Temporal: remove legacy_amqp build tag
**What it fixes:**
- `backend/cmd/e2e_temporal/main.go`: removed `//go:build legacy_amqp` constraint
- The constraint excluded this file from normal Linux/amd64 builds, causing `go run ./backend/cmd/e2e_temporal` to find no package

**Depends on:** nothing (independent)

**Merge method:** rebase on post-#96 main, `--admin` merge

**What CI must show (pre-merge):**
- `E2E Temporal`: green (was failing because `go run` found no package to run on Linux/amd64 without the legacy_amqp tag)

**What to watch for:**
- The e2e test requires Docker (RabbitMQ + Temporal containers started by `scripts/e2e_temporal.sh`) — confirm the workflow can reach Docker in CI
- If Docker is unavailable in the CI environment, this test may be skipped or use a different runner

**Post-merge action:** none

---

### 6. PR #97 — init-db.sql: replace invalid string 'default' with proper UUIDs
**What it fixes:**
- `init-db.sql`: `INSERT INTO tenants (id, name) VALUES ('default', ...)` — `'default'` is not a valid UUID
- `init-db.sql`: tenant datasource inserts also used `'default'` for uuid columns
- Fixed with `00000000-0000-0000-0000-000000000001` and `00000000-0000-0000-0000-000000000002`

**Depends on:** nothing (independent)

**Merge method:** rebase on post-#98 main, `--admin` merge

**What CI must show (pre-merge):**
- `Integration - Gateway Roles`: green (was failing with "invalid input syntax for type uuid: default")

**What to watch for:**
- **Verify the fixed file is committed, not gitignored**: run `git show HEAD:init-db.sql | grep "00000000"` and confirm it returns the UUID fix, not the old `'default'` string
- If the file is in `.gitignore` or was applied via a different mechanism (e.g., a seed script), confirm the actual file in the repo has the UUID fix

**Post-merge action:** confirm Integration workflow is green on main

---

### 7. PR #101 — CUE deletion: remove dead cue/ + test + workflow steps
**What it fixes:**
- Deleted `cue/` (7 files, 659 lines): untouched since initial import 2026-06-26, zero runtime consumers
- Deleted `tests/cueValidation.test.ts`: not picked up by any test runner
- Removed `Setup CUE` and `Validate CUE pack` steps from `hybrid_ci.yml`

**Depends on:** #92 (peer-deps unblocked CI), #93 (clean backend signal), #94 (migration), #96 (health check), #97 (integration), #98 (e2e temporal)

**Merge method:** rebase on post-#97 main, `--admin` merge

**What CI must show (pre-merge):**
- `Hybrid Stack CI`: green — this is the **real verification** of the workflow edit. The first CI run of the edited `hybrid_ci.yml` on this PR proves the CUE step removal was clean (no dropped dependencies, no broken job graph)

**What to watch for:**
- The diff is small (9 files: 8 deletions + 1 workflow modification) — review the diff before merging to confirm nothing unexpected
- After merge: `hybrid_ci.yml` should have no reference to `cue` or `CUE` anywhere

**Post-merge action:** watch main run of Hybrid Stack CI — this is the definitive proof the workflow edit was clean

---

## Post-Batch Deploy Cycle

After all 7 PRs are merged:

```bash
# 1. Build from merged main (Linux amd64 binary — must cross-compile on macOS)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/uisce-server-fixed ./cmd/server/main.go

# 2. SCP to server
scp /tmp/uisce-server-fixed eganpj@100.84.50.65:/home/eganpj/uisce-server-fixed

# 3. Restart via the authoritative script (lives on the deployment mount)
ssh eganpj@100.84.50.65 "/mnt/github/uisce/START_BACKEND.sh"

# 4. Verify
# - /health returns {"status":"healthy"}
# - logs show "✅ SemanticMappingSvc: Gemini provider initialized"
# - logs show NO red bootstrap warning (if bootstrap succeeded — its clean absence is now meaningful)
```

**Observability check:**
- If `START_BACKEND.sh` prints the loud red bootstrap warning: bootstrap is failing — check `INFISICAL_TOKEN` in the shell environment on the server
- If no warning: bootstrap succeeded — the fail-loud signal is working as designed

---

## Dependency Summary

```
#92 ──┬── #93 ─── #94 ──┬── #96 ──┬── #98 ──┬── #97 ─── #101
      │                  │         │         │
      │                  │         │         └── (independent)
      │                  │         └── (independent)
      │                  └── (independent) ← #94 is migration; #96 is minio pin
      └── (unblocks CI signal for #93)
```

**About the chain:** #94 (migration), #96 (minio), #98 (build tag), #97 (UUID) are technically independent — the linear chain is a **serialization choice for clean signal attribution** (knowing exactly which merge introduced a regression), not a technical dependency. The rebase-before-merge gate between each is what keeps the attribution clean.

---

## Decision Record

- Merge order based on: CI signal dependency (some PRs fix CI, others need CI fixed to show clean signal), and attribution clarity (sequential merging of independent PRs so each post-merge main run is attributable to a single PR)
- All merges use `--admin` due to red CI baseline — deliberate, not silent
- Rebase-before-merge prevents branch drift and ensures each PR tests against the latest main
- PR #101 (CUE deletion) is last because its own workflow run is the real test of the edit
- #94 (migration BEGIN/COMMIT), #96 (minio pin), #98 (build tag), #97 (UUID) order is interchangeable — merged sequentially for attribution clarity only
