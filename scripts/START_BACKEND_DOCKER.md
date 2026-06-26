Start backend Docker services (using local Postgres)

This file documents the recommended way to bring up the backend-related Docker services while running Postgres on your host (macOS).

Why
----
The repository's `docker-compose.backend.yml` is configured to point Hasura, Temporal and other services at `host.docker.internal:5432` so containers will use the Postgres running on your machine. The helper script `./scripts/start-backend.sh` performs a pre-flight probe to ensure the host Postgres is reachable before starting containers.

Usage
-----
# Start backend docker services (build + detach)
./scripts/start-backend.sh

# If you run Postgres on a different host/port
LOCAL_PG_HOST=localhost LOCAL_PG_PORT=5432 ./scripts/start-backend.sh

# Skip the pre-flight Postgres check (not recommended)
SKIP_PG_CHECK=1 ./scripts/start-backend.sh

Notes
-----
- The script checks TCP connectivity to `LOCAL_PG_HOST:LOCAL_PG_PORT` (defaults to `host.docker.internal:5432`) and will exit early if it cannot connect. This avoids bringing up dependent services that will immediately fail.
- Ensure your Postgres accepts connections from Docker (check `listen_addresses` and `pg_hba.conf`).
- To stop services started by the compose file run: `docker compose -f docker-compose.backend.yml down` or `./scripts/docker-start.sh down`.

Troubleshooting
---------------
- If containers cannot connect to Postgres, verify that `host.docker.internal` resolves from your Docker environment (Docker Desktop on macOS sets this up by default).
- If you prefer containers to run a Postgres container instead of the host DB, modify `docker-compose.backend.yml` or use a different compose file that defines a Postgres service.
