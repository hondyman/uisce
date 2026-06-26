#!/bin/bash

# Quick Local Services Startup
# Ensures Docker is running and starts local services

set -e

echo "🚀 Quick Start Local Services"
echo "============================"
echo ""

# Check if Docker is running
echo "🐳 Checking Docker status..."
if ! docker ps >/dev/null 2>&1; then
    echo "❌ Docker is not running!"
    echo ""
    echo "Please start Docker Desktop:"
    echo "1. Open Docker Desktop application"
    echo "2. Wait for it to fully start (whale icon in menu bar)"
    echo "3. Run this script again"
    echo ""
    echo "Or run: open -a Docker"
    exit 1
fi

echo "✅ Docker is running"
echo ""

# Optional: Quick cleanup if requested
if [ "$1" = "--clean" ]; then
    echo "🧹 Running quick cleanup..."
    docker system prune -f >/dev/null 2>&1
    echo "✅ Cleanup complete"
    echo ""
fi

# Start services
echo "🐳 Starting local services..."
docker-compose -f docker-compose.local-apps.yml up -d

# Wait for services
echo "⏳ Waiting for services to initialize..."
sleep 30

# Check status
echo "📊 Service status:"
docker-compose -f docker-compose.local-apps.yml ps

echo ""
echo "✅ Local services started!"
echo ""
echo "🌐 Access points:"
echo "   - Hasura Console: http://localhost:8085"
echo "   - Backend API: http://localhost:8082"
echo "   - API Gateway: http://localhost:8001"
echo ""
echo ""