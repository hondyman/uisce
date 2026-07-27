#!/usr/bin/env bash
# polaris-bootstrap-schema.sh
# Extracts the bundled postgres/schema-v4.sql from the Apache Polaris
# relational-jdbc JAR inside the running uisce-polaris container and
# applies it to the polaris database on the host Postgres.
#
# This is a ONE-SHOT operation — run once after the polaris DB has been
# created but before uisce-polaris first starts serving traffic.
#
# Prerequisites (already done):
#   psql -h <host> -U postgres -c 'CREATE DATABASE polaris;'
#
# Usage:
#   chmod +x scripts/polaris-bootstrap-schema.sh
#   ./scripts/polaris-bootstrap-schema.sh <remote-host>
#   e.g.:  ./scripts/polaris-bootstrap-schema.sh eganpj@100.84.50.65

set -euo pipefail

REMOTE="${1:-eganpj@100.84.50.65}"
CONTAINER="uisce-polaris"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_USER="${DB_USER:-postgres}"
DB_PASS="${DB_PASS:-postgres}"
DB_NAME="${DB_NAME:-polaris}"

JAR_PATH="/deployments/lib/main/org.apache.polaris.polaris-relational-jdbc-1.6.0.jar"
SCHEMA_OUT="postgres/schema-v4.sql"

echo "=== Polaris JDBC Schema Bootstrap ==="
echo "Container : $CONTAINER"
echo "JAR path  : $JAR_PATH"
echo "Target DB : $DB_NAME on $DB_HOST"
echo ""

# Step 1: Verify the JAR exists inside the container and has the schema
echo "[1/4] Verifying JAR and schema file inside container..."
SCHEMA_EXISTS=$(ssh "$REMOTE" "docker exec $CONTAINER python3 -c \
  \"import zipfile; z=zipfile.ZipFile('$JAR_PATH'); print('OK' if '$SCHEMA_OUT' in z.namelist() else 'MISSING')\"")
if [ "$SCHEMA_EXISTS" != "OK" ]; then
  echo "ERROR: $SCHEMA_OUT not found inside $JAR_PATH in container $CONTAINER"
  exit 1
fi
echo "      Schema file found in JAR."

# Step 2: Check if schema already applied (idempotency guard)
echo "[2/4] Checking if schema already applied..."
TABLE_COUNT=$(ssh "$REMOTE" \
  "PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
    -t -c \"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'polaris_schema';\"" 2>/dev/null || echo "0")
TABLE_COUNT=$(echo "$TABLE_COUNT" | tr -d '[:space:]')
if [ "$TABLE_COUNT" -gt 0 ]; then
  echo "      Schema already applied ($TABLE_COUNT tables in polaris_schema). Nothing to do."
  echo "      To re-apply, drop the polaris_schema namespace first:"
  echo "        PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c 'DROP SCHEMA polaris_schema CASCADE;'"
  exit 0
fi
echo "      No schema found — proceeding."

# Step 3: Extract and apply schema
echo "[3/4] Extracting schema from JAR and applying to database..."
ssh "$REMOTE" "docker exec $CONTAINER python3 -c \
  \"import zipfile; z=zipfile.ZipFile('$JAR_PATH'); print(z.read('$SCHEMA_OUT').decode())\"" \
  | PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1
echo "      Schema applied successfully."

# Step 4: Verify
echo "[4/4] Verifying schema..."
RESULT=$(ssh "$REMOTE" \
  "PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
    -t -c \"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'polaris_schema';\"" 2>/dev/null | tr -d '[:space:]')
if [ "$RESULT" -gt 0 ]; then
  echo "      SUCCESS: $RESULT tables in polaris_schema."
  echo ""
  echo "=== Schema bootstrap complete ==="
  echo "Next: restart uisce-polaris to bootstrap the POLARIS realm root principal"
  echo "  ssh $REMOTE"
  echo "  cd /path/to/repo  # wherever docker-compose.remote.yml lives"
  echo "  docker compose -f docker-compose.remote.yml restart uisce-polaris"
  echo ""
  echo "Then verify with:"
  echo "  curl -s -X POST http://localhost:8185/api/catalog/v1/oauth/tokens \\"
  echo "    -H 'Content-Type: application/json' \\"
  echo "    -d '{\"name\":\"root\",\"client_id\":\"root\",\"client_secret\":\"secret\",\"grant_type\":\"client_credentials\",\"scope\":\"PRINCIPAL_ROLE:ALL\"}'"
else
  echo "      ERROR: schema verification failed — expected tables in polaris_schema"
  exit 1
fi
