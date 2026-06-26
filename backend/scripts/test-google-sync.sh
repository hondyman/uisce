#!/bin/bash
# Test script for Google Calendar Sync flow
set -e

API_URL="http://localhost:8080"
TENANT_ID="test-tenant"
USER_ID="test-user"

echo "Starting Google Sync Test..."

# 1. List Calendars
echo "Checking List Calendars endpoint..."
# We expect 500 or 401 if not mocked, but we just want to ensure the route exists
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-User-ID: $USER_ID" -H "X-Tenant-ID: $TENANT_ID" "$API_URL/sync/google/calendars")
echo "Endpoint /sync/google/calendars returning: $HTTP_CODE"

if [ "$HTTP_CODE" == "404" ]; then
    echo "Error: Endpoint not found!"
    exit 1
fi

# 2. Start Sync
echo "Attempting to start sync..."
SYNC_RESP=$(curl -s -X POST "$API_URL/sync/google/sync" \
    -H "Content-Type: application/json" \
    -H "X-User-ID: $USER_ID" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "google_calendar_id": "primary",
        "start_time": "2023-01-01T00:00:00Z",
        "end_time": "2023-02-01T00:00:00Z"
    }')

echo "Start Sync Response: $SYNC_RESP"

if echo "$SYNC_RESP" | grep -q "id"; then
    SYNC_ID=$(echo "$SYNC_RESP" | jq -r .id)
    echo "Sync Job ID: $SYNC_ID"
    
    # 3. Check Status
    sleep 1
    STATUS_RESP=$(curl -s -H "X-User-ID: $USER_ID" "$API_URL/sync/google/status/$SYNC_ID")
    echo "Sync Status: $STATUS_RESP"
    
    # 4. List Active
    ACTIVE_RESP=$(curl -s -H "X-User-ID: $USER_ID" "$API_URL/sync/google/active")
    echo "Active Syncs: $ACTIVE_RESP"
fi

echo "Test script complete."
