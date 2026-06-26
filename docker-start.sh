#!/bin/bash

###############################################################################
#                                                                             #
#                    SEMLAYER DOCKER COMPOSE STARTER                         #
#                                                                             #
#  This script provides easy commands to manage the Semlayer microservices  #
#  stack using Docker Compose.                                               #
#                                                                             #
#  Usage:                                                                    #
#    ./docker-start.sh up           - Start all services                     #
#    ./docker-start.sh down         - Stop all services                      #
#    ./docker-start.sh restart      - Restart all services                   #
#    ./docker-start.sh logs         - Show logs for all services             #
#    ./docker-start.sh logs <service> - Show logs for specific service       #
#    ./docker-start.sh ps           - List all running services              #
#    ./docker-start.sh backend      - Start only backend services            #
#    ./docker-start.sh infra        - Start only infrastructure services     #
#                                                                             #
###############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
COMPOSE_FILE="docker-compose.yml"
OVERRIDE_FILE="docker-compose.override.yml"
INFRA_FILE="infrastructure/docker/docker-compose.yml"
PROJECT_NAME="semlayer"
LOCAL_DB_OVERRIDE="docker-compose.localdb.yml"
BACKEND_LOCAL_DB_OVERRIDE="docker-compose.backend.localdb.yml"
ENV_FILE_LOCAL=".env.local"

# Helper to run docker compose with optional local-db override when USE_LOCAL_POSTGRES=true
run_compose() {
    # When using local Postgres we want to include both the local-db override
    # and a small env-file that points service envs at the host (host.docker.internal)
        # Prefer the small backend-specific local db override if present; otherwise fall back
        # to the global localdb that contains overrides for many services.
        OVERRIDE_TO_USE="$LOCAL_DB_OVERRIDE"
        if [ -f "$BACKEND_LOCAL_DB_OVERRIDE" ]; then
            OVERRIDE_TO_USE="$BACKEND_LOCAL_DB_OVERRIDE"
        fi

        if [ "${USE_LOCAL_POSTGRES:-false}" = "true" ] && [ -f "$OVERRIDE_TO_USE" ]; then
        # Build file argument list dynamically so we can include the infra compose
        # if it exists. This avoids errors when infra services like graphql-engine
        # are defined in separate compose files.
        FILE_ARGS=(-f "$COMPOSE_FILE")
        if [ -f "$INFRA_FILE" ]; then
            FILE_ARGS+=( -f "$INFRA_FILE" )
        fi
        FILE_ARGS+=( -f "$OVERRIDE_FILE" -f "$OVERRIDE_TO_USE" )
        if [ -f "$ENV_FILE_LOCAL" ]; then
            $COMPOSE_CMD --env-file "$ENV_FILE_LOCAL" "${FILE_ARGS[@]}" -p "$PROJECT_NAME" "$@"
        else
            $COMPOSE_CMD "${FILE_ARGS[@]}" -p "$PROJECT_NAME" "$@"
        fi
    else
        FILE_ARGS=(-f "$COMPOSE_FILE")
        if [ -f "$INFRA_FILE" ]; then
            FILE_ARGS+=( -f "$INFRA_FILE" )
        fi
        FILE_ARGS+=( -f "$OVERRIDE_FILE" )
        if [ "${USE_LOCAL_POSTGRES:-false}" = "true" ] && [ -f "$ENV_FILE_LOCAL" ]; then
            $COMPOSE_CMD --env-file "$ENV_FILE_LOCAL" "${FILE_ARGS[@]}" -p "$PROJECT_NAME" "$@"
        else
            $COMPOSE_CMD "${FILE_ARGS[@]}" -p "$PROJECT_NAME" "$@"
        fi
    fi
}

# Function to print messages
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to check if docker and docker-compose are available
check_dependencies() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
}

# Function to get docker compose command
get_compose_cmd() {
    if docker compose version &> /dev/null; then
        echo "docker compose"
    else
        echo "docker-compose"
    fi
}

