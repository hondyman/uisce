# Scripts helper

This directory contains helper scripts to start and stop local services used during development.

## start-backend-local.sh

A convenience wrapper to start the backend locally with safety guards:

- Kills stale `.backend.pid` if present
- Kills any process listening on the configured `PORT` (default 8080)
- Starts backend using `go run ./cmd/server` when `go` is available
- Falls back to running `./server` binary if present
- Writes logs to `logs/backend_<TIMESTAMP>.log` and pid to `.backend.pid`

Usage:

```bash
# Start backend on default port 8080 (interactive, tails log)
./scripts/start-backend-local.sh

# Start backend on a different port
PORT=29080 ./scripts/start-backend-local.sh

# Run in background (nohup)
PORT=8080 nohup ./scripts/start-backend-local.sh >/dev/null 2>&1 &
```

## create_semantic_terms_and_edges.sql

Creates representative semantic term nodes (node_type_id = 820b942a-9c9e-4abc-acdc-84616db33098) for database columns and links them using the "mapped to" edge (97d82101-2b84-47a6-9ec0-f930fe389c3c). See the `scripts/create_semantic_terms_and_edges.sql` script for details and run instructions.

Usage example:

```bash
psql -d postgres://postgres:postgres@localhost:5432/alpha \
	-v tenant_id='00000000-0000-0000-0000-000000000000' \
	-v tenant_datasource_id='11111111-1111-1111-1111-111111111111' \
	-f scripts/create_semantic_terms_and_edges.sql
```

### Alternative: use the backend API for single column mapping

If you only need to create a semantic term for one column and map it, the backend exposes an endpoint that mimics the script logic and is safer for single rows:

POST /api/semantic-mappings/apply-custom

Body: { "column_node_id": "<COLUMN_NODE_ID>", "semantic_term_name": "<TERM_NAME>" }

This call will create the semantic term if it doesn't exist and create the mapping edge. It uses the server config for edge types (default `99c86836-...`) rather than the hard-coded `97d82101-...` used by the bulk SQL script.

This script is also used by `START_FULL_SYSTEM.sh` when present, providing a single, robust backend starter for both manual and full-system runs.

## sync_cube_tenants.go

Generates the tenant metadata snapshot consumed by Cube:

- Reads `tenant_product_datasource` (and related tables) using `DATABASE_URL` / `ALPHA_DB_URL`.
- Writes `cube/generated/tenant-scopes.json`, which Cube loads for `scheduledRefreshContexts`, QoS routing, and header validation.
- Materializes any `schema_overrides` JSON into `cube/schema/tenants/<tenant>/<datasource>/auto/*.yml` so the `repositoryFactory` automatically overlays tenant files on top of the base schema.

Usage:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable \
	go run ./scripts/sync_cube_tenants.go

# or via Make
make sync-cube-tenants
```

The generated files are ignored by git (except for the `.gitkeep` placeholders) and can be safely regenerated at any time.
