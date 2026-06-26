# Cube.js Semantic Layer

Multi-tenant semantic layer built with Cube.js, StarRocks (hot tier), and Trino/Iceberg (cold tier).

## Features

- **Tenant Isolation**: Row-level security via `queryRewrite`, per-tenant query queues and caches
- **Multi-Source**: StarRocks for real-time queries, Trino/Iceberg for historical analytics
- **Governance**: Rollup-only mode to protect data lake from ad-hoc queries
- **Scalability**: Horizontal scaling with multiple API nodes and dedicated refresh workers
- **Observability**: Comprehensive metrics, logs, and SLO monitoring

## Quick Start

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your StarRocks and Trino credentials
vim .env

# Start services
docker-compose up -d

# View logs
docker-compose logs -f cube_api_1

# Access Cube.js Playground
open http://localhost:4000

# Access Traefik Dashboard
open http://localhost:8080
```

## Architecture

```
Client → Traefik (LB) → Cube API (×2) → Redis → Refresh Workers (hot/cold)
                                              ↓
                                        StarRocks (hot)
                                        Trino/Iceberg (cold)
```

## Tenant Isolation

Every query requires `X-Tenant-Id` header. The system automatically:
1. Routes to tenant-specific queue (`contextToAppId`)
2. Injects tenant filter via `queryRewrite`
3. Applies tenant-specific concurrency and cache limits
4. Loads tenant schema overlays if present

## Schema Organization

- `schema/base/` - Shared cube definitions
- `schema/tenants/<tenant_id>/` - Tenant-specific overlays

## Pre-Aggregations

All cubes use external pre-aggregations stored in StarRocks:
- Hot data: 5-minute refresh
- Incremental builds with 7-day window
- Automatic partition pruning

## Security

- Row-level security enforced at query time
- JWT validation (configure in `cube.js`)
- Audit logging for all queries
- Resource quotas per tenant tier

## Monitoring

See `services/cube-semantic/docs/observability.md` for:
- Prometheus metrics
- Grafana dashboards
- SLO definitions
- Alert policies

## Development

```bash
# Install dependencies
npm install

# Run linter
npm run lint

# Run tests
npm test
```

## Production Deployment

See `docs/deployment.md` for Kubernetes/Helm deployment guides.
