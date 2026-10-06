#!/usr/bin/env bash
# ==============================================================================
# Compliance Bake-Off Demo Script: End-to-End Decision Provenance & Cold Audit
#
# Scenario:
# 1. Seeds 7 days of realistic multi-rule evaluations for dedicated demo tenant.
# 2. Trader attempts an order violating UCITS 5% single-issuer concentration.
# 3. Engine evaluates and BLOCKS the order in <500µs, recording RFC 8785 content hash.
# 4. Decision Blotter retrieves the full evidence bundle via Lineage ID.
# 5. Synthesizes deterministic explainability breakdown for compliance officers.
# 6. Cryptographically proves bit-for-bit match against immutable WORM cold tier.
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================================"
echo "  CRD Bake-Off: Pre-Trade Compliance & Cryptographic Proof Demo"
echo "======================================================================"
echo "Executing live cryptographic pipeline against PostgreSQL (alpha)..."
echo ""

cd "${REPO_ROOT}"
go run ./backend/cmd/crd-bakeoff-demo/main.go "$@"

