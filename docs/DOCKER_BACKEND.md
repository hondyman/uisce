# Running the backend services (without Postgres)

This document explains how to run the backend-only development stack without starting a Postgres container.

Files:
- `docker-compose.backend.yml` - canonical compose file (does NOT start Postgres). Services talk to host Postgres via `host.docker.internal`.
- `.env.example` - example environment variables. Copy to `.env` and edit as needed.

Prerequisites
- Docker Desktop running on macOS (or any Docker engine where `host.docker.internal` resolves to the host).
- A Postgres instance running on your host configured with the databases and credentials referenced by `docker-compose.backend.yml` (default: `postgres:postgres`, DB `alpha`, and temporal DB `temporal`).

docker compose -f docker-compose.backend.yml up -d
Start the stack

```bash
# (from repository root)
cp .env.example .env   # edit .env if you need different creds
# Full stack (this will build many services and can be slow):
docker compose -f docker-compose.backend.yml up -d

# Fast dev mode — start only the minimal services required for backend development
# (Hasura, RabbitMQ, Backend). This avoids building all microservices:
./scripts/docker-start.sh up-minimal
```

docker compose -f docker-compose.backend.yml run --rm runner "./migrate"
Run a one-off script (example: run migrations using the migrate binary built into the backend image)

```bash
docker compose -f docker-compose.backend.yml run --rm runner "./migrate"
```

Open a shell in the backend image

```bash
docker compose -f docker-compose.backend.yml run --rm runner "sh"
```

Notes and quick checks

- If your host Postgres uses different credentials or different DB names, update `.env` accordingly.
- If Docker refuses to reach `host.docker.internal`, ensure Docker Desktop is running and that your Docker engine supports that DNS name (macOS does by default).
- Ensure required databases/users exist on your host Postgres. Quick idempotent SQL you can run locally (adjust creds as needed):

```sql
-- Connect as the postgres superuser and run:
CREATE DATABASE IF NOT EXISTS alpha;
CREATE DATABASE IF NOT EXISTS temporal;
DO $$
BEGIN
	IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'temporal') THEN
		CREATE ROLE temporal WITH LOGIN PASSWORD 'temporal';
	END IF;
END$$;
GRANT ALL PRIVILEGES ON DATABASE temporal TO temporal;
GRANT ALL PRIVILEGES ON DATABASE alpha TO postgres;
```

- If you want the compose stack to start its own Postgres instead, see `infrastructure/docker/docker-compose.workflows.yml` which contains a full stack including Postgres.

Tips
- Use `./scripts/docker-start.sh up-minimal` for quick dev startups.
- Use `./scripts/docker-start.sh up` to bring up the full stack when you need all services.
- Consider adding a `Makefile` with shortcuts like `make up` and `make migrate` if this becomes your daily workflow.
