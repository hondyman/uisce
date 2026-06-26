#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "🧪 Starting BP triggers end-to-end test"
echo "======================================="

echo "📦 Checking docker services..."
docker compose -f docker-compose.workflows.local.yml ps

echo "⏳ Services should be running. Sending test event..."

# Send the pg_notify event
psql -h localhost -p 5435 -U postgres -d northwind -c "SELECT pg_notify('entity_events', '{\"tenant_id\": \"22222222-2222-2222-2222-222222222222\", \"entity\": \"Employee\", \"action\": \"created\", \"entity_id\": \"44444444-4444-4444-4444-444444444444\", \"data\": {\"name\": \"Jane Doe\", \"department\": \"Engineering\"}, \"timestamp\": \"2025-10-21T10:00:00Z\"}')"

echo "✅ Test event sent"
echo "🎯 Now start the trigger engine in another terminal:"
echo "   go run -tags bp_versioned ./backend/cmd/triggers"
