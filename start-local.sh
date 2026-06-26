#!/bin/bash

# Local Services Startup Script
# Starts local Golang applications with PostgreSQL and Hasura

set -e

echo "🚀 Starting SemLayer Local Services"
echo "==================================="
echo ""

# Start services
echo "🐳 Starting local services (PostgreSQL, Hasura, and Golang apps)..."
docker-compose -f docker-compose.local-apps.yml up -d

# Wait for services to initialize
echo "⏳ Waiting for PostgreSQL and Hasura to initialize..."
sleep 30

# Check service status
echo "📊 Checking service status..."
docker-compose -f docker-compose.local-apps.yml ps

echo ""
echo "✅ Local services deployment complete!"
echo ""
echo "🌐 Local Service Endpoints:"
echo "   - Hasura Console: http://localhost:8085"
echo "   - Hasura Admin Secret: myadminsecret"
echo "   - PostgreSQL: localhost:5432"
echo "   - Backend API: http://localhost:8082"
echo "   - API Gateway: http://localhost:8001"
echo ""
echo "🔗 Remote Infrastructure (via Tailscale):"
echo "   - Temporal UI: http://100.84.126.19:8086"
echo "   - MinIO Console: http://100.84.126.19:9001"
echo "   - Trino: http://100.84.126.19:8084"
echo ""