#!/bin/bash
# Quick start script for ATR module

set -e

echo "🚀 Starting AI Trade Reconciliation..."

# Check prerequisites
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Please install Go 1.24+"
    exit 1
fi

# Check environment
if [ -z "$XAI_API_KEY" ]; then
    echo "⚠️  XAI_API_KEY not set. AI matching will fail."
    echo "   Set: export XAI_API_KEY=your-key-here"
fi

# Build and start
echo "📦 Building services..."
docker-compose build

echo "🔌 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services..."
sleep 10

# Apply migrations
echo "📊 Applying database migrations..."
docker-compose exec -T atr-db psql -U postgres -d alpha < db/migrations/001_create_reconciliation_tables.sql

echo "✅ Services running:"
echo "   - Backend API: http://localhost:8080"
echo "   - Frontend: http://localhost:3000"
echo "   - Temporal UI: http://localhost:8233"
echo "   - Database: localhost:5432"

echo ""
echo "🧪 Test endpoints:"
echo "   curl http://localhost:8080/health"
echo "   curl http://localhost:8080/api/reconciliation/results"
echo ""
echo "📖 View logs: docker-compose logs -f atr-backend"
echo "🛑 Stop services: docker-compose down"
