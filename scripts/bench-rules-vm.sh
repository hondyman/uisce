#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

echo "Running VM Benchmarks..."
go test -bench=BenchmarkVM -benchmem -count=10 ./backend/internal/rules/vm/... | tee BENCH_REPORT.md