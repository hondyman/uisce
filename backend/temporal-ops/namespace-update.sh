#!/usr/bin/env sh
# Update a Temporal namespace settings (example: retention)
# Usage: ./namespace-update.sh <namespace> <retention-duration> [address]
if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Usage: $0 <namespace> <retention-duration (e.g. 168h)> [address]"
  exit 2
fi
NS=$1
RETENTION=$2
ADDRESS=${3:-localhost:7233}
temporal operator namespace update --address "$ADDRESS" --namespace "$NS" --retention "$RETENTION"
