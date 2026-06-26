#!/bin/bash

# Generate bcrypt password hashes for test users
# This script generates the hashes needed for create_test_users.sql

set -e

echo "🔐 Generating Password Hashes for Test Users"
echo "=============================================="
echo ""

# Check if htpasswd is available
if ! command -v htpasswd &> /dev/null; then
    echo "❌ htpasswd not found. Installing..."
    echo "On macOS: brew install httpd"
    echo "On Ubuntu: sudo apt-get install apache2-utils"
    exit 1
fi

echo "Generating hashes (this may take a moment)..."
echo ""

# Generate hash for password123
HASH_PASSWORD123=$(htpasswd -bnBC 10 "" password123 | tr -d ':\n' | sed 's/^//')
echo "Password: password123"
echo "Hash: $HASH_PASSWORD123"
echo ""

# Generate hash for admin123
HASH_ADMIN123=$(htpasswd -bnBC 10 "" admin123 | tr -d ':\n' | sed 's/^//')
echo "Password: admin123"
echo "Hash: $HASH_ADMIN123"
echo ""

echo "=============================================="
echo "✅ Hashes generated successfully!"
echo ""
echo "Now updating create_test_users.sql with actual hashes..."

# Update the SQL file with actual hashes
sed -i.bak "s|\$2a\$10\$rKZLvVZvKxF7Y9jXxZ8eJOqKqH5vYxYxYxYxYxYxYxYxYxYxYxYxY|$HASH_PASSWORD123|g" scripts/create_test_users.sql

echo "✅ SQL file updated!"
echo ""
echo "You can now run:"
echo "  psql -U postgres -d alpha -f scripts/create_test_users.sql"
