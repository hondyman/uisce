#!/bin/bash

# Generate a proper JWT token for testing
# This script creates a valid HS256 JWT with required claims

generate_jwt() {
    local secret="$1"
    local user_id="${2:-test-user-$(date +%s)}"
    local tenant_id="${3:-test-tenant}"
    
    # Use Python to generate JWT with proper claims
    python3 << 'PYTHON_EOF'
import hmac
import hashlib
import base64
import json
import time
import sys

# Get parameters from shell  
secret = sys.argv[1] if len(sys.argv) > 1 else "dev-jwt-secret-key-change-in-production"
user_id = sys.argv[2] if len(sys.argv) > 2 else "test-user"
tenant_id = sys.argv[3] if len(sys.argv) > 3 else "test-tenant"

# Create JWT
def b64_encode(data):
    return base64.urlsafe_b64encode(data.encode()).decode().rstrip('=')

# Header
header = b64_encode(json.dumps({"alg":"HS256","typ":"JWT"}))

# Payload with required claims for the middleware
now = int(time.time())
payload = b64_encode(json.dumps({
    "sub": user_id,
    "user_id": user_id,
    "tenant_id": tenant_id,
    "tenant_ids": [tenant_id],
    "email": f"{user_id}@example.com",
    "iat": now,
    "exp": now + 3600
}))

# Signature
message = f"{header}.{payload}".encode()
signature = base64.urlsafe_b64encode(
    hmac.new(secret.encode(), message, hashlib.sha256).digest()
).decode().rstrip('=')

print(f"{header}.{payload}.{signature}")
PYTHON_EOF
}

# Get parameters
SECRET="${JWT_SECRET:-dev-jwt-secret-key-change-in-production}"
USER_ID="${1:-test-user-phase5-2}"
TENANT_ID="${2:-test-tenant}"

# Generate token
TOKEN=$(generate_jwt "$SECRET" "$USER_ID" "$TENANT_ID")
echo "$TOKEN"
