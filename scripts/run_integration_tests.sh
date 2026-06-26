#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
echo "Starting integration test environment..."

docker compose -f "$ROOT/docker-compose.test.yml" up -d --build

echo "Waiting for Postgres to accept connections..."
for i in {1..30}; do
  if docker exec "$(docker ps -q -f ancestor=postgres:14 | head -n1)" pg_isready -U semlayer_user -d semlayer_db >/dev/null 2>&1; then
    echo "Postgres is ready"
    break
  fi
  sleep 1
done

echo "Running migrations into test DB (if init script exists)..."
if [ -f "$ROOT/init-db.sql" ]; then
  docker exec -i "$(docker ps -q -f ancestor=postgres:14 | head -n1)" psql -U semlayer_user -d semlayer_db -f /semlayer/init-db.sql || true
fi


echo "Starting backend via 'go run' pointing at test DB..."
(
  cd "$ROOT/backend"
  PGHOST=127.0.0.1 PGPORT=55432 PGUSER=semlayer_user PGPASSWORD=semlayer_pass PGDATABASE=semlayer_db PORT=3001 go run ./cmd/server &
  echo $! > "$ROOT/.integration_backend_pid"
)
BGPID=$(cat "$ROOT/.integration_backend_pid" 2>/dev/null || echo "")

echo "Waiting for backend to accept connections on :3001..."
for i in {1..30}; do
  if nc -z 127.0.0.1 3001 >/dev/null 2>&1; then
    echo "Backend is listening on 3001"
    break
  fi
  sleep 1
done

echo "Running HTTP checks..."
set +e
OUT=$(curl -s -o /dev/stderr -w "%{http_code}" -X POST http://localhost:3001/api/fabric/models/generate -H 'Content-Type: application/json' -d '{}')
if [ "$OUT" != "400" ] && [ "$OUT" != "422" ]; then
  echo "Expected 4xx validation response but got $OUT"
  kill $BGPID || true
  docker compose -f "$ROOT/docker-compose.test.yml" down -v
  exit 1
fi

echo "Fetching body to validate structured error..."
BODY=$(curl -s -X POST http://localhost:3001/api/fabric/models/generate -H 'Content-Type: application/json' -d '{}')
echo "Response body: $BODY"
echo "$BODY" | grep -q 'error_code' || (echo "Missing error_code in response" && exit 2)

echo "Integration checks passed. Cleaning up..."
kill $BGPID || true
docker compose -f "$ROOT/docker-compose.test.yml" down -v

echo "Done"
