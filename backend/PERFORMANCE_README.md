# High-Concurrency Performance Implementation

This document outlines the implementation of high-concurrency performance optimizations for the Semlayer backend, based on the provided performance plan.

## Implemented Features

### 1. Enhanced Caching System (`cache.go`)
- **Sharded Cache**: 16-shard cache with consistent hashing for reduced lock contention
- **Version-based Invalidation**: Cache keys include claims and policy versions for efficient invalidation
- **LRU Eviction**: Automatic cleanup of least-recently-used entries
- **Atomic Operations**: Lock-free reads using atomic operations on access timestamps

### 2. Performance Monitoring (`performance_monitor.go`)
- **Real-time Metrics**: CPU, memory, GC, and request statistics
- **pprof Integration**: Automatic profiling endpoints at `/debug/pprof/`
- **Expvar Metrics**: Published metrics for monitoring systems
- **Continuous Monitoring**: Background goroutines for GC and stats collection

### 3. Load Testing Framework (`load_tester.go`)
- **Configurable Tests**: Customizable concurrency, duration, and request patterns
- **Latency Analysis**: P50, P95, P99 latency calculations
- **Progress Reporting**: Real-time progress updates during tests
- **Warmup Support**: Optional warmup phase for cache population

### 4. Async Audit Logging
- **Non-blocking Audit**: Buffered channel for audit events
- **Worker Pool**: 4 goroutines for processing audit events
- **Backpressure Handling**: Drops events when channel is full to prevent blocking

### 5. Concurrency Controls
- **Token Bucket**: Limits concurrent evaluations to prevent overload
- **Graceful Degradation**: Returns 429 when concurrency limit exceeded
- **Object Pooling**: sync.Pool for governance context reuse

## API Endpoints

### Load Testing
```
POST /load-test
```
Runs a 30-second load test with 10 concurrent workers at 100 req/s.

### Performance Monitoring
```
GET /debug/pprof/          # pprof index
GET /debug/pprof/profile   # CPU profile
GET /debug/pprof/heap      # Heap profile
GET /debug/pprof/goroutine # Goroutine profile
```

### Metrics
```
GET /debug/vars            # expvar metrics
```

## Configuration

### Cache Configuration
```go
cache := NewShardedCache(16, 1000) // 16 shards, 1000 entries per shard
```

### Connection Pool (Already Optimized)
```go
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(10 * time.Minute)
db.SetConnMaxIdleTime(5 * time.Minute)
```

### Audit Configuration
```go
auditChan := make(chan *AuditEvent, 1000) // 1000 event buffer
```

## Performance Targets

Based on the plan, the implementation targets:

- **p50 decision latency**: ≤ 3ms
- **p95 decision latency**: ≤ 8ms
- **p99 decision latency**: ≤ 15ms
- **Error rate**: < 0.1% under steady load
- **Cache invalidation**: ≤ 1s from event

## Usage Examples

### Running Load Tests
```bash
curl -X POST http://localhost:8080/load-test
```

### Monitoring Performance
```bash
# Get current metrics
curl http://localhost:8080/debug/vars

# Generate CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile
```

### Cache Statistics
The cache provides real-time statistics including hit rates, eviction counts, and shard utilization.

## Architecture Improvements

1. **Hot Path Optimization**: Cache-first lookups with version-based invalidation
2. **Lock Reduction**: Sharded locks and atomic operations reduce contention
3. **Memory Efficiency**: Object pooling and reduced allocations
4. **Async Processing**: Non-blocking audit logging and background processing
5. **Backpressure Control**: Token buckets prevent cascade failures

## Monitoring and Alerting

The system provides comprehensive monitoring through:
- Request latency histograms
- Cache hit/miss ratios
- GC pressure metrics
- Concurrency utilization
- Error rates and patterns

## Future Enhancements

1. **Redis Integration**: Distributed cache for cross-instance sharing
2. **Metrics Export**: Prometheus/Grafana integration
3. **Adaptive Concurrency**: Dynamic token bucket sizing
4. **Circuit Breakers**: Automatic failure detection and recovery
5. **Advanced Profiling**: Custom profiling for specific code paths
