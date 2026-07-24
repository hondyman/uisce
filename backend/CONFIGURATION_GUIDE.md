# SemLayer Configuration Guide

This guide shows you how to easily configure the SemLayer platform for different environments.

## Quick Start

### Option 1: Use Environment Variables (Easiest)

```bash
# Copy this to your .env file or set as environment variables
export ENVIRONMENT=development
export DRIVER=postgres
export DSN="host=localhost port=5432 user=postgres password=mypassword dbname=semlayer sslmode=disable"
export PORT=":8080"
export JWT_SECRET="your-super-secret-jwt-key-change-in-production"
export LOG_LEVEL=debug
export ENABLE_METRICS=true
export ENABLE_CACHING=true
export ENABLE_SECURITY=true
```

### Option 2: Use Configuration File

Create a `config.yaml` file:

```yaml
# Basic configuration for development
yaml_dir: "./config"
driver: "postgres"
dsn: "host=localhost port=5432 user=postgres password=mypassword dbname=semlayer sslmode=disable"
port: ":8080"
environment: "development"
log_level: "debug"
enable_metrics: true
enable_caching: true
enable_security: true
jwt_secret: "your-super-secret-jwt-key-change-in-production"
```

## Configuration Examples

### Development Environment

```yaml
yaml_dir: "./config"
driver: "postgres"
dsn: "host=localhost port=5432 user=postgres password=devpassword dbname=semlayer sslmode=disable"
port: ":8080"
environment: "development"
log_level: "debug"
enable_metrics: true
enable_caching: true
enable_security: false  # Disable security for easier development
jwt_secret: "dev-jwt-secret-key"
db_max_open_conns: 10
db_max_idle_conns: 2
cache_num_shards: 4
cache_max_size_per_shard: 5000
```

### Production Environment

```yaml
yaml_dir: "/app/config"
driver: "postgres"
dsn: "postgres://produser:prodpassword@prod-db-host:5432/proddb?sslmode=require"
port: ":8080"
environment: "production"
log_level: "warn"
enable_metrics: true
enable_caching: true
enable_security: true
jwt_secret: "${JWT_SECRET}"  # Use environment variable
redis_addr: "redis-cluster:6379"
db_max_open_conns: 50
db_max_idle_conns: 10
cache_num_shards: 32
cache_max_size_per_shard: 50000
security_rate_limit_requests: 1000
```

### Staging Environment

```yaml
yaml_dir: "/app/config"
driver: "postgres"
dsn: "postgres://stageuser:stagepassword@stage-db-host:5432/stagedb?sslmode=require"
port: ":8080"
environment: "staging"
log_level: "info"
enable_metrics: true
enable_caching: true
enable_security: true
jwt_secret: "${JWT_SECRET}"
redis_addr: "redis-stage:6379"
db_max_open_conns: 30
db_max_idle_conns: 8
cache_num_shards: 16
cache_max_size_per_shard: 25000
security_rate_limit_requests: 500
```

## Environment Variables Reference

### Core Settings
- `ENVIRONMENT`: Environment (development/staging/production)
- `YAML_DIR`: Directory containing YAML configuration files
- `DRIVER`: Database driver (postgres/snowflake/mssql)
- `DSN`: Database connection string
- `PORT`: HTTP server port (default: :8080)
- `PG_PORT`: PostgreSQL wire port (default: :5432)

### Enhanced Settings
- `REDIS_ADDR`: Redis server address for caching
- `JWT_SECRET`: Secret key for JWT token signing
- `GRAPHQL_URL`: GraphQL endpoint URL
- `LOG_LEVEL`: Logging level (debug/info/warn/error)

### Feature Flags
- `ENABLE_METRICS`: Enable metrics collection (true/false)
- `ENABLE_CACHING`: Enable advanced caching (true/false)
- `ENABLE_SECURITY`: Enable security features (true/false)

### Database Settings
- `DB_MAX_OPEN_CONNS`: Maximum open database connections
- `DB_MAX_IDLE_CONNS`: Maximum idle database connections

### Cache Settings
- `CACHE_NUM_SHARDS`: Number of cache shards
- `CACHE_MAX_SIZE_PER_SHARD`: Maximum entries per shard
- `CACHE_DEFAULT_TTL`: Default cache TTL duration

### Security Settings
- `RATE_LIMIT_ENABLED`: Enable rate limiting (true/false)
- `RATE_LIMIT_REQUESTS`: Number of requests allowed per window
- `RATE_LIMIT_WINDOW`: Rate limit time window (e.g., "1m", "5m")

## Docker Configuration

### docker-compose.yml for Development

