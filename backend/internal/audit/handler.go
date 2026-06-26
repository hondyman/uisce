package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
)

// Handler handles audit-related HTTP requests
type Handler struct {
	auditService *Service
}

// NewHandler creates a new audit handler
func NewHandler(auditService *Service) *Handler {
	return &Handler{
		auditService: auditService,
	}
}

// GetAuditEvents handles GET /api/audit/events
func (h *Handler) GetAuditEvents(w http.ResponseWriter, r *http.Request) {
	filter := &models.AuditEventFilter{}
	query := r.URL.Query()

	// Parse query parameters
	if userID := query.Get("user_id"); userID != "" {
		filter.UserID = &userID
	}
	if tenantID := query.Get("tenant_id"); tenantID != "" {
		filter.TenantID = &tenantID
	}
	if eventType := query.Get("event_type"); eventType != "" {
		et := models.AuditEventType(eventType)
		filter.EventType = &et
	}
	if severity := query.Get("severity"); severity != "" {
		s := models.AuditEventSeverity(severity)
		filter.Severity = &s
	}
	if resourceType := query.Get("resource_type"); resourceType != "" {
		filter.ResourceType = &resourceType
	}
	if resourceID := query.Get("resource_id"); resourceID != "" {
		filter.ResourceID = &resourceID
	}
	if startTimeStr := query.Get("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}
	if endTimeStr := query.Get("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}
	if ipAddress := query.Get("ip_address"); ipAddress != "" {
		filter.IPAddress = &ipAddress
	}
	if successStr := query.Get("success"); successStr != "" {
		if success, err := strconv.ParseBool(successStr); err == nil {
			filter.Success = &success
		}
	}
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	events, err := h.auditService.QueryEvents(r.Context(), filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query audit events"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

// GetAuditSummary handles GET /api/audit/summary
func (h *Handler) GetAuditSummary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var tenantID *string
	if tid := query.Get("tenant_id"); tid != "" {
		tenantID = &tid
	}

	// Default to last 30 days
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)

	if startStr := query.Get("start_time"); startStr != "" {
		if st, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = st
		}
	}
	if endStr := query.Get("end_time"); endStr != "" {
		if et, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = et
		}
	}

	summary, err := h.auditService.GetAuditSummary(r.Context(), tenantID, startTime, endTime)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get audit summary"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetAuditEvent handles GET /api/audit/events/{id}
func (h *Handler) GetAuditEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")

	filter := &models.AuditEventFilter{
		Limit: 1,
	}

	// We can't directly filter by ID, so we'll need to get all events and find the matching one
	// In a production system, you'd want to add an ID filter to the service
	events, err := h.auditService.QueryEvents(r.Context(), filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query audit events"})
		return
	}

	for _, event := range events {
		if event.ID == eventID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(event)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Audit event not found"})
}

// ExportAuditEvents handles POST /api/audit/export
func (h *Handler) ExportAuditEvents(w http.ResponseWriter, r *http.Request) {
	var exportRequest struct {
		Filter     *models.AuditEventFilter `json:"filter"`
		Format     string                   `json:"format"` // "json", "csv", "pdf"
		ReportName string                   `json:"report_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&exportRequest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request format"})
		return
	}

	if exportRequest.Filter == nil {
		exportRequest.Filter = &models.AuditEventFilter{}
	}

	// Set default limit for export
	if exportRequest.Filter.Limit == 0 {
		exportRequest.Filter.Limit = 10000 // Max 10k records for export
	}

	events, err := h.auditService.QueryEvents(r.Context(), exportRequest.Filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query audit events for export"})
		return
	}

	// Generate report based on format
	switch exportRequest.Format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=audit_export.json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"events":      events,
			"exported_at": time.Now(),
			"total_count": len(events),
		})
	case "csv":
		h.exportAsCSV(w, events, exportRequest.ReportName)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unsupported export format"})
	}
}

// CreateComplianceReport handles POST /api/audit/compliance-reports
func (h *Handler) CreateComplianceReport(w http.ResponseWriter, r *http.Request) {
	var reportRequest struct {
		ReportType string    `json:"report_type"`
		StartDate  time.Time `json:"start_date"`
		EndDate    time.Time `json:"end_date"`
		TenantID   *string   `json:"tenant_id"`
		ReportName string    `json:"report_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reportRequest); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request format"})
		return
	}

	// Get user ID from header
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "system"
	}

	// Generate summary
	summary, err := h.auditService.GetAuditSummary(r.Context(), reportRequest.TenantID, reportRequest.StartDate, reportRequest.EndDate)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate compliance report"})
		return
	}

	// Create compliance report record
	report := &models.ComplianceReport{
		ID:          uuid.New().String(),
		ReportType:  reportRequest.ReportType,
		TimeRange:   models.AuditTimeRange{Start: reportRequest.StartDate, End: reportRequest.EndDate},
		GeneratedAt: time.Now(),
		GeneratedBy: userID,
		Summary:     *summary,
		Status:      "completed",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(report)
}

// GetComplianceReports handles GET /api/audit/compliance-reports
func (h *Handler) GetComplianceReports(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you'd query the compliance_reports table
	// For now, return empty array
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reports": []models.ComplianceReport{},
		"count":   0,
	})
}

// CleanupAuditEvents handles POST /api/audit/cleanup
func (h *Handler) CleanupAuditEvents(w http.ResponseWriter, r *http.Request) {
	// Only allow admin users to perform cleanup
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "steward" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Insufficient permissions"})
		return
	}

	err := h.auditService.CleanupOldEvents(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cleanup audit events"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Audit cleanup completed successfully"})
}

// RegisterRoutes registers audit routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/audit", func(r chi.Router) {
		r.Get("/events", h.GetAuditEvents)
		r.Get("/events/{id}", h.GetAuditEvent)
		r.Get("/summary", h.GetAuditSummary)
		r.Post("/export", h.ExportAuditEvents)
		r.Get("/compliance-reports", h.GetComplianceReports)
		r.Post("/compliance-reports", h.CreateComplianceReport)
		r.Post("/cleanup", h.CleanupAuditEvents)
	})
}

// exportAsCSV exports audit events as CSV
func (h *Handler) exportAsCSV(w http.ResponseWriter, events []models.AuditEvent, reportName string) {
	w.Header().Set("Content-Type", "text/csv")
	if reportName == "" {
		reportName = "audit_export"
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+reportName+".csv")

	// Write CSV header
	w.Write([]byte("ID,Timestamp,Event Type,Severity,User ID,Tenant ID,Resource Type,Resource ID,Action,Success,Error Message\n"))

	// Write CSV rows
	for _, event := range events {
		w.Write([]byte(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%t,%s\n",
			event.ID,
			event.Timestamp.Format(time.RFC3339),
			string(event.EventType),
			string(event.Severity),
			event.UserID,
			event.TenantID,
			event.ResourceType,
			event.ResourceID,
			event.Action,
			event.Success,
			event.ErrorMessage,
		)))
	}

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
