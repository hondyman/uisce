# SemLayer API Layer

This directory contains the complete API layer for the SemLayer semantic catalog system, providing REST, GraphQL, and OpenAPI support.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST/OpenAPI  │    │   API Gateway   │    │     Hasura      │
│    Clients      │───▶│   (Go/Gin)     │───▶│  GraphQL Engine │
│                 │    │                 │    │                 │
│ • Swagger UI    │    │ • REST → GraphQL│    │ • GraphQL API   │
│ • Postman       │    │ • OpenAPI Spec  │    │ • Auto-generated│
│ • Custom Apps   │    │ • Multi-tenant  │    │ • Real-time     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │  PostgreSQL     │    │  Semantic       │
                       │  Database       │    │  Service        │
                       │                 │    │  (Go)           │
                       │ • Catalog data  │    │                 │
                       │ • Business terms│    │ • Business logic│
                       │ • Lineage info  │    │ • Validation    │
                       └─────────────────┘    └─────────────────┘
```

## Components

### 1. API Gateway (`api-gateway/`)
A Go-based service that provides REST endpoints and converts them to GraphQL queries.

**Features:**
- REST API endpoints for business term operations
- Automatic conversion to Hasura GraphQL queries
- OpenAPI/Swagger documentation
- Multi-tenant support
- Health monitoring

**Endpoints:**
- `GET /health` - Health check
- `POST /api/search/business-terms` - Search business terms
- `POST /api/validate/business-term` - Validate business term
- `GET /api/lineage/semantic` - Get semantic lineage
- `POST /api/graphql` - Direct GraphQL proxy
- `GET /docs/` - Swagger UI
- `GET /api/openapi.yaml` - OpenAPI specification

### 2. Hasura GraphQL Engine (`hasura/`)
Auto-generated GraphQL API with real-time subscriptions.

**Features:**
- Auto-generated GraphQL schema from PostgreSQL
- Real-time subscriptions
- Role-based permissions
- Custom actions for complex business logic
- Metadata-driven configuration

### 3. PostgreSQL Database (`init-db.sql`)
Database schema and initial data for the semantic catalog.

**Tables:**
- `tenants` - Multi-tenant configuration
- `tenant_datasources` - Data source connections
- `catalog_node` - Generic catalog nodes
- `catalog_edge` - Relationships between nodes
- `catalog_node_type` - Node type definitions
-- `catalog_edge_types` - Edge type definitions

**Views:**
- `business_terms` - Business term view
- `semantic_models` - Semantic model view
- `semantic_columns` - Semantic column view

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)

### 1. Start the Services
```bash
# From the project root
docker-compose up -d
```

### 2. Verify Services are Running
```bash
# Check all services
docker-compose ps

# View logs
docker-compose logs api-gateway
docker-compose logs graphql-engine
```

### 3. Access the APIs

**REST API (API Gateway):**
- Base URL: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/docs/`
- OpenAPI Spec: `http://localhost:8080/api/openapi.yaml`

**GraphQL API (Hasura):**
- GraphQL Endpoint: `http://localhost:8081/v1/graphql`
- Console: `http://localhost:8081/console`

**PostgreSQL:**
- Host: `localhost:5432`
- Database: `semlayer_db`
- User: `semlayer_user`
- Password: `semlayer_password`

### 4. Test the APIs

```bash
# Run API tests
cd api-gateway && ./test-api.sh

# Or test manually
curl http://localhost:8080/health
curl http://localhost:8080/api/openapi.yaml
```

## Development

### Local Development Setup

1. **API Gateway:**
```bash
cd api-gateway
go mod tidy
go run main.go
```

2. **Database Migrations:**
```bash
# Apply database schema
docker-compose exec postgres psql -U semlayer_user -d semlayer_db -f /docker-entrypoint-initdb.d/init-db.sql
```

3. **Hasura Metadata:**
```bash
# Apply Hasura metadata
docker-compose exec graphql-engine hasura metadata apply
```

### Testing

```bash
# Run API gateway tests
cd api-gateway && go test ./...

# Test with curl
curl -X POST http://localhost:8080/api/search/business-terms \
  -H "Content-Type: application/json" \
  -d '{"query": "test", "tenant_id": "default"}'
```

