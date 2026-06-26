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
