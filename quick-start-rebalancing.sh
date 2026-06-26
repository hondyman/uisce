#!/bin/bash

# Quick Start - Portfolio Rebalancing System
# Get the system running in 60 seconds

echo "🚀 Portfolio Rebalancing System - Quick Start"
echo "=============================================="
echo ""

# Navigate to rebalancing directory
cd "$(dirname "$0")/rebalancing" || {
    echo "❌ Error: Cannot find rebalancing directory"
    exit 1
}

echo "📋 Checking prerequisites..."
docker --version > /dev/null 2>&1 || { echo "❌ Docker not installed"; exit 1; }
docker-compose --version > /dev/null 2>&1 || { echo "❌ Docker Compose not installed"; exit 1; }
echo "✅ Prerequisites OK"
echo ""

echo "📝 Setting up environment..."
if [ ! -f .env ]; then
    cp .env.example .env
    echo "✅ Created .env file from template"
else
    echo "✅ .env file already exists"
fi
echo ""

echo "🐳 Starting Docker services..."
docker-compose up -d
echo "✅ Services starting..."
echo ""

echo "⏳ Waiting for services to be healthy (30 seconds)..."
sleep 30

# Check if services are running
if docker-compose ps | grep -q "healthy"; then
    echo "✅ Services are healthy!"
else
    echo "⚠️  Services still starting, checking in 30 seconds..."
    sleep 30
fi
echo ""

echo "🎉 System is ready!"
echo ""
echo "Access your system:"
echo "  🌐 Dashboard:       http://localhost:3000"
echo "  📊 Hasura Console:  http://localhost:8080"
echo "  ⏱️  Temporal UI:      http://localhost:8081"
echo "  📬 RabbitMQ Admin:   http://localhost:15672 (guest/guest)"
echo "  🔌 API:              http://localhost:8090/health"
echo ""
echo "Useful commands:"
echo "  📋 View logs:        docker-compose logs -f"
echo "  🛑 Stop services:    docker-compose down"
echo "  ♻️  Restart service:   docker-compose restart rebalance-api"
echo "  🗄️  Connect to DB:    docker-compose exec postgres psql -U postgres -d portfolio"
echo ""
echo "📚 Documentation:"
echo "  Complete guide:     ./DOCKER_DEPLOYMENT.md"
echo "  Getting started:    ./README.md"
echo "  Quick reference:    ../REBALANCING_INDEX.md"
echo ""
echo "👉 Next: Open http://localhost:3000 to access the dashboard"