### Building

```bash
# Build API gateway
cd api-gateway && docker build -t semlayer-api-gateway .

# Build all services
docker-compose build
```

## Configuration

### Environment Variables

**API Gateway:**
- `PORT`: Service port (default: 8080)
- `HASURA_URL`: Hasura endpoint URL
- `HASURA_ADMIN_SECRET`: Hasura admin secret
- `DATABASE_URL`: PostgreSQL connection string

**Hasura:**
- `HASURA_GRAPHQL_DATABASE_URL`: Database connection
- `HASURA_GRAPHQL_ADMIN_SECRET`: Admin secret
- `HASURA_GRAPHQL_ENABLE_CONSOLE`: Enable console (dev mode)

**PostgreSQL:**
- `POSTGRES_USER`: Database user
- `POSTGRES_PASSWORD`: Database password
- `POSTGRES_DB`: Database name

### Multi-Tenancy

The system supports multi-tenant operation through:

1. **Tenant Header:** `X-Tenant-ID` header in API requests
2. **Database Isolation:** Tenant-specific data partitioning
3. **Hasura Permissions:** Role-based access control per tenant

## API Examples

### Search Business Terms
```bash
curl -X POST http://localhost:8080/api/search/business-terms \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: default" \
  -d '{
    "query": "customer",
    "limit": 10
  }'
```

### GraphQL Query
```bash
curl -X POST http://localhost:8081/v1/graphql \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: myadminsecretkey" \
  -d '{
    "query": "query { business_terms { id name description } }"
  }'
```

### Validate Business Term
```bash
curl -X POST http://localhost:8080/api/validate/business-term \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Customer ID",
    "description": "Unique customer identifier",
    "category": "Customer Data"
  }'
```

## Monitoring & Observability

### Health Checks
- API Gateway: `GET /health`
- Hasura: Health check via GraphQL introspection

### Logs
```bash
# View all service logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f api-gateway
docker-compose logs -f graphql-engine
```

### Metrics
- Hasura provides built-in metrics
- API Gateway can be extended with Prometheus metrics

## Troubleshooting

### Common Issues

1. **Services not starting:**
   ```bash
   docker-compose logs <service-name>
   ```

2. **Database connection issues:**
   ```bash
   docker-compose exec postgres pg_isready -U semlayer_user -d semlayer_db
   ```

3. **Hasura metadata issues:**
   ```bash
   docker-compose exec graphql-engine hasura metadata reload
   ```

4. **API gateway compilation errors:**
   ```bash
   cd api-gateway && go mod tidy && go build
   ```

### Reset Everything
```bash
# Stop and remove all containers
docker-compose down -v

# Rebuild and start
docker-compose up --build -d

# Reinitialize database
docker-compose exec postgres psql -U semlayer_user -d semlayer_db -f /docker-entrypoint-initdb.d/init-db.sql
```

## Contributing

1. **API Gateway:** Make changes in `api-gateway/` directory
2. **Database Schema:** Update `init-db.sql`
3. **Hasura Config:** Modify `hasura/metadata/`
4. **Docker:** Update `docker-compose.yml`

### Testing Changes
```bash
# Test API gateway
cd api-gateway && go test ./...

# Test integration
docker-compose up -d && ./api-gateway/test-api.sh
```

## Security Considerations

- **API Keys:** Use Hasura admin secret securely
- **Database Credentials:** Store in environment variables
- **Network Security:** Configure proper firewall rules
- **HTTPS:** Enable SSL/TLS in production
- **Authentication:** Implement proper auth mechanisms

## Production Deployment

1. **Environment Variables:** Use secure secret management
2. **Database:** Use managed PostgreSQL service
3. **Load Balancing:** Deploy multiple API gateway instances
4. **Monitoring:** Set up comprehensive monitoring
5. **Backups:** Configure database backups
6. **Scaling:** Use Docker Swarm or Kubernetes

## Support

For issues and questions:
1. Check the logs: `docker-compose logs`
2. Review the API documentation: `http://localhost:8080/docs/`
3. Test with the provided scripts: `./api-gateway/test-api.sh`
4. Check Hasura console: `http://localhost:8081/console`
