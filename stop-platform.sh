#!/bin/bash

# 🛑 Stop UMA Alpha + Attribution Alpha Platform

set -e

echo "🛑 Stopping UMA Alpha + Attribution Alpha Platform..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PROJECT_ROOT="/Users/eganpj/GitHub/semlayer"

# Function to print status
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Stop all services
stop_all() {
    cd "$PROJECT_ROOT"

    # Stop background processes
    if [ -f "backend/backend.pid" ]; then
        print_status "Stopping backend..."
        kill $(cat backend/backend.pid) 2>/dev/null || true
        rm backend/backend.pid
        print_success "Backend stopped"
    fi

    if [ -f "frontend/frontend.pid" ]; then
        print_status "Stopping frontend..."
        kill $(cat frontend/frontend.pid) 2>/dev/null || true
        rm frontend/frontend.pid
        print_success "Frontend stopped"
    fi

    if [ -f "temporal/temporal.pid" ]; then
        print_status "Stopping Temporal workers..."
        kill $(cat temporal/temporal.pid) 2>/dev/null || true
        rm temporal/temporal.pid
        print_success "Temporal workers stopped"
    fi

    # Stop Docker services
    if [ -f "docker-compose.yml" ]; then
        print_status "Stopping Docker services..."
        docker-compose down
        print_success "Docker services stopped"
    fi

    # Clean up log files
    print_status "Cleaning up log files..."
    rm -f backend/*.log frontend/*.log temporal/*.log
    print_success "Log files cleaned"

    echo
    print_success "UMA Alpha + Attribution Alpha Platform stopped successfully"
    echo
    echo "💡 To restart: ./deploy-platform.sh"
    echo
}

stop_all</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/stop-platform.sh