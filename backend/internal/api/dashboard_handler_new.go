package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/middleware"
	"github.com/hondyman/semlayer/backend/internal/security"
)

// ============================================================================
// Dashboard Handler - Risk & Compliance Console
// ============================================================================

type DashboardHandler struct {
	db *sqlx.DB
}

func NewDashboardHandler(db *sqlx.DB) *DashboardHandler {
	return &DashboardHandler{
		db: db,
	}
}

// RegisterRoutes registers all dashboard routes
func (h *DashboardHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/dashboard", func(r chi.Router) {
		r.Get("/compliance", h.GetComplianceMetrics)
		r.Get("/risk", h.GetRiskMetrics)
		r.Get("/sparklines", h.GetSparklines)
		r.Get("/etl-health", h.GetETLHealth)
		r.Get("/alerts", h.GetAlerts)
		r.Post("/etl/trigger", h.TriggerETL)
	})
}

// ============================================================================
// Request/Response Types
// ============================================================================

// ComplianceMetric represents compliance data
type ComplianceMetric struct {
	RuleID      string  `json:"ruleId"`
	RuleName    string  `json:"ruleName"`
	Status      string  `json:"status"`   // "Pass" | "Fail" | "Warning"
	PassRate    float64 `json:"passRate"` // 0-100%
	LastChecked string  `json:"lastChecked"`
	Description string  `json:"description"`
}

type ComplianceResponse struct {
	Critical  int64              `json:"critical"` // Number of failed rules
	Warning   int64              `json:"warning"`  // Number of warning rules
	Passing   int64              `json:"passing"`  // Number of passing rules
	Rules     []ComplianceMetric `json:"rules"`
	Timestamp string             `json:"timestamp"`
}

// RiskMetric represents risk data
type RiskMetric struct {
	MetricID    string  `json:"metricId"`
	MetricName  string  `json:"metricName"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"` // e.g., "%" or "bps"
	Threshold   float64 `json:"threshold"`
	Status      string  `json:"status"` // "Normal" | "Warning" | "Alert"
	LastUpdated string  `json:"lastUpdated"`
}

type RiskResponse struct {
	Volatility float64      `json:"volatility"` // 7.5
	VaRPercent float64      `json:"varPercent"` // 95
	VaRValue   float64      `json:"varValue"`   // 2.3M
	BetaMarket float64      `json:"betaMarket"` // 1.2
	Drawdown   float64      `json:"drawdown"`   // -15.2
	Metrics    []RiskMetric `json:"metrics"`
	Timestamp  string       `json:"timestamp"`
}

// SparklineDataPoint represents a single data point
type SparklineDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type SparklineMetric struct {
	MetricName string               `json:"metricName"`
	Data       []SparklineDataPoint `json:"data"` // Last 7 days
}

type SparklinesResponse struct {
	Metrics   []SparklineMetric `json:"metrics"`
	Period    string            `json:"period"` // "7d"
	Timestamp string            `json:"timestamp"`
}

// ETLHealthStatus represents ETL run status
type ETLHealthStatus struct {
	RunID            string  `json:"runId"`
	Status           string  `json:"status"` // "Running" | "Success" | "Failed"
	StartTime        string  `json:"startTime"`
	EndTime          *string `json:"endTime,omitempty"`
	RecordsProcessed int64   `json:"recordsProcessed"`
	RecordsFailed    int64   `json:"recordsFailed"`
	Duration         int     `json:"duration"` // seconds
	ErrorMessage     *string `json:"errorMessage,omitempty"`
}

type ETLHealthResponse struct {
	LastRun         ETLHealthStatus `json:"lastRun"`
	RunCount24h     int64           `json:"runCount24h"`
	SuccessRate     float64         `json:"successRate"`
	AverageDuration int             `json:"averageDuration"`
	Timestamp       string          `json:"timestamp"`
}

// AlertItem represents a single alert
type AlertItem struct {
	AlertID   string `json:"alertId"`
	Title     string `json:"title"`
	Severity  string `json:"severity"` // "Critical" | "Warning" | "Info"
	Message   string `json:"message"`
	Source    string `json:"source"` // "Compliance" | "Risk" | "Operations"
	CreatedAt string `json:"createdAt"`
	Status    string `json:"status"` // "Open" | "Acknowledged" | "Resolved"
}

