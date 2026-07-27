#!/usr/bin/env bash
# polaris-bootstrap-tenant.sh
# Creates an Iceberg REST catalog in Apache Polaris using the official
# Management REST API and grants full access to the service_admin principal role.
#
# Usage:
#   ./scripts/polaris-bootstrap-tenant.sh <catalog_name> [polaris_host_port]
#   e.g.: ./scripts/polaris-bootstrap-tenant.sh tenant-alpha http://localhost:8185

set -euo pipefail

CATALOG_NAME="${1:-tenant-alpha}"
POLARIS_URL="${2:-http://localhost:8185}"
CLIENT_ID="${POLARIS_CLIENT_ID:-root}"
CLIENT_SECRET="${POLARIS_CLIENT_SECRET:-secret}"

S3_BUCKET="${S3_BUCKET:-iceberg-warehouse}"
S3_ENDPOINT="${S3_ENDPOINT:-http://uisce-minio:9000}"

echo "=== Polaris Tenant Catalog Bootstrap ==="
echo "Catalog Name: $CATALOG_NAME"
echo "Polaris URL : $POLARIS_URL"
echo "S3 Bucket   : s3://${S3_BUCKET}/${CATALOG_NAME}"
echo "S3 Endpoint : $S3_ENDPOINT"
echo ""

# 1. Fetch OAuth Access Token
echo "[1/3] Requesting OAuth access token..."
TOKEN_RESP=$(curl -s -f -X POST "${POLARIS_URL}/api/catalog/v1/oauth/tokens" \
  -d "grant_type=client_credentials&client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&scope=PRINCIPAL_ROLE:ALL")

ACCESS_TOKEN=$(echo "$TOKEN_RESP" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
  echo "ERROR: Failed to retrieve access token from Polaris"
  exit 1
fi
echo "      Successfully authenticated as principal '${CLIENT_ID}'."

# 2. Create Catalog via Management API
echo "[2/3] Registering catalog '${CATALOG_NAME}'..."
CREATE_HTTP_CODE=$(curl -s -o /tmp/polaris_create.json -w "%{http_code}" -X POST "${POLARIS_URL}/api/management/v1/catalogs" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
     \"catalog\": {
       \"name\": \"${CATALOG_NAME}\",
       \"type\": \"INTERNAL\",
       \"readOnly\": false,
       \"properties\": {
         \"default-base-location\": \"s3://${S3_BUCKET}/${CATALOG_NAME}\",
         \"s3.credentials-type\": \"MANUAL\",
         \"s3.endpoint\": \"${S3_ENDPOINT}\",
         \"s3.path-style-access\": \"true\"
       },
       \"storageConfigInfo\": {
         \"storageType\": \"S3\",
         \"allowedLocations\": [\"s3://${S3_BUCKET}/${CATALOG_NAME}\"]
       }
     }
   }")

if [ "$CREATE_HTTP_CODE" -eq 201 ]; then
  echo "      Catalog '${CATALOG_NAME}' created successfully."
elif [ "$CREATE_HTTP_CODE" -eq 409 ]; then
  echo "      Catalog '${CATALOG_NAME}' already exists (409 Conflict)."
else
  echo "ERROR: Catalog creation failed with HTTP status $CREATE_HTTP_CODE"
  cat /tmp/polaris_create.json
  exit 1
fi

# 3. Grant catalog_admin role and CATALOG_MANAGE_CONTENT privilege
echo "[3/3] Granting privileges on catalog '${CATALOG_NAME}'..."
GRANT_ROLE_CODE=$(curl -s -o /tmp/polaris_grant_role.json -w "%{http_code}" -X PUT "${POLARIS_URL}/api/management/v1/principal-roles/service_admin/catalog-roles/${CATALOG_NAME}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"catalogRole\": {
      \"name\": \"catalog_admin\"
    }
  }")

GRANT_PRIV_CODE=$(curl -s -o /tmp/polaris_grant_priv.json -w "%{http_code}" -X PUT "${POLARIS_URL}/api/management/v1/catalogs/${CATALOG_NAME}/catalog-roles/catalog_admin/grants" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"grant\": {
      \"type\": \"catalog\",
      \"privilege\": \"CATALOG_MANAGE_CONTENT\"
    }
  }")

if [ "$GRANT_ROLE_CODE" -eq 201 ] || [ "$GRANT_ROLE_CODE" -eq 200 ] || [ "$GRANT_ROLE_CODE" -eq 409 ]; then
  echo "      Granted 'catalog_admin' catalog role to 'service_admin' principal role."
else
  echo "WARNING: Granting catalog_admin role returned HTTP status $GRANT_ROLE_CODE"
fi

if [ "$GRANT_PRIV_CODE" -eq 201 ] || [ "$GRANT_PRIV_CODE" -eq 200 ] || [ "$GRANT_PRIV_CODE" -eq 409 ]; then
  echo "      Granted 'CATALOG_MANAGE_CONTENT' privilege on catalog '${CATALOG_NAME}'."
else
  echo "WARNING: Granting CATALOG_MANAGE_CONTENT returned HTTP status $GRANT_PRIV_CODE"
fi

echo ""
echo "=== Polaris Catalog Bootstrap Complete ==="
