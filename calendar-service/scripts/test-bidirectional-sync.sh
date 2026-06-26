#!/bin/bash
set -e

echo "🔄 Testing Bi-directional Sync"

# Configuration
USER_ID="test-user-$(date +%s)"
TENANT_ID="550e8400-e29b-41d4-a716-446655440000"
API_BASE="http://localhost:8081"

# Test 1: Connect Google Calendar
echo ""
echo "📅 Test 1: Connect Google Calendar"
# (Manual OAuth flow - assume already connected)

# Test 2: Create internal event
echo ""
echo "✏️  Test 2: Create Internal Event"
EVENT_RESPONSE=$(curl -s -X POST "$API_BASE/api/v1/events" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "title": "Test Event from Internal",
    "start_time": "2026-02-20T10:00:00Z",
    "end_time": "2026-02-20T11:00:00Z",
    "description": "Created in internal system"
  }')

EVENT_ID=$(echo $EVENT_RESPONSE | jq -r '.id')
echo "Created internal event: $EVENT_ID"

# Test 3: Push to Google
echo ""
echo "🔄 Test 3: Push to Google Calendar"
curl -s -X POST "$API_BASE/api/v1/sync/google/push/$EVENT_ID" \
  -H "X-User-ID: $USER_ID" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

# Wait for sync
sleep 5

# Test 4: Verify in Google
echo ""
echo "✅ Test 4: Verify Event in Google"
# (Manual verification in Google Calendar UI)

# Test 5: Update internal event
echo ""
echo "✏️  Test 5: Update Internal Event"
curl -s -X PUT "$API_BASE/api/v1/events/$EVENT_ID" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "title": "Updated Test Event",
    "description": "Updated in internal system"
  }' | jq .

# Wait for sync
sleep 5

# Test 6: Sync all events
echo ""
echo "🔄 Test 6: Sync All Events to Google"
curl -s -X POST "$API_BASE/api/v1/sync/google/sync-all" \
  -H "X-User-ID: $USER_ID" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

# Test 7: Check sync direction
echo ""
echo "📊 Test 7: Check Sync Direction"
curl -s "$API_BASE/api/v1/sync/google/direction" \
  -H "X-User-ID: $USER_ID" | jq .

echo ""
echo "✅ Bi-directional sync test complete!"