type AlertsResponse struct {
	Critical  int64       `json:"critical"`
	Warning   int64       `json:"warning"`
	Info      int64       `json:"info"`
	Alerts    []AlertItem `json:"alerts"`
	Timestamp string      `json:"timestamp"`
}

type TriggerETLRequest struct {
	DataSourceID string `json:"dataSourceId,omitempty"`
	Priority     string `json:"priority,omitempty"` // "high" | "normal" | "low"
}

type TriggerETLResponse struct {
	RunID     string `json:"runId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	StartedAt string `json:"startedAt"`
}

// ============================================================================
// Handler Methods
// ============================================================================

// GetComplianceMetrics returns compliance dashboard metrics
// GET /api/dashboard/compliance?tenant_id=xxx&valuation_date=yyyy-mm-dd
func (h *DashboardHandler) GetComplianceMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Verify authentication
	userID, auth, err := h.verifyAuthentication(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Verify dashboard access permission
	if !h.hasPermission(auth, "dashboard:read") {
		h.logSecurityEvent(ctx, userID, "", "dashboard_access_denied", "compliance", "", r.RemoteAddr)
		http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
		return
	}

	// 3. Extract tenant_id
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	// 4. Verify tenant access control
	if !h.verifyTenantAccess(auth, tenantID) {
		h.logSecurityEvent(ctx, userID, tenantID, "dashboard_cross_tenant_attempt", "compliance", "", r.RemoteAddr)
		http.Error(w, "Forbidden: access denied to this tenant", http.StatusForbidden)
		return
	}

	// 5. Log audit trail
	h.logSecurityEvent(ctx, userID, tenantID, "dashboard_compliance_accessed", "compliance", "", r.RemoteAddr)

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	// Query compliance metrics from database
	// For now, return mock data matching the TypeScript contract
	response := ComplianceResponse{
		Critical:  2,
		Warning:   5,
		Passing:   18,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Rules: []ComplianceMetric{
			{
				RuleID:      "rule-001",
				RuleName:    "Portfolio Diversification",
				Status:      "Pass",
				PassRate:    95.2,
				LastChecked: time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
				Description: "Ensures portfolio meets minimum diversification requirements",
			},
			{
				RuleID:      "rule-002",
				RuleName:    "Sector Concentration Limit",
				Status:      "Warning",
				PassRate:    87.5,
				LastChecked: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
				Description: "Monitors sector concentration against policy limits",
			},
			{
				RuleID:      "rule-003",
				RuleName:    "Cash Position Rule",
				Status:      "Fail",
				PassRate:    65.0,
				LastChecked: time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339),
				Description: "Ensures minimum cash position requirement",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetRiskMetrics returns risk dashboard metrics
// GET /api/dashboard/risk?tenant_id=xxx&valuation_date=yyyy-mm-dd
func (h *DashboardHandler) GetRiskMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Verify authentication
	userID, auth, err := h.verifyAuthentication(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Verify dashboard access permission
	if !h.hasPermission(auth, "dashboard:read") {
		h.logSecurityEvent(ctx, userID, "", "dashboard_access_denied", "risk", "", r.RemoteAddr)
		http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
		return
	}

	// 3. Extract tenant_id
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	// 4. Verify tenant access control
	if !h.verifyTenantAccess(auth, tenantID) {
		h.logSecurityEvent(ctx, userID, tenantID, "dashboard_cross_tenant_attempt", "risk", "", r.RemoteAddr)
		http.Error(w, "Forbidden: access denied to this tenant", http.StatusForbidden)
		return
	}

	// 5. Log audit trail
	h.logSecurityEvent(ctx, userID, tenantID, "dashboard_risk_accessed", "risk", "", r.RemoteAddr)

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	response := RiskResponse{
		Volatility: 7.5,
		VaRPercent: 95.0,
		VaRValue:   2300000.0,
		BetaMarket: 1.2,
		Drawdown:   -15.2,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Metrics: []RiskMetric{
			{
				MetricID:    "risk-001",
				MetricName:  "Volatility (Annualized)",
				Value:       7.5,
				Unit:        "%",
				Threshold:   8.0,
				Status:      "Normal",
				LastUpdated: time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339),
			},
			{
				MetricID:    "risk-002",
				MetricName:  "Value at Risk (95%)",
				Value:       2.3,
				Unit:        "M",
				Threshold:   2.5,
				Status:      "Normal",
				LastUpdated: time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339),
			},
			{
				MetricID:    "risk-003",
				MetricName:  "Maximum Drawdown",
				Value:       -15.2,
				Unit:        "%",
				Threshold:   -20.0,
				Status:      "Warning",
				LastUpdated: time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSparklines returns 7-day trend data
// GET /api/dashboard/sparklines?tenant_id=xxx&valuation_date=yyyy-mm-dd&time_range=day
func (h *DashboardHandler) GetSparklines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Verify authentication
	userID, auth, err := h.verifyAuthentication(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Verify dashboard access permission
	if !h.hasPermission(auth, "dashboard:read") {
		h.logSecurityEvent(ctx, userID, "", "dashboard_access_denied", "sparklines", "", r.RemoteAddr)
		http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
		return
	}

	// 3. Extract and validate tenant_id
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	// 4. Extract and validate time_range parameter
	timeRange := r.URL.Query().Get("time_range")
	validatedTimeRange, err := h.validateTimeRange(timeRange)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query parameter: %v", err), http.StatusBadRequest)
		return
	}

	// 5. Verify tenant access control
	if !h.verifyTenantAccess(auth, tenantID) {
		h.logSecurityEvent(ctx, userID, tenantID, "dashboard_cross_tenant_attempt", "sparklines", "", r.RemoteAddr)
		http.Error(w, "Forbidden: access denied to this tenant", http.StatusForbidden)
		return
	}

	// 6. Log audit trail
	h.logSecurityEvent(ctx, userID, tenantID, fmt.Sprintf("dashboard_sparklines_accessed_range=%s", validatedTimeRange), "sparklines", "", r.RemoteAddr)

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	now := time.Now()
	response := SparklinesResponse{
		Period:    "7d",
		Timestamp: now.UTC().Format(time.RFC3339),
		Metrics: []SparklineMetric{
			{
				MetricName: "Portfolio Value",
				Data:       generateSparklineData(now, 7, 10000000, 0.015),
			},
			{
				MetricName: "Daily Return %",
				Data:       generateSparklineData(now, 7, 0, 0.005),
			},
			{
				MetricName: "Volatility (30d)",
				Data:       generateSparklineData(now, 7, 7.5, 0.2),
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetETLHealth returns ETL health status
// GET /api/dashboard/etl-health?tenant_id=xxx
func (h *DashboardHandler) GetETLHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Verify authentication
	userID, auth, err := h.verifyAuthentication(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Verify dashboard access permission
	if !h.hasPermission(auth, "dashboard:read") {
		h.logSecurityEvent(ctx, userID, "", "dashboard_access_denied", "etl-health", "", r.RemoteAddr)
		http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
		return
	}

	// 3. Extract tenant_id
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	// 4. Verify tenant access control
	if !h.verifyTenantAccess(auth, tenantID) {
		h.logSecurityEvent(ctx, userID, tenantID, "dashboard_cross_tenant_attempt", "etl-health", "", r.RemoteAddr)
		http.Error(w, "Forbidden: access denied to this tenant", http.StatusForbidden)
		return
	}

	// 5. Log audit trail
	h.logSecurityEvent(ctx, userID, tenantID, "dashboard_etl_health_accessed", "etl-health", "", r.RemoteAddr)

	endTime := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	response := ETLHealthResponse{
		LastRun: ETLHealthStatus{
			RunID:            "etl-run-20260222-001",
			Status:           "Success",
			StartTime:        time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339),
			EndTime:          &endTime,
			RecordsProcessed: 1250000,
			RecordsFailed:    1250,
			Duration:         900,
			ErrorMessage:     nil,
		},
		RunCount24h:     48,
		SuccessRate:     98.5,
		AverageDuration: 850,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetAlerts returns active alerts
// GET /api/dashboard/alerts?tenant_id=xxx&severity=Critical|Warning|Info
func (h *DashboardHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Verify authentication
	userID, auth, err := h.verifyAuthentication(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Verify dashboard access permission
	if !h.hasPermission(auth, "dashboard:read") {
		h.logSecurityEvent(ctx, userID, "", "dashboard_access_denied", "alerts", "", r.RemoteAddr)
		http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
		return
	}

	// 3. Extract and validate tenant_id
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	// 4. Extract and validate severity parameter
	severity := r.URL.Query().Get("severity")
	validatedSeverity, err := h.validateSeverity(severity)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query parameter: %v", err), http.StatusBadRequest)
		return
	}

	// 5. Verify tenant access control
	if !h.verifyTenantAccess(auth, tenantID) {
		h.logSecurityEvent(ctx, userID, tenantID, "dashboard_cross_tenant_attempt", "alerts", "", r.RemoteAddr)
		http.Error(w, "Forbidden: access denied to this tenant", http.StatusForbidden)
		return
	}

	// 6. Log audit trail
	action := "dashboard_alerts_accessed"
	if validatedSeverity != "" {
		action = fmt.Sprintf("dashboard_alerts_accessed_severity=%s", validatedSeverity)
	}
	h.logSecurityEvent(ctx, userID, tenantID, action, "alerts", "", r.RemoteAddr)

	response := AlertsResponse{
		Critical:  3,
		Warning:   12,
		Info:      45,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Alerts: []AlertItem{
			{
				AlertID:   "alert-001",
				Title:     "Sector Concentration Alert",
				Severity:  "Critical",
				Message:   "Technology sector concentration exceeds 35% limit",
				Source:    "Compliance",
				CreatedAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
				Status:    "Open",
			},
			{
				AlertID:   "alert-002",
				Title:     "Portfolio Volatility Rising",
				Severity:  "Warning",
				Message:   "30-day volatility increased 2% above rolling average",
				Source:    "Risk",
				CreatedAt: time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
				Status:    "Open",
			},
			{
				AlertID:   "alert-003",
				Title:     "ETL Performance Degradation",
				Severity:  "Warning",
				Message:   "Last ETL run took 15% longer than average",
				Source:    "Operations",
				CreatedAt: time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339),
				Status:    "Acknowledged",
			},
		},
	}

	// Filter by severity if provided and validated
	if validatedSeverity != "" {
		filtered := []AlertItem{}
		for _, alert := range response.Alerts {
			if alert.Severity == validatedSeverity {
				filtered = append(filtered, alert)
			}
		}
		response.Alerts = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// TriggerETL triggers an ETL run asynchronously
// POST /api/dashboard/etl/trigger?tenant_id=xxx
func (h *DashboardHandler) TriggerETL(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	var req TriggerETLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Priority == "" {
		req.Priority = "normal"
	}

	// TODO: Queue ETL job asynchronously
	// For now, return success response
	response := TriggerETLResponse{
		RunID:     fmt.Sprintf("etl-run-%d", time.Now().Unix()),
		Status:    "Queued",
		Message:   "ETL run has been queued successfully",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// Security Helpers - Authentication, Authorization & Audit Logging
// ============================================================================

// verifyAuthentication extracts and verifies user identity from request context
// Sets up JWT claims from AuthContextMiddleware
func (h *DashboardHandler) verifyAuthentication(r *http.Request) (string, security.AuthInfo, error) {
	ctx := r.Context()

	// Extract user ID from context (set by AuthContextMiddleware)
	userID, hasUserID := identity.ActorIDFromContext(ctx)
	if !hasUserID || userID == "" {
		return "", security.AuthInfo{}, fmt.Errorf("user identity not found in context")
	}

	// Extract auth info including roles and tenant IDs
	auth, hasAuth := security.AuthInfoFromContext(ctx)
	if !hasAuth {
		return userID, security.AuthInfo{}, fmt.Errorf("auth info not found in context")
	}

	if len(auth.TenantIDs) == 0 && auth.UserID == "" {
		return userID, auth, fmt.Errorf("user has no tenant access")
	}

	return userID, auth, nil
}

// verifyTenantAccess ensures the requested tenant_id matches the user's authorized tenants
func (h *DashboardHandler) verifyTenantAccess(auth security.AuthInfo, requestedTenantID string) bool {
	if requestedTenantID == "" {
		return false
	}

	// Check if requested tenant is in user's authorized tenant list
	for _, tenantID := range auth.TenantIDs {
		if tenantID == requestedTenantID {
			return true
		}
	}

	return false
}

// hasPermission checks if user has required permission based on roles
// Dashboard access requires admin, analyst, compliance_officer, or risk_manager roles
func (h *DashboardHandler) hasPermission(auth security.AuthInfo, requiredPermission string) bool {
	// Roles that have dashboard access
	allowedRoles := map[string]bool{
		"admin":              true,
		"analyst":            true,
		"compliance_officer": true,
		"risk_manager":       true,
	}

	for _, role := range auth.Roles {
		if allowedRoles[role] {
			return true
		}
	}

	return false
}

// validateTimeRange validates the time_range query parameter
func (h *DashboardHandler) validateTimeRange(timeRange string) (string, error) {
	validRanges := map[string]bool{
		"hour":  true,
		"day":   true,
		"week":  true,
		"month": true,
	}

	if timeRange == "" {
		return "day", nil // default
	}

	if validRanges[timeRange] {
		return timeRange, nil
	}

	return "", fmt.Errorf("invalid time_range: %s (valid: hour, day, week, month)", timeRange)
}

// validateSeverity validates the severity query parameter for alerts
func (h *DashboardHandler) validateSeverity(severity string) (string, error) {
	validSeverities := map[string]bool{
		"Critical": true,
		"Warning":  true,
		"Info":     true,
	}

	if severity == "" {
		return "", nil // no filter
	}

	if validSeverities[severity] {
		return severity, nil
	}

	return "", fmt.Errorf("invalid severity: %s (valid: Critical, Warning, Info)", severity)
}

// logSecurityEvent logs dashboard access for audit trail
func (h *DashboardHandler) logSecurityEvent(ctx context.Context, userID, tenantID, action, resource, resourceID, ipAddress string) {
	if h.db == nil {
		log.Printf("[DASHBOARD-AUDIT] %s | user=%s | tenant=%s | action=%s | resource=%s | ip=%s",
			time.Now().Format(time.RFC3339), userID, tenantID, action, resource, ipAddress)
		return
	}

	event := middleware.SecurityAuditLog{
		UserID:     userID,
		TenantID:   tenantID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ipAddress,
		UserAgent:  "dashboard-api",
	}

	// Convert *sqlx.DB to *sql.DB for middleware logging
	middleware.LogSecurityEvent(ctx, h.db.DB, event)
}

// secureHandler wraps a handler with authentication and authorization checks
func (h *DashboardHandler) secureHandler(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract and verify authentication from context
		userID, auth, err := h.verifyAuthentication(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		// 2. Verify user has dashboard access permission
		if !h.hasPermission(auth, "dashboard:read") {
			h.logSecurityEvent(r.Context(), userID, "", "dashboard_access_denied", "dashboard", "", r.RemoteAddr)
			http.Error(w, "Forbidden: insufficient permissions for dashboard access", http.StatusForbidden)
			return
		}

		// Call the actual handler
		handler(w, r)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateSparklineData generates 7 days of synthetic trend data
func generateSparklineData(baseTime time.Time, days int, baseValue float64, volatility float64) []SparklineDataPoint {
	data := make([]SparklineDataPoint, days)
	for i := 0; i < days; i++ {
		date := baseTime.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")
		// Add some variance to make it realistic
		variance := float64(i) * volatility
		value := baseValue + (baseValue * variance)
		data[i] = SparklineDataPoint{
			Date:  date,
			Value: value,
		}
	}
	return data
}
