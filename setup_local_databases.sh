#!/bin/bash

# Preaggregation Database Setup for Local Development
# Sets up both alpha (metadata/config) and northwind (aggregates) databases

set -e

echo "🚀 Setting up Preaggregation Databases for Local Development"
echo "=========================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

print_status "Docker is running ✓"

# Create network for database communication
print_status "Creating Docker network..."
docker network create semlayer-network 2>/dev/null || print_warning "Network already exists"

# Start Alpha database (for metadata and config)
print_status "Starting Alpha database container..."
docker run --name postgres-alpha \
  --network semlayer-network \
  -e POSTGRES_DB=alpha \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  -d postgres:13

# Start Northwind database (for aggregates)
print_status "Starting Northwind database container..."
docker run --name postgres-northwind \
  --network semlayer-network \
  -e POSTGRES_DB=northwind \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5433:5432 \
  -d postgres:13

print_status "Waiting for databases to start..."
sleep 15

# Function to wait for database to be ready
wait_for_db() {
    local container_name=$1
    local db_name=$2
    local port=$3

    print_status "Waiting for $db_name database to be ready..."
    for i in {1..30}; do
        if docker exec $container_name pg_isready -U postgres -d $db_name > /dev/null 2>&1; then
            print_success "$db_name database is ready!"
            return 0
        fi
        sleep 2
    done

    print_error "$db_name database failed to start"
    return 1
}

# Wait for both databases
wait_for_db postgres-alpha alpha 5432
wait_for_db postgres-northwind northwind 5433

# Setup Alpha database (metadata and config)
print_status "Setting up Alpha database schema..."

# Create semantic layer schema in Alpha
docker exec postgres-alpha psql -U postgres -d alpha -c "
CREATE SCHEMA IF NOT EXISTS semantic_layer;
CREATE SCHEMA IF NOT EXISTS config;
" 2>/dev/null || print_warning "Some schema creation may have failed"

# Setup Northwind database (aggregates)
print_status "Setting up Northwind database schema..."

# Create aggregates schema in Northwind
docker exec postgres-northwind psql -U postgres -d northwind -c "
CREATE SCHEMA IF NOT EXISTS aggregates;
" 2>/dev/null || print_warning "Aggregates schema creation may have failed"

# Copy and run the preaggregation schema migration
print_status "Setting up preaggregation tables..."

# Copy migration file to containers
docker cp /Users/eganpj/GitHub/semlayer/backend/migrations/000015_preaggregation_schema.sql postgres-alpha:/tmp/ 2>/dev/null || print_warning "Could not copy migration file to alpha"
docker cp /Users/eganpj/GitHub/semlayer/backend/migrations/000015_preaggregation_schema.sql postgres-northwind:/tmp/ 2>/dev/null || print_warning "Could not copy migration file to northwind"

# Run migration in Alpha (for metadata/config)
print_status "Running preaggregation migration in Alpha database..."
docker exec postgres-alpha psql -U postgres -d alpha -f /tmp/000015_preaggregation_schema.sql 2>/dev/null || print_warning "Migration may have partial failures in alpha"

# Run migration in Northwind (for aggregates) - modify schema name
print_status "Setting up aggregation tables in Northwind database..."
docker exec postgres-northwind psql -U postgres -d northwind -c "
-- Create aggregation tables in northwind database
CREATE SCHEMA IF NOT EXISTS aggregates;

