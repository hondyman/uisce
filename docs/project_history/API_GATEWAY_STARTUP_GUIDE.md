# API Gateway - Startup Guide (Docker-Only)

This guide explains how to run the API Gateway and all SemLayer services in Docker Compose. Only PostgreSQL runs locally on your machine.

## Architecture Overview

The API Gateway is the main entry point for the SemLayer platform. It:
- Proxies requests to backend services (fabric-builder, hasura, etc.)
- Handles JWT authentication and authorization
- Enforces tenant scoping via `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers
- Manages API rate limiting and IP whitelisting
- Routes requests to appropriate microservices

## Prerequisites

### Local Setup Required

Only **PostgreSQL** needs to run locally:

1. **PostgreSQL** (on localhost:5432)
   - Database: `alpha`
   - User: `postgres`
   - Password: `postgres`
   - SSL: disabled

All other services (Hasura, Backend, Fabric Builder, Temporal, etc.) run in Docker Compose.

### System Requirements

- Docker & Docker Compose installed
- PostgreSQL running locally
- ~2GB available RAM
- Network access to localhost

## Running the API Gateway & Services

### Quick Start (Recommended)

```bash
cd /Users/eganpj/GitHub/semlayer

# Start all services in Docker Compose
./start-docker.sh
```

This script will:
1. ✓ Verify PostgreSQL is accessible
2. ✓ Create `.env` file with defaults
3. ✓ Build Docker images
4. ✓ Start all services
5. ✓ Show health status

### Manual Docker Compose

```bash
cd /Users/eganpj/GitHub/semlayer

# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f api-gateway

# Check status
docker-compose ps
```

### Stop Services

```bash
# Using the script
./stop-docker.sh

# Or manually
docker-compose down
```

## Environment Variables

The `.env` file is automatically created by `start-docker.sh` with these defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8001` | Port to listen on (in container) |
| `API_GATEWAY_HOST_PORT` | `8001` | Port exposed on host |
| `BACKEND_URL` | `http://backend:8080` | Backend service URL (Docker network) |
| `HASURA_URL` | `http://hasura:8080` | Hasura GraphQL endpoint (Docker network) |
| `HASURA_ADMIN_SECRET` | `newadminsecretkey` | Hasura admin secret |
| `JWT_SECRET` | `your-jwt-secret-key` | Secret for signing JWTs |
| `DEV_ALLOW_UNAUTH_FABRIC` | `true` | Allow unauthenticated access in dev mode |
| `IP_WHITELIST_ENFORCE` | `false` | Disable IP whitelist for development |

### Custom Environment Variables

To customize settings, edit `.env` before running `start-docker.sh`:

```bash
# Example: Change API Gateway port to 9000
echo "API_GATEWAY_HOST_PORT=9000" >> .env

# Restart services
docker-compose down && docker-compose up -d
```

## Testing the API Gateway

All services are automatically healthy before the startup script completes. You can verify:

### Health Check

```bash
curl http://localhost:8001/health
```

Expected response:
```json
{"status":"ok"}
```

### Backend Health

```bash
curl http://localhost:8080/health
```

### Debug Tenant Headers

```bash
curl -H "X-Tenant-ID: test-tenant" \
     -H "X-Tenant-Datasource-ID: test-datasource" \
     http://localhost:8001/api/_debug/headers
```

Expected response:
```json
{
  "received_tenant_id": "test-tenant",
  "received_datasource_id": "test-datasource",
  "query_params": {}
}
```

### GraphQL Proxy to Hasura

```bash
curl -X POST http://localhost:8001/api/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}'
```

### Fabric Builder Integration

Verify the API Gateway correctly proxies to Fabric Builder:

```bash
curl http://localhost:8001/api/fabric/bundles \
  -H "X-Tenant-ID: default" \
  -H "X-Tenant-Datasource-ID: default"
```

## Tenant Context

The API Gateway **requires** tenant context for all protected endpoints:

