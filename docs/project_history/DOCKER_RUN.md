# Starting services (frontend/backend)

This repository provides convenience scripts to start frontend and backend environments.

Prerequisites
- Docker and Docker Compose (Docker Desktop) installed
- Node.js and npm (for frontend dev)
- Local Postgres running on host:5432 (alpha DB)
- RabbitMQ (the backend script will start it in Docker)

Start Backend (includes RabbitMQ and backend microservices)

```bash
# from repository root
./scripts/start-backend.sh
```

This will use `docker-compose.backend.yml` which expects a local Postgres accessible via `host.docker.internal`.

Start Frontend (dev server)

```bash
# from repository root
./scripts/start-frontend.sh
```

This runs the frontend dev server using `npm run dev`.

Notes
- If you prefer to use docker-compose multi-file configs, you can run:
  docker compose -f docker-compose.yml -f docker-compose.observability.yml up --build -d

- The `.env` file contains DATABASE_URL_DOCKER and DSN_DOCKER values used by containers to reach host Postgres.

Troubleshooting
- If the frontend doesn't start, delete `node_modules/.vite` and try again (common Vite issue):
  rm -rf frontend/node_modules/.vite

- If RabbitMQ UI is not reachable, check `docker compose -f docker-compose.backend.yml ps` and view logs with:
  docker compose -f docker-compose.backend.yml logs -f rabbitmq