-- Preaggregated metrics table
CREATE TABLE IF NOT EXISTS aggregates.preaggregated_metrics (
    id VARCHAR(255) PRIMARY KEY,
    node_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    value DECIMAL(20, 8) NOT NULL,
    grain JSONB NOT NULL,
    grain_values JSONB NOT NULL,
    last_refresh TIMESTAMP WITH TIME ZONE NOT NULL,
    refresh_schedule VARCHAR(50) NOT NULL,
    source_formula TEXT NOT NULL,
    data_quality JSONB NOT NULL,
    business_context TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for fast querying
CREATE INDEX IF NOT EXISTS idx_agg_node_id ON aggregates.preaggregated_metrics(node_id);
CREATE INDEX IF NOT EXISTS idx_agg_grain ON aggregates.preaggregated_metrics USING GIN(grain);
CREATE INDEX IF NOT EXISTS idx_agg_grain_values ON aggregates.preaggregated_metrics USING GIN(grain_values);
CREATE INDEX IF NOT EXISTS idx_agg_last_refresh ON aggregates.preaggregated_metrics(last_refresh);

-- Audit table
CREATE TABLE IF NOT EXISTS aggregates.preaggregation_audit (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(255) NOT NULL,
    metric_node_id VARCHAR(255) NOT NULL,
    grain JSONB NOT NULL,
    records_processed INTEGER NOT NULL,
    execution_time_ms INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Sample data for testing
INSERT INTO aggregates.preaggregated_metrics (
    id, node_id, name, value, grain, grain_values, last_refresh,
    refresh_schedule, source_formula, data_quality, business_context
) VALUES
    (
        'sample_net_irr_fund001_2024_09',
        'private_markets_net_irr',
        'Net IRR',
        0.1234,
        '[\"fund_id\", \"month\"]',
        '{\"fund_id\": \"FUND001\", \"month\": \"2024-09-01T00:00:00Z\"}',
        NOW(),
        'daily',
        '=XIRR({cash_flows}, {dates})',
        '{\"completeness_score\": 0.95, \"freshness_hours\": 0, \"source_count\": 24, \"last_validated\": \"' || NOW()::text || '\"}',
        'Net Internal Rate of Return after fees - preaggregated for performance monitoring'
    )
ON CONFLICT (id) DO NOTHING;
" 2>/dev/null || print_warning "Northwind setup may have partial failures"

# Update configuration file
print_status "Updating configuration file..."

# Backup original config
cp /Users/eganpj/GitHub/semlayer/backend/config.yaml /Users/eganpj/GitHub/semlayer/backend/config.yaml.backup 2>/dev/null || true

# Update config to use localhost databases
cat > /Users/eganpj/GitHub/semlayer/backend/config.yaml << EOF
yaml_dir: ./models
driver: postgres
# Alpha database for metadata and config
dsn: "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
# Northwind database for aggregates
aggregates_dsn: "postgres://postgres:postgres@localhost:5433/northwind?sslmode=disable"
port: :8080
pg_port: :5432
graphql_url: "http://graphql-engine:8080/v1/graphql"
EOF

print_success "Configuration updated!"

# Verify setup
print_status "Verifying database setup..."

# Check Alpha database
ALPHA_TABLES=$(docker exec postgres-alpha psql -U postgres -d alpha -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'semantic_layer';" 2>/dev/null || echo "0")
if [ "$ALPHA_TABLES" -gt "0" ]; then
    print_success "Alpha database: $ALPHA_TABLES tables created"
else
    print_warning "Alpha database: No tables found"
fi

# Check Northwind database
NORTHWIND_TABLES=$(docker exec postgres-northwind psql -U postgres -d northwind -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'aggregates';" 2>/dev/null || echo "0")
if [ "$NORTHWIND_TABLES" -gt "0" ]; then
    print_success "Northwind database: $NORTHWIND_TABLES tables created"
else
    print_warning "Northwind database: No tables found"
fi

# Test connections
print_status "Testing database connections..."
if docker exec postgres-alpha psql -U postgres -d alpha -c "SELECT 1;" > /dev/null 2>&1; then
    print_success "Alpha database connection: OK"
else
    print_error "Alpha database connection: FAILED"
fi

if docker exec postgres-northwind psql -U postgres -d northwind -c "SELECT 1;" > /dev/null 2>&1; then
    print_success "Northwind database connection: OK"
else
    print_error "Northwind database connection: FAILED"
fi

print_success "🎉 Database setup completed!"
echo ""
echo "📋 Database Configuration:"
echo "   Alpha (metadata/config):    localhost:5432/alpha"
echo "   Northwind (aggregates):     localhost:5433/northwind"
echo "   User: postgres"
echo "   Password: postgres"
echo ""
echo "🧪 Test the setup:"
echo "   cd /Users/eganpj/GitHub/semlayer/backend/cmd/preaggregation"
echo "   go run main.go"
echo ""
echo "🛑 To stop databases:"
echo "   docker stop postgres-alpha postgres-northwind"
echo "   docker rm postgres-alpha postgres-northwind"
echo ""
echo "📊 To view database contents:"
echo "   docker exec -it postgres-alpha psql -U postgres -d alpha"
echo "   docker exec -it postgres-northwind psql -U postgres -d northwind"
