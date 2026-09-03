# FinOps Predictive: Verification Record

This file is the durable home for the runtime evidence produced while
shipping the FinOps Predictive feature. Captured Sept 3 2026 against
live `alpha` (database `100.84.50.65:5432/alpha`, accessed via libpq
mTLS env vars against `~/.uisce/certs/postgres-client.{crt,key}`).
Live-database access used:
```bash
export PGHOST=100.84.50.65 PGPORT=5432 PGUSER=postgres \
       PGSSLMODE=verify-full \
       PGSSLCERT="$HOME/.uisce/certs/postgres-client.crt" \
       PGSSLKEY="$HOME/.uisce/certs/postgres-client.key" \
       PGSSLROOTCERT="$HOME/.uisce/certs/ca.crt"
```

## Live DB authoritative: alpha

Disambiguation — `to_regclass` against both DBs plus a polaris probe
(from `postgres` DB so the probe doesn't depend on polaris hba state):

| DB        | finops.prewarm_execution_ledger | oms.migration_log | public.schema_migrations | polaris (via pg_database) |
|-----------|---------------------------------|-------------------|--------------------------|---------------------------|
| alpha     | present                         | present           | present (empty)          | —                         |
| postgres  | absent                          | absent            | present                  | —                         |
| polaris   | —                               | —                 | —                        | EXISTS                    |

Three-part query output:
```
alpha:    finops.prewarm_execution_ledger, oms.migration_log, public.schema_migrations
postgres: —, —, schema_migrations
polaris:  EXISTS
```

The deployed DATABASE_URL points at `alpha` (mTLS DSN, cert material
at `~/.uisce/certs/` — verified one actual connection succeeds as
`current_database=alpha`, `current_user=postgres`).

## SHA + applied_at reconciliation (all four files, byte-for-byte)

Captured at `4.2/4.4` — before/after the server restart, so the
write-once proof has both static (grep) and empirical (before/after
diff) legs.

```
20261017_create_catalog_view_definitions.up.sql   2e512e00cd6825b4b0dd54e8a1f74a4371bce5aa85294d7dcda4e9d5cc6586c3   2026-09-02 23:40:58.567965+00
20261016_predictive_finops_and_smoothing.up.sql   903e2ec5e74b38cc5cb68652a8f2230e976023eee851db17d5f4bc404a15af8a   2026-09-03 02:22:42.233573+00
20261017_forecast_feedback_loop.up.sql            b5e5e589772728a2464d1b3b9dcf974304f5c9d910f152f5ef7b71f7b4978523   2026-09-03 02:22:42.392472+00
20261018_prewarm_job_tracking.up.sql              02d59a00ffa84b597be7ed517184bd0bde7c10b70138738be85d4a265b43dc0b   2026-09-03 02:22:42.498673+00
```

Empirical before/after diff: empty (write-once proven behaviorally;
no re-apply happened between captures). Static proof: `grep -rn
migration_log backend/ --include="*.go"` returns only two
write sites — `internal/migrations/runner.go:101` and `:110`, both
INSERT paths on the no-existing-row branch, with zero UPDATE or
DELETE statements on `oms.migration_log`. Combined, the empirical
diff and the static grep are the two-way write-once proof.

DBeaver reconciliation (hypothesis, not machine-provable):
`oms.migration_log` is written only by the runner; the round-1
walkthrough evidence is consistent with both DBeaver-applied
(runner no-op'd via `IF NOT EXISTS` guards at the runner's first
pass today) and runner-applied (file content in working tree at
the moment of apply). applied_at cannot distinguish because it
records the runner's stamp regardless of which path the schema
followed to exist. Marked as inference, not finding.

## Live smoke test

Triggered via dashboard click + dev-token JWT (token issued by
`backend/cmd/devjwt` with `admin,finops_manager` roles). Response
captured:
```
HTTP 202
{
  "jobId": "677f5a15-36b3-4f8c-8ff9-5d309c3b4f43",
  "status": "PENDING",
  "tenantId": "99e99e99-99e9-49e9-89e9-99e99e99e999"
}
```

SQL verify at T+2s, T+5s, T+10s (`SELECT ... WHERE job_id = '<jobId>'`):
```
job_id     : 677f5a15-36b3-4f8c-8ff9-5d309c3b4f43
status     : SKIPPED_BELOW_THRESHOLD
bo_id      : NULL
target_metric : ALL
completed_at  : 2026-09-03 12:22:40.507225+00 (executed_at 12:22:40.348924+00)
```

Proofs:
- **jobId continuity**: response `jobId` == SQL `job_id` byte-for-byte
- **Run-level UPDATE consistency**: terminal row has `bo_id IS NULL`
  AND `target_metric = 'ALL'` — the load-bearing Phase 0 fix proven
  on the live DB, not just in sqlmock
- **lifecycle**: PENDING → terminal in ~160ms (sub-second; PENDING
  capture would have missed terminal — the synchronous PENDING
  write is covered by
  `TestForecastHandler_PrewarmTrigger_WritesPendingAndReturns202`)

`SKIPPED_BELOW_THRESHOLD` is the optimal outcome, not a failure: today
is Sept 3, so "tomorrow" = Sept 4 (mid-month, multiplier 1.0, peak
probability 0, AND gate fails on `peak_probability < 0.70`). The
SKIPPED path routes through the run-level UPDATE branch — the exact
statement Phase 0 repaired.

## Server restart output (4.2)

```
main.go:49: Connected to database successfully
main.go:99: [INFO] AI Semantic Bridge ledger monitor: no SLO_SLACK_WEBHOOK_URL or PAGERDUTY_ROUTING_KEY configured — breaches will only be logged, not paged.
main.go:116: AI Semantic Bridge ledger monitor started (checking every 15m0s)
main.go:64: [INFO] prewarm startup sweep: clean (no orphan PENDING rows)
runner.go:80: ⚠️  WARNING: migration file 20260731_calc_fields.up.sql content has changed since it was applied; skipping
```

Three signals captured:
- **Prewarm sweep present**: `[INFO] prewarm startup sweep: clean
  (no orphan PENDING rows)` — wiring is alive
- **No `✅ Applied migration:` for any 2026101x file**: all four
  finops files were already in `oms.migration_log` before restart
- **One `⚠️ WARNING:` for `20260731_calc_fields.up.sql`**: the
  drift alarm fired on its first restart. See the triage branch
  below.

## 20260731_calc_fields.up.sql drift — triage

The Sept 3 restart emitted the standing drift alarm for this file.
Under skip-on-mismatch semantics, the file's current content has
**NEVER EXECUTED on live** — the schema change the editor intended
is silently unapplied. Three branches:

1. **Intent matches file current content + live schema matches** →
   capture as a new versioned migration (e.g.,
   `20260904_calc_fields_drift_fix.up.sql` with `IF NOT EXISTS`
   guards) and revert the file edit so the warning stops. The
   capturing migration's `IF NOT EXISTS` guards make it no-op where
   live already matches.
2. **Intent matches file current content + live schema does NOT
   match** → the change was never applied; capture the intent as
   a new versioned migration and revert the file edit.
3. **The change was hand-applied via psql/DBeaver-style direct
   DDL** (this repo's round-1 DBeaver pattern) → revert the file
   edit AND capture a migration that describes the
   post-hand-apply state, so live is in a state at least one file
   describes. Reverting alone stops the warning but leaves live
   in a state no file describes.

calc_fields is core schema — this is **triage**, not cleanup.
Captured in commit `3c90f2895` as the first post-commit action.

Tracking issue (paste-ready body): `docs/ops/.issues/20260731-calc-fields-drift.md`
(Replace the path with the GitHub issue URL after the issue is created.)

## ca.key (Uisce PKI signing private key) — escalation

Post-mTLS handoff reality, captured Sept 3:
- Files at `~/.uisce/certs/` on this host: `ca.crt`,
  `postgres-client.crt`, `postgres-client.key`,
  `postgres-client.pk8` — one role's client material (postgres)
  plus the CA cert.
- `find $HOME -name 'ca.key'` (Library-excluded; no Trash
  exclusion) returned only gRPC testdata fixtures
  (`/Users/eganpj/go/pkg/mod/google.golang.org/grpc@*/testdata/spiffe_end2end/ca.key`).
  `~/.Trash` was denied by macOS TCC (`ls ~/.Trash/` →
  "Operation not permitted").

Implications for this host:
- No future cert can be issued from here (rotation, re-issuance,
  new roles all impossible) — needs ca.key, which is not found
  in the captured scope.
- Issuance is frozen; verification works for postgres (the only
  role with cert material at `~/.uisce/certs/`). Other roles'
  certs live wherever the issuing host placed them
  (containers/mounts per the handoff) — verify-from-here is
  untested.

Required action (in order):
1. **Check Finder's Trash GUI** (or run `find` from a Full Disk
   Access terminal) — if `ca.key` is in `~/.Trash`, the
   escalation is a recovery (drag out, store durably).
2. If not in Trash: locate `ca.key` on the original issuing host
   and copy it to durable storage (encrypted backup, password
   manager, or Infisical). Without it, the PKI is frozen at its
   current 9-role surface and new roles require full PKI
   regeneration.

Captured in commit `b8103da33` (runner.go authoritative-
bookkeeping doc references the PKI gap); the Trash step itself is
not captured in the repo because macOS TCC denies the sandboxed
terminal access to `~/.Trash`.

Tracking issue (paste-ready body): `docs/ops/.issues/uiece-pki-ca-key-recovery.md`
(Replace the path with the GitHub issue URL after the issue is created.)

## Migration collision structure (committed tree at HEAD)

Captured via `git ls-files backend/db/migrations/2026{0903,1017}_*.sql`
at commit time of `3c90f2895`:

| Prefix   | Files committed | .up.sql | duplicate-version violations for golang-migrate |
|----------|-----------------|---------|---------------------------------------------------|
| 20260903 | 9               | 7       | 6                                                 |
| 20261017 | 3               | 2       | 1                                                 |

golang-migrate has been structurally unusable in this repo since
at least 20260903. The hash-runner (`internal/migrations/runner.go`,
`oms.migration_log`) is the permanent answer. Sub-numbered suffixes
(`_001`, `_002`, the earlier `_b_`) disambiguate *filenames* for
the hash-runner but not *versions* for golang-migrate — a `cmd/apply-migrations`
wrapper around the hash-runner is tracked as a B6 deliverable to
retire the CLI dependency for scratch + drift-detector workflows.

## Commit-time evidence trail

The work captured in this record is spread across the following
commits (chronological, oldest first):

- `a6ed54977` feat(finops): wire FinOps forecast and smoothing routes into SetupRouter
- `dc110685d` fix(finops): add CREATE SCHEMA IF NOT EXISTS finops
- `761bd575a` feat(semantic-layer): centralize compliance/MDM on rule engine
- `a2cd0d245` chore(migrations): renumber semantic AI bridge migration
- `fa4c9d63c` feat(finops): forecast feedback loop, prewarm job tracking, startup sweep
- `054ed3f31` feat(datapipeline): retire legacy pipelines table, wire BO CRUD trigger dispatch
- `c5d5f8998` chore: remove hardcoded plaintext DB credential fallbacks from cmd tools
- `b8103da33` docs: migration ownership notes, runbook/README updates; stop tracking pid files
- `5e23708db` feat(graph): port BOLineageGraphTab/LineageGraph/ImpactGraph onto CatalogGraph
- `3c90f2895` test(finops): close target_metric matrix; document drift cousins

The 5→9 file appearance mid-session was real concurrent work by a
parallel agent (commits `054ed3f31`, `a2cd0d245`) — see
`docs/ops/migration-drift-alarm.md` § "A third cousin" for the
operational lesson.