```yaml
version: '3.8'
services:
  semlayer:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ENVIRONMENT=development
      - DRIVER=postgres
      - DSN=host=db port=5432 user=postgres password=password dbname=semlayer sslmode=disable
      - PORT=:8080
      - JWT_SECRET=dev-secret-key
      - LOG_LEVEL=debug
      - ENABLE_METRICS=true
      - ENABLE_CACHING=true
      - ENABLE_SECURITY=false
    depends_on:
      - db
      - redis

  db:
    image: postgres:13
    environment:
      POSTGRES_DB: semlayer
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

### Production Docker Configuration

```yaml
version: '3.8'
services:
  semlayer:
    image: semlayer:latest
    ports:
      - "8080:8080"
      - "9090:9090"  # Metrics port
    environment:
      - ENVIRONMENT=production
      - DRIVER=postgres
      - DSN=${DATABASE_URL}
      - PORT=:8080
      - JWT_SECRET=${JWT_SECRET}
      - REDIS_ADDR=redis:6379
      - LOG_LEVEL=warn
      - ENABLE_METRICS=true
      - ENABLE_CACHING=true
      - ENABLE_SECURITY=true
      - DB_MAX_OPEN_CONNS=50
      - CACHE_NUM_SHARDS=32
    depends_on:
      - redis
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  redis_data:
```

## Kubernetes Configuration

### ConfigMap for Application Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: semlayer-config
data:
  config.yaml: |
    yaml_dir: "/app/config"
    driver: "postgres"
    dsn: "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):5432/$(DB_NAME)?sslmode=require"
    port: ":8080"
    environment: "production"
    log_level: "info"
    enable_metrics: true
    enable_caching: true
    enable_security: true
    redis_addr: "redis-service:6379"
    db_max_open_conns: "50"
    cache_num_shards: "32"
    cache_max_size_per_shard: "50000"
```

### Deployment Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: semlayer
spec:
  replicas: 3
  selector:
    matchLabels:
      app: semlayer
  template:
    metadata:
      labels:
        app: semlayer
    spec:
      containers:
      - name: semlayer
        image: semlayer:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090  # Metrics
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: semlayer-secrets
              key: jwt-secret
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: db-secrets
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secrets
              key: password
        - name: DB_HOST
          valueFrom:
            configMapKeyRef:
              name: db-config
              key: host
        - name: DB_NAME
          valueFrom:
            configMapKeyRef:
              name: db-config
              key: database
        volumeMounts:
        - name: config-volume
          mountPath: /app/config
      volumes:
      - name: config-volume
        configMap:
          name: semlayer-config
```

## Configuration Validation

The application automatically validates your configuration on startup:

```bash
# The app will show validation errors if configuration is invalid
./semlayer --config config.yaml

# Or with environment variables
ENVIRONMENT=production JWT_SECRET=your-secret ./semlayer
```

## Troubleshooting

### Common Configuration Issues

1. **Database Connection Failed**
   ```bash
   # Check your DSN format
   export DSN="host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable"
   ```

2. **JWT Secret Not Set**
   ```bash
   # Generate a secure secret
   openssl rand -base64 32
   export JWT_SECRET="generated-secret-here"
   ```

3. **Redis Connection Failed**
   ```bash
   # Check Redis is running
   redis-cli ping
   export REDIS_ADDR="localhost:6379"
   ```

4. **Port Already in Use**
   ```bash
   # Find what's using the port
   lsof -i :8080
   # Change port
   export PORT=":8081"
   ```

### Configuration Debugging

```bash
# Enable debug logging to see configuration loading
export LOG_LEVEL=debug

# The app will print the loaded configuration (with secrets masked)
./semlayer --config config.yaml --print-config
```

## Best Practices

1. **Never commit secrets** to version control
2. **Use environment variables** for sensitive data
3. **Validate configuration** before deployment
4. **Use different settings** for each environment
5. **Monitor configuration** changes in production
6. **Document custom configurations** for your team

## Quick Setup Scripts

### Development Setup Script

```bash
#!/bin/bash
# setup-dev.sh

# Create .env file for development
cat > .env << EOF
ENVIRONMENT=development
DRIVER=postgres
DSN=host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable
PORT=:8080
JWT_SECRET=dev-secret-key-for-development-only
LOG_LEVEL=debug
ENABLE_METRICS=true
ENABLE_CACHING=true
ENABLE_SECURITY=false
DB_MAX_OPEN_CONNS=10
CACHE_NUM_SHARDS=4
EOF

echo "Development environment configured!"
echo "Run: source .env && ./semlayer"
```

### Production Setup Script

```bash
#!/bin/bash
# setup-prod.sh

# Generate secure JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Create production environment file
cat > .env.prod << EOF
ENVIRONMENT=production
DRIVER=postgres
DSN=\${DATABASE_URL}
PORT=:8080
JWT_SECRET=${JWT_SECRET}
REDIS_ADDR=\${REDIS_URL}
LOG_LEVEL=warn
ENABLE_METRICS=true
ENABLE_CACHING=true
ENABLE_SECURITY=true
DB_MAX_OPEN_CONNS=50
CACHE_NUM_SHARDS=32
RATE_LIMIT_REQUESTS=1000
EOF

echo "Production environment configured!"
echo "JWT Secret: ${JWT_SECRET}"
echo "Make sure to set DATABASE_URL and REDIS_URL environment variables"
```

This configuration system makes it incredibly easy to set up and manage the SemLayer platform across different environments while maintaining security and performance best practices.
