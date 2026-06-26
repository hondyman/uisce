package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/sirupsen/logrus"
)

// AuditReportHandler handles audit report generation and export requests
type AuditReportHandler struct {
	service services.AuditReportService
	logger  *logrus.Entry
}

// NewAuditReportHandler creates a new audit report handler
func NewAuditReportHandler(svc services.AuditReportService, logger *logrus.Entry) *AuditReportHandler {
	return &AuditReportHandler{
		service: svc,
		logger:  logger.WithField("handler", "audit_report"),
	}
}

// GenerateReport handles POST /api/v1/audit/reports
// Generates an audit report in the requested format (json or csv)
func (h *AuditReportHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract authenticated context
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	if userID == "" || tenantID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req services.AuditReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate and sanitize request
	req.TenantID = tenantID // Override with authenticated tenant
	if req.Format == "" {
		req.Format = "json"
	}

	// Validate format
	if req.Format != "json" && req.Format != "csv" {
		http.Error(w, "Invalid format - must be 'json' or 'csv'", http.StatusBadRequest)
		return
	}

	// Validate date range
	if !req.EndDate.IsZero() && !req.StartDate.IsZero() {
		if req.EndDate.Before(req.StartDate) {
			http.Error(w, "end_date must be after start_date", http.StatusBadRequest)
			return
		}
	}

	// Default to last 30 days if not specified
	if req.StartDate.IsZero() {
		req.StartDate = time.Now().AddDate(0, 0, -30)
	}
	if req.EndDate.IsZero() {
		req.EndDate = time.Now()
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"tenant_id":   tenantID,
		"start_date":  req.StartDate,
		"end_date":    req.EndDate,
		"format":      req.Format,
		"entity_type": req.EntityType,
		"action":      req.Action,
	}).Info("Generating audit report")

	// Generate report
	report, err := h.service.GenerateReport(ctx, req)
	if err != nil {
		h.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to generate audit report")
		http.Error(w, "Failed to generate report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response format based on requested format
	if req.Format == "csv" {
		// CSV download
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=audit-report-%s.csv", time.Now().Format("20060102-150405")))
		w.WriteHeader(http.StatusOK)

		if err := h.service.ExportToCSV(w, report.Records); err != nil {
			h.logger.WithError(err).Error("Failed to export to CSV")
			// Response already started, can't send error
			return
		}
	} else {
		// JSON response
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(report); err != nil {
			h.logger.WithError(err).Error("Failed to encode JSON response")
		}
	}
}

// GetSummary handles GET /api/v1/audit/summary
// Returns summary statistics of audit activity
func (h *AuditReportHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := middleware.ExtractTenantIDFromContext(ctx)
	if tenantID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	if startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = t
		}
	}

	if endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = t
		}
	}

	summary, err := h.service.GetSummary(ctx, tenantID, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get summary")
		http.Error(w, "Failed to get summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary)
}

// VerifyCompliance handles GET /api/v1/audit/compliance
// Verifies if tenant meets compliance requirements
func (h *AuditReportHandler) VerifyCompliance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := middleware.ExtractTenantIDFromContext(ctx)
	if tenantID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	required, err := h.service.VerifyComplianceRequired(ctx, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to verify compliance")
		http.Error(w, "Failed to verify compliance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":           tenantID,
		"compliance_required": required,
		"last_check":          time.Now().UTC(),
		"status":              "ok",
	})
}
