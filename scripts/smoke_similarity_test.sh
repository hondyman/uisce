#!/usr/bin/env bash
# Wrapper to run the Python smoke test, sourcing local env first.
# Usage: ./scripts/smoke_similarity_test.sh <tenant_datasource_id> "query text"

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <tenant_datasource_id> \"query text\""
  exit 1
fi

TD_ID="$1"
shift
QUERY="$*"

# load local env (if present)
if [ -f "$(pwd)/.env.local" ]; then
  # shellcheck disable=SC1090
  source scripts/load-local-env.sh
fi

PYTHON="/Users/$(whoami)/GitHub/semlayer/.venv/bin/python"
if [ ! -x "$PYTHON" ]; then
  PYTHON="python3"
fi

$PYTHON scripts/smoke_similarity_test.py --tenant-datasource "$TD_ID" --query "$QUERY"
