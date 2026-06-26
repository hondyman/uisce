#!/usr/bin/env bash
# Load local environment variables from .env.local into the current shell
# Usage: source scripts/load-local-env.sh

ENV_FILE="$(pwd)/.env.local"
if [ ! -f "$ENV_FILE" ]; then
  echo ".env.local not found at $ENV_FILE"
  return 1
fi

# Export variables defined as KEY=VALUE, ignoring comments and blank lines
set -o allexport
# shellcheck disable=SC1090
# Use sed to remove leading/trailing spaces and ignore lines starting with #
# then source via envsubst to expand variables if needed
while IFS='=' read -r key value; do
  # skip comments and empty lines
  [[ "$key" =~ ^# ]] && continue
  [[ -z "$key" ]] && continue
  # trim whitespace
  key=$(echo "$key" | sed -E 's/^\s+|\s+$//g')
  value=$(echo "$value" | sed -E 's/^\s+|\s+$//g')
  # remove optional surrounding quotes
  value=$(echo "$value" | sed -E 's/^"|"$//g' | sed -E "s/^'|'$//g")
  export "$key"="$value"
done < <(grep -E "^[A-Za-z_][A-Za-z0-9_]*\s*=" "$ENV_FILE")
set +o allexport

echo "Loaded environment from $ENV_FILE"
