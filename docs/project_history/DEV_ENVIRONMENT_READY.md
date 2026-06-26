# Development Environment Ready 🚀

## Current Status: ALL SYSTEMS GO ✅

### Backend Services (Running in Docker)
```bash
./scripts/start-backend.sh
```
- **Redpanda (Pandaproxy)**: http://localhost:8082 (Pandaproxy)  
  (broker: localhost:9092)
- **Hasura GraphQL**: http://localhost:8080
- **Event Router**: http://localhost:8081
- **Backend Service**: http://localhost:29080 (placeholder)

### Frontend Dev Server (Running)
```bash
cd frontend && npm run dev
# or use
./scripts/start-frontend.sh
```
- **Vite Dev Server**: http://localhost:5173/

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend (Vite)                         │
│                  http://localhost:5173                      │
└───────────────────────┬─────────────────────────────────────┘
                        │
          ┌─────────────┼─────────────┐
          │             │             │
    ┌─────▼────┐   ┌────▼────┐  ┌───▼──────┐
    │  Backend │   │ Hasura  │  │ Event    │
    │  Service │   │ GraphQL │  │ Router   │
    │:29080    │   │ :8080   │  │ :8081    │
    └─────┬────┘   └────┬────┘  └───┬──────┘
          │             │           │
          └─────────────┼───────────┘
                        │
                  ┌─────▼──────┐
                  │  RabbitMQ  │
                  │   :5672    │
                  │   :15672   │
                  └────────────┘
                        │
                        ▼
                  ┌──────────────┐
                  │   Local      │
                  │  PostgreSQL  │
                  │  (alpha DB)  │
                  └──────────────┘
```

## Database Connection
- **Host**: `host.docker.internal` (from containers) / `localhost` (from host)
- **Port**: 5432
- **Database**: alpha
- **User**: postgres
- **Password**: postgres

## Key Configuration Files

| File | Purpose |
|------|---------|
| `docker-compose.backend.yml` | Docker Compose for backend services |
| `.env` | Environment variables (Hasura secrets, JWT config) |
| `config.yaml` | Application configuration |
| `scripts/start-backend.sh` | Start backend services script |
| `scripts/start-frontend.sh` | Start frontend dev server script |

## Next Steps

1. **Test the integration**: Open http://localhost:5173 in your browser
2. **Configure Hasura**: Add database tables and metadata at http://localhost:8080
3. **Verify tenant scoping**: Use the Fabric Builder tenant selector (see agents.md for details)
4. **Test API calls**: Frontend should make requests through the tenant scope shim

## Troubleshooting

### Services won't start
```bash
# Clean up and restart
docker compose -f docker-compose.backend.yml down
./scripts/start-backend.sh
```

### PostgreSQL not reachable
Ensure local Postgres is running:
```bash
# Check if Postgres is listening
lsof -i :5432
```

### Port conflicts
If ports are in use, modify `docker-compose.backend.yml` or `frontend/scripts/start-dev.sh`

### Frontend not hot-reloading
Check that Vite is running on port 5173 and accessible
