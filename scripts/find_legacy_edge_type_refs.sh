#!/usr/bin/env bash
# Script: find_legacy_edge_type_refs.sh
# Purpose: Search repository for occurrences of legacy singular `catalog_edge_type` and compare with canonical `catalog_edge_types`
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")"/.. && pwd)"
cd "$ROOT_DIR"

echo "Searching for legacy singular 'catalog_edge_type' occurrences..."
if command -v rg >/dev/null 2>&1; then
	rg -n "catalog_edge_type" || true
else
	grep -RIn "catalog_edge_type" . || true
fi

echo "\nSearching for canonical 'catalog_edge_types' occurrences..."
if command -v rg >/dev/null 2>&1; then
	rg -n "catalog_edge_types" || true
else
	grep -RIn "catalog_edge_types" . || true
fi

echo "\nNOTE: Review code references to determine if they need to use the canonical table or fallback logic." 
