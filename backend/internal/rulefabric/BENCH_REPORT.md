# Rules VM — Benchmark Report (Phase 7 — Rulefabric Migration)

Generated: 2026-07-31 (Phase 7 — rulefabric migrated)
CPU:       Apple M1 Pro (arm64)
Go:        go1.x

## Phase 7 Headline — `rulefabric` Migrated

| Benchmark | ns/op | B/op | allocs/op | evals/sec/core | Speedup vs Recursive |
|-----------|------:|-----:|----------:|---------------:|---------------------:|
| `BenchmarkEvaluate_Recursive` (legacy) | **12,448** | 6,091 | 150 | 80K | 1× (baseline) |
| `BenchmarkEvaluate_VM` (full Evaluate path) | 12,323 | 6,188 | 152 | 81K | ~1× (JSON unmarshal dominates) |
| `BenchmarkEvaluate_VM_ManagedHotPath` (pre-compiled) | **109** | 96 | 2 | **9.2M** | **114×** |
| `BenchmarkEvaluate_VM_ManagedHotPath_Parallel` (pre-compiled) | **205** | 96 | 2 | **4.9M** | **60×** (parallel) |

The `BenchmarkEvaluate_VM_ManagedHotPath` benchmark reflects the **production pattern**: rules are compiled once at startup, then re-evaluated many times. The 114× speedup vs the legacy recursive path comes from:

1. **Eliminating JSON unmarshal per call** — the recursive path re-parses `ConditionJSON` on every Evaluate
2. **Eliminating interface dispatch** — the VM's flat bytecodes replace recursive tree walks
3. **Eliminating `fmt.Sprintf` in operator comparisons** — the recursive path does `fmt.Sprintf("%v", ...)` on every leaf; the VM does direct integer/string compares
4. **Eliminating map allocations** — the VM uses pre-sized slices (`FastRecord`) instead of allocating maps per evaluation

## Verification

```
go test ./backend/internal/rulefabric/...               → PASS (parity, fallback, sticky, missing-field)
go test -race -run TestRulefabricVM ./backend/...        → PASS
go build ./backend/...                                  → PASS
```

## Operator Coverage (v1 VM subset in rulefabric)

| Operator | Status |
|----------|--------|
| `==`, `!=`, `>`, `<`, `>=`, `<=` (numeric) | ✅ VM |
| `==`, `!=` (string / enum) | ✅ VM |
| `in` (numeric set, ≤256 elements) | ✅ VM |
| `is_null`, `is_not_null` | ✅ VM |
| `AND`, `OR`, `NOT` (groups) | ✅ VM (with short-circuit + nested groups) |
| `regex`, `matches_regex` | ⚠️ Fallback to recursive |
| `contains`, `not_contains` | ⚠️ Fallback to recursive |
| `starts_with`, `ends_with` | ⚠️ Fallback (Phase 9) |
| `between` | ⚠️ Fallback |
| `date_*` | ⚠️ Fallback (Phase 9) |
| CEL scoring formula | ⚠️ Fallback (computed via CEL after VM returns bool) |

## Architecture Achieved (All Phases)

| Phase | What landed | Status |
|-------|------------|--------|
| 1 | `SymbolDict`, `EnumDict`, `FastRecord` | ✅ |
| 2 | 8-byte packed `Instruction`, `Compile()` | ✅ |
| 3 | BCE-clean VM hot loop, stack pool | ✅ |
| 4 | Benchmarks + acceptance tests (0 allocs, <50 ns) | ✅ |
| 5 | `RuleEngine` wiring + compile cache + fallback | ✅ |
| 6 | Zero-alloc JSON streaming decoder | ✅ |
| **7** | **`rulefabric` migrated: 12 of 18 operators on VM, 114× speedup** | ✅ |

## Remaining Work

- **Phase 8** — production rollout (`useVM` flag, gradual traffic shift, soak period).
- **Phase 9** — operator coverage (`contains`, `starts_with`, `ends_with`, `date_*` via zero-alloc primitives).
- **Cross-package elimination** — `rulefabric.mergeContextMaps` adds 1 alloc per call; Phase 9 cleanup.