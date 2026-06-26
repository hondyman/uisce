# Backend Service Commands Reference

## Current Status Commands

```bash
# Check if backend is running
docker ps | grep backend
# Expected: semlayer-backend-1 ... Up ...

# Check all services
docker compose ps

# View backend logs
docker logs -f semlayer-backend-1

# Check recent errors
docker logs semlayer-backend-1 | grep -i error

# Test API endpoint
curl http://localhost:8082/swagger/index.html

# Check database connection
curl -X GET http://localhost:8082/api/abbreviations/
```

## Container Management

```bash
# Start backend
docker run -d --name semlayer-backend-1 \
  --network semlayer_default \
  -e POSTGRES_HOST=postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=semlayer \
  -p 8082:8080 \
  semlayer-backend:latest

# Stop backend
docker stop semlayer-backend-1

# Restart backend
docker restart semlayer-backend-1

# Remove and restart
docker rm -f semlayer-backend-1
# Then re-run the start command above

# Rebuild image
docker compose build backend
```

## Debugging Commands

```bash
# Get detailed container info
docker inspect semlayer-backend-1

# Check resource usage
docker stats semlayer-backend-1

# Execute command in container
docker exec semlayer-backend-1 /bin/sh

# Check port binding
lsof -i :8082

# Network connectivity test
docker exec semlayer-backend-1 ping postgres
docker exec semlayer-backend-1 nc -zv postgres 5432

# View network
docker network ls
docker network inspect semlayer_default
```

## Database Commands

```bash
# Connect to PostgreSQL
psql postgres://postgres:postgres@localhost:55432/semlayer

# Check tables
psql -h localhost -p 55432 -U postgres -d semlayer -c "\dt"

# View metric registry
psql -h localhost -p 55432 -U postgres -d semlayer \
  -c "SELECT * FROM semantic_layer.metric_registry LIMIT 5;"
```

## Development Workflow

```bash
# Watch logs while developing
docker logs -f semlayer-backend-1

# Rebuild on code change
docker compose build backend
docker restart semlayer-backend-1

# Check compilation errors
docker compose build backend 2>&1 | grep -i error

# Full system restart
docker compose down
docker compose up -d

# Fresh start (remove volumes)
docker compose down -v
docker compose up -d
```

## Testing Endpoints

```bash
# List available routes
curl http://localhost:8082/_routes

# Get metrics registry
curl -X GET http://localhost:8082/api/metrics-registry \
  -H "X-Tenant-ID: test"

# Health check variations
curl http://localhost:8082/health
curl http://localhost:8082/_health
curl http://localhost:8082/api/health

# Swagger API
curl http://localhost:8082/swagger/

# Debug info
curl http://localhost:8082/_debug
curl http://localhost:8082/api/_debug/amqp-metrics
```

## Common Issues & Fixes

```bash
# Port already in use
lsof -i :8082
kill -9 <PID>

# Container stuck/crashed
docker logs semlayer-backend-1 | tail -100
docker restart semlayer-backend-1

# Database connection error
docker logs semlayer-backend-1 | grep -i "database\|postgres"
# Check postgres is running: docker ps | grep postgres

# Memory issues
docker stats semlayer-backend-1
# Increase Docker memory allocation if needed

# Network connectivity
docker exec semlayer-backend-1 ping postgres
docker network inspect semlayer_default
```

## Monitoring Commands

```bash
# Real-time stats
watch -n 1 'docker ps | grep backend'

# Log monitoring
docker logs -f semlayer-backend-1 --timestamps

# Performance metrics
docker stats semlayer-backend-1

# Connection metrics
curl http://localhost:8082/debug/pprof/

# Database connections
psql -h localhost -p 55432 -U postgres -d semlayer \
  -c "SELECT datname, usename, application_name, state, count(*) FROM pg_stat_activity GROUP BY 1,2,3,4;"
```

## Deployment Commands

```bash
# Build fresh image
docker compose build --no-cache backend

# Deploy to local
docker compose up -d backend

# Check deployment
docker compose ps backend

# View logs after deployment
docker logs semlayer-backend-1 | tail -20

# Verify health
curl http://localhost:8082/swagger/index.html
```

---

**Quick Status Check** (copy & paste):
```bash
docker ps | grep backend && curl -s http://localhost:8082/swagger/index.html | head -1 && echo "✅ Backend is running"
```

**Emergency Restart** (copy & paste):
```bash
docker rm -f semlayer-backend-1 && \
docker run -d --name semlayer-backend-1 \
  --network semlayer_default \
  -e POSTGRES_HOST=postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=semlayer \
  -p 8082:8080 \
  semlayer-backend:latest && \
sleep 5 && \
docker logs semlayer-backend-1 | tail -5
```
