package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug    LogLevel = "DEBUG"
	LogLevelInfo     LogLevel = "INFO"
	LogLevelWarning  LogLevel = "WARNING"
	LogLevelError    LogLevel = "ERROR"
	LogLevelCritical LogLevel = "CRITICAL"
)

// StructuredLog represents a structured log entry with trace correlation
type StructuredLog struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        LogLevel               `json:"level"`
	Message      string                 `json:"message"`
	ServiceName  string                 `json:"service_name"`
	TraceID      string                 `json:"trace_id,omitempty"`
	SpanID       string                 `json:"span_id,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	DatasourceID string                 `json:"datasource_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	Environment  string                 `json:"environment"`
	Version      string                 `json:"version"`
	Fields       map[string]interface{} `json:"fields,omitempty"`
	Error        string                 `json:"error,omitempty"`
	StackTrace   string                 `json:"stack_trace,omitempty"`
	Duration     int64                  `json:"duration_ms,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Action       string                 `json:"action,omitempty"`
	Source       string                 `json:"source"`
}

// StructuredLogger provides structured logging with trace correlation
type StructuredLogger struct {
	serviceName        string
	environment        string
	version            string
	traceProvider      *TracerProvider
	logger             *log.Logger
	outputFile         *os.File
	mu                 sync.Mutex
	contextPropagators map[string]bool
}

// NewStructuredLogger creates a new structured logger instance
func NewStructuredLogger(serviceName, environment, version string, tp *TracerProvider) *StructuredLogger {
	// Use stdout for structured logs (container will forward to Loki)
	return &StructuredLogger{
		serviceName:        serviceName,
		environment:        environment,
		version:            version,
		traceProvider:      tp,
		logger:             log.New(os.Stdout, "", 0),
		outputFile:         os.Stdout,
		contextPropagators: make(map[string]bool),
	}
}

// LogWithContext logs a message with trace context from request context
func (sl *StructuredLogger) LogWithContext(ctx context.Context, level LogLevel, message string, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Extract trace context from context if available
	traceID := ""
	spanID := ""
	if ctx != nil {
		if tid, ok := ctx.Value(ctxKeyTraceID).(string); ok {
			traceID = tid
		}
		if sid, ok := ctx.Value(ctxKeySpanID).(string); ok {
			spanID = sid
		}
	}

	sl.logEntry(level, message, traceID, spanID, "", "", "", "", fields, "", "", 0, 0, "", "", "")
}

// LogInfo logs an info-level message
func (sl *StructuredLogger) LogInfo(message string, fields map[string]interface{}) {
	sl.logSimple(LogLevelInfo, message, fields)
}

// LogWarning logs a warning-level message
func (sl *StructuredLogger) LogWarning(message string, fields map[string]interface{}) {
	sl.logSimple(LogLevelWarning, message, fields)
}

// LogError logs an error-level message with error details
func (sl *StructuredLogger) LogError(message string, err error, fields map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	sl.logSimple(LogLevelError, message, fields)
	if errStr != "" && fields == nil {
		fields = map[string]interface{}{"error": errStr}
	}
	sl.logSimple(LogLevelError, message, fields)
}

// LogCritical logs a critical-level message
func (sl *StructuredLogger) LogCritical(message string, fields map[string]interface{}) {
	sl.logSimple(LogLevelCritical, message, fields)
}

// LogHTTPRequest logs HTTP request details
func (sl *StructuredLogger) LogHTTPRequest(method, url, traceID, spanID, tenantID, userID string, statusCode int, durationMs int64, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["http_method"] = method
	fields["http_url"] = url
	fields["status_code"] = statusCode
	fields["duration_ms"] = durationMs

	sl.logEntry(LogLevelInfo, method+" "+url, traceID, spanID, tenantID, "", userID, "", fields, "", "", durationMs, statusCode, "", "", "")
}

// LogValidationEvent logs validation-related events
func (sl *StructuredLogger) LogValidationEvent(event string, traceID, spanID, tenantID, validationID string, passed bool, durationMs int64, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["validation_id"] = validationID
	fields["validation_passed"] = passed

	message := "Validation: " + event
	sl.logEntry(LogLevelInfo, message, traceID, spanID, tenantID, "", "", "", fields, "", "", durationMs, 0, "validation", validationID, "validate")
}

// LogRuleEvent logs rule engine events
func (sl *StructuredLogger) LogRuleEvent(event string, traceID, spanID, tenantID, ruleID string, outcome bool, durationMs int64, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["rule_id"] = ruleID
	fields["rule_outcome"] = outcome

	message := "Rule Engine: " + event
	sl.logEntry(LogLevelInfo, message, traceID, spanID, tenantID, "", "", "", fields, "", "", durationMs, 0, "rule", ruleID, "evaluate")
}

// LogNotificationEvent logs notification delivery events
func (sl *StructuredLogger) LogNotificationEvent(event string, traceID, spanID, tenantID, notificationID string, delivered bool, durationMs int64, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["notification_id"] = notificationID
	fields["delivered"] = delivered

	message := "Notification: " + event
	sl.logEntry(LogLevelInfo, message, traceID, spanID, tenantID, "", "", "", fields, "", "", durationMs, 0, "notification", notificationID, "send")
}

// LogSearchEvent logs search-related events
func (sl *StructuredLogger) LogSearchEvent(event string, traceID, spanID, tenantID, queryID string, resultCount int, durationMs int64, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["query_id"] = queryID
	fields["result_count"] = resultCount

	message := "Search: " + event
	sl.logEntry(LogLevelInfo, message, traceID, spanID, tenantID, "", "", "", fields, "", "", durationMs, 0, "search", queryID, "query")
}

// LogBusinessEvent logs business metrics
func (sl *StructuredLogger) LogBusinessEvent(eventType string, traceID, spanID, tenantID string, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if fields == nil {
		fields = make(map[string]interface{})
	}

	message := "Business Event: " + eventType
	sl.logEntry(LogLevelInfo, message, traceID, spanID, tenantID, "", "", "", fields, "", "", 0, 0, "", "", "")
}

// logSimple logs a simple message without trace context
func (sl *StructuredLogger) logSimple(level LogLevel, message string, fields map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.logEntry(level, message, "", "", "", "", "", "", fields, "", "", 0, 0, "", "", "")
}

// logEntry writes a structured log entry
func (sl *StructuredLogger) logEntry(
	level LogLevel,
	message string,
	traceID string,
	spanID string,
	tenantID string,
	datasourceID string,
	userID string,
	requestID string,
	fields map[string]interface{},
	errorMsg string,
	stackTrace string,
	duration int64,
	statusCode int,
	resourceType string,
	resourceID string,
	action string,
) {
	logEntry := StructuredLog{
		Timestamp:    time.Now().UTC(),
		Level:        level,
		Message:      message,
		ServiceName:  sl.serviceName,
		TraceID:      traceID,
		SpanID:       spanID,
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		RequestID:    requestID,
		UserID:       userID,
		Environment:  sl.environment,
		Version:      sl.version,
		Fields:       fields,
		Error:        errorMsg,
		StackTrace:   stackTrace,
		Duration:     duration,
		StatusCode:   statusCode,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Source:       sl.serviceName,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(logEntry)
	if err != nil {
		sl.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	// Write to stdout (will be forwarded to Loki by container)
	fmt.Fprintln(sl.outputFile, string(jsonBytes))
}

// LogContextWrapper wraps a context with trace information
func (sl *StructuredLogger) LogContextWrapper(ctx context.Context, traceID, spanID string) context.Context {
	return WithTraceContext(ctx, traceID, spanID)
}

// ExtractTraceContext extracts trace information from context
func (sl *StructuredLogger) ExtractTraceContext(ctx context.Context) (traceID, spanID string) {
	if ctx == nil {
		return "", ""
	}
	if tid, ok := ctx.Value(ctxKeyTraceID).(string); ok {
		traceID = tid
	}
	if sid, ok := ctx.Value(ctxKeySpanID).(string); ok {
		spanID = sid
	}
	return traceID, spanID
}

// Flush flushes any pending log entries
func (sl *StructuredLogger) Flush() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if sl.outputFile != nil && sl.outputFile != os.Stdout {
		return sl.outputFile.Sync()
	}
	return nil
}

// Close closes the logger
func (sl *StructuredLogger) Close() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if sl.outputFile != nil && sl.outputFile != os.Stdout {
		return sl.outputFile.Close()
	}
	return nil
}
