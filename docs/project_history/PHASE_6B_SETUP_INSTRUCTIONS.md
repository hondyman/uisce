// Phase 6b: Distributed Tracing with Jaeger - Installation Instructions

## Required Go Module Additions

Add these dependencies to backend/go.mod:

```
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/otel/exporters/jaeger@latest
go get go.opentelemetry.io/otel/exporters/zipkin@latest
go get go.opentelemetry.io/otel/sdk/resource@latest
go get go.opentelemetry.io/otel/sdk/trace@latest
go get go.opentelemetry.io/otel/semconv@latest
go get go.opentelemetry.io/otel/trace@latest
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@latest
```

Or add directly to go.mod:

```go
require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/exporters/jaeger v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0
)
```

Then run:
```bash
cd backend
go mod download
go mod tidy
```

## Files to Create in Phase 6b

1. backend/internal/observability/tracer_provider.go
   - TracerProvider struct with Jaeger initialization
   - OpenTelemetry SDK setup
   - Tracer factory methods

2. backend/internal/observability/http_middleware.go
   - HTTP span middleware
   - Request/response span creation
   - Trace ID extraction and propagation

3. backend/internal/observability/rabbitmq_propagator.go
   - Trace context propagation for RabbitMQ
   - Message header injection
   - Trace ID extraction from messages

4. backend/internal/observability/context_propagator.go
   - W3C Trace Context implementation
   - Baggage propagation
   - Trace ID generation

5. All 7 services instrumentation:
   - backend/cmd/main.go - tracer initialization
   - backend/cmd/validation-service/main.go
   - backend/cmd/rule-engine-service/main.go
   - backend/cmd/notifications-service/main.go
   - backend/cmd/policy-service/main.go
   - backend/cmd/search-service/main.go
   - backend/cmd/event-router/main.go

6. Testing utilities:
   - backend/internal/observability/test_exporter.go
   - backend/internal/observability/span_assertions.go

## Implementation Timeline

This will provide:
- ✅ Request flow tracing across all services
- ✅ Service dependency visualization in Jaeger
- ✅ Performance bottleneck identification
- ✅ Error tracking with full request context
- ✅ Tenant-scoped trace isolation
- ✅ Custom attributes for business metrics

## Expected Output

After Phase 6b completion:
- 500+ lines of tracing infrastructure code
- All 7 services sending spans to Jaeger
- Service dependency graph visible in Jaeger UI
- Trace context propagation through RabbitMQ
- Request flow visualization
- Performance analysis capabilities
