# SemLayer - Docker-Only Setup

This document explains how to run SemLayer entirely in Docker Compose with only PostgreSQL running locally.

## Quick Start

```bash
# 1. Ensure PostgreSQL is running locally
# psql postgres://postgres:postgres@localhost:5432/alpha

# 2. Start all services in Docker
./start-docker.sh

# 3. Access the services
# - API Gateway: http://localhost:8001
# - Frontend: npm start (in frontend directory)
# - Hasura Console: http://localhost:8080
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Your Machine                         │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   PostgreSQL (Local)                  │  │
│  │                   localhost:5432                      │  │
│  └───────────────────────────────────────────────────────┘  │
└──────────────────────────┬──────────────────────────────────┘
                           │ (TCP: host.docker.internal:5432)
┌──────────────────────────┴──────────────────────────────────┐
│                      Docker Network                         │
│                   (semlayer-network)                        │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │               API Gateway (8001)                     │  │
│  │            (Forwards requests to services)          │  │
│  └──────────────────────────────────────────────────────┘  │
│    ↓                  ↓                  ↓                   │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ Backend    │  │ Hasura     │  │ Temporal   │            │
│  │ (8080)     │  │ (8080)     │  │ (7233)     │            │
│  └────────────┘  └────────────┘  └────────────┘            │
│    ↓              ↓                                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ Fabric     │  │ RabbitMQ   │  │ Semantic   │            │
│  │ Builder    │  │ (5672)     │  │ Engine     │            │
│  │ (8081)     │  └────────────┘  └────────────┘            │
│  └────────────┘                                             │
│                         ↓                                    │
│                   PostgreSQL (local)                        │
│                   via host.docker.internal                  │
└──────────────────────────────────────────────────────────────┘
```

## Services

### Infrastructure

| Service | Port | Image | Role |
|---------|------|-------|------|
| PostgreSQL | 5432 | Local | Primary data store |
| Hasura | 8080 | hasura/graphql-engine:v2.46.0 | GraphQL API layer |
| Temporal | 7233 | temporalio/auto-setup:1.22.0 | Workflow orchestration |
| Temporal UI | 8088 | temporalio/ui:2.21.3 | Temporal dashboard |
| RabbitMQ | 5672,15672 | rabbitmq:3-management | Message broker |

### Application Services

| Service | Port | Role |
|---------|------|------|
| API Gateway | 8001 | Request routing & authentication |
| Backend | 8080 | Business logic & APIs |
| Fabric Builder | 8081 | Semantic fabric management |
| AI Builder | 8082 | AI-powered features |
| Semantic Engine | 8083 | Semantic metadata management |
| Governance | 8084 | Governance & compliance |
| Compliance Engine | 8085 | Compliance checks |

## Commands

### Start Services

```bash
# Automatic (recommended)
./start-docker.sh

# Manual
docker-compose up -d
```

### Stop Services

```bash
# Clean stop
./stop-docker.sh

# Or manually
docker-compose down

# Remove volumes too
docker-compose down -v
```

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api-gateway
docker-compose logs -f backend
docker-compose logs -f hasura

# Last N lines
docker-compose logs --tail=50 api-gateway

# Follow with timestamps
docker-compose logs -f --timestamps
```

### Manage Services

```bash
# Status
docker-compose ps

# Restart a service
docker-compose restart api-gateway

# Restart all services
docker-compose restart

# Stop a service
docker-compose stop backend

# Start a service
docker-compose start backend

# Rebuild a service
docker-compose build --no-cache api-gateway

# Execute command in container
docker-compose exec api-gateway curl http://localhost:8001/health
```

## Configuration

### Environment File (.env)

The `start-docker.sh` script automatically creates `.env` with defaults:

```bash
# View current settings
cat .env

# Modify a setting
sed -i '' 's/KEY=.*/KEY=new-value/' .env

# Add a new setting
echo "NEW_KEY=value" >> .env

# Apply changes
docker-compose restart
```

### Common Customizations

```bash
# Change API Gateway port
sed -i '' 's/API_GATEWAY_HOST_PORT=.*/API_GATEWAY_HOST_PORT=9000/' .env
docker-compose up -d

# Disable authentication for development
sed -i '' 's/DEV_ALLOW_UNAUTH_FABRIC=.*/DEV_ALLOW_UNAUTH_FABRIC=true/' .env
docker-compose restart

# Enable IP whitelist
sed -i '' 's/IP_WHITELIST_ENFORCE=.*/IP_WHITELIST_ENFORCE=true/' .env
docker-compose restart
```

## PostgreSQL Setup

### Local PostgreSQL

```bash
# Start PostgreSQL (macOS with Homebrew)
brew services start postgresql@15

# Verify it's running
psql -h localhost -U postgres -d alpha -c "SELECT 1"

# Create database if needed
createdb -h localhost -U postgres alpha

# Connect
psql postgres://postgres:postgres@localhost:5432/alpha
```

### Docker PostgreSQL (Alternative)

If you prefer PostgreSQL in Docker:

```bash
# Start PostgreSQL container
docker run -d \
  --name semlayer-postgres \
  --network semlayer_semlayer-network \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=alpha \
  -p 5432:5432 \
  postgres:latest
```

Then modify docker-compose.yml:
- Change `POSTGRES_HOST=host.docker.internal` to `POSTGRES_HOST=postgres`
- Restart services

## Frontend Integration

### Connect Frontend to Services

1. Start the frontend dev server:

```bash
cd frontend
npm install
npm start
```

2. Configure tenant context in browser localStorage:

```javascript
// In browser console
localStorage.setItem('selected_tenant', JSON.stringify({
  id: '00000000-0000-0000-0000-000000000000',
  display_name: 'Default Tenant'
}));
localStorage.setItem('selected_datasource', JSON.stringify({
  id: '11111111-1111-1111-1111-111111111111',
  source_name: 'Default Datasource'
}));
// Reload page
window.location.reload();
```

3. Access the frontend: http://localhost:5173

## Troubleshooting

### Services Won't Start

```bash
# Check Docker is running
docker ps

# Check PostgreSQL is accessible
psql postgres://postgres:postgres@localhost:5432/alpha

# View startup errors
docker-compose logs --tail=100
```

### Port Conflicts

```bash
# Find process using port
lsof -i :8001

# Kill process
kill -9 <PID>

# Or change port in .env
echo "API_GATEWAY_HOST_PORT=9001" >> .env
docker-compose up -d
```

### Service Won't Connect to PostgreSQL

```bash
# Verify PostgreSQL is accessible from Docker
docker-compose exec api-gateway curl -v postgres://host.docker.internal:5432

# Check POSTGRES_HOST environment variables
docker-compose exec backend env | grep POSTGRES

# Test connection
docker-compose exec backend psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "SELECT 1"
```

### Rebuild After Code Changes

```bash
# Rebuild specific service
docker-compose build --no-cache backend

# Restart service
docker-compose up -d backend

# Or full rebuild and restart
docker-compose build --no-cache && docker-compose up -d
```

### Clear Everything and Start Fresh

```bash
# Stop and remove everything
docker-compose down -v

# Remove unused images
docker image prune -a -f

# Start fresh
./start-docker.sh
```

## Production Deployment

For production, see:
- `PRODUCTION_README.md` in the backend directory
- Docker Hub registry configuration
- Kubernetes manifests (if applicable)
- Environment-specific configuration

## Support

For detailed information:
- See `API_GATEWAY_STARTUP_GUIDE.md` for API Gateway details
- See `agents.md` for tenant scoping information
- See individual service READMEs
- Check logs: `docker-compose logs -f <service>`
