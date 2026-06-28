package observability

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type ctxKey string

const (
	ctxKeyTraceID      ctxKey = "trace-id"
	ctxKeyRequestStart ctxKey = "request_start"
	ctxKeyCurrentSpan  ctxKey = "current-span"
	ctxKeySpanID       ctxKey = "span-id"
)

// HTTPSpanMiddleware creates HTTP request spans for tracing
func HTTPSpanMiddleware(tp *TracerProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from request headers
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = r.Header.Get("X-B3-TraceId")
			}
			if traceID == "" {
				traceID = r.Header.Get("traceparent")
			}

			// Create new context with trace ID
			ctx := r.Context()
			if traceID != "" {
				ctx = context.WithValue(ctx, ctxKeyTraceID, traceID)
			}

			// Prepare span attributes
			attributes := map[string]interface{}{
				"http.method":      r.Method,
				"http.url":         r.URL.String(),
				"http.target":      r.URL.Path,
				"http.host":        r.Host,
				"http.scheme":      r.URL.Scheme,
				"http.user_agent":  r.UserAgent(),
				"http.client_ip":   getClientIP(r),
				"http.remote_addr": r.RemoteAddr,
			}

			// Add tenant information if available
			if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
				attributes["tenant.id"] = claims.TenantID
			}
			if datasourceID := r.Header.Get("X-Tenant-Datasource-ID"); datasourceID != "" {
				attributes["tenant.datasource_id"] = datasourceID
			}

			// Start span
			span, newCtx := tp.StartSpan(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path), attributes)

			// Write trace ID back to response headers
			w.Header().Set("X-Trace-ID", span.TraceID)
			w.Header().Set("X-Span-ID", span.SpanID)

			// Create response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler with new context
			next.ServeHTTP(rw, r.WithContext(newCtx))

			// End span with status
			status := "ok"
			if rw.statusCode >= 400 {
				status = "error"
			}
			tp.EndSpan(span, status, http.StatusText(rw.statusCode))

			// Add response details
			tp.SetAttribute(span, "http.status_code", rw.statusCode)
			tp.SetAttribute(span, "http.response_size", rw.responseSize)

			// Add route information
			if route := chi.RouteContext(newCtx); route != nil {
				tp.SetAttribute(span, "http.route", route.RoutePattern())
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
	wroteHeader  bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write captures response data
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.statusCode = http.StatusOK
		rw.wroteHeader = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(n)
	return n, err
}

// HTTPErrorMiddleware adds error details to spans when errors occur
func HTTPErrorMiddleware(tp *TracerProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					ctx := r.Context()
					if span, ok := ctx.Value(ctxKeyCurrentSpan).(*Span); ok {
						tp.EndSpan(span, "error", fmt.Sprintf("panic: %v", err))
						tp.SetAttribute(span, "error", true)
						tp.SetAttribute(span, "error.kind", "panic")
						tp.AddEvent(span, "error", map[string]interface{}{
							"message": fmt.Sprintf("%v", err),
						})
					}
					panic(err)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RequestTimingMiddleware adds request timing metrics to spans
func RequestTimingMiddleware(tp *TracerProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Mark request start in context
			requestStart := time.Now()
			ctx = context.WithValue(ctx, ctxKeyRequestStart, requestStart)

			// Wrap response writer to capture timing
			rw := &timedResponseWriter{
				ResponseWriter: w,
				startTime:      requestStart,
			}

			next.ServeHTTP(rw, r.WithContext(ctx))

			// Add timing to span
			if span, ok := ctx.Value(ctxKeyCurrentSpan).(*Span); ok {
				tp.SetAttribute(span, "http.request_duration_ms", rw.duration.Milliseconds())
			}
		})
	}
}

// timedResponseWriter tracks response timing
type timedResponseWriter struct {
	http.ResponseWriter
	startTime   time.Time
	duration    time.Duration
	wroteHeader bool
}

// WriteHeader captures the status code and calculates duration
func (trw *timedResponseWriter) WriteHeader(code int) {
	if !trw.wroteHeader {
		trw.duration = time.Since(trw.startTime)
		trw.wroteHeader = true
		trw.ResponseWriter.WriteHeader(code)
	}
}

// Write updates duration and delegates
func (trw *timedResponseWriter) Write(b []byte) (int, error) {
	if !trw.wroteHeader {
		trw.duration = time.Since(trw.startTime)
		trw.wroteHeader = true
	}
	return trw.ResponseWriter.Write(b)
}

// Helper function to get client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// InjectTraceContext injects trace context into request headers
func InjectTraceContext(r *http.Request, traceID, spanID string) {
	if traceID != "" {
		r.Header.Set("X-Trace-ID", traceID)
		r.Header.Set("X-B3-TraceId", traceID)

		if spanID != "" {
			r.Header.Set("X-Span-ID", spanID)
			r.Header.Set("X-B3-SpanId", spanID)
		}
	}
}

// ExtractTraceContext extracts trace context from request headers
func ExtractTraceContext(r *http.Request) (traceID, spanID string) {
	// Try X-Trace-ID first
	traceID = r.Header.Get("X-Trace-ID")
	if traceID == "" {
		// Try B3 format
		traceID = r.Header.Get("X-B3-TraceId")
	}
	if traceID == "" {
		// Try W3C format
		if tp := r.Header.Get("traceparent"); tp != "" {
			// Format: version-trace_id-span_id-trace_flags
			parts := strings.Split(tp, "-")
			if len(parts) >= 3 {
				// parts[1] is trace_id, parts[2] is span_id
				traceID = parts[1]
				// spanID will be set below after trying the X-Span-ID header
			}
		}
	}

	// Try span ID
	spanID = r.Header.Get("X-Span-ID")
	if spanID == "" {
		spanID = r.Header.Get("X-B3-SpanId")
	}

	return traceID, spanID
}
