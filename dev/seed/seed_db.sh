#!/usr/bin/env bash
# Simple local seeding script for alpha DB and Temporal namespace (dev use only).
# Usage:
#   ALPHA_DB_URL="postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable" \
#   TEMPORAL_CLI="docker run --rm --network host temporalio/cli:1.27.0" \
#   ./dev/seed/seed_db.sh

set -euo pipefail

ALPHA_DB_URL="${ALPHA_DB_URL:-postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable}"
TEMPORAL_CLI="${TEMPORAL_CLI:-docker run --rm --network host temporalio/cli:1.27.0}"
TEMPORAL_NAMESPACE="semlayer-dev"

echo "Using ALPHA_DB_URL=$ALPHA_DB_URL"

# Wait for DB
echo "Waiting for alpha DB to be ready..."
until pg_isready -d "$ALPHA_DB_URL" >/dev/null 2>&1; do
  sleep 1
done

# Insert a demo tenant (id chosen deterministically if you want to re-run)
DEMO_TENANT_ID="00000000-0000-0000-0000-000000000001"
DEMO_TENANT_NAME="local-demo"
DEMO_TENANT_DISPLAY="Local Demo Tenant"

echo "Inserting demo tenant (id=$DEMO_TENANT_ID)..."
psql "$ALPHA_DB_URL" -v ON_ERROR_STOP=1 <<SQL
INSERT INTO tenants (id, name, display_name, tenant_code, is_active, created_at)
VALUES ('$DEMO_TENANT_ID', '$DEMO_TENANT_NAME', '$DEMO_TENANT_DISPLAY', 'local-demo', true, now())
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name;
SQL

# Insert a demo datasource
DEMO_DS_ID="00000000-0000-0000-0000-000000000002"
DEMO_DS_NAME="local-ds"

echo "Inserting demo datasource (id=$DEMO_DS_ID)..."
psql "$ALPHA_DB_URL" -v ON_ERROR_STOP=1 <<SQL
INSERT INTO alpha_datasource (id, datasource_name, datasource_code, datasource_type, is_active, created_at)
VALUES ('$DEMO_DS_ID', '$DEMO_DS_NAME', 'local-ds', 'Postgres', true, now())
ON CONFLICT (id) DO UPDATE SET datasource_name=EXCLUDED.datasource_name;
SQL

# Map tenant -> datasource if tenant_datasources exists
if psql "$ALPHA_DB_URL" -c "\d tenant_datasources" >/dev/null 2>&1; then
  echo "Ensuring tenant_datasources mapping..."
  psql "$ALPHA_DB_URL" -v ON_ERROR_STOP=1 <<SQL
INSERT INTO tenant_datasources (tenant_id, datasource_id)
VALUES ('$DEMO_TENANT_ID', '$DEMO_DS_ID')
ON CONFLICT (tenant_id, datasource_id) DO NOTHING;
SQL
else
  echo "Note: table tenant_datasources not present, skipping mapping step."
fi

# (Optional) Run migrations if you have a migrations CLI
if [ -x ./backend/migrations/cmd/migrate ]; then
  echo "Running backend migrations..."
  ./backend/migrations/cmd/migrate -database "$ALPHA_DB_URL" -path ./backend/migrations
else
  echo "Migration CLI not found at ./backend/migrations/cmd/migrate — skipping migrations."
fi

# Create Temporal namespace (assumes docker-compose for temporal is up and CLI can reach it)
echo "Creating Temporal namespace $TEMPORAL_NAMESPACE (if not exists)..."
$TEMPORAL_CLI --address 127.0.0.1:7233 namespace register --namespace $TEMPORAL_NAMESPACE || true

cat <<EOF
Seeding complete.
Use the following environment to run the server with the demo tenant:

export selected_tenant='{"id":"$DEMO_TENANT_ID","display_name":"$DEMO_TENANT_DISPLAY"}'
export selected_datasource='{"id":"$DEMO_DS_ID","source_name":"$DEMO_DS_NAME"}'

Then start the server and include X-Tenant-ID / X-Tenant-Datasource-ID headers in requests.
EOF
