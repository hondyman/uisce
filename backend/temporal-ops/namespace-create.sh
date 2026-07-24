#!/usr/bin/env sh
# Create/register a Temporal namespace
# Usage: ./namespace-create.sh <namespace> [address]
if [ -z "$1" ]; then
  echo "Usage: $0 <namespace> [address]"
  exit 2
fi
NS=$1
ADDRESS=${2:-localhost:7233}
temporal operator namespace create --address "$ADDRESS" --namespace "$NS"
