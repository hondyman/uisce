#!/bin/bash

# SemLayer Docker Stop Script
# Stops all running SemLayer services

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Stopping SemLayer services...${NC}"
echo ""

docker compose down --remove-orphans

echo ""
echo -e "${GREEN}✓ All services stopped${NC}"
