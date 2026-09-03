# Migration Drift Alarm

## Background

Uisce's authoritative migration bookkeeping is `oms.migration_log`
(populated exclusively by `internal/migrations/runner.go`). The runner is
**skip-on-mismatch**: when an applied `.up.sql` file's SHA-256 differs
from the value stored in `oms.migration_log`, the runner emits a warning
and skips re-execution. It never reconciles.

Two consequences:

1. **Applied migrations are write-once.** Any post-apply edit to a
   `.up.sql` produces a warning + skip on the next restart, not a
   re-apply. The correct way to alter an applied schema is via a new
   versioned migration file.
2. **`oms.migration_log.applied_at` is write-once.** No `UPDATE` or
   `DELETE` exists against that table anywhere in the repo (grep
   confirms: only `internal/migrations/runner.go` lines 101, 110 issue
   `INSERT`, and only on the no-existing-row path). The timestamp
   therefore proves first-apply time, not "last seen."

## The alarm

The skip-on-mismatch warning is the **only** signal the runner emits
when files and DB have diverged. Any future restart that logs

```
⚠️  WARNING: migration file X content has changed since it was applied; skipping
```

is a drift event requiring a new versioned migration. **The alarm is
unscoped**: the runner doesn't know which files the team cares about, so
a drift warning for *any* `.up.sql` is signal. This includes files
unrelated to the current change set — for example, the Sept 3 2026
restart of the FinOps work triggered an unrelated drift warning for
`20260731_calc_fields.up.sql`, which is a separate pre-existing drift
problem worth its own ticket.

## Responding to the alarm

When the warning appears in a server's startup log:

1. **Don't edit the file in place.** Under skip semantics, edits have no
   effect — the runner will keep warning and never re-apply.
2. **Create a new versioned migration file** with a higher
   `YYYYMMDD_NNN_name.up.sql` prefix. Apply it via the runner; it will
   execute because no row exists yet in `oms.migration_log` for that
   filename.
3. **If the file change is harmless** (e.g., a comment-only edit that
   produces a SHA mismatch but no schema change), the warning is
   safe to ignore — but the warning will appear at every restart
   until the file is reverted. The right action is to revert, not
   to suppress.

## Standing detector (recommended)

Grep the server's startup output on every restart:

```bash
journalctl -u uisce-server | grep -E 'WARNING: migration file|Applied migration:'
```

Or, for a one-shot capture:

```bash
./uisce-server 2>&1 | tee /tmp/startup.log \
  | grep -E 'WARNING: migration file|Applied migration:|prewarm startup sweep'
```

Expected outcomes after a clean restart (no drift introduced by this
deployment):

- **No `WARNING: migration file … content has changed since it was applied`**
  for any file the current change set touched.
- **No `✅  Applied migration:`** for any file the current change set
  introduced — the runner no-op'd because the SHA was already in
  `oms.migration_log`.
- **`[INFO] prewarm startup sweep: clean (no orphan PENDING rows)`** —
  the sweep ran (it lives at composition root, see commit history).
- **Absence of all three lines is ambiguous** — it means the sweep
  isn't wired, not that it succeeded. A wiring regression masquerading
  as clean output. If any line is missing where one was expected, the
  wiring is suspect.

## Provenance and write-once

The empirical proof that `applied_at` is write-once: query the same
`SELECT filename, sha256, applied_at FROM oms.migration_log WHERE filename
LIKE '…' ORDER BY applied_at` immediately **before and after** a
restart. Unchanged timestamps + unchanged row count prove write-once
behaviorally and prove the runner did not re-apply. This check is part
of every FinOps Predictive deployment.
