#!/usr/bin/env bash
set -euo pipefail

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required to generate an AES-256 key" >&2
  exit 1
fi

KEY=$(openssl rand -base64 32)
cat <<EOF
# AES-256 key for OAuth token encryption
# Store this securely (Vault, AWS Secrets Manager, etc.)
# Example env entry:
# OAUTH_TOKEN_ENCRYPTION_KEY=$KEY
EOF
