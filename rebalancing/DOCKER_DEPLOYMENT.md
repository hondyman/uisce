# Rebalancing System - Docker Deployment Guide

## 🐳 Complete Stack

This `docker-compose.yml` orchestrates a full production-ready rebalancing system with 9 services:

### Services Included

| Service | Port | Purpose | Status Check |
|---------|------|---------|--------------|
| **PostgreSQL** | 5432 | Primary data store | `pg_isready -U postgres` |
| **Temporal** | 7233+ | Workflow orchestration | `curl http://localhost:7233` |
| **Temporal UI** | 8081 | Workflow dashboard | http://localhost:8081 |
| **Redpanda (Kafka)** | 9092/8082 | Event streaming (Kafka) | `rpk cluster info` |
| **Hasura** | 8080 | GraphQL API | http://localhost:8080 |
| **Redis** | 6379 | Caching layer | `redis-cli ping` |
| **API Server** | 8090 | REST backend | http://localhost:8090/health |
| **Worker** | - | Activity executor | Logs to Docker |
| **React Frontend** | 3000 | Dashboard UI | http://localhost:3000 |

## 🚀 Quick Start (5 minutes)

### 1. Prerequisites
```bash
# Ensure Docker and Docker Compose installed
docker --version          # v20.10+
docker-compose --version  # v1.29+
```

### 2. Set up environment
```bash
cd /path/to/semlayer/rebalancing

# Create .env from template
cat > .env << 'ENV'
HASURA_SECRET=your-secure-secret
XAI_API_KEY=your-xai-api-key
FINNHUB_API_KEY=your-finnhub-api-key
ENV

# Or use example values for local dev
cp .env.example .env
```

### 3. Start the stack
```bash
# Spin up all services
docker-compose up -d

# Watch startup (2-3 minutes)
docker-compose logs -f

# Check all services healthy
docker-compose ps
```

### 4. Verify deployment
```bash
# Test each endpoint
curl http://localhost:5432 -v          # PostgreSQL (will fail, but connects)
curl http://localhost:8080/v1/metadata # Hasura
curl http://localhost:8090/health      # API
curl http://localhost:3000/             # Frontend
curl http://localhost:8081/             # Temporal UI
```

### 5. Access interfaces
- **React Dashboard**: http://localhost:3000
- **Hasura Console**: http://localhost:8080 (admin secret: `secret`)
- **Temporal UI**: http://localhost:8081
- **Pandaproxy (Kafka HTTP)**: http://localhost:8082 (HTTP Kafka proxy)

## 🔧 Configuration

### Environment Variables (.env)

```bash
# Security
HASURA_SECRET=your-admin-secret-here
HASURA_JWT_SECRET='{"type":"HS256","key":"your-256-bit-secret"}'

# External APIs
XAI_API_KEY=your-xai-api-key
FINNHUB_API_KEY=your-finnhub-api-key

# Frontend build args (optional)
VITE_API_URL=http://localhost:8090
VITE_GRAPHQL_URL=http://localhost:8080/v1/graphql
VITE_TEMPORAL_UI_URL=http://localhost:8081
```

### Service Configuration

#### PostgreSQL
- Database: `portfolio`
- User: `postgres`
- Password: `postgres`
- Schema auto-loaded from: `./schema.sql`
- Volumes: `postgres_data` (persistent)

#### Hasura
- Endpoint: http://localhost:8080
- GraphQL: http://localhost:8080/v1/graphql
- Admin Secret: Value of `HASURA_SECRET`
- Console: http://localhost:8080/console

#### Temporal
- Server: `localhost:7233`
- UI: http://localhost:8081
- Database: Uses PostgreSQL backend

#### Redpanda (Kafka)
- Brokers: `localhost:9092` (Kafka broker)
- Pandaproxy (HTTP): http://localhost:8082

#### API Server
- Endpoint: http://localhost:8090
- Health: http://localhost:8090/health
- Connects to: Temporal, Hasura, Redpanda (Kafka), Redis

#### React Frontend
- Endpoint: http://localhost:3000
- Connects to: Hasura GraphQL (8080), API (8090)

## 📊 Health Checks

All services include automated health checks. Monitor status:

```bash
# Watch health status
docker-compose ps

# Check specific service
docker-compose exec postgres pg_isready -U postgres

# View service logs
docker-compose logs service-name -f

# Inspect full health status
docker-compose exec rebalance-api curl http://localhost:8080/v1/metadata
```

## 🔄 Common Operations

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f rebalance-frontend

# Last 50 lines, follow updates
docker-compose logs -f --tail=50 rebalance-worker
```

### Restart Service
```bash
# Restart one service
docker-compose restart rebalance-api

# Stop and start (fresh start)
docker-compose stop rebalance-frontend
docker-compose start rebalance-frontend
```

### Database Operations
```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U postgres -d portfolio

