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

## A second cousin: different directories, not directory lag

applied_at clusters can also surface directory mismatches between
processes. Sept 3 2026: 17_catalog stamped `2026-09-02 23:40:58Z`
while 16/17_feedback/18 stamped `2026-09-03 02:22:42Z`. File birth
times (`stat -f '%SB'` in UTC; local EDT would inflate offsets by
4h if naively compared to UTC applied_at):

| File                      | Birth (UTC)         | Existed at 23:40Z? |
|---------------------------|---------------------|--------------------|
| 20261016                  | 2026-09-02 21:38Z   | Yes — 2h02m before |
| 17_feedback               | 2026-09-02 23:18Z   | Yes — 22min before |
| 17_catalog                | 2026-09-02 23:33Z   | Yes — 7min before  |
| 20261018                  | 2026-09-03 02:08Z   | No — born AFTER    |

18's non-application at 23:40Z needs no hypothesis — it didn't exist
yet. The remaining anomaly: 17_catalog (7min old) was applied while
16 (2h02m old) and 17_feedback (22min old) were skipped. The 7-minute
gap favors a workstream-scoped migrations directory — the 23:40Z
process restarted for the catalog rollout, saw only its own files.
Inspect each runner process's working directory; correlate with the
git state of that directory at the recorded applied_at. Distinct
clusters → distinct directories → look at process mounts, not "lag."

## A third cousin: concurrent writers to a shared migrations dir

The migrations directory has concurrent writers. Two listings of the
same glob (`ls backend/db/migrations/20260903_*.sql | cat -n`), taken
minutes apart during one working session, returned 5 files and then 9.
At listing time the 4 new files were untracked WIP from a
concurrent workstream — untracked means not in HEAD, present on
disk, golang-migrate sees them. Between the two listings, those
files were committed by a parallel agent (commit `054ed3f31` —
"retire legacy pipelines table" — explains the two
`migrate_legacy_pipelines.{up,down}.sql`; commit `a2cd0d245` —
"renumber semantic AI bridge" — explains the `20260903_001_create_semantic_ai_bridge.up.sql`
flip from untracked to tracked). The lesson is stronger than the
listing alone reveals: tracked status can flip *twice within one
session*, and the same file can be both untracked (working-tree
arithmetic) and committed (HEAD arithmetic) at different capture
moments. Re-verify directory contents AND `git status` before
recording any count in a permanent artifact. Counts, SHAs, tracked
status, and directory contents are all point-in-time.

