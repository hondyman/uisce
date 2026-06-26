# Semlayer Development Port Configuration

## Port Assignments (Always Consistent)

| Service | Port | URL | Description |
|---------|------|-----|-------------|
| Frontend (Vite) | 5173 | http://localhost:5173 | React application |
| Dev Proxy | 5175 | http://localhost:5175 | Development API proxy |
| Backend (Docker) | 8080 | http://localhost:8080 | Go backend service |
| API Gateway (Docker) | 8001 | http://localhost:8001 | Go API gateway |
| Hasura (Docker) | 8081 | http://localhost:8081 | GraphQL engine |
| Swagger (Docker) | 8082 | http://localhost:8082 | API documentation |

## Development Ports (Local)
| Service | Port | URL | Description |
|---------|------|-----|-------------|
| Backend (Local Dev) | 9090 | http://localhost:9090 | For direct backend development |

## Data Flow

```
Frontend (5173) 
    ↓ 
Dev Proxy (5175) 
    ↓ 
API Gateway (8001) 
    ↓ 
Backend (8080) ←→ Hasura (8081)
```

## Quick Start

```bash
# Start all services
./scripts/start-services.sh

# Stop all services  
./scripts/stop-services.sh
```

## Manual Commands

```bash
# Stop all services and clean ports
./scripts/stop-services.sh

# Start Docker services only
docker-compose up -d

# Start development proxy
cd frontend/dev-tools && node dev-proxy.cjs

# Start frontend
cd frontend && npm run dev
```

## Environment Variables

The dev-proxy uses these environment variables (with defaults):
- `API_TARGET=http://localhost:8001` (API Gateway)
- `CATALOG_TARGET=http://localhost:8080` (Backend directly for catalog)
- `PORT=5175` (Dev proxy port)

## Notes

- All ports are cleaned up before starting to ensure no conflicts
- Docker services run the production-like setup
- Dev proxy forwards frontend API calls to the appropriate backend services
- Frontend always runs on port 5173 for consistency