# Run SQL query
docker-compose exec postgres psql -U postgres -d portfolio -c "SELECT * FROM v_rebalance_summary LIMIT 1;"

# Backup database
docker-compose exec postgres pg_dump -U postgres portfolio > backup.sql

# Restore database
docker-compose exec -T postgres psql -U postgres portfolio < backup.sql
```

### Hasura Operations
```bash
# Apply migrations
docker-compose exec hasura hasura-cli migrate apply

# Export metadata
docker-compose exec hasura hasura-cli metadata export

# Reload metadata
curl -X POST http://localhost:8080/v1/metadata \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: secret" \
  -d '{"type": "reload_metadata", "args": {}}'
```

### Clean Up
```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes data)
docker-compose down -v

# Remove images
docker-compose down --rmi local

# Full reset
docker-compose down -v --rmi all
```

## 🐛 Troubleshooting

### Services Won't Start
```bash
# Check for port conflicts
lsof -i :5432
lsof -i :8080
lsof -i :3000

# Free up ports or change docker-compose ports
# Then restart
docker-compose up -d
```

### PostgreSQL Won't Connect
```bash
# Check PostgreSQL logs
docker-compose logs postgres

# Verify data directory
docker-compose exec postgres ls -la /var/lib/postgresql/data

# Reset PostgreSQL (⚠️ loses data)
docker-compose down -v
docker-compose up -d postgres
```

### Temporal Not Connecting
```bash
# Check Temporal logs
docker-compose logs temporal

# Verify PostgreSQL is healthy first
docker-compose ps postgres

# Temporal can take 30+ seconds to start
sleep 30 && docker-compose logs temporal
```

### Frontend Can't Reach API
```bash
# Check network connectivity
docker-compose exec rebalance-frontend curl http://rebalance-api:8090/health

# Verify API is running
docker-compose exec rebalance-api curl http://localhost:8080/v1/metadata

# Check frontend environment variables
docker-compose exec rebalance-frontend env | grep VITE
```

### Hasura Errors
```bash
# Check Hasura logs
docker-compose logs hasura

# Verify database connection
docker-compose exec hasura curl -X POST http://localhost:8080/v1/metadata \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: secret" \
  -d '{"type": "test_database_connection", "args": {}}'
```

## 📈 Performance & Scaling

### Resource Limits (Optional)
Add to docker-compose.yml services:
```yaml
services:
  rebalance-api:
    ...
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### Horizontal Scaling
```bash
# Run multiple workers
docker-compose up -d --scale rebalance-worker=3

# Load balance with nginx (separate service)
# See infrastructure/nginx/ for config
```

## 🔐 Production Deployment

### Security Checklist
- [ ] Change all default passwords
- [ ] Set strong `HASURA_SECRET` and JWT secret
- [ ] Use environment-specific `.env` files
- [ ] Enable HTTPS with reverse proxy (nginx/Traefik)
- [ ] Restrict network access (use firewall rules)
- [ ] Enable PostgreSQL encryption at rest
- [ ] Rotate API keys regularly
- [ ] Enable audit logging for Hasura

### Production Compose File
See `docker-compose.prod.yml` for production-ready configuration with:
- Resource limits
- Restart policies
- Volume persistence
- Health checks
- Logging configuration

## 📝 Monitoring

### Docker Stats
```bash
# Real-time resource usage
docker stats

# Watch specific service
docker stats rebalance-worker
```

### Log Aggregation
```bash
# View combined logs with timestamps
docker-compose logs --timestamps rebalance-api rebalance-worker

# Stream logs to file
docker-compose logs -f > /tmp/rebalancing.log &
```

## 🧪 Testing

### Run Integration Tests
```bash
# Inside API container
docker-compose exec rebalance-api go test ./...

# Inside worker container
docker-compose exec rebalance-worker go test ./...

# Frontend tests
docker-compose exec rebalance-frontend npm test
```

### Load Testing
```bash
# Start load test against API
docker-compose run --rm load-test k6 run test.js

# Or use curl loop
for i in {1..100}; do
  curl -X POST http://localhost:8090/api/rebalance/start \
    -H "Content-Type: application/json" \
    -d '{"portfolio_id":"test"}'
done
```

## 📚 Additional Resources

- **Docker Docs**: https://docs.docker.com/
- **Docker Compose**: https://docs.docker.com/compose/
- **PostgreSQL**: https://www.postgresql.org/docs/
- **Hasura**: https://hasura.io/docs/
- **Temporal**: https://docs.temporal.io/
- **RabbitMQ**: https://www.rabbitmq.com/documentation.html

## 🤝 Support

For issues:
1. Check logs: `docker-compose logs service-name`
2. Verify health: `docker-compose ps`
3. Review REBALANCING_GUIDE.md for integration details
4. Check firewall and port availability

---

**Last Updated**: Oct 30, 2025  
**Version**: 1.0.0
