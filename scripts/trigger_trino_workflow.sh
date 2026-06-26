#!/usr/bin/env bash
# Trigger the GitHub Actions workflow via GitHub CLI
set -euo pipefail

WORKFLOW=${1:-verify-snapshot-backfill.yml}
REF=${2:-main}

echo "Triggering workflow $WORKFLOW on ref $REF"
gh workflow run "$WORKFLOW" -f ref="$REF"
echo "Workflow dispatched. Use 'gh run list' to monitor runs."
