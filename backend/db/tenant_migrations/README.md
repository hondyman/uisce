# Tenant migrations

Schema migrations for a **tenant's own database** (ADR-030), one directory per app:

    backend/db/tenant_migrations/<app>/NNNN_name.up.sql

Applied by `internal/migrations.TenantRunner` to a target `tenant:<tenant id>:<app>`.
`alpha` is NOT migrated from here; it keeps using `backend/db/migrations` and `ApplyMigrations`.

Rules the runner enforces:

- **Append-only.** An applied file is never edited or removed (sha256 is recorded in the
  database's `ivy_meta.migration_log`). A new file must sort *after* every applied one.
- **Drift stops the target.** An edited, missing or out-of-order file makes the runner refuse to
  apply anything and report it, rather than warn and skip as the alpha runner does.
- **No COMMIT / ROLLBACK** in a file; the runner wraps each file and its log row in one
  transaction.
- A file that fails rolls back whole; a later run resumes from it. A never-applied file may be
  corrected.

There are no app directories yet: a tenant database starts as a clone of the gold-copy schema, and
the saga records that clone's stand with `TenantRunner.Baseline`.
