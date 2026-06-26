# Semlayer Docker Compose Setup

This directory contains a comprehensive Docker Compose setup for running all Semlayer backend microservices with a single command.

## 🚀 Quick Start

### Start All Services
```bash
./docker-start.sh up
```

### Stop All Services
```bash
./docker-start.sh down
```

### View Service Status
```bash
./docker-start.sh ps
```

### View Logs
```bash
./docker-start.sh logs          # All services
./docker-start.sh logs backend  # Specific service
```

## 📋 Service Architecture

### Infrastructure Services
- **PostgreSQL** (5432) - Primary database
- **Hasura** (8080) - GraphQL engine and API gateway
- **Temporal** (7233) - Workflow orchestration
- **Temporal UI** (8088) - Workflow monitoring
- **RabbitMQ** (5672, 15672) - Message queue
- **AI Service** (8000) - External AI API

### Backend Microservices
- **Backend** (8080) - Main API server
- **Fabric Builder** (8081) - Data fabric management
- **Wealth Management** (8082) - Wealth management features
- **AI Builder** (8083) - AI-powered features
- **Semantic Engine** (8084) - Semantic data processing
- **Governance** (8085) - Data governance
- **Compliance Engine** (8086) - Regulatory compliance
- **Validation Service** (8087) - Data validation
- **Rule Engine** (8089) - Business rules
- **Notifications** (8090) - Notification system
- **Policy Service** (8091) - Access policies
- **Search Service** (8092) - Search functionality
- **Event Router** (8093) - Event processing

### API Gateway
- **API Gateway** (80, 443) - Load balancer and reverse proxy

## 🔧 Development Setup

For development, use the override file which includes additional tools:

```bash
# Start with development tools
docker-compose -f docker-compose.yml -f docker-compose.override.yml up -d

# Or use the helper script
./docker-start.sh up
```

### Development Services
- **Frontend** (5173) - React development server
- **Adminer** (8089) - Database management UI
- **Redis** (6379) - Caching layer
- **Prometheus** (9090) - Metrics collection
- **Grafana** (3000) - Monitoring dashboards
- **Swagger UI** (8094) - API documentation

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the root directory:

```bash
# Database
POSTGRES_DB=semlayer
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# Hasura
HASURA_ADMIN_SECRET=your-admin-secret
JWT_SECRET=your-jwt-secret

# AI Service
XAI_API_KEY=your-xai-api-key

# Other services
RABBITMQ_DEFAULT_USER=guest
RABBITMQ_DEFAULT_PASS=guest
```

### Service-Specific Configuration

- **Hasura**: Metadata in `hasura/metadata/`
- **Temporal**: Dynamic config in `configs/temporal-dynamicconfig/`
- **RabbitMQ**: Definitions in `rabbitmq-definitions.json`
- **Nginx**: Config in `configs/nginx/`

## 🏗️ Building Services

Services are automatically built when starting. To rebuild specific services:

```bash
# Rebuild and restart a specific service
docker-compose build backend
docker-compose up -d backend

# Rebuild all services
docker-compose build
```

## 🔍 Troubleshooting

### Check Service Health
```bash
# View all running containers
docker-compose ps

# Check logs for a specific service
docker-compose logs backend

# Check resource usage
docker stats
```

### Common Issues

1. **Port conflicts**: Ensure ports 8080-8094, 5432, 7233, etc. are available
2. **Database connection**: Wait for PostgreSQL to be healthy before starting dependent services
3. **Memory issues**: Increase Docker memory limit if services crash
4. **Network issues**: Check that all services are on the same network

### Reset Everything
```bash
# Stop and remove all containers, volumes, and networks
docker-compose down -v --remove-orphans

# Clean up unused images
docker image prune -f
```

## 📊 Monitoring

### Service URLs
- **Hasura Console**: http://localhost:8080/console
- **Temporal UI**: http://localhost:8088
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Adminer**: http://localhost:8089
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9091
- **Swagger UI**: http://localhost:8094

### Health Checks
All services include health checks that run automatically. Check service health:

```bash
# Check specific service health
curl http://localhost:8080/health

# Check Hasura health
curl http://localhost:8080/v1/version
```

## 🛠️ Advanced Usage

### Start Only Infrastructure
```bash
./docker-start.sh infra
```

### Start Only Backend Services
```bash
./docker-start.sh backend
```

### Scale Services
```bash
# Scale a service to multiple instances
docker-compose up -d --scale backend=3
```

### Custom Configuration
```bash
# Use custom compose files
docker-compose -f docker-compose.yml -f docker-compose.custom.yml up
```

## 📁 File Structure

```
.
├── docker-compose.yml              # Main compose file
├── docker-compose.override.yml     # Development overrides
├── docker-start.sh                 # Management script
├── configs/                        # Service configurations
│   ├── nginx/
│   └── temporal-dynamicconfig/
├── hasura/                         # Hasura metadata and migrations
├── monitoring/                     # Prometheus/Grafana configs
├── services/                       # Microservice source code
└── init-db.sql                     # Database initialization
```

## 🤝 Contributing

When adding new services:

1. Add the service definition to `docker-compose.yml`
2. Update dependencies with `depends_on`
3. Add appropriate environment variables
4. Include health checks
5. Update this README
6. Test the complete stack

## 📞 Support

For issues with the Docker setup:
1. Check service logs: `./docker-start.sh logs <service>`
2. Verify environment variables in `.env`
3. Ensure all required ports are available
4. Check Docker resources (memory, CPU)