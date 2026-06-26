#!/bin/bash

# RabbitMQ Setup Script for Semlayer
# This script sets up RabbitMQ for local development with event publishing

set -e

echo "🚀 Starting RabbitMQ setup for Semlayer..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ============================================================================
# STEP 1: Check if Docker is installed
# ============================================================================
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ Docker not found. Please install Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker found${NC}"

# ============================================================================
# STEP 2: Start RabbitMQ container
# ============================================================================
echo ""
echo "Starting RabbitMQ container..."

if docker ps -a --format '{{.Names}}' | grep -q '^semlayer-rabbitmq$'; then
    echo "RabbitMQ container already exists. Starting it..."
    docker start semlayer-rabbitmq || true
else
    echo "Creating and starting RabbitMQ container..."
    docker-compose -f docker-compose.rabbitmq.yml up -d
fi

echo -e "${GREEN}✓ RabbitMQ container started${NC}"

# ============================================================================
# STEP 3: Wait for RabbitMQ to be ready
# ============================================================================
echo ""
echo "Waiting for RabbitMQ to be ready (up to 60 seconds)..."

for i in {1..60}; do
    if docker exec semlayer-rabbitmq rabbitmq-diagnostics -q ping 2>/dev/null; then
        echo -e "${GREEN}✓ RabbitMQ is ready!${NC}"
        break
    fi
    if [ $i -eq 60 ]; then
        echo -e "${RED}✗ RabbitMQ failed to start${NC}"
        exit 1
    fi
    echo -n "."
    sleep 1
done

# ============================================================================
# STEP 4: Display connection information
# ============================================================================
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ RabbitMQ Setup Complete!${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "📊 RabbitMQ Management UI:"
echo "   URL: http://localhost:15672"
echo "   Username: guest"
echo "   Password: guest"
echo ""
echo "📮 AMQP Connection:"
echo "   URL: amqp://guest:guest@localhost:5672"
echo ""
echo "🔗 Environment Variables:"
echo "   export RABBITMQ_URL=amqp://guest:guest@localhost:5672"
echo ""
echo "📝 Docker Commands:"
echo "   View logs: docker logs -f semlayer-rabbitmq"
echo "   Stop: docker stop semlayer-rabbitmq"
echo "   Start: docker start semlayer-rabbitmq"
echo "   Remove: docker-compose -f docker-compose.rabbitmq.yml down"
echo ""
echo "🧪 Test Publishing Events:"
echo "   See RABBITMQ_ARCHITECTURE_DECISION.md for curl examples"
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
