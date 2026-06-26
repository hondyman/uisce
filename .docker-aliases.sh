#!/bin/bash

# Add these aliases to your ~/.zshrc or ~/.bashrc for easier docker-compose management

# Semlayer Docker Compose Helpers
alias dc="docker compose -f docker-compose.dev.simple.yml"
alias dcup="docker compose -f docker-compose.dev.simple.yml up -d"
alias dcdown="docker compose -f docker-compose.dev.simple.yml down"
alias dcrestart="docker compose -f docker-compose.dev.simple.yml restart"
alias dclogs="docker compose -f docker-compose.dev.simple.yml logs -f"
alias dcps="docker compose -f docker-compose.dev.simple.yml ps"
alias dcstatus="./scripts/check-services.sh"

# Example usage:
# dcup          - Start all services
# dcdown        - Stop all services  
# dcps          - Show container status
# dclogs        - Follow all logs
# dclogs hasura - Follow hasura logs only
# dcstatus      - Check which services are responding
