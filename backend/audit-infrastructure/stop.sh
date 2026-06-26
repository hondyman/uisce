#!/bin/bash
set -e

echo "🔧 Stopping Audit Infrastructure..."
echo ""

cd "$(dirname "$0")"

# Check if running
if ! docker-compose ps | grep -q "Up"; then
    echo "No services running"
    exit 0
fi

# Stop gracefully
echo "Stopping containers..."
docker-compose down

echo ""
echo "✓ All services stopped"
echo ""
echo "To remove all data (Kafka, S3, Iceberg):"
echo "  docker-compose down -v"
echo ""
