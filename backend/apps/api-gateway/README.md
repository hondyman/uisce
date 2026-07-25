# SemLayer API Gateway

A REST API gateway that provides OpenAPI-compliant endpoints for the SemLayer semantic catalog system. This service converts REST/OpenAPI requests to GraphQL queries that are executed against Hasura.

## Features

- **REST API Endpoints**: Clean REST endpoints for business term operations
- **OpenAPI Documentation**: Auto-generated Swagger UI documentation
- **GraphQL Integration**: Seamless conversion to Hasura GraphQL queries
- **Multi-tenant Support**: Tenant isolation via headers
- **Health Checks**: Service monitoring endpoints

## API Endpoints

### Health Check
- `GET /health` - Service health status

### Business Terms
- `POST /api/search/business-terms` - Search business terms
- `POST /api/validate/business-term` - Validate business term definition

### Lineage
- `GET /api/lineage/semantic` - Get semantic lineage for a node

### GraphQL Proxy
- `POST /api/graphql` - Direct GraphQL query execution

### Documentation
- `GET /docs/` - Swagger UI documentation
- `GET /api/openapi.yaml` - OpenAPI specification

## Usage Examples

### Search Business Terms
```bash
curl -X POST http://localhost:8080/api/search/business-terms \
  -H "Content-Type: application/json" \
  -d '{
    "query": "customer",
    "limit": 10,
    "tenant_id": "default"
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

### Get Semantic Lineage
```bash
curl "http://localhost:8080/api/lineage/semantic?node_id=bt_123&depth=3"
```

## Environment Variables

- `PORT`: Service port (default: 8080)
- `HASURA_URL`: Hasura GraphQL endpoint URL
- `HASURA_ADMIN_SECRET`: Hasura admin secret
- `DATABASE_URL`: PostgreSQL connection string
- `TENANT_ID`: Default tenant ID

## Development

### Prerequisites
- Go 1.21+
- Docker & Docker Compose

### Running Locally
```bash
# Install dependencies
go mod tidy

# Run the service
go run main.go
```

### Building Docker Image
```bash
docker build -t semlayer-api-gateway .
```

## Docker Compose

The API gateway is part of the larger SemLayer docker-compose stack:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs api-gateway

# Stop services
docker-compose down
```

## Compose modes (safe vs standalone)

This repository provides two ways to run the API Gateway compose depending on your workflow:

- Safe / co-located mode (default)
  - File: `docker-compose.yml`
  - This compose uses `expose` for internal service ports so it can be run alongside the repo-root compose without binding host ports. Use this when you already run the full stack from the repository root and only need the gateway services on an internal network.

- Standalone mode (host-exposed)
  - File: `docker-compose.override.yml`
  - This override binds host ports to the gateway/hasura/backend so you can run the api-gateway compose by itself and access services on `http://localhost:8000`, `http://localhost:8081`, and `http://localhost:3000`.
  - To start in standalone mode:

```bash
cd api-gateway
docker compose -f docker-compose.yml -f docker-compose.override.yml up -d --build
```

Notes:
- Do not run both the repo-root compose and the standalone override at the same time — they will conflict on host ports. Use the default `docker-compose.yml` for co-located workflows.
- If you want I can add an env-driven helper script to toggle host port binding automatically.


## Architecture

```
REST/OpenAPI Request → API Gateway → GraphQL Query → Hasura → PostgreSQL
```

The API gateway:
1. Receives REST requests with JSON payloads
2. Converts them to GraphQL queries
3. Forwards to Hasura GraphQL engine
4. Returns formatted JSON responses
5. Provides OpenAPI documentation via Swagger UI

## Multi-tenancy

Tenant isolation is handled via:
- `X-Tenant-ID` header in requests
- Database-level tenant separation
- Hasura role-based permissions

## Error Handling

The API gateway provides consistent error responses:
```json
{
  "error": "Error message description"
}
```

GraphQL errors are also properly formatted and returned to clients.
