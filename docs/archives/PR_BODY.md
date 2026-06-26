PR: chore/triage-u1000-shims

Summary

This branch contains a set of safe, non-destructive compatibility fixes to tolerate Phase-B schema changes (promoting `config` in catalog types).

What changed

- backend/migrations/002600_backfill_catalog_edge_types_config.sql
  - Idempotent migration that ensures `catalog_edge_types.config` exists and backfills from `properties` when present; creates a GIN index on config.
- backend/internal/api/node_types_routes.go
  - Selects now prefer `config` but fall back to `properties` via COALESCE for compatibility.
- backend/internal/api/glossary_handler.go
  - SELECT/RETURNING use COALESCE for `config`/`properties` textual scanning.
- phaseb_playbook.sql, phaseb_final_cleanup.sql, backend/totalddl.sql
  - Example and node view DDL adjusted to prefer `config` with a `properties` fallback.
- backend/internal/api/validation_triggers_handlers_test.go
  - Fixed unit test mock rule types to match validation engine expectations (cardinality vs field_format).

What I ran/verified

- Applied the migration locally against the `alpha` DB.
- Started the backend locally (port 8085) and ran tenant-scoped smoke checks:
  - GET /api/edge-types?tenant_id=...&datasource_id=...  => 200 OK
  - GET /api/node-types?tenant_id=...&datasource_id=...  => 200 OK
- Ran focused unit tests for the failing validation test and fixed it locally.

Remaining items / follow-ups

- Full repository `go test ./...` fails at the top-level build due to duplicate `main` declarations in root-level files (false-positive for this PR). Recommend running CI or a scoped test target (backend packages) for CI gating.
- Decide how to handle `catalog_edge_types.properties` situation (edge view): either add a light migration to expose `properties` or rely on the `config` backfill migration; this PR uses the latter approach for safety.
- Consider a cautious repo-wide sweep for `config` occurrences across front-end docs and other services — do in small batches.

How to review/run locally

1. Apply DB migration (idempotent):

   psql "postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable" -f backend/migrations/002600_backfill_catalog_edge_types_config.sql

2. Start server locally:

   PORT=8085 DSN='postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable' go run ./backend/cmd/server

3. Run smoke checks (example):

   curl -H "X-Tenant-ID: <TENANT_ID>" -H "X-Tenant-Datasource-ID: <DATASOURCE_ID>" "http://localhost:8085/api/node-types?tenant_id=<TENANT_ID>&datasource_id=<DATASOURCE_ID>"

Notes

- The branch commit is on `chore/triage-u1000-shims` (local). If pushing fails due to permissions, you can open a PR manually after pushing.
- I can push the branch and open a PR for you if you want me to try (may require Git credentials/CLI access).