Local backend scaffold for development and smoke tests.

Run with Docker Compose (uses docker-compose.dev.yml):

```bash
docker compose -f docker-compose.dev.yml up --build
```

Or build and run locally:

```bash
cd backend/local
go build -o backend-local
HASURA_URL=http://localhost:8080/v1/graphql ./backend-local
```

## Using host Postgres with docker-start (USE_LOCAL_POSTGRES)

If you prefer to run Postgres locally instead of the Postgres container, the repo provides a helper flow.

1. Set `USE_LOCAL_POSTGRES=true` in your `.env.local` or environment. This prevents the Docker Compose stack from starting the `postgres` container.
2. The `docker-start.sh` script will include the `infrastructure/docker/docker-compose.yml` override so the backend can resolve the `graphql-engine` infra service. It also provides a small `docker-compose.backend.localdb.yml` override to update `HASURA_URL` for the backend when using the host DB.
3. If you see name resolution errors for `graphql-engine` inside the backend container, make sure the semlayer container is joined to the `semlayer_default` network created by `infrastructure/docker/docker-compose.yml` and restart the backend:

```bash
# from repo root
USE_LOCAL_POSTGRES=true ./docker-start.sh

# In some cases you need to re-attach the network:
docker network connect semlayer_default semlayer-semlayer-1 || true
```

4. Troubleshooting tips:
- Use `docker exec -it semlayer-semlayer-1 sh -c "getent hosts graphql-engine"` to confirm DNS.
- Use `docker exec -it semlayer-semlayer-1 sh -c "curl -v http://graphql-engine:8080/v1/version"` to ensure the backend can reach Hasura.
- If you get intermittent "connection reset by peer" errors, restart both `semlayer` and `graphql-engine` containers and re-run the curl tests. This is usually transient after the network changes.
