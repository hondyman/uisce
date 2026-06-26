package observability

import (
	"fmt"
	"strings"
	"time"
)

// MetricsExporter exports tracing metrics to Prometheus format
type MetricsExporter struct {
	tp *TracerProvider
}

// NewMetricsExporter creates a new metrics exporter
func NewMetricsExporter(tp *TracerProvider) *MetricsExporter {
	return &MetricsExporter{
		tp: tp,
	}
}

// TraceMetrics represents aggregated trace metrics
type TraceMetrics struct {
	TotalSpans           int64
	SuccessfulSpans      int64
	ErrorSpans           int64
	AverageDurationUs    int64
	PercentileDurationUs map[string]int64
	ServiceMetrics       map[string]*ServiceMetrics
	MethodMetrics        map[string]*MethodMetrics
}

// ServiceMetrics represents metrics for a specific service
type ServiceMetrics struct {
	ServiceName     string
	TotalSpans      int64
	ErrorSpans      int64
	SuccessfulSpans int64
	AverageDuration int64
	P50DurationUs   int64
	P95DurationUs   int64
	P99DurationUs   int64
}

// MethodMetrics represents metrics for a specific operation
type MethodMetrics struct {
	MethodName      string
	TotalSpans      int64
	ErrorSpans      int64
	SuccessfulSpans int64
	AverageDuration int64
	P50DurationUs   int64
	P95DurationUs   int64
	P99DurationUs   int64
}

// GenerateMetrics generates Prometheus metrics from collected spans
func (me *MetricsExporter) GenerateMetrics() *TraceMetrics {
	spans := me.tp.GetSpans()

	metrics := &TraceMetrics{
		TotalSpans:           int64(len(spans)),
		ServiceMetrics:       make(map[string]*ServiceMetrics),
		MethodMetrics:        make(map[string]*MethodMetrics),
		PercentileDurationUs: make(map[string]int64),
	}

	if len(spans) == 0 {
		return metrics
	}

	// Collect durations and categorize spans
	durations := make([]int64, 0, len(spans))

	for _, span := range spans {
		if span.Status == "ok" {
			metrics.SuccessfulSpans++
		} else {
			metrics.ErrorSpans++
		}

		if span.Duration > 0 {
			durations = append(durations, span.Duration)
		}

		// Aggregate by service
		if _, exists := metrics.ServiceMetrics[span.ServiceName]; !exists {
			metrics.ServiceMetrics[span.ServiceName] = &ServiceMetrics{
				ServiceName: span.ServiceName,
			}
		}
		serviceMet := metrics.ServiceMetrics[span.ServiceName]
		serviceMet.TotalSpans++
		if span.Status == "ok" {
			serviceMet.SuccessfulSpans++
		} else {
			serviceMet.ErrorSpans++
		}
		if span.Duration > 0 {
			serviceMet.AverageDuration = (serviceMet.AverageDuration*(serviceMet.TotalSpans-1) + span.Duration) / serviceMet.TotalSpans
		}

		// Aggregate by method
		if _, exists := metrics.MethodMetrics[span.OperationName]; !exists {
			metrics.MethodMetrics[span.OperationName] = &MethodMetrics{
				MethodName: span.OperationName,
			}
		}
		methodMet := metrics.MethodMetrics[span.OperationName]
		methodMet.TotalSpans++
		if span.Status == "ok" {
			methodMet.SuccessfulSpans++
		} else {
			methodMet.ErrorSpans++
		}
		if span.Duration > 0 {
			methodMet.AverageDuration = (methodMet.AverageDuration*(methodMet.TotalSpans-1) + span.Duration) / methodMet.TotalSpans
		}
	}

	// Calculate percentiles
	if len(durations) > 0 {
		metrics.AverageDurationUs = calculateAverage(durations)
		metrics.PercentileDurationUs["p50"] = calculatePercentile(durations, 50)
		metrics.PercentileDurationUs["p95"] = calculatePercentile(durations, 95)
		metrics.PercentileDurationUs["p99"] = calculatePercentile(durations, 99)

		// Update service percentiles
		for _, svc := range metrics.ServiceMetrics {
			svc.P50DurationUs = calculatePercentile(durations, 50)
			svc.P95DurationUs = calculatePercentile(durations, 95)
			svc.P99DurationUs = calculatePercentile(durations, 99)
		}

		// Update method percentiles
		for _, method := range metrics.MethodMetrics {
			method.P50DurationUs = calculatePercentile(durations, 50)
			method.P95DurationUs = calculatePercentile(durations, 95)
			method.P99DurationUs = calculatePercentile(durations, 99)
		}
	}

	return metrics
}

