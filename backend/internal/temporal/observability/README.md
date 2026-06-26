# Temporal Observability Package

This package provides OpenTelemetry integration for Temporal workflows, enabling **semantic observability** of metadata-driven business logic.

## Problem

In a metadata-driven architecture, workflows are interpreted from JSON DSL:
```json
{
  "workflow_id": "nba_retraining",
  "steps": [
    {"name": "ExtractTrainingData", ...},
    {"name": "RetrainModel", ...}
  ]
}
```

Standard traces only show generic `Interpreter.Execute()` spans, providing **zero business context**.

## Solution

This package injects custom OTel spans with business-level names and attributes, making the "virtual" DSL execution visible in Jaeger/Datadog.

## Usage

### 1. Initialize Tracer (in `main.go`)

```go
import "github.com/yourusername/semlayer/backend/internal/temporal/observability"

func main() {
    cleanup, err := observability.InitTracer("semlayer-backend", "localhost:14268")
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup(context.Background())

    // Start Temporal worker...
}
```

### 2. Wrap Workflow Activities

**Before** (invisible):
```go
err := workflow.ExecuteActivity(ctx, acts.ExtractTrainingData, params).Get(ctx, &result)
```

**After** (visible in Jaeger):
```go
result, err := observability.TracedActivityWithMetadata(
    ctx,
    "ExtractTrainingData",
    map[string]string{
        "model_id": modelID,
        "training_window_days": "90",
    },
    func(ctx context.Context) (interface{}, error) {
        return acts.ExtractTrainingData(ctx, params)
    },
)
```

### 3. View in Jaeger

Navigate to `http://localhost:16686` and search for traces.

**You'll see:**
- `DSL Step: ExtractTrainingData` (200ms)
- `DSL Step: RetrainModel` (5s)
- `DSL Step: ValidateModel` (100ms)

Each span includes business metadata like `app.model_id`, `app.training_window_days`.

## Architecture

```
API Request (TraceID: A)
   ↓
Temporal Workflow (TraceID: A) ← Context Propagation
   ↓
Activity: "DSL Step: ExtractTrainingData" ← Semantic Span Injection
   ↓
Database Query (TraceID: A)
```

## Files

- `tracer.go` - OTel setup with Jaeger exporter
- `activity_wrapper.go` - Span injection for activities
- `workflow_interceptor.go` - Context propagation across Temporal boundaries

## Configuration

Set via environment variables:
```bash
export JAEGER_ENDPOINT=localhost:14268
export OTEL_SERVICE_NAME=semlayer-backend
```

## Performance Impact

- Span creation: <1ms overhead per activity
- No impact on workflow determinism (uses LocalActivity)
- Sampling: Configure via `sdktrace.WithSampler()`
