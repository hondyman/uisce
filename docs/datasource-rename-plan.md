# Datasource Rename Plan (tenant_instance_id → datasource_id)

Overview
- Goal: Hard-rename occurrences of `tenant_instance` → `datasource` across the stack (headers, query params, GraphQL/Hasura metadata, DB column names, Go code, and frontend) and add tests + CI to make the change safe and auditable.

Scope
- Backend (Go): replace header `X-Tenant-Instance-ID` -> `X-Tenant-Datasource-ID`, change query/payload keys `tenant_instance_id` -> `datasource_id`, update models and SQL, add unit and integration tests.
- DB: rename columns `tenant_instance_id` -> `datasource_id` using migration scripts (draft in `backend/migrations/20260207_rename_tenant_instance_to_datasource.sql`).
- Hasura/GraphQL: update metadata and queries to use `datasource_id` variables and relationships.
- Frontend: already partially updated — ensure all call sites use `X-Tenant-Datasource-ID` and `datasource_id`.
- CI: add workflows to run backend unit tests and integration tests, and E2E runs for UI.

Phased Rollout (strict cutover)
1. Implement backend code changes and tests (done: initial set of handler updates + tests).
2. Add DB migration scripts (DRAFT created) and schedule a maintenance window for schema changes if needed.
3. Update Hasura metadata and migrations for GraphQL variable names.
4. Update frontend to the final naming and run full E2E tests in staging.
5. Run CI, deploy backend + database migration, and then deploy frontend.
6. Validate in staging then production.

Testing & CI
- Unit tests: Add tests for header enforcement and parsing in handlers.
- Integration tests: Spin up test DB/Hasura instances and exercise key API flows.
- E2E tests: Cypress tests to validate BO listing, creation, and semantic mapping flows using the new headers.
- CI: New GitHub Actions workflow `datasource-rename-backend.yml` (created) to run backend tests.

Rollback Plan
- Keep DB snapshot before migration.
- If issues occur, rollback migration by restoring DB snapshot and revert PRs.

Notes & Next steps
- Finish replacing header usage across all Go files and update error messages to reference `X-Tenant-Datasource-ID` (in progress).
- Finalize and test DB migration thoroughly (indexes, FK constraints, unique constraints updates).
- Update Hasura metadata and re-run metadata apply in staging.
- Consolidate frontend changes and add E2E tests.

Please review and confirm next action: continue making code changes across the rest of the backend (replace all header occurrences and tests), or switch to drafting the DB/Hasura migration plan in detail first.