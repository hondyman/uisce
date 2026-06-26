# Remote Infrastructure Setup

This setup splits your SemLayer services between local application services (Golang) and remote infrastructure services (data stores, message queues, etc.) running on a Tailscale-accessible server.

## Architecture

### Remote Server (100.84.126.19 via Tailscale)
- **Redpanda** (Kafka-compatible streaming) - Port 9092
- **Temporal** (workflow engine) - Port 7233
- **Temporal UI** - Port 8086
- **Debezium** (CDC connector) - Port 8083
- **Kafka Connect Iceberg** - Port 8098
- **MinIO** (object storage) - Ports 9000/9001
- **Redis** (caching) - Port 6379
- **Trino** (query engine) - Port 8084

### Local Machine
- All Golang application services (backend, workers, APIs)
- **PostgreSQL** (runs on localhost:5432)
- **Hasura** (GraphQL engine) - Port 8085

## Setup Instructions

### 1. On Remote Tailscale Server (100.84.126.19)

```bash
# Copy the remote compose file
scp docker-compose.remote.yml user@100.84.126.19:~/semlayer/

# Start remote infrastructure services
cd ~/semlayer
docker-compose -f docker-compose.remote.yml up -d
```

### 2. Prerequisites (on Local Machine)

Ensure these services are running on your localhost:

```bash
# PostgreSQL should be running on localhost:5432
# Hasura should be running on localhost:8085

# Check PostgreSQL
psql -h localhost -p 5432 -U postgres -d alpha -c "SELECT version();"

# Check Hasura
curl http://localhost:8085/healthz
```

### 3. Start Local Golang Services

#### Option A: Quick Start (Recommended)
```bash
# Start Docker Desktop first, then run:
./quick-start.sh

# Or with cleanup:
./quick-start.sh --clean
```

#### Option B: Full Startup Script
```bash
./start-local.sh
```

#### Option C: Manual Docker Compose
```bash
docker-compose -f docker-compose.local-apps.yml up -d
```

### 4. Verify Setup

```bash
# Check all Golang services are running
docker-compose -f docker-compose.local-apps.yml ps

# Test local prerequisites
curl http://localhost:8085/healthz  # Hasura
psql -h localhost -p 5432 -U postgres -d alpha -c "SELECT 1;"  # PostgreSQL

# Test remote infrastructure
curl -I http://100.84.126.19:8086  # Temporal UI
curl -I http://100.84.126.19:9001  # MinIO Console
```

### 5. Docker Cleanup (if needed)

If you need to free up disk space:

```bash
# Quick cleanup
./docker-cleanup.sh

# Or manual cleanup
docker system prune -af
docker volume prune -f
docker image prune -af
```

## Environment Variables

The local services are configured to connect to remote infrastructure:

- `REDIS_ADDR=100.84.126.19:6379`
- `TEMPORAL_HOSTPORT=100.84.126.19:7233`
- `KAFKA_BROKERS=100.84.126.19:9092`
- `HASURA_URL=http://host.docker.internal:8085`
- `TRINO_DSN=http://admin@100.84.126.19:8084?catalog=iceberg&schema=audit`

## Benefits

1. **Reduced Local Resource Usage**: Infrastructure services run remotely
2. **Better Performance**: Local machine focuses on application logic
3. **Scalability**: Infrastructure can be scaled independently
4. **Network Security**: Services communicate over Tailscale VPN

## Monitoring Remote Services

```bash
# Check remote services
ssh user@100.84.126.19 "docker-compose -f semlayer/docker-compose.remote.yml ps"

# View logs
ssh user@100.84.126.19 "docker-compose -f semlayer/docker-compose.remote.yml logs -f [service-name]"
```

## Troubleshooting

1. **Connection Issues**: Ensure Tailscale is running and connected
2. **Port Conflicts**: Check that remote ports are accessible
3. **DNS Resolution**: Verify Tailscale IP is reachable
4. **Firewall**: Ensure Tailscale traffic is allowed

## Migration Notes

- Database remains local (host.docker.internal) for development
- All service discovery updated to use Tailscale IPs
- Volumes persist on remote server for data durability
- Network isolation maintained between local and remote services