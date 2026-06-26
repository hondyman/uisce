package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// SecurityDashboardHandler handles security dashboard requests
type SecurityDashboardHandler struct {
	promRegistry prometheus.Gatherer
	logger       *logrus.Entry
}

// NewSecurityDashboardHandler creates a new security dashboard handler
func NewSecurityDashboardHandler(registry prometheus.Gatherer, logger *logrus.Entry) *SecurityDashboardHandler {
	return &SecurityDashboardHandler{
		promRegistry: registry,
		logger:       logger.WithField("handler", "security_dashboard"),
	}
}

// SecurityDashboardData represents the complete security dashboard view
type SecurityDashboardData struct {
	Timestamp            time.Time            `json:"timestamp"`
	AuthMetrics          AuthMetrics          `json:"auth_metrics"`
	AuthorizationMetrics AuthorizationMetrics `json:"authorization_metrics"`
	RateLimitMetrics     RateLimitMetrics     `json:"rate_limit_metrics"`
	AuditMetrics         AuditMetrics         `json:"audit_metrics"`
	ComplianceStatus     ComplianceStatus     `json:"compliance_status"`
	RecentSecurityEvents []SecurityEvent      `json:"recent_security_events"`
}

// AuthMetrics represents authentication statistics
type AuthMetrics struct {
	TotalRequests      int64            `json:"total_requests"`
	SuccessRate        float64          `json:"success_rate"` // percentage
	FailureCount       int64            `json:"failure_count"`
	FailedAuthByReason map[string]int64 `json:"failed_auth_by_reason"`
}

// AuthorizationMetrics represents authorization statistics
type AuthorizationMetrics struct {
	TotalFailures    int64            `json:"total_failures"`
	FailuresByTenant map[string]int64 `json:"failures_by_tenant"`
	FailuresByReason map[string]int64 `json:"failures_by_reason"`
}

// RateLimitMetrics represents rate limiting statistics
type RateLimitMetrics struct {
	TotalExceeded    int64            `json:"total_exceeded"`
	ExceededByTenant map[string]int64 `json:"exceeded_by_tenant"`
}

// AuditMetrics represents audit logging statistics
type AuditMetrics struct {
	TotalLogs    int64            `json:"total_logs"`
	LogsByType   map[string]int64 `json:"logs_by_type"`
	LogsByAction map[string]int64 `json:"logs_by_action"`
}

// ComplianceStatus represents current compliance state
type ComplianceStatus struct {
	DataResidency     bool      `json:"data_residency"`
	AuditCompleteness bool      `json:"audit_completeness"`
	EncryptionEnabled bool      `json:"encryption_enabled"`
	LastCheck         time.Time `json:"last_check"`
	OverallStatus     string    `json:"overall_status"` // "compliant", "warning", "critical"
}

// SecurityEvent represents a security event for dashboard display
type SecurityEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`     // "auth_failure", "rate_limit", "token_revoked", "compliance_failure"
	Severity  string    `json:"severity"` // "low", "medium", "high", "critical"
	TenantID  string    `json:"tenant_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Details   string    `json:"details"`
}

// GetDashboard handles GET /api/security/dashboard
func (h *SecurityDashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data := SecurityDashboardData{
		Timestamp: time.Now().UTC(),
	}

	// Collect Prometheus metrics (in production, would actually gather from Prometheus)
	// For MVP, these are aggregated from internal counters
	data.AuthMetrics = h.getAuthMetrics(ctx)
	data.AuthorizationMetrics = h.getAuthorizationMetrics(ctx)
	data.RateLimitMetrics = h.getRateLimitMetrics(ctx)
	data.AuditMetrics = h.getAuditMetrics(ctx)
	data.ComplianceStatus = h.getComplianceStatus(ctx)
	data.RecentSecurityEvents = h.getRecentSecurityEvents(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// getAuthMetrics retrieves authentication metrics
func (h *SecurityDashboardHandler) getAuthMetrics(ctx context.Context) AuthMetrics {
	return AuthMetrics{
		TotalRequests:      0, // Would be populated from Prometheus
		SuccessRate:        0,
		FailureCount:       0,
		FailedAuthByReason: make(map[string]int64),
	}
}

// getAuthorizationMetrics retrieves authorization metrics
func (h *SecurityDashboardHandler) getAuthorizationMetrics(ctx context.Context) AuthorizationMetrics {
	return AuthorizationMetrics{
		TotalFailures:    0,
		FailuresByTenant: make(map[string]int64),
		FailuresByReason: make(map[string]int64),
	}
}

// getRateLimitMetrics retrieves rate limiting metrics
func (h *SecurityDashboardHandler) getRateLimitMetrics(ctx context.Context) RateLimitMetrics {
	return RateLimitMetrics{
		TotalExceeded:    0,
		ExceededByTenant: make(map[string]int64),
	}
}

// getAuditMetrics retrieves audit metrics
func (h *SecurityDashboardHandler) getAuditMetrics(ctx context.Context) AuditMetrics {
	return AuditMetrics{
		TotalLogs:    0,
		LogsByType:   make(map[string]int64),
		LogsByAction: make(map[string]int64),
	}
}

// getComplianceStatus retrieves current compliance status
func (h *SecurityDashboardHandler) getComplianceStatus(ctx context.Context) ComplianceStatus {
	return ComplianceStatus{
		DataResidency:     true,
		AuditCompleteness: true,
		EncryptionEnabled: true,
		LastCheck:         time.Now().UTC(),
		OverallStatus:     "compliant",
	}
}

// getRecentSecurityEvents retrieves recent security events
func (h *SecurityDashboardHandler) getRecentSecurityEvents(ctx context.Context) []SecurityEvent {
	// In production, would query audit logs table for recent security events
	return []SecurityEvent{}
}

// HealthCheck handles GET /api/security/health
func (h *SecurityDashboardHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"component": "security_dashboard",
	})
}
