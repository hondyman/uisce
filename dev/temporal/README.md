Dev Temporal + seeding

This folder contains a simple docker-compose and a seed script you can use to bring up Temporal locally and seed the local `alpha` DB with a demo tenant and datasource for running tests and smoke checks.

Quick start

1. Start Temporal and its Postgres (will expose Temporal on 7233 and DB on 5433):

```bash
cd dev/temporal
docker compose up -d
```

2. Run the seed script (it will insert demo tenant/datasource into your alpha DB):

```bash
ALPHA_DB_URL="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" \
TEMPORAL_CLI="docker run --rm --network host temporalio/cli:1.27.0" \
./dev/seed/seed_db.sh
```

Notes

- The docker-compose uses the `temporalio/auto-setup` image and a Postgres instance for Temporal. This is for local development only.
- The seed script uses `psql` and expects `pg_isready`/`psql` to be on your PATH.
- The seed script is intentionally conservative and uses `ON CONFLICT` so it is safe to re-run.
- You may need to adapt the seed SQL if your database schema differs from the one assumed by this project.
