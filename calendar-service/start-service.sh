#!/bin/bash
# Start Calendar Service with Real Google OAuth Credentials
# Phase 5.2 - Google Calendar Integration

set -e

SERVICE_DIR="/Users/eganpj/GitHub/semlayer/calendar-service"
cd "$SERVICE_DIR"

echo "🚀 Starting Calendar Service with Real Google Credentials..."
echo ""

# Kill any existing instances
echo "⏹️  Stopping any existing instances..."
pkill -f "bin/calendar-service" || true
sleep 2

# Build the binary if needed
if [ ! -f "$SERVICE_DIR/bin/calendar-service" ]; then
    echo "📦 Building calendar-service binary..."
    go build -o bin/calendar-service ./cmd/server
fi

# Export Google OAuth credentials
export GOOGLE_CLIENT_ID="607288898719-qkpbcdrdjshm55112pr9ld7h29c33u73.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="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"
export GOOGLE_REDIRECT_URL="http://localhost:9081/api/v1/oauth/google/callback"

# Start the service
echo "🎬 Starting service on port 9081..."
./bin/calendar-service \
  -port 9081 \
  -db-host localhost \
  -db-port 5432 \
  -db-name alpha \
  -db-user postgres \
  -db-password postgres \
  -redis-dsn redis://localhost:6379/0 \
  -hasura-endpoint http://localhost:8080/v1/graphql \
  -loglevel debug > /tmp/calendar-service.log 2>&1 &

SERVICE_PID=$!
echo "✅ Service started (PID: $SERVICE_PID)"
echo ""

# Wait for service to be ready
echo "⏳ Waiting for service to initialize..."
sleep 4

# Check if service is responding
if curl -s http://localhost:9081/api/v1/health | grep -q "healthy"; then
    echo "✅ Service is healthy and responding"
    echo ""
    echo "📱 Generate JWT Token:"
    echo "   TOKEN=\$(./bin/jwt_gen)"
    echo ""
    echo "📍 Get Google Auth URL:"
    echo "   curl -s 'http://localhost:9081/api/v1/sync/google/auth-url-pkce?user_id=test&tenant_id=test' \\"
    echo "     -H \"Authorization: Bearer \$TOKEN\" \\"
    echo "     -H 'X-Tenant-ID: test' | jq ."
    echo ""
    echo "📚 For more info, see PHASE5_2_COMPLETE.md"
else
    echo "❌ Service failed to start. Check logs:"
    echo "   tail -50 /tmp/calendar-service.log"
    exit 1
fi
