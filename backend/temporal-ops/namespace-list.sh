#!/usr/bin/env sh
# List Temporal namespaces using the Temporal CLI
# Usage: ./namespace-list.sh [address]
ADDRESS=${1:-localhost:7233}
temporal operator namespace list --address "$ADDRESS"
