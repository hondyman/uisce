# Investment Management LLM Platform - Deployment Guide

## Quick Start

### 1. Prerequisites
- Docker & Docker Compose
- Gemini API key (or OpenAI/Anthropic)
- PostgreSQL 14+ with pgvector

### 2. Environment Setup
```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API keys
GEMINI_API_KEY=your-key-here
```

### 3. Deploy with Docker
```bash
# Build and start all services
docker-compose up -d

# Check logs
docker-compose logs -f backend

# Stop
docker-compose down
```

### 4. Load Sample Data
```bash
# Connect to database
docker exec -it semlayer-postgres psql -U postgres -d semlayer

# Load sample data
\i /docker-entrypoint-initdb.d/sample_data.sql
```

### 5. Test the API
```bash
# Test pricing query
curl -X POST http://localhost:8080/nlq/ask \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "demo-tenant-123",
    "question": "What is MSFT trading at?"
  }'

# Test NAV calculation
curl -X POST http://localhost:8080/nlq/ask \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "demo-tenant-123",
    "question": "What is my portfolio NAV?",
    "portfolio_id": "demo-portfolio"
  }'
```

## Local Development

### Backend
```bash
cd backend
go run cmd/server/main.go
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

### Database Migrations
```bash
psql -h localhost -U postgres -d semlayer < migrations/*.sql
```

## Production Deployment

### Kubernetes
See `k8s/` directory for Kubernetes manifests (create separately).

### Environment Variables (Production)
- Use Kubernetes secrets for API keys
- Set `DATABASE_URL` to production database
- Enable TLS/SSL for PostgreSQL
- Set up monitoring (Prometheus/Grafana)

## Monitoring

### Health Check
```bash
curl http://localhost:8080/health
```

### Metrics
- Audit logs: Query `audit_log` table
- Usage analytics: Track request counts, latency
- Data quality: Monitor freshness gates

## Troubleshooting

### Backend won't start
- Check DATABASE_URL is correct
- Ensure migrations ran successfully
- Verify API keys are set

### Pricing not working
- Yahoo Finance is free but unofficial (may break)
- Alpha Vantage requires API key for >5 req/min
- Bloomberg requires licensed feed

### NAV calculation fails
- Ensure sample data is loaded
- Check holdings exist in catalog
- Verify pricing provider is accessible

## Next Steps
1. Add your actual portfolio data
2. Configure production LLM provider
3. Set up CI/CD pipeline
4. Enable monitoring and alerting
5. Configure backup and disaster recovery