### Headers

All API requests should include:

```
X-Tenant-ID: <tenant-uuid>
X-Tenant-Datasource-ID: <datasource-uuid>
```

### Query Parameters

Alternatively, use query parameters:

```
?tenant_id=<tenant-uuid>&datasource_id=<datasource-uuid>
```

### Frontend Context

For the frontend (Vite dev server), pre-populate localStorage:

```javascript
localStorage.setItem('selected_tenant', JSON.stringify({
  id: '00000000-0000-0000-0000-000000000000',
  display_name: 'Default Tenant'
}));
localStorage.setItem('selected_datasource', JSON.stringify({
  id: '11111111-1111-1111-1111-111111111111',
  source_name: 'Default Datasource'
}));
```

## Common Issues

### PostgreSQL Connection Error on Startup

**Cause**: PostgreSQL not running or not accessible at localhost:5432.

**Solution**:
```bash
# Start PostgreSQL (adjust for your setup)
brew services start postgresql@15

# Verify it's running
psql -h localhost -U postgres -d alpha -c "SELECT 1"

# Then run start-docker.sh
./start-docker.sh
```

### 502 Bad Gateway

**Cause**: Backend or Hasura service failed to start.

**Solution**:
```bash
# Check service status
docker-compose ps

# View backend logs
docker-compose logs backend

# View Hasura logs
docker-compose logs hasura

# Restart services
docker-compose restart
```

### 403 Forbidden (IP Whitelist)

**Cause**: Client IP not in tenant's whitelist.

**Solution**:
IP whitelist is disabled by default (`IP_WHITELIST_ENFORCE=false` in `.env`). If you enabled it:

```bash
# Disable IP whitelist for development
sed -i '' 's/IP_WHITELIST_ENFORCE=.*/IP_WHITELIST_ENFORCE=false/' .env

# Restart
docker-compose restart api-gateway
```

### 401 Unauthorized

**Cause**: Missing or invalid JWT token.

**Solution**:
In development, authentication is optional (`DEV_ALLOW_UNAUTH_FABRIC=true`). If you disabled it:

```bash
# Enable development mode
sed -i '' 's/DEV_ALLOW_UNAUTH_FABRIC=.*/DEV_ALLOW_UNAUTH_FABRIC=true/' .env

# Restart
docker-compose restart api-gateway
```

### Docker Image Build Fails

**Cause**: Missing dependencies or compilation errors.

**Solution**:
```bash
# Clean and rebuild
docker-compose down --remove-orphans
docker system prune -f
./start-docker.sh
```

### Port Already in Use

**Cause**: Another service is using port 8001, 8080, etc.

**Solution**:
```bash
# Option 1: Find and kill process using the port
lsof -i :8001
kill -9 <PID>

# Option 2: Change port in .env
echo "API_GATEWAY_HOST_PORT=9001" >> .env
docker-compose restart api-gateway

# Then access at http://localhost:9001
```

## Logs

### View All Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api-gateway
docker-compose logs -f backend
docker-compose logs -f hasura

# Follow with timestamps
docker-compose logs -f --timestamps api-gateway

# Last 100 lines
docker-compose logs --tail=100 api-gateway
```

### Debug Mode Logs

Enable verbose logging by modifying `.env`:

```bash
# Enable Gin debug mode
echo "GIN_MODE=debug" >> .env

