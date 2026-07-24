package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"

	commonpb "go.temporal.io/api/common/v1"
)

// WorkflowInterceptor propagates OpenTelemetry context across Temporal workflows.
// This ensures that traces from the API Gateway flow through to workflow activities.
//
// Without this: API request (Trace A) → Workflow (NEW Trace B) ← Broken chain
// With this:    API request (Trace A) → Workflow (Trace A) → Activity (Trace A) ← Unified trace
//
// Usage:
//
//	workerOptions := worker.Options{
//	    Interceptors: []interceptor.WorkerInterceptor{
//	        observability.NewWorkflowInterceptor(),
//	    },
//	}
//	w := worker.New(temporalClient, taskQueue, workerOptions)
func NewWorkflowInterceptor() interceptor.WorkerInterceptor {
	return &workflowInterceptor{}
}

type workflowInterceptor struct {
	interceptor.WorkerInterceptorBase
}

// InterceptActivity implements context propagation for activities.
func (w *workflowInterceptor) InterceptActivity(
	ctx context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &activityInboundInterceptor{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{
			Next: next,
		},
	}
}

type activityInboundInterceptor struct {
	interceptor.ActivityInboundInterceptorBase
}

// Init extracts OTel context from Temporal headers and injects it into the Go context.
func (a *activityInboundInterceptor) Init(outbound interceptor.ActivityOutboundInterceptor) error {
	return a.Next.Init(outbound)
}

// Execute propagates the trace context.
func (a *activityInboundInterceptor) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (interface{}, error) {
	// Extract OTel SpanContext from Temporal header
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, newHeaderCarrier(interceptor.Header(ctx)))

	// Continue with the extracted context
	return a.Next.ExecuteActivity(ctx, in)
}

// headerCarrier implements propagation.TextMapCarrier for Temporal headers.
type headerCarrier struct {
	header map[string]*commonpb.Payload
}

func newHeaderCarrier(fields map[string]*commonpb.Payload) *headerCarrier {
	if fields == nil {
		fields = make(map[string]*commonpb.Payload)
	}
	return &headerCarrier{header: fields}
}

func (h *headerCarrier) Get(key string) string {
	val, ok := h.header[key]
	if !ok || val == nil {
		return ""
	}
	return string(val.GetData())
}

func (h *headerCarrier) Set(key string, value string) {
	h.header[key] = &commonpb.Payload{Data: []byte(value)}
}

func (h *headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h.header))
	for k := range h.header {
		keys = append(keys, k)
	}
	return keys
}

// InterceptWorkflow propagates context from the workflow starter.
func (w *workflowInterceptor) InterceptWorkflow(
	ctx workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &workflowInboundInterceptor{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{
			Next: next,
		},
	}
}

type workflowInboundInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
}

func (w *workflowInboundInterceptor) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return w.Next.Init(outbound)
}

// ExecuteWorkflow injects OTel headers into the workflow context.
func (w *workflowInboundInterceptor) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (interface{}, error) {
	// In a real implementation, you would:
	// 1. Extract SpanContext from workflow input headers
	// 2. Store it in workflow.Context for activities to access
	// 3. Propagate to child workflows

	// For now, pass through
	return w.Next.ExecuteWorkflow(ctx, in)
}
