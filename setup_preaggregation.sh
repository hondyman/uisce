#!/bin/bash

# Preaggregation Schema Setup Script
# This script sets up the preaggregation schema for the semantic layer

echo "🚀 Preaggregation Schema Setup"
echo "================================"

# Check if PostgreSQL is available
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL client (psql) not found. Please install PostgreSQL."
    exit 1
fi

# Database connection details (adjust as needed)
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="alpha"
DB_USER="postgres"
DB_PASSWORD="postgres"

echo "📋 Database Configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"

# Test database connection
echo ""
echo "🔍 Testing database connection..."
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();" &> /dev/null; then
    echo "✅ Database connection successful"
else
    echo "❌ Database connection failed"
    echo ""
    echo "💡 Troubleshooting:"
    echo "  1. Make sure PostgreSQL is running"
    echo "  2. Check database credentials"
    echo "  3. Create database if it doesn't exist:"
    echo "     createdb -h $DB_HOST -p $DB_PORT -U $DB_USER $DB_NAME"
    echo ""
    echo "  Or run with Docker:"
    echo "  docker run --name postgres-semlayer -e POSTGRES_DB=$DB_NAME -e POSTGRES_USER=$DB_USER -e POSTGRES_PASSWORD=$DB_PASSWORD -p $DB_PORT:5432 -d postgres:13"
    exit 1
fi

# Run the migration
echo ""
echo "📄 Running preaggregation schema migration..."
MIGRATION_FILE="/Users/eganpj/GitHub/semlayer/backend/migrations/000015_preaggregation_schema.sql"

if [ ! -f "$MIGRATION_FILE" ]; then
    echo "❌ Migration file not found: $MIGRATION_FILE"
    exit 1
fi

PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$MIGRATION_FILE"

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Preaggregation schema migration completed successfully!"
    echo ""
    echo "🔍 Verifying setup..."

    # Check schema creation
    SCHEMA_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = 'semantic_layer';" | tr -d ' ')

    if [ "$SCHEMA_COUNT" -gt 0 ]; then
        echo "✅ Semantic layer schema created"
    else
        echo "❌ Semantic layer schema not found"
    fi

    # Check table creation
    TABLE_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'semantic_layer' AND table_name = 'preaggregated_metrics';" | tr -d ' ')

    if [ "$TABLE_COUNT" -gt 0 ]; then
        echo "✅ Preaggregated metrics table created"
    else
        echo "❌ Preaggregated metrics table not found"
    fi

    # Check sample data
    SAMPLE_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM semantic_layer.preaggregated_metrics WHERE id LIKE 'sample_%';" | tr -d ' ')

    if [ "$SAMPLE_COUNT" -gt 0 ]; then
        echo "✅ Sample data inserted ($SAMPLE_COUNT records)"
    else
        echo "ℹ️  No sample data found (this is normal)"
    fi

    echo ""
    echo "🎉 Setup complete! Next steps:"
    echo "1. Run the preaggregation demo:"
    echo "   cd /Users/eganpj/GitHub/semlayer/backend/cmd/preaggregation && go run main.go"
    echo ""
    echo "2. Set up automated cron jobs:"
    echo "   # Example cron job for daily preaggregation (6 AM daily)"
    echo "   0 6 * * * cd /Users/eganpj/GitHub/semlayer/backend/cmd/preaggregation && /usr/local/go/bin/go run main.go"
    echo ""
    echo "3. Configure monitoring:"
    echo "   - Set up dashboards for data quality metrics"
    echo "   - Configure alerts for preaggregation failures"
    echo "   - Monitor query performance improvements"

else
    echo ""
    echo "❌ Migration failed. Please check the error messages above."
    exit 1
fi
