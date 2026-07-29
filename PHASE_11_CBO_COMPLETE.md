# Phase 11 Complete: Cost-Based Query Optimizer (CBO)

## Objective
Build a telemetry-driven CBO that routes queries to the optimal execution tier (StarRocks hot / Iceberg cold) based on real-time failure rates and latency, evolving beyond the static 90-day watermark rule.

## Architecture
```
semantic_query_history_v2
        │
        ▼
TelemetryRouter (cbo/telemetry_router.go)
  ├─ cache-aside (Redis, 60s TTL, tenant:{tid}:cbo:{boKey})
  ├─ heuristic rules (failure rate, latency thresholds)
  └─ OverrideReason + TelemetryFlavorDecision
        │
        ├─► BOSQLGenerator.ResolveEffectiveDialect (Rule 5)
        │     boresolver/bo_sql_generator.go
        │
        ├─► UnifiedCalcEngine.determineQueryMode
        │     calcengine/unified_engine.go
        │
        └─► SemanticPublisher.PublishCBOReroute
              events/semantic_publisher.go → "cbo.events" topic
```

## Heuristic Rules
| Rule | Trigger | Action |
|------|---------|--------|
| Insufficient telemetry | `sample_count < CBO_MIN_SAMPLES` (default 5) | Return default flavor |
| Failover | `failure_rate > 0.15` AND `default=STARROCKS` | Route to ICEBERG |
| Degraded latency | `avg_latency > 2500ms` AND `default=STARROCKS` | Route to ICEBERG |
| Healthy | Otherwise | Return default |

## New Files
| File | Purpose |
|------|---------|
| `internal/cbo/redis_client.go` | RedisClient interface + go-redis wrapper + NoopRedisClient |
| `internal/cbo/telemetry_cache.go` | Cache-aside layer (tenant:{tid}:cbo:{boKey}, JSON, TTL) |
| `internal/cbo/telemetry_router.go` | Core TelemetryRouter |
| `internal/cbo/telemetry_router_test.go` | 14 unit tests |
| `internal/calcengine/cbo_adapter.go` | TelemetryRouter port + CBOAdapter + NoOpOptimizer |
| `migrations/20260729000009_cbo_telemetry_indexes.up.sql` | Covering index |
| `migrations/20260729000009_cbo_telemetry_indexes.down.sql` | Rollback |

## Modified Files
| File | Change |
|------|--------|
| `internal/cbo/cbo_types.go` | Added TelemetryFlavorDecision, OverrideReason, CBOConfig, FlavorConstants |
| `internal/boresolver/bo_sql_generator.go` | + TelemetryRouter field, SetTelemetryRouter(), NewBOSQLGeneratorWithCBO(), Rule 5, dialectToFlavor() |
| `internal/boresolver/datafusion_dialect_test.go` | + 5 CBO override tests |
| `internal/calcengine/unified_engine.go` | + Optimizer field, WithOptimizer(), CBO override in determineQueryMode, test helpers |
| `internal/calcengine/watermark_router_test.go` | + 4 CBO override tests |
| `internal/events/event_types.go` | + EventTypeCBOReroute + CBORerouteEvent |
| `internal/events/semantic_publisher.go` | + CBOPublisher interface + PublishCBOReroute() |
| `internal/api/api.go` | + newCBOTelemetryRouter(), SetTelemetryRouter() on boGenerator |

## Service Wiring
- **BOSQLGenerator**: `api.go` — `SetTelemetryRouter()` called at `CBO_ENABLED=true`
- **UnifiedCalcEngine**: `WithOptimizer(tr)` available for callers; `determineQueryMode` uses it when set
- **CBO event publishing**: `TelemetryRouter.WithPublisher(semanticPublisher)` — fires `cbo.reroute` events when flavor differs from default

## Env-Var Configuration
```bash
CBO_ENABLED=true               # Enable/disable CBO (default true)
CBO_WINDOW_MINUTES=60          # Telemetry lookback window
CBO_MIN_SAMPLES=5             # Min samples before applying heuristics
CBO_FAILURE_THRESHOLD=0.15     # Failure rate % that triggers failover
CBO_LATENCY_THRESHOLD_MS=2500 # Latency ms that triggers degraded routing
CBO_CACHE_TTL_SECONDS=60      # Redis cache TTL
REDIS_URL=redis://...         # Optional; falls back to degraded mode
```

## Tests
```
internal/cbo             14 tests — all pass
internal/calcengine       4 new CBO tests — all pass
internal/boresolver       5 new CBO tests — all pass
```

## Pre-existing Issues (Not Fixed Here)
- `internal/api` package has multiple pre-existing build errors (metadata redeclared, undefined auth, etc.) — unrelated to CBO work
