#!/bin/bash
# setup-local-dev.sh - Local development environment setup for Calendar Service

set -e

echo "📦 Calendar Service - Local Development Setup"
echo "=============================================="

# Check prerequisites
echo "✓ Checking prerequisites..."

if ! command -v postgres &> /dev/null; then
    echo "⚠️  PostgreSQL not found. Install with:"
    echo "   brew install postgresql@16  (macOS)"
    echo "   sudo apt-get install postgresql (Linux)"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "⚠️  Go not found. Install from https://golang.org/dl/"
    exit 1
fi

echo "✓ Go $(go version | awk '{print $3}')"
echo "✓ PostgreSQL $(psql --version)"

# Setup database
echo ""
echo "🗄️  Setting up database..."

# Start PostgreSQL if not running
if ! pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo "⚠️  PostgreSQL not running. Starting..."
    if command -v brew &> /dev/null; then
        brew services start postgresql@16
    else
        sudo systemctl start postgresql
    fi
    sleep 2
fi

# Create database and user
DB_NAME="calendar_service"
DB_USER="calendar_user"
DB_PASSWORD="calendar_password"

echo "Creating database: $DB_NAME"
createdb $DB_NAME 2>/dev/null || echo "  (database already exists)"

# Create user
echo "Creating database user: $DB_USER"
psql $DB_NAME -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null || true
psql $DB_NAME -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>/dev/null || true

echo "✓ Database ready"

# Build binaries
echo ""
echo "🔨 Building binaries..."

go build -o bin/calendar-service ./cmd/server
echo "✓ Built: bin/calendar-service"

go build -o bin/migrate ./cmd/migrate
echo "✓ Built: bin/migrate"

# Run migrations
echo ""
echo "📝 Running migrations..."

./bin/migrate \
    -host localhost \
    -port 5432 \
    -user $DB_USER \
    -password $DB_PASSWORD \
    -db $DB_NAME \
    -action up

echo ""
echo "✓ Migration status:"
./bin/migrate \
    -host localhost \
    -port 5432 \
    -user $DB_USER \
    -password $DB_PASSWORD \
    -db $DB_NAME \
    -action status

# Show next steps
echo ""
echo "✅ Setup complete!"
echo ""
echo "📋 Next steps:"
echo ""
echo "1️⃣  Start the service:"
echo "   ./bin/calendar-service -port 8080 -db-host localhost -db-user $DB_USER -db-password $DB_PASSWORD"
echo ""
echo "2️⃣  Test health endpoint:"
echo "   curl http://localhost:8080/api/v1/health"
echo ""
echo "3️⃣  View database (psql):"
echo "   psql $DB_NAME -U $DB_USER"
echo ""
echo "📚 Database documentation:"
echo "   docs/DATABASE.md"
echo ""
