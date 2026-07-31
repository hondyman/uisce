#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

echo "Running BCE diagnostic for VM.Run..."
out=$(go test -gcflags="-d=ssa/check_bce/debug=1" \
        -run TestBCE_CleanVMHotLoop ./backend/internal/rules/vm/ 2>&1) || true

# Print a summary of bounds checks per file for review.
echo ""
echo "--- BCE summary (vm.go dispatch loop only, lines >= 53) ---"
echo "$out" | grep "vm.go:" | awk -F: '$2+0 >= 53' | sort -u | head -40
vm_count=$(echo "$out" | grep "vm.go:" | awk -F: '$2+0 >= 53' | wc -l | tr -d ' ')

# compiler.go bounds checks are in error-formatting code paths; they only
# fire when Compile() returns an error. We ignore those.
fastrecord_count=$(echo "$out" | grep "fastrecord.go:" | wc -l | tr -d ' ')

echo ""
echo "vm.go dispatch loop: $vm_count bounds checks"
echo "fastrecord.go (off-hot-path, Project() helper): $fastrecord_count bounds checks"

# Heuristic threshold: Go's BCE is conservative on runtime indices. Empirically
# the dispatch loop emits ~1 bounds check per indexed access. We allow up to 30
# (10 load cases × ~3 each). This catches gross regressions without false-positives
# on the BCE pattern.
MAX_VM_BCE=30
if [ "$vm_count" -gt "$MAX_VM_BCE" ]; then
    echo ""
    echo "FAIL: vm.go dispatch loop emits $vm_count bounds checks (> $MAX_VM_BCE)"
    echo "Investigate new slice accesses or regressions in slice+guard patterns."
    exit 1
fi

echo ""
echo "PASS: vm.go dispatch loop is within BCE budget ($vm_count / $MAX_VM_BCE)"
echo "  (Note: Go's compiler is conservative for runtime-indexed slice accesses."
echo "   The real binding contracts — 0 allocs/op and <50 ns/op — are enforced by"
echo "   TestBenchmarkAcceptance.)"