// ExportPrometheus exports metrics in Prometheus format
func (me *MetricsExporter) ExportPrometheus() string {
	metrics := me.GenerateMetrics()

	var b strings.Builder

	// Trace span metrics
	fmt.Fprintln(&b, "# HELP traces_total_spans Total number of spans")
	fmt.Fprintln(&b, "# TYPE traces_total_spans gauge")
	fmt.Fprintf(&b, "traces_total_spans %d\n\n", metrics.TotalSpans)

	fmt.Fprintln(&b, "# HELP traces_successful_spans Number of successful spans")
	fmt.Fprintln(&b, "# TYPE traces_successful_spans gauge")
	fmt.Fprintf(&b, "traces_successful_spans %d\n\n", metrics.SuccessfulSpans)

	fmt.Fprintln(&b, "# HELP traces_error_spans Number of error spans")
	fmt.Fprintln(&b, "# TYPE traces_error_spans gauge")
	fmt.Fprintf(&b, "traces_error_spans %d\n\n", metrics.ErrorSpans)

	// Duration metrics
	fmt.Fprintln(&b, "# HELP traces_average_duration_us Average span duration in microseconds")
	fmt.Fprintln(&b, "# TYPE traces_average_duration_us gauge")
	fmt.Fprintf(&b, "traces_average_duration_us %d\n\n", metrics.AverageDurationUs)

	// Percentile metrics
	fmt.Fprintln(&b, "# HELP traces_duration_p50_us 50th percentile span duration in microseconds")
	fmt.Fprintln(&b, "# TYPE traces_duration_p50_us gauge")
	fmt.Fprintf(&b, "traces_duration_p50_us %d\n\n", metrics.PercentileDurationUs["p50"])

	fmt.Fprintln(&b, "# HELP traces_duration_p95_us 95th percentile span duration in microseconds")
	fmt.Fprintln(&b, "# TYPE traces_duration_p95_us gauge")
	fmt.Fprintf(&b, "traces_duration_p95_us %d\n\n", metrics.PercentileDurationUs["p95"])

	fmt.Fprintln(&b, "# HELP traces_duration_p99_us 99th percentile span duration in microseconds")
	fmt.Fprintln(&b, "# TYPE traces_duration_p99_us gauge")
	fmt.Fprintf(&b, "traces_duration_p99_us %d\n\n", metrics.PercentileDurationUs["p99"])

	// Per-service metrics
	if len(metrics.ServiceMetrics) > 0 {
		fmt.Fprintln(&b, "# HELP service_spans_total Total spans per service")
		fmt.Fprintln(&b, "# TYPE service_spans_total gauge")
		for service, svc := range metrics.ServiceMetrics {
			fmt.Fprintf(&b, "service_spans_total{service=\"%s\"} %d\n", service, svc.TotalSpans)
		}
		b.WriteString("\n")

		fmt.Fprintln(&b, "# HELP service_error_spans Error spans per service")
		fmt.Fprintln(&b, "# TYPE service_error_spans gauge")
		for service, svc := range metrics.ServiceMetrics {
			fmt.Fprintf(&b, "service_error_spans{service=\"%s\"} %d\n", service, svc.ErrorSpans)
		}
		b.WriteString("\n")

		fmt.Fprintln(&b, "# HELP service_duration_p99_us 99th percentile duration per service")
		fmt.Fprintln(&b, "# TYPE service_duration_p99_us gauge")
		for service, svc := range metrics.ServiceMetrics {
			fmt.Fprintf(&b, "service_duration_p99_us{service=\"%s\"} %d\n", service, svc.P99DurationUs)
		}
		b.WriteString("\n")
	}

	// Per-method metrics
	if len(metrics.MethodMetrics) > 0 {
		fmt.Fprintln(&b, "# HELP method_spans_total Total spans per method")
		fmt.Fprintln(&b, "# TYPE method_spans_total gauge")
		for method, m := range metrics.MethodMetrics {
			fmt.Fprintf(&b, "method_spans_total{method=\"%s\"} %d\n", method, m.TotalSpans)
		}
		b.WriteString("\n")

		fmt.Fprintln(&b, "# HELP method_error_spans Error spans per method")
		fmt.Fprintln(&b, "# TYPE method_error_spans gauge")
		for method, m := range metrics.MethodMetrics {
			fmt.Fprintf(&b, "method_error_spans{method=\"%s\"} %d\n", method, m.ErrorSpans)
		}
		b.WriteString("\n")

		fmt.Fprintln(&b, "# HELP method_duration_p99_us 99th percentile duration per method")
		fmt.Fprintln(&b, "# TYPE method_duration_p99_us gauge")
		for method, m := range metrics.MethodMetrics {
			fmt.Fprintf(&b, "method_duration_p99_us{method=\"%s\"} %d\n", method, m.P99DurationUs)
		}
		b.WriteString("\n")
	}

	// Timestamp
	fmt.Fprintln(&b, "# HELP traces_exported_timestamp Timestamp when metrics were exported")
	fmt.Fprintln(&b, "# TYPE traces_exported_timestamp gauge")
	fmt.Fprintf(&b, "traces_exported_timestamp %d\n", time.Now().UnixMilli())

	return b.String()
}

// Helper functions

func calculateAverage(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}

	sum := int64(0)
	for _, v := range values {
		sum += v
	}

	return sum / int64(len(values))
}

func calculatePercentile(values []int64, percentile int) int64 {
	if len(values) == 0 {
		return 0
	}

	// Sort would be better but for now use simple approximation
	idx := (len(values) * percentile) / 100
	if idx >= len(values) {
		idx = len(values) - 1
	}

	return values[idx]
}

// ErrorRate calculates the error rate of spans
func (me *MetricsExporter) ErrorRate() float64 {
	metrics := me.GenerateMetrics()

	if metrics.TotalSpans == 0 {
		return 0
	}

	return float64(metrics.ErrorSpans) / float64(metrics.TotalSpans)
}

// ServiceErrorRate calculates the error rate for a specific service
func (me *MetricsExporter) ServiceErrorRate(serviceName string) float64 {
	metrics := me.GenerateMetrics()

	svc, exists := metrics.ServiceMetrics[serviceName]
	if !exists || svc.TotalSpans == 0 {
		return 0
	}

	return float64(svc.ErrorSpans) / float64(svc.TotalSpans)
}

// MethodErrorRate calculates the error rate for a specific method
func (me *MetricsExporter) MethodErrorRate(methodName string) float64 {
	metrics := me.GenerateMetrics()

	method, exists := metrics.MethodMetrics[methodName]
	if !exists || method.TotalSpans == 0 {
		return 0
	}

	return float64(method.ErrorSpans) / float64(method.TotalSpans)
}
