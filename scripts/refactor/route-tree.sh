#!/opt/homebrew/bin/bash
# scripts/refactor/route-tree.sh — transitive route-registration audit
#
# Encodes two lessons from the 2026-09 tenant-spoof audit:
#   1. A handler package can define register functions that are never called
#      (RegisterDynamicHandlers) — "exists in code" ≠ "reachable".
#   2. Registration is TRANSITIVE (main → api.go → sub-router). Grepping the
#      entrypoint alone underreports the mounted surface (the rulefabric miss).
#
# Usage:  ./scripts/refactor/route-tree.sh [root] [entrypoint-rel-path] [mount-root-rel-path]
#         default: ./scripts/refactor/route-tree.sh backend cmd/server/main.go internal/api/api.go
#
# Limitations (documented deliberately):
#   - Name-based analysis; cannot see interface dispatch or reflection mounting.
#   - File-level transitivity over-reports mounted (conservative/safe direction).
#   - Same-named register funcs in different packages are conflated (rare).
set -euo pipefail

ROOT="${1:-backend}"
ENTRY="${2:-cmd/server/main.go}"
MOUNT="${3:-internal/api/api.go}"
[[ -d "$ROOT" ]] || { echo "FAIL: root '$ROOT' not found"; exit 1; }
[[ -f "$ROOT/$ENTRY" ]] || { echo "FAIL: entrypoint '$ROOT/$ENTRY' not found"; exit 1; }
[[ -f "$ROOT/$MOUNT" ]] || { echo "FAIL: mount root '$ROOT/$MOUNT' not found"; exit 1; }

REG_PAT='Register[A-Za-z]*(Routes|Handlers)'

# --- Collect all register-function definitions ---
defs=()
while IFS= read -r line; do
  defs+=("$line")
done < <(grep -rnE "func (\([^)]*\) )?${REG_PAT%%(*}\(" "$ROOT" --include='*.go' 2>/dev/null \
  | grep -v '_test.go' | sort -u || true)

[[ ${#defs[@]} -gt 0 ]] || { echo "NOTE: no Register*Routes/Handlers definitions found"; exit 0; }

declare -A deffile=()
declare -A seen_name=()
names=()
for d in "${defs[@]}"; do
  file="${d%%:*}"
  name=$(grep -oE "${REG_PAT%%(*}" <<<"$d" | head -1)
  # last definition wins on collision — acceptable per limitations
  if [[ -z "${seen_name[$name]:-}" ]]; then
    seen_name[$name]=1
    names+=("$name")
  fi
  deffile[$name]="$file"
done

echo "=== Check 1: definitions with ZERO callers outside their own file ==="
dead=0
for name in "${names[@]}"; do
  callers=$(grep -rlE "\b${name}\(" "$ROOT" --include='*.go' 2>/dev/null \
            | grep -v '_test.go' | grep -vx "${deffile[$name]}" | wc -l | tr -d ' ' || true)
  if [[ "$callers" == "0" ]]; then
    echo "  UNMOUNTED: ${name}  (defined ${deffile[$name]})"
    dead=$((dead + 1))
  fi
done
[[ $dead -eq 0 ]] && echo "  all ${#names[@]} register functions have callers"
echo ""

echo "=== Check 2: transitive mount tree from ${MOUNT} (SetupRouter / api.go) ==="
declare -A mounted=()
queue=$(grep -oE "\b${REG_PAT%%(*}\(" "$ROOT/$MOUNT" 2>/dev/null \
        | sed 's/($//; s/(//' | sort -u || true)
[[ -n "$queue" ]] || echo "  NOTE: entrypoint registers no routes directly (normal if it delegates)"
depth=0
while [[ -n "$queue" ]]; do
  depth=$((depth + 1))
  next=""
  for name in $queue; do
    [[ -n "${mounted[$name]:-}" ]] && continue
    mounted[$name]=1
    echo "  $(printf '%*s' $((depth * 2)) '')MOUNTED: ${name}  (${deffile[$name]:-?})"
    df="${deffile[$name]:-}"
    [[ -z "$df" || ! -f "$df" ]] && continue
    sub=$(grep -oE "\b${REG_PAT%%(*}\(" "$df" 2>/dev/null | sed 's/($//; s/(//' | sort -u || true)
    for s in $sub; do
      [[ -n "${mounted[$s]:-}" ]] || next="$next $s"
    done
  done
  queue=$(echo "$next" | tr ' ' '\n' | sed '/^$/d' | sort -u)
done
echo ""
echo "SUMMARY: ${#names[@]} defined / ${#mounted[@]} transitively mounted / ${dead} unmounted"
