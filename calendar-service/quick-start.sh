#!/bin/bash
# Quick start script for Calendar Service development

set -e

echo "🚀 Calendar Service - Local Development Setup"
echo "============================================="

# Check prerequisites
echo "✓ Checking prerequisites..."
command -v go >/dev/null 2>&1 || { echo "Go 1.21+ required"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Docker required"; exit 1; }
command -v psql >/dev/null 2>&1 || { echo "PostgreSQL client required"; exit 1; }

# Load environment
if [ ! -f .env.local ]; then
  echo "Creating .env.local..."
  cat > .env.local << EOF
CALENDAR_SERVICE_PORT=8081
LOG_LEVEL=debug
ENVIRONMENT=dev

HASURA_ENDPOINT=http://localhost:8080/v1/graphql
HASURA_ADMIN_SECRET=myadminsecretkey

REDIS_URL=redis://localhost:6379
REDPANDA_BROKERS=localhost:9092
ENABLE_CDC=true

TEMPORAL_HOST_PORT=localhost:7233
CACHE_TTL_MINUTES=60
EOF
  echo "✓ Created .env.local (update with your values)"
fi

# Download dependencies
echo "📦 Downloading Go dependencies..."
go mod download

# Tidy modules
echo "🧹 Tidying Go modules..."
go mod tidy

# Build
echo "🔨 Building Calendar Service..."
go build -v -o bin/calendar-service ./cmd/server

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Start infrastructure: docker-compose up -d"
echo "2. Apply schema: psql -f docs/schema.sql"
echo "3. Run service: ./bin/calendar-service"
echo "4. Test: curl http://localhost:8081/health"
echo ""
echo "Development endpoints:"
echo "  - Health: http://localhost:8081/health"
echo "  - Readiness: http://localhost:8081/ready"
echo "  - Calendar API: http://localhost:8081/api/v1/calendars"
echo ""
