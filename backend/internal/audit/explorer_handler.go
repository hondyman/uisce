package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
)

// ExplorerHandler handles audit explorer API requests
type ExplorerHandler struct {
	service *ExplorerService
}

// NewExplorerHandler creates a new audit explorer handler
func NewExplorerHandler(service *ExplorerService) *ExplorerHandler {
	return &ExplorerHandler{
		service: service,
	}
}

// RegisterRoutes registers all audit explorer routes
func (h *ExplorerHandler) RegisterRoutes(r chi.Router) {
	r.Route("/audit-explorer", func(r chi.Router) {
		// All routes require authentication and multi-tenant scoping
		r.Use(TenantScopeMiddlewareChi)

		// Timeline tab
		r.Post("/events", h.ListEvents)

		// Entities tab
		r.Get("/entities/{entityType}/{entityID}", h.GetEntityAudit)

		// Incidents tab
		r.Get("/incidents", h.ListIncidents)
		r.Get("/incidents/{incidentID}", h.GetIncident)

		// Compliance tab
		r.Get("/compliance-events", h.ListComplianceEvents)

		// AI explanation
		r.Post("/explain", h.ExplainAuditEvent)

		// Dashboards (role-specific)
		r.Get("/dashboard/global-admin", h.GetGlobalAdminDashboard)
		r.Get("/dashboard/global-ops", h.GetGlobalOpsDashboard)
		r.Get("/dashboard/tenant-admin/{tenantID}", h.GetTenantAdminDashboard)
		r.Get("/dashboard/tenant-ops/{tenantID}", h.GetTenantOpsDashboard)
	})
}

// ListEvents lists audit events with filters
// POST /audit-explorer/events
func (h *ExplorerHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	var req ListEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Default time range to last 7 days if not specified
	if req.TimeRange.From.IsZero() {
		req.TimeRange.From = time.Now().AddDate(0, 0, -7)
	}
	if req.TimeRange.To.IsZero() {
		req.TimeRange.To = time.Now()
	}

	if req.Limit == 0 {
		req.Limit = 50
	}

	resp, err := h.service.ListEvents(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetEntityAudit retrieves all audit events for an entity
// GET /audit-explorer/entities/{entityType}/{entityID}
func (h *ExplorerHandler) GetEntityAudit(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityID := chi.URLParam(r, "entityID")

	// Parse query parameters
	from := time.Now().AddDate(0, -3, 0) // Default: last 3 months
	to := time.Now()

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	result, err := h.service.GetEntityAudit(r.Context(), entityType, entityID, from, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get entity audit: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ListIncidents lists incident clusters
// GET /audit-explorer/incidents
func (h *ExplorerHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, 0, -7) // Default: last 7 days
	to := time.Now()

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	limit := 50
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
	}

	incidents, err := h.service.ListIncidents(r.Context(), from, to, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list incidents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"incidents": incidents,
	})
}

// GetIncident retrieves a single incident
// GET /audit-explorer/incidents/{incidentID}
func (h *ExplorerHandler) GetIncident(w http.ResponseWriter, r *http.Request) {
	incidentID := chi.URLParam(r, "incidentID")

	incident, err := h.service.GetIncident(r.Context(), incidentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get incident: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incident)
}

// ListComplianceEvents lists compliance events
// GET /audit-explorer/compliance-events
func (h *ExplorerHandler) ListComplianceEvents(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, -1, 0) // Default: last month
	to := time.Now()

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}

	violationTypes := []string{}
	if violationStr := r.URL.Query().Get("violationTypes"); violationStr != "" {
		// Parse comma-separated violation types
		if err := json.Unmarshal([]byte(violationStr), &violationTypes); err != nil {
			violationTypes = []string{}
		}
	}

	limit := 50
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	events, err := h.service.ListComplianceEvents(r.Context(), from, to, violationTypes, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list compliance events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
	})
}

// ExplainAuditEvent generates AI explanation for audit events
// POST /audit-explorer/explain
func (h *ExplorerHandler) ExplainAuditEvent(w http.ResponseWriter, r *http.Request) {
	var req ExplainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.AuditRecords) == 0 {
		http.Error(w, "at least one audit record is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ExplainAuditEvent(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to explain event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetGlobalAdminDashboard returns platform-wide dashboard (Global Admin only)
// GET /audit-explorer/dashboard/global-admin
func (h *ExplorerHandler) GetGlobalAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// Check role
	if !hasRole(r, "global_admin") {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return
	}

	from := time.Now().AddDate(0, 0, -7) // Last 7 days
	to := time.Now()

	dashboard, err := h.service.GetGlobalAdminDashboard(r.Context(), from, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get dashboard: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// GetGlobalOpsDashboard returns multi-tenant ops dashboard (Global Ops only)
// GET /audit-explorer/dashboard/global-ops
func (h *ExplorerHandler) GetGlobalOpsDashboard(w http.ResponseWriter, r *http.Request) {
	// Check role
	if !hasRole(r, "global_ops") {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return
	}

	from := time.Now().AddDate(0, 0, -1) // Last 24 hours
	to := time.Now()

	dashboard, err := h.service.GetGlobalOpsDashboard(r.Context(), from, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get dashboard: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// GetTenantAdminDashboard returns tenant-specific dashboard (Tenant Admin only)
// GET /audit-explorer/dashboard/tenant-admin/{tenantID}
func (h *ExplorerHandler) GetTenantAdminDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")

	// Check role and tenant access
	if !hasRole(r, "tenant_admin") {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return
	}

	userTenantID := auth.TenantIDFromContext(r.Context())
	if userTenantID != tenantID {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	dashboard, err := h.service.GetTenantAdminDashboard(r.Context(), tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get dashboard: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// GetTenantOpsDashboard returns tenant ops dashboard (Tenant Ops only)
// GET /audit-explorer/dashboard/tenant-ops/{tenantID}
func (h *ExplorerHandler) GetTenantOpsDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")

	// Check role and tenant access
	if !hasRole(r, "tenant_ops") {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return
	}

	userTenantID := auth.TenantIDFromContext(r.Context())
	if userTenantID != tenantID {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	dashboard, err := h.service.GetTenantOpsDashboard(r.Context(), tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get dashboard: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// Helper function to check if user has a specific role
func hasRole(r *http.Request, role string) bool {
	roles := auth.RolesFromContext(r.Context())
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
