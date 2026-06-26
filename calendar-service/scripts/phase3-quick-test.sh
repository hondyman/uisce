#!/bin/bash
# Phase 3: Generate Valid JWT Token and Test API

SERVICE_URL="http://127.0.0.1:9081"
SERVICE_API="${SERVICE_URL}/api/v1"
JWT_SECRET="dev-jwt-secret-key-change-in-production"
TENANT_ID="870361a8-87e2-4171-95ad-0473cc93791e"

echo "Generating JWT token..."

# Generate JWT token using Python
TOKEN=$(python3 << 'PYTHON_EOF'
import base64
import json
import hmac
import hashlib
import time

secret = "dev-jwt-secret-key-change-in-production"
tenant_id = "870361a8-87e2-4171-95ad-0473cc93791e"

# Create header and payload
header = base64.urlsafe_b64encode(json.dumps({"alg": "HS256", "typ": "JWT"}).encode()).decode().rstrip('=')
payload = base64.urlsafe_b64encode(json.dumps({
    "user_id": "test-user-123",
    "tenant_id": tenant_id,
    "email": "test@example.com",
    "role": "user",
    "exp": int(time.time()) + 3600,
    "iat": int(time.time()),
}).encode()).decode().rstrip('=')

# Sign it
message = f"{header}.{payload}"
sig = base64.urlsafe_b64encode(
    hmac.new(secret.encode(), message.encode(), hashlib.sha256).digest()
).decode().rstrip('=')

print(f"{message}.{sig}")
PYTHON_EOF
)

echo "Token generated: ${TOKEN:0:50}..."
echo ""

# Test 1: List calendars
echo "=== TEST 1: List Calendars ==="
RESPONSE=$(curl -s http://127.0.0.1:9081/api/v1/calendars \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json")
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
echo ""

# Test 2: List profiles
echo "=== TEST 2: List Profiles ==="
RESPONSE=$(curl -s http://127.0.0.1:9081/api/v1/profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json")
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
echo ""

# Test 3: Check availability
echo "=== TEST 3: Check Availability for 2026-02-20 ==="
RESPONSE=$(curl -s -X POST http://127.0.0.1:9081/api/v1/availability \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"profile_name":"test-default","date":"2026-02-20"}')
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
echo ""

# Test 4: Availability metrics
echo "=== TEST 4: Availability Metrics ==="
RESPONSE=$(curl -s http://127.0.0.1:9081/api/v1/availability/metrics \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json")
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

echo ""
echo "✅ Phase 3 Tests Complete"