# Restart service
docker-compose restart api-gateway
```

## Key Endpoints

| Path | Method | Auth | Service | Purpose |
|------|--------|------|---------|---------|
| `/health` | GET | No | API Gateway | Health check |
| `/api/_debug/headers` | GET | No | API Gateway | Debug tenant headers |
| `/api/auth/login` | POST | No | API Gateway | Obtain JWT token |
| `/api/graphql` | POST | Optional | API Gateway → Hasura | GraphQL proxy |
| `/api/fabric/*` | Any | Dev Only | API Gateway → Fabric Builder | Fabric CRUD operations |
| `/api/bundles` | Any | Dev Only | API Gateway → Backend | Bundle management |
| `/api/models` | Any | Dev Only | API Gateway → Backend | Model catalog |
| `/api/views` | Any | Dev Only | API Gateway → Backend | View definitions |

## Service Ports

| Service | Docker Port | Host Port | URL |
|---------|-------------|-----------|-----|
| API Gateway | 8001 | 8001 | http://localhost:8001 |
| Backend | 8080 | 8080 | http://localhost:8080 |
| Hasura | 8080 | (internal) | http://hasura:8080 |
| Fabric Builder | 8081 | 8081 | http://localhost:8081 |
| Temporal | 7233 | 7233 | http://localhost:7233 |
| Temporal UI | 8080 | 8088 | http://localhost:8088 |
| RabbitMQ | 5672,15672 | 5672,15672 | http://localhost:15672 |
| PostgreSQL | (local) | 5432 | postgres://localhost:5432 |

## Troubleshooting

### Check Docker Network

Verify containers can communicate:

```bash
# Check if containers are on the same network
docker network inspect semlayer_semlayer-network

# Test connectivity from one container to another
docker-compose exec api-gateway curl http://backend:8080/health
```

### Rebuild Images

If code changes aren't reflected:

```bash
# Rebuild specific service
docker-compose build --no-cache api-gateway

# Rebuild all
docker-compose build --no-cache

# Start fresh
./start-docker.sh
```

### Clear Docker Cache

```bash
# Remove all stopped containers
docker container prune -f

# Remove unused volumes
docker volume prune -f

# Full cleanup
docker system prune -a --volumes -f
```

### View Detailed Service Info

```bash
# Inspect a service
docker-compose exec api-gateway env

# Check network
docker-compose exec api-gateway ping backend

# View process info
docker-compose exec api-gateway ps aux
```

## Workflow

### Initial Setup (One Time)

```bash
# 1. Ensure PostgreSQL is running
psql -h localhost -U postgres -d alpha -c "SELECT 1"

# 2. Clone the repository
git clone https://github.com/hondyman/semlayer.git
cd semlayer

# 3. Start all Docker services
./start-docker.sh

# 4. Wait for services to be ready (~30 seconds)
```

### Daily Startup

```bash
# Start all services
./start-docker.sh

# Services automatically start in the correct order
# Dependent services wait for their dependencies to be healthy
```

### Frontend Development

```bash
# In another terminal
cd frontend
npm install
npm start

# Frontend will be available at http://localhost:5173
# It connects to API Gateway at http://localhost:8001
```

### Making Code Changes

When you modify service code (backend, fabric-builder, etc.):

```bash
# Rebuild the affected service
docker-compose build --no-cache <service-name>

# Examples
docker-compose build --no-cache backend
docker-compose build --no-cache fabric-builder
docker-compose build --no-cache api-gateway

# Restart the service
docker-compose up -d <service-name>

# Or restart all
docker-compose restart
```

### Shutting Down

```bash
# Clean stop (preserves data)
./stop-docker.sh

# Full cleanup (removes containers and volumes)
docker-compose down -v
```

## Additional Resources

- **Tenant Scoping**: See `/Users/eganpj/GitHub/semlayer/agents.md`
- **Authentication**: See `/Users/eganpj/GitHub/semlayer/api-gateway/README.md`
- **Backend API**: See `/Users/eganpj/GitHub/semlayer/backend/README.md`
- **Fabric Builder**: See `/Users/eganpj/GitHub/semlayer/services/fabric-builder`

## Support

For issues:

1. Check logs: `docker-compose logs -f <service>`
2. Verify PostgreSQL: `psql postgres://postgres:postgres@localhost:5432/alpha`
3. Test endpoints: `curl http://localhost:8001/health`
4. Review error messages in service logs
5. Check docker network: `docker network ls`