# Function to start all services
start_all() {
    print_info "Starting all Semlayer microservices..."
    if [ "${USE_LOCAL_POSTGRES:-false}" = "true" ]; then
        print_warning "USE_LOCAL_POSTGRES=true -> will NOT start the 'postgres' container. Using local DB on localhost."
        # Build list of services from compose config and filter out 'postgres'
        # filter out both postgres and postgres-dev services which bind host 5432
        SERVICES=$(run_compose config --services 2>/dev/null | grep -E -v '^postgres(-dev)?$' | xargs)
        if [ -z "$SERVICES" ]; then
            print_error "No services found to start after filtering postgres. Aborting."
            exit 1
        fi
    # Use --no-deps so docker-compose does not attempt to start dependency containers
    # (we expect services to connect to your local Postgres instead).
    run_compose up -d --remove-orphans --no-deps $SERVICES
    else
        run_compose up -d
    fi
    print_success "All services started!"
    print_info "Use './docker-start.sh logs' to view logs"
    print_info "Use './docker-start.sh ps' to see running services"
}

# Function to stop all services
stop_all() {
    print_info "Stopping all Semlayer microservices..."
    run_compose down
    print_success "All services stopped!"
}

# Function to restart all services
restart_all() {
    print_info "Restarting all Semlayer microservices..."
    run_compose restart
    print_success "All services restarted!"
}

# Function to show logs
show_logs() {
    if [ -n "$2" ]; then
        print_info "Showing logs for service: $2"
        run_compose logs -f "$2"
    else
        print_info "Showing logs for all services (Ctrl+C to exit)"
        run_compose logs -f
    fi
}

# Function to list services
list_services() {
    print_info "Running services:"
    run_compose ps
}

