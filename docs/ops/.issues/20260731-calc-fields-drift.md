# fix(finops): triage 20260731_calc_fields.up.sql drift — schema change never applied

## Summary

The standing drift alarm (`docs/ops/migration-drift-alarm.md`) detected
`migration file 20260731_calc_fields.up.sql content has changed since it
was applied; skipping` on the Sept 3 2026 server restart. Under
skip-on-mismatch semantics (`internal/migrations/runner.go:79-81`), the
file's current content has **NEVER EXECUTED on live** — the schema
change the editor intended is silently unapplied. `calc_fields` is
core schema; this is triage, not cleanup.

## Triage branches

1. **Intent matches file current content + live schema matches** →
   capture as a new versioned migration (e.g.,
   `20260904_calc_fields_drift_fix.up.sql` with `IF NOT EXISTS`
   guards) and revert the file edit so the warning stops. The
   capturing migration's `IF NOT EXISTS` guards make it no-op where
   live already matches.
2. **Intent matches file current content + live schema does NOT match**
   → the change was never applied; capture the intent as a new
   versioned migration and revert the file edit.
3. **The change was hand-applied via psql/DBeaver-style direct DDL**
   (this repo's round-1 DBeaver pattern) → revert the file edit AND
   capture a migration that describes the post-hand-apply state, so
   live is in a state at least one file describes. Reverting alone
   stops the warning but leaves live in a state no file describes.

## Acceptance

- No `⚠️  WARNING:` line for `20260731_calc_fields.up.sql` in the
  next server startup output.
- Either the schema state is captured in a versioned migration
  (branches 1 or 2) or the file revert is paired with a
  state-describing migration (branch 3).

## Reference

- Verification record: `docs/ops/finops-predictive-verification.md`
  § "20260731_calc_fields.up.sql drift — triage"
- Drift alarm doc: `docs/ops/migration-drift-alarm.md`
- First surfaced: `3c90f2895` (matrix close commit), then
  `c0a614ae8` (verification record commit)
