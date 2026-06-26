# Local Databases for Development

This guide explains how to run Postgres and Ignite locally for development while keeping the rest of the platform running in Docker Compose.

## Strategy
- Keep most services in Docker Compose (default `docker compose up -d`).
- Run Postgres and Apache Ignite on your host machine (recommended for low-latency local development), or run them in Docker only when you want (use the `local-db` profile).

## Options

1) Run Postgres on macOS with Homebrew

```bash
brew install postgresql
brew services start postgresql
# Create database and user
createdb -U postgres alpha || true
# If password needed:
# psql -U postgres -c "ALTER USER postgres WITH PASSWORD 'postgres';"
```

2) Run Postgres in Docker (optional)

```bash
docker run -d --name local-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=alpha -p 5432:5432 postgres:15-alpine
```

3) Run Apache Ignite locally (two ways)

- Quick (Docker):
```bash
# Runs Ignite server on host ports (10800 client, 8082 REST)
docker run -d --name local-ignite -p 10800:10800 -p 8082:8082 apacheignite/ignite:2.16.0
```

- Manual install: download a local distribution from https://ignite.apache.org and follow their start instructions.

## Configuration
- Containers that need to reach host-local Postgres/Ignite should point to `host.docker.internal`.
- The repository includes `docker-compose.override.yml` which already sets these environment variables for the `backend` service:
  - DATABASE_URL=postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable
  - IGNITE_ADDR=host.docker.internal:10800

## Compose Usage
- Default (do not start Postgres/Ignite in Docker):
```bash
docker compose up -d
```

- Start compose and also start Postgres/Ignite inside compose (for ephemeral/dev-only DBs):
```bash
docker compose --profile local-db up -d
```

## Notes & Troubleshooting
- If you are running Postgres locally, ensure it's listening on 0.0.0.0 or Docker's `host.docker.internal` can reach it.
- If Ignite is running locally, make sure the port `10800` (thin client) is available and that your local firewall allows connections.
- If you prefer the backend to run on your host (not in Docker), set the same env variables in your shell before starting the backend process.

Example env for host backend developers:
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
export IGNITE_ADDR="localhost:10800"
```

If you'd like, I can also add a small note to the repo README linking to this page. Let me know if you'd like me to do that.