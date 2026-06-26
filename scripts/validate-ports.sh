#!/bin/bash

# ============================================================================
# PORT VALIDATION SCRIPT
# ============================================================================
# This script validates that:
# 1. All ports in .env.ports are unique (no duplicates)
# 2. All ports are valid (1-65535)
# 3. All required environment variables are defined
# 4. All services have a corresponding port defined
#
# Usage: bash scripts/validate-ports.sh
# ============================================================================

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env.ports exists
if [ ! -f ".env.ports" ]; then
  echo -e "${RED}❌ ERROR: .env.ports not found${NC}"
  echo "   Please create .env.ports in the root directory"
  exit 1
fi

echo "=================================================="
echo "PORT VALIDATION REPORT"
echo "=================================================="

# Load the port variables
source .env.ports

# Array to store all ports for uniqueness check
declare -a all_ports

# Function to validate a port
validate_port() {
  local port_name=$1
  local port_value=$2

  if [ -z "$port_value" ]; then
    echo -e "${RED}❌ $port_name: NOT SET${NC}"
    return 1
  fi

  # Check if port is a valid number between 1-65535
  if ! [[ "$port_value" =~ ^[0-9]+$ ]] || [ "$port_value" -lt 1 ] || [ "$port_value" -gt 65535 ]; then
    echo -e "${RED}❌ $port_name=$port_value: INVALID (must be 1-65535)${NC}"
    return 1
  fi

  echo -e "${GREEN}✓ $port_name=$port_value${NC}"
  all_ports+=("$port_value")
  return 0
}

echo ""
echo "1. CHECKING INDIVIDUAL PORTS"
echo "----------------------------"

# Validate each port
validate_port "PORT_BACKEND_API" "$PORT_BACKEND_API"
validate_port "PORT_FABRIC_BUILDER" "$PORT_FABRIC_BUILDER"
validate_port "PORT_LEGACY_GATEWAY" "$PORT_LEGACY_GATEWAY"
validate_port "PORT_HASURA_GRAPHQL" "$PORT_HASURA_GRAPHQL"
validate_port "PORT_RABBITMQ_AMQP" "$PORT_RABBITMQ_AMQP"
validate_port "PORT_RABBITMQ_MANAGEMENT" "$PORT_RABBITMQ_MANAGEMENT"
validate_port "PORT_TEMPORAL_SERVER" "$PORT_TEMPORAL_SERVER"
validate_port "PORT_TEMPORAL_UI" "$PORT_TEMPORAL_UI"
validate_port "PORT_VITE_DEV_SERVER" "$PORT_VITE_DEV_SERVER"
validate_port "PORT_POSTGRES_HOST" "$PORT_POSTGRES_HOST"

echo ""
echo "2. CHECKING FOR DUPLICATE PORTS"
echo "--------------------------------"

# Check for duplicates
duplicates=0
for port in "${all_ports[@]}"; do
  count=$(echo "${all_ports[@]}" | tr ' ' '\n' | grep -c "^$port$")
  if [ "$count" -gt 1 ]; then
    echo -e "${RED}❌ Port $port is used $count times (DUPLICATE)${NC}"
    duplicates=$((duplicates + 1))
  fi
done

if [ "$duplicates" -eq 0 ]; then
  echo -e "${GREEN}✓ All ports are unique${NC}"
else
  echo -e "${RED}❌ Found $duplicates duplicate port(s)${NC}"
  exit 1
fi

echo ""
echo "3. CHECKING ENVIRONMENT VARIABLES"
echo "----------------------------------"

# Check required VITE_* variables
if [ -z "$VITE_GRAPHQL_ENDPOINT" ]; then
  echo -e "${YELLOW}⚠ VITE_GRAPHQL_ENDPOINT: NOT SET (uses .env fallback)${NC}"
else
  echo -e "${GREEN}✓ VITE_GRAPHQL_ENDPOINT: $VITE_GRAPHQL_ENDPOINT${NC}"
fi

if [ -z "$VITE_API_BASE_URL" ]; then
  echo -e "${YELLOW}⚠ VITE_API_BASE_URL: NOT SET (uses .env fallback)${NC}"
else
  echo -e "${GREEN}✓ VITE_API_BASE_URL: $VITE_API_BASE_URL${NC}"
fi

if [ -z "$VITE_GRAPHQL_ADMIN_SECRET" ]; then
  echo -e "${RED}❌ VITE_GRAPHQL_ADMIN_SECRET: NOT SET (required)${NC}"
  exit 1
else
  echo -e "${GREEN}✓ VITE_GRAPHQL_ADMIN_SECRET: [SET]${NC}"
fi

echo ""
echo "4. PORT SUMMARY"
echo "---------------"
echo "Total ports allocated: ${#all_ports[@]}"
echo ""
echo "Ports by range:"
echo "  Backend Services (8000-8099):"
echo "    - Backend API: $PORT_BACKEND_API"
echo "    - Fabric Builder: $PORT_FABRIC_BUILDER"
echo "    - Legacy Gateway: $PORT_LEGACY_GATEWAY"
echo ""
echo "  GraphQL & Data (8200-8299):"
echo "    - Hasura GraphQL: $PORT_HASURA_GRAPHQL"
echo ""
echo "  Message Queue (5600-5700):"
echo "    - RabbitMQ AMQP: $PORT_RABBITMQ_AMQP"
echo "    - RabbitMQ Management: $PORT_RABBITMQ_MANAGEMENT"
echo ""
echo "  Workflow Engine (7200-7300):"
echo "    - Temporal Server: $PORT_TEMPORAL_SERVER"
echo "    - Temporal UI: $PORT_TEMPORAL_UI"
echo ""
echo "  Frontend (5000-5200):"
echo "    - Vite Dev Server: $PORT_VITE_DEV_SERVER"
echo ""
echo "  Database (5400-5500):"
echo "    - PostgreSQL (Host): $PORT_POSTGRES_HOST"
echo ""

echo "=================================================="
echo -e "${GREEN}✓ ALL VALIDATIONS PASSED${NC}"
echo "=================================================="
echo ""
echo "Next steps:"
echo "  1. Start services: docker compose --env-file .env.ports -f docker-compose.dev.simple.yml up -d"
echo "  2. Start frontend: cd frontend && npm run dev"
echo "  3. Access app at: http://localhost:$PORT_VITE_DEV_SERVER"
echo ""
