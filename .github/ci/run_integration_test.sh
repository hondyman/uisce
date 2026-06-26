#!/usr/bin/env bash
set -euo pipefail

POSTGRES_CONN="postgresql://ops:ops_pass@localhost:5432/ops"
MINIO_ALIAS=local
MINIO_BUCKET=iceberg-staging
COMMIT_URL=http://localhost:8081/commit
TRINO_URL=http://localhost:8080

# Wait helper
wait_for_port() {
  local host=$1 port=$2 timeout=${3:-60}
  for i in $(seq 1 $timeout); do
    if nc -z $host $port >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  return 1
}

echo "Create idempotency table in Postgres"
psql "$POSTGRES_CONN" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS committed_manifests (
  manifest_id TEXT PRIMARY KEY,
  table_name TEXT NOT NULL,
  committed_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  uploader TEXT
);
SQL

# Configure mc for MinIO
mc alias set $MINIO_ALIAS http://localhost:9000 $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD || true
mc mb $MINIO_ALIAS/$MINIO_BUCKET || true

# Create an Iceberg schema/table via Trino
echo "Creating Iceberg schema and table via Trino"
SQL_CREATE_SCHEMA="CREATE SCHEMA IF NOT EXISTS iceberg.ops;"
SQL_CREATE_TABLE="CREATE TABLE IF NOT EXISTS iceberg.ops.incidents (
  incident_id varchar,
  tenant_id varchar,
  region varchar,
  status varchar,
  severity varchar,
  created_at timestamp
) WITH (format = 'PARQUET');"

# Submit SQL via Trino HTTP API
submit_trino_sql() {
  local sql="$1"
  # POST SQL, then follow nextUri to get completion
  local resp=$(curl -s -X POST --data "$sql" $TRINO_URL/v1/statement)
  local next=$(echo "$resp" | jq -r '.nextUri // empty')
  while [ -n "$next" ]; do
    resp=$(curl -s "$next")
    next=$(echo "$resp" | jq -r '.nextUri // empty')
    sleep 0.2
  done
}

submit_trino_sql "$SQL_CREATE_SCHEMA"
submit_trino_sql "$SQL_CREATE_TABLE"

# Generate a test Parquet file
PARQUET_FILE=/tmp/test-$(date +%s).parquet
python3 .github/ci/generate_test_parquet.py $PARQUET_FILE

# Upload to MinIO
OBJECT_PATH="ops/incidents/region=us-east-1/date=$(date -u +%Y-%m-%d)/$(basename $PARQUET_FILE)"
mc cp $PARQUET_FILE $MINIO_ALIAS/$MINIO_BUCKET/$OBJECT_PATH

# Build manifest JSON
MANIFEST_ID="manifest-$(date +%s)-$RANDOM"
cat > /tmp/manifest.json <<EOF
{
  "manifest_id": "$MANIFEST_ID",
  "catalog": "iceberg",
  "table": "ops.incidents",
  "tenant_id": "t1",
  "region": "us-east-1",
  "uploaded_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "uploader": "ci-run",
  "files": [
    {
      "path": "s3://$MINIO_BUCKET/$OBJECT_PATH",
      "file_size": 0,
      "record_count": 1,
      "partition": {"region":"us-east-1","date":"$(date -u +%Y-%m-%d)"}
    }
  ]
}
EOF

# POST manifest to commit service
echo "Posting manifest to $COMMIT_URL"
for i in {1..10}; do
  if curl -sS -X POST -H "Content-Type: application/json" --data @/tmp/manifest.json $COMMIT_URL -o /tmp/commit_resp.json; then
    cat /tmp/commit_resp.json
    break
  else
    echo "Waiting for commit service..."
    sleep 2
  fi
done

# Poll Trino until the row is visible
echo "Polling Trino for committed row"
SQL_COUNT="SELECT count(*) FROM iceberg.ops.incidents WHERE incident_id = 'inc_test';"
for i in {1..30}; do
  resp=$(curl -s -X POST --data "$SQL_COUNT" $TRINO_URL/v1/statement)
  next=$(echo "$resp" | jq -r '.nextUri // empty')
  data=''
  while [ -n "$next" ]; do
    resp=$(curl -s "$next")
    next=$(echo "$resp" | jq -r '.nextUri // empty')
    data=$(echo "$resp" | jq -r '.data // empty')
    if [ -n "$data" ] && [ "$data" != "null" ]; then
      count=$(echo "$data" | jq -r '.[0][0]')
      if [ "$count" = "1" ]; then
        echo "Success: row found"
        exit 0
      fi
    fi
    sleep 1
  done
  sleep 2
done

echo "ERROR: row not found in Trino after timeout"
exit 1
