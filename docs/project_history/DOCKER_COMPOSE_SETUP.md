# Docker Compose Setup Guide

## ✅ Current Status

Your Docker Compose is **up and running** with the following services:

### Infrastructure Services ✓
- **Hasura GraphQL** → http://localhost:8888
- **RabbitMQ** → amqp://localhost:5672 (Management: http://localhost:15672)
- **Temporal Workflow** → localhost:7233
- **Temporal UI** → http://localhost:8088

### Frontend ✓
- **Frontend Dev Server** → http://localhost:5173

## 🚀 Quick Start

### 1. Start Docker Compose
```bash
cd /Users/eganpj/GitHub/semlayer
docker compose -f docker-compose.dev.simple.yml up -d
```

### 2. Check Service Status
```bash
./scripts/check-services.sh
```

### 3. View Logs
```bash
# All services
docker compose -f docker-compose.dev.simple.yml logs -f

# Specific service
docker compose -f docker-compose.dev.simple.yml logs -f hasura
docker compose -f docker-compose.dev.simple.yml logs -f rabbitmq
docker compose -f docker-compose.dev.simple.yml logs -f temporal
```

### 4. Stop Services
```bash
docker compose -f docker-compose.dev.simple.yml down
```

## 🔌 Service Details

### Hasura GraphQL (Port 8888)
- **URL**: http://localhost:8888
- **GraphQL Endpoint**: http://localhost:8888/v1/graphql
- **Admin Console**: http://localhost:8888/console
- **Status**: ✓ Running and healthy
- **Connected to**: PostgreSQL on host.docker.internal:5432

### RabbitMQ (Port 5672, Management 15672)
- **AMQP URL**: amqp://guest:guest@localhost:5672/
- **Management Console**: http://localhost:15672 (guest/guest)
- **Status**: ✓ Running and healthy
- **Use for**: Message queues, event streaming

### Temporal (Port 7233)
- **Server**: localhost:7233
- **UI**: http://localhost:8088
- **Status**: ✓ Running
- **Connected to**: PostgreSQL on host.docker.internal:5432
- **Use for**: Workflow orchestration, temporal task scheduling

## 📋 Environment Variables

Your `.env.local` is already configured for these services:
```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_BACKEND_TARGET=http://localhost:8080
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
VITE_GRAPHQL_WS_ENDPOINT=ws://localhost:8080/v1/graphql
```

## 🔧 Troubleshooting

### Service won't start
```bash
# Remove old containers and networks
docker compose -f docker-compose.dev.simple.yml down --remove-orphans
docker compose -f docker-compose.dev.simple.yml up -d
```

### Logs show connection errors
```bash
# Check if PostgreSQL is running on host
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -c "SELECT 1"
```

### Port conflicts
```bash
# Find what's using a port
lsof -i :8888  # or any port
kill -9 <PID>  # if needed
```

### View resource usage
```bash
docker stats semlayer-hasura semlayer-rabbitmq semlayer-temporal
```

## 📝 Next Steps

1. **Start your backend**:
   ```bash
   cd /Users/eganpj/GitHub/semlayer/services/fabric-builder
   go run main.go
   ```

2. **Start your frontend** (if not running):
   ```bash
   cd /Users/eganpj/GitHub/semlayer/frontend
   npm run dev
   ```

3. **Access the app**:
   - Frontend: http://localhost:5173
   - Hasura: http://localhost:8888
   - RabbitMQ: http://localhost:15672

## 📊 Monitor Services

Use the status checker script:
```bash
./scripts/check-services.sh
```

This will show you:
- Which services are running
- Which ports they're on
- Direct links to web interfaces
- Environment variables

## 🎯 Development Workflow

1. **Database changes**: Update PostgreSQL directly (it's on your host machine)
2. **API changes**: Restart your backend service
3. **Frontend changes**: Auto-reload on save (Vite)
4. **Workflow changes**: Update Temporal configurations and restart
5. **Message queues**: RabbitMQ persists across restarts

## 🔐 Credentials

- **RabbitMQ**: guest/guest
- **Hasura Admin Secret**: admin-secret (set in docker-compose)
- **JWT Secret**: development-secret-key (set in docker-compose)
- **PostgreSQL**: postgres/postgres (running on your host at :5432)

## 📚 References

- [Hasura Docs](https://hasura.io/docs/)
- [RabbitMQ Docs](https://www.rabbitmq.com/documentation.html)
- [Temporal Docs](https://docs.temporal.io/)
- [Docker Compose Docs](https://docs.docker.com/compose/)
