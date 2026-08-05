#!/usr/bin/env bash
set -euo pipefail

# Configuration
API_URL="${API_URL:-http://localhost:8080/api}"
TENANT_ID="${TENANT_ID:-}"  # Required: set via environment or pass --tenant-id
RESOURCE_TYPE="tenant_instance"

if [[ -z "$TENANT_ID" ]]; then
    echo "ERROR: TENANT_ID environment variable is required" >&2
    echo "Usage: TENANT_ID=<uuid> $0" >&2
    exit 1
fi

echo "=== 1. Simulating Instance Lifecycle (Writing Logs) ==="

# Step A: Create instance with unique name and URL
UNIQUE_NAME="audit-test-$(date +%s)"
UNIQUE_URL="http://$UNIQUE_NAME.local"
echo "-> Creating instance with name: $UNIQUE_NAME..."
CREATE_RESP=$(curl -s -X POST "$API_URL/tenant-ops/instances" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{
    \"instance_name\": \"$UNIQUE_NAME\",
    \"display_name\": \"Audit Test Instance\",
    \"url\": \"$UNIQUE_URL\",
    \"is_active\": true,
    \"config\": \"{}\"
  }")

echo "Create response: $CREATE_RESP"

# Extract actual instance ID from response
INSTANCE_ID=$(echo "$CREATE_RESP" | jq -r '.id')
echo "-> Created instance with ID: $INSTANCE_ID"

# Step B: Update instance (triggers CDC change log)
echo "-> Updating instance..."
UPDATE_RESP=$(curl -s -X PATCH "$API_URL/tenant-ops/instances/$INSTANCE_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{
    \"instance_name\": \"$UNIQUE_NAME\",
    \"display_name\": \"Audit Test Instance UPDATED\",
    \"url\": \"$UNIQUE_URL\",
    \"is_active\": true,
    \"config\": \"{}\"
  }")

echo "Update response: $UPDATE_RESP"

echo -e "\n=== 2. Waiting for DataFusion Write ==="
sleep 2

echo -e "\n=== 3. Querying Read API ==="
HISTORY_RESPONSE=$(curl -s -X GET "$API_URL/audit/history/$RESOURCE_TYPE/$INSTANCE_ID")

echo "History response: $HISTORY_RESPONSE"

# Check if we got valid JSON with history array
if ! echo "$HISTORY_RESPONSE" | jq -e '.history' > /dev/null 2>&1; then
  echo "❌ Error: Response does not contain 'history' field"
  echo "Response received: $HISTORY_RESPONSE"
  exit 1
fi

RECORD_COUNT=$(echo "$HISTORY_RESPONSE" | jq '.history | length')
echo "✅ Successfully retrieved history! Found $RECORD_COUNT audit log entry/entries."

echo -e "\n=== 4. Validating Payload Integrity ==="

# Map: test uses .timestamp → actual is .system_from
echo "$HISTORY_RESPONSE" | jq -r '.history[] | "Timestamp: \(.system_from) | Action: \(.change_type) | Actor: \(.changed_by)"'

# Assert UPDATE action was recorded
HAS_UPDATE=$(echo "$HISTORY_RESPONSE" | jq '[.history[] | select(.change_type == "update_instance")] | length > 0')
if [ "$HAS_UPDATE" = "true" ]; then
  echo "✅ Success: update_instance action found in history log."
  echo "Parsed JSON Details payload:"
  echo "$HISTORY_RESPONSE" | jq '.history[] | select(.change_type == "update_instance") | .entity_data'
else
  echo "❌ Failure: Expected update_instance action was not found."
  exit 1
fi

echo -e "\n=== 5. Testing Recent Changes (GetRecentChanges) ==="
RECENT_RESPONSE=$(curl -s -X GET "$API_URL/audit/changes?limit=10")
echo "Recent changes response: $RECENT_RESPONSE" | head -c 500
echo ""

echo "✅ All validations passed!"
