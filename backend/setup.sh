#!/bin/bash

# SemLayer Quick Setup Script
# This script helps you get started with SemLayer in minutes!

set -e

echo "🚀 Welcome to SemLayer Quick Setup!"
echo "===================================="

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to generate a secure JWT secret
generate_jwt_secret() {
    if command_exists openssl; then
        openssl rand -base64 32
    else
        # Fallback to a simple random string
        echo "fallback-jwt-secret-$(date +%s)-$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 16)"
    fi
}

# Detect environment
detect_environment() {
    if [ -n "$CI" ] || [ -n "$CONTINUOUS_INTEGRATION" ]; then
        echo "ci"
    elif [ -f "docker-compose.yml" ] || [ -f "Dockerfile" ]; then
        echo "docker"
    else
        echo "local"
    fi
}

# Docker setup function
setup_docker() {
    echo "🐳 Setting up for Docker environment..."

    # Create docker-compose.yml if it doesn't exist
    if [ ! -f "docker-compose.yml" ]; then
        cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  semlayer:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ENVIRONMENT=development
      - DRIVER=postgres
      - DSN=host=db port=5432 user=postgres password=password dbname=semlayer sslmode=disable
      - PORT=:8080
      - JWT_SECRET=dev-secret-key
      - LOG_LEVEL=debug
      - ENABLE_METRICS=true
      - ENABLE_CACHING=true
      - ENABLE_SECURITY=false
    depends_on:
      - db
      - redis

  db:
    image: postgres:13
    environment:
      POSTGRES_DB: semlayer
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
EOF
        echo "Created docker-compose.yml"
    fi

    # Create .env file for docker
    cat > .env << EOF
# SemLayer Docker Environment Configuration
ENVIRONMENT=development
DRIVER=postgres
DSN=host=db port=5432 user=postgres password=password dbname=semlayer sslmode=disable
PORT=:8080
JWT_SECRET=dev-secret-key
LOG_LEVEL=debug
ENABLE_METRICS=true
ENABLE_CACHING=true
ENABLE_SECURITY=false
EOF

    echo "Created .env file for Docker"
    echo ""
    echo "To start the application:"
    echo "  docker-compose up -d"
    echo "  docker-compose logs -f semlayer"
}

# CI setup function
setup_ci() {
    echo "🔄 Setting up for CI environment..."

    # Generate JWT secret for CI
    JWT_SECRET=$(generate_jwt_secret)

    cat > .env << EOF
# SemLayer CI Environment Configuration
ENVIRONMENT=ci
DRIVER=postgres
DSN=host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable
PORT=:8080
JWT_SECRET=${JWT_SECRET}
LOG_LEVEL=info
ENABLE_METRICS=false
ENABLE_CACHING=true
ENABLE_SECURITY=false
EOF

    echo "Created .env file for CI"
}

# Local setup function
setup_local() {
    echo "💻 Setting up for local development..."

    # Check if PostgreSQL is running
    if command_exists psql; then
        echo "PostgreSQL client found. Make sure your database is running."
    else
        echo "⚠️  PostgreSQL client not found. Please install PostgreSQL."
        echo "   On macOS: brew install postgresql"
        echo "   On Ubuntu: sudo apt-get install postgresql"
    fi

    # Check if Redis is available
    if command_exists redis-cli; then
        echo "Redis client found."
        REDIS_ADDR="localhost:6379"
    else
        echo "⚠️  Redis not found. Install Redis or disable caching."
        REDIS_ADDR=""
    fi

    # Generate JWT secret
    JWT_SECRET=$(generate_jwt_secret)

    # Create .env file
    cat > .env << EOF
# SemLayer Local Development Configuration
ENVIRONMENT=development
DRIVER=postgres
DSN=host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable
PORT=:8080
JWT_SECRET=${JWT_SECRET}
LOG_LEVEL=debug
ENABLE_METRICS=true
ENABLE_CACHING=true
ENABLE_SECURITY=false
REDIS_ADDR=${REDIS_ADDR}
EOF

    echo "Created .env file for local development"

    # Create config.yaml as well
    cat > config.yaml << EOF
yaml_dir: "./config"
driver: "postgres"
dsn: "host=localhost port=5432 user=postgres password=password dbname=semlayer sslmode=disable"
port: ":8080"
environment: "development"
log_level: "debug"
enable_metrics: true
enable_caching: true
enable_security: false
jwt_secret: "${JWT_SECRET}"
redis_addr: "${REDIS_ADDR}"
EOF

    echo "Created config.yaml for local development"

    # Copy example config files
    cp config.example.dev.yaml config.dev.yaml
    cp config.example.prod.yaml config.prod.yaml
    cp config.example.staging.yaml config.staging.yaml

    echo "Copied example configuration files"
    echo ""
    echo "To start developing:"
    echo "1. Make sure PostgreSQL is running"
    echo "2. Create the database: createdb semlayer"
    echo "3. Run: go run ./cmd/server"
    echo ""
    echo "Or use the configuration files:"
    echo "  go run ./cmd/server --config config.dev.yaml"
}

ENV_TYPE=$(detect_environment)

echo "Detected environment type: $ENV_TYPE"
echo ""

# Setup based on environment type
case $ENV_TYPE in
    "docker")
        setup_docker
        ;;
    "ci")
        setup_ci
        ;;
    "local")
        setup_local
        ;;
    *)
        setup_local
        ;;
esac

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Review the generated configuration files"
echo "2. Update database credentials and secrets"
echo "3. Run: go run ./cmd/server"
echo ""
echo "For detailed configuration options, see CONFIGURATION_GUIDE.md"

# Create a simple database setup script
cat > setup-db.sh << 'EOF'
#!/bin/bash
# Database setup script for SemLayer

echo "Setting up SemLayer database..."

# Database connection details
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-semlayer}

echo "Connecting to PostgreSQL at ${DB_HOST}:${DB_PORT}"

# Create database if it doesn't exist
createdb -h $DB_HOST -p $DB_PORT -U $DB_USER $DB_NAME 2>/dev/null || echo "Database $DB_NAME already exists"

echo "Database setup complete!"
echo "You can now run: go run ./cmd/server"
EOF

chmod +x setup-db.sh
echo "Created setup-db.sh script"

# Make the setup script executable
chmod +x "$0"

echo ""
echo "📁 Files created:"
echo "  - .env (environment variables)"
if [ "$ENV_TYPE" = "local" ]; then
    echo "  - config.yaml (YAML configuration)"
    echo "  - config.dev.yaml (development config)"
    echo "  - config.prod.yaml (production config)"
    echo "  - config.staging.yaml (staging config)"
    echo "  - setup-db.sh (database setup script)"
elif [ "$ENV_TYPE" = "docker" ]; then
    echo "  - docker-compose.yml (Docker setup)"
fi
echo "  - CONFIGURATION_GUIDE.md (detailed documentation)"