# Function to start only backend services
start_backend() {
    print_info "Starting backend microservices..."
    BACKEND_SERVICES=(backend fabric-builder wealth-management ai-builder semantic-engine governance compliance-engine validation-service rule-engine-service notifications-service policy-service search-service event-router)

    # Helper: determine likely Dockerfile locations for a service and skip missing builds
    services_to_start=()
    for svc in "${BACKEND_SERVICES[@]}"; do
        skip=false
        # Common backend-local dockerfiles
        case "$svc" in
            backend|validation-service|rule-engine-service|notifications-service|policy-service|search-service)
                        # check for Dockerfile variants in ./backend (explicit filenames)
                        base="./backend/Dockerfile"
                        df_rule="./backend/Dockerfile.rule-engine"
                        df_notifications="./backend/Dockerfile.notifications"
                        df_validation="./backend/Dockerfile.validation"
                        df_policy="./backend/Dockerfile.policy"
                        df_search="./backend/Dockerfile.search"
                        case "$svc" in
                            rule-engine-service)
                                [ -f "$df_rule" ] && services_to_start+=("$svc") || print_warning "Skipping service '$svc' because $df_rule not found."
                                ;;
                            notifications-service)
                                [ -f "$df_notifications" ] && services_to_start+=("$svc") || print_warning "Skipping service '$svc' because $df_notifications not found."
                                ;;
                            validation-service)
                                [ -f "$df_validation" ] && services_to_start+=("$svc") || print_warning "Skipping service '$svc' because $df_validation not found."
                                ;;
                            policy-service)
                                [ -f "$df_policy" ] && services_to_start+=("$svc") || print_warning "Skipping service '$svc' because $df_policy not found."
                                ;;
                            search-service)
                                [ -f "$df_search" ] && services_to_start+=("$svc") || print_warning "Skipping service '$svc' because $df_search not found."
                                ;;
                            *)
                                # fallback to default Dockerfile
                                if [ -f "$base" ]; then
                                    services_to_start+=("$svc")
                                else
                                    print_warning "Skipping service '$svc' because no Dockerfile found in ./backend."
                                fi
                                ;;
                        esac
                ;;
            fabric-builder|wealth-management|ai-builder|semantic-engine|governance|compliance-engine)
                dir="./services/${svc//-/_}"
                dir_alt="./services/$(echo $svc | sed 's/-/\//')"
                # check typical locations
                if [ -f "./services/${svc//-/_}/Dockerfile" ] || [ -f "./services/${svc}/Dockerfile" ] || [ -f "./services/$svc/Dockerfile" ]; then
                    services_to_start+=("$svc")
                else
                    # try mapping hyphen to path components
                    mapped="./services/$(echo $svc | sed 's/-/\//')/Dockerfile"
                    if [ -f "$mapped" ]; then
                        services_to_start+=("$svc")
                    else
                        print_warning "Skipping service '$svc' because its Dockerfile was not found under ./services (expected a Dockerfile in service dir)."
                    fi
                fi
                ;;
            event-router)
                # event-router may live under backend or services/event-router
                if [ -f "./backend/Dockerfile.event-router" ] || [ -f "./services/event-router/Dockerfile" ]; then
                    services_to_start+=("$svc")
                else
                    print_warning "Skipping 'event-router' because no Dockerfile was found."
                fi
                ;;
            *)
                services_to_start+=("$svc")
                ;;
        esac
    done

    if [ ${#services_to_start[@]} -eq 0 ]; then
        print_error "No backend services to start (all Dockerfiles missing)."
        exit 1
    fi
    # Optionally only start the primary 'backend' service (skip microservices)
    if [ "${ONLY_BACKEND:-false}" = "true" ]; then
        print_info "ONLY_BACKEND=true -> starting only 'backend' service"
    # Use --no-deps so compose does not try to start postgres or other dependent containers
    run_compose up -d --no-deps backend
        print_success "Backend (primary) started."
        return
    fi
    print_info "Starting backend services: ${services_to_start[*]}"
    if [ "${USE_LOCAL_POSTGRES:-false}" = "true" ]; then
        run_compose up -d --remove-orphans "${services_to_start[@]}"
    else
        run_compose up -d "${services_to_start[@]}"
    fi
    print_success "Backend services started!"
}

# Function to start only infrastructure services
start_infra() {
    print_info "Starting infrastructure services..."
    INFRA_SERVICES=(postgres hasura temporal temporal-ui rabbitmq ai-service api-gateway)
    # When USE_LOCAL_POSTGRES=true the local-db override file will ensure a dummy postgres
    # service is used that doesn't bind host ports. We can safely start infra services.
    if [ "${USE_LOCAL_POSTGRES:-false}" = "true" ]; then
        run_compose up -d --remove-orphans "${INFRA_SERVICES[@]}"
    else
        run_compose up -d "${INFRA_SERVICES[@]}"
    fi
    print_success "Infrastructure services started!"
}

# Main script
check_dependencies
COMPOSE_CMD=$(get_compose_cmd)

case "${1:-help}" in
    "up"|"start")
        start_all
        ;;
    "down"|"stop")
        stop_all
        ;;
    "restart")
        restart_all
        ;;
    "logs")
        show_logs "$@"
        ;;
    "ps"|"status")
        list_services
        ;;
    "backend")
        # Support a CLI flag --only to start only the primary backend container
        for arg in "$@"; do
            if [ "$arg" = "--only" ] || [ "$arg" = "-o" ]; then
                ONLY_BACKEND=true
            fi
        done
        start_backend
        ;;
    "infra")
        start_infra
        ;;
    "help"|*)
        echo "Semlayer Docker Compose Manager"
        echo ""
        echo "Usage: $0 <command>"
        echo ""
        echo "Commands:"
        echo "  up         Start all services"
        echo "  down       Stop all services"
        echo "  restart    Restart all services"
        echo "  logs       Show logs for all services"
        echo "  logs <svc> Show logs for specific service"
        echo "  ps         List running services"
        echo "  backend    Start only backend services"
        echo "  infra      Start only infrastructure services"
        echo "  help       Show this help"
        echo ""
        echo "Examples:"
        echo "  $0 up"
        echo "  $0 logs backend"
        echo "  $0 backend"
        ;;
esac