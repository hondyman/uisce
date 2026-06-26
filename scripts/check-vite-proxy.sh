#!/usr/bin/env bash
set -euo pipefail

TENANT_ID=${1:-99e99e99-99e9-49e9-89e9-99e99e99e999}
DATASOURCE_ID=${2:-11111111-1111-1111-1111-111111111111}

echo "Checking Vite proxy at http://localhost:5173/api/debug/headers"
if curl -s -D - -H "X-Tenant-ID: $TENANT_ID" -H "X-Tenant-Instance-ID: $DATASOURCE_ID" http://localhost:5173/api/debug/headers | jq . >/dev/null 2>&1; then
  echo "Vite dev server responded via proxy (success)."
else
  echo "Vite dev server did not respond or returned non-JSON. Falling back to backend host-mapped port." 
  echo "Checking backend directly at http://localhost:8082/api/debug/headers"
  if curl -s -D - -H "X-Tenant-ID: $TENANT_ID" -H "X-Tenant-Instance-ID: $DATASOURCE_ID" http://localhost:8082/api/debug/headers | jq . >/dev/null 2>&1; then
    echo "Backend responded directly. If you are running frontend locally, ensure VITE_USE_PROXY=true and VITE_BACKEND_TARGET=http://localhost:8082 in frontend/.env.local and restart the dev server (npm run dev)."
  else
    echo "No response from Vite proxy or backend. Check that frontend dev server (Vite) and backend container are running and ports 5173/8082 are reachable."
    exit 2
  fi
fi

echo "Done."
