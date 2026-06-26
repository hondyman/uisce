# Port Configuration Guide

## Overview

All SemLayer service ports are centralized in the `.env.ports` file for easy configuration and management.

## Quick Start

1. Copy the example file:
   ```bash
   cp .env.ports.example .env.ports
   ```

2. Customize ports as needed (optional - defaults work out of the box)

3. Restart services to apply changes:
   ```bash
   docker-compose up -d
   ```

## Port Allocation

### Infrastructure Services
| Service | Default Port | Description |
|---------|--------------|-------------|
| Hasura | 8085 | GraphQL Engine |
| Metastore DB | 5433 | Postgres for Trino/Iceberg |
| Redpanda | 9092, 19092, 9644, 8081 | Kafka-compatible streaming |
| Redpanda Console | 8096 | Kafka UI |
| Debezium | 8083 | CDC connector |
| MinIO | 9000 (API), 9001 (Console) | Object storage |
| RisingWave | 4566 | Real-time analytics |
| Trino | 8084 | Query engine |
| Cube.js | 4000 (API), 15432 (PG) | Semantic layer |

### Application Services (Port Range: 8001-8095)
| Service | Default Port | Description |
|---------|--------------|-------------|
| API Gateway | 8001 | Main API gateway |
| Backend | 8082 | Core backend service |
| BP Backend | 8086 | Business process backend |
| Entity Manager | 8087 | Entity management |
| NBA ML Service | 8088 | ML service |
| Notifications | 8089 | Notification service |
| Validation Engine | 8090 | Data validation |
| Rule Engine | 8091 | Business rules |
| Search Service | 8092 | Search functionality |
| Policy Engine | 8093 | Policy management |
| Analytics Engine | 8094 | Analytics processing |
| Compliance Engine | 8095 | Compliance checks |

## Changing Ports

To change a service port:

1. Edit `.env.ports` and update the desired port variable
2. Restart the affected service:
   ```bash
   docker-compose up -d <service-name>
   ```

Example - change backend from 8082 to 8097:
```bash
# In .env.ports
BACKEND_PORT=8097
```

```bash
# Restart the service
docker-compose up -d backend
```

## Port Allocation Strategy

- **4000-4999**: Analytics & Data services
- **5000-5999**: Databases and database-like services
- **8000-8099**: Core platform and microservices
- **9000-9999**: Infrastructure (Message queues, object storage)
- **15000+**: Alternate protocols

## Internal vs External Ports

Most application services use **port 8080 internally** within the Docker network. The ports listed above are the **external/host ports** that you use to access the services.

For example, the backend service:
- Internal (Docker network): `http://backend:8080`
- External (host): `http://localhost:8082`

## Troubleshooting

### Port Already in Use

If you see "address already in use" errors:

1. Find what's using the port:
   ```bash
   lsof -i :<port>
   ```

2. Either:
   - Stop the conflicting process, OR
   - Change the port in `.env.ports`

### Service Not Starting

Check the service logs:
```bash
docker-compose logs <service-name>
```

Common issues:
- Port conflict (see above)
- Missing environment variables
- Service dependencies not running

## Related Files

- `.env.ports.example` - Template with all available port variables
- `.env.ports` -Your active port configuration (git-ignored)
- `docker-compose.yml` - Service definitions using port variables
