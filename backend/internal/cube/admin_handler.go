package cube

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CubeAdminHandler provides HTTP handlers for the Cube admin console
type CubeAdminHandler struct {
	service *CubeAdminService
}

// NewCubeAdminHandler creates a new admin handler
func NewCubeAdminHandler(service *CubeAdminService) *CubeAdminHandler {
	return &CubeAdminHandler{service: service}
}

// RegisterRoutes registers all admin console routes
func (h *CubeAdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/cube-admin", func(r chi.Router) {
		// Dashboard
		r.Get("/dashboard/stats", h.GetDashboardStats)

		// Organizations (super admin)
		r.Get("/organizations", h.ListOrganizations)
		r.Post("/organizations", h.CreateOrganization)
		r.Get("/organizations/{orgId}", h.GetOrganization)
		r.Put("/organizations/{orgId}", h.UpdateOrganization)

		// Tenants (org admin + super admin)
		r.Get("/tenants", h.ListTenants)
		r.Post("/tenants", h.CreateTenantConfig)
		r.Get("/tenants/{tenantId}", h.GetTenantConfig)
		r.Put("/tenants/{tenantId}", h.UpdateTenantConfig)
		r.Delete("/tenants/{tenantId}", h.DeleteTenantConfig)

		// Cube Catalog
		r.Get("/cubes", h.ListCubes)
		r.Post("/cubes", h.CreateCube)
		r.Get("/cubes/{cubeId}", h.GetCube)
		r.Put("/cubes/{cubeId}", h.UpdateCube)
		r.Delete("/cubes/{cubeId}", h.DeleteCube)
		r.Post("/cubes/{cubeId}/validate", h.ValidateCube)
		r.Post("/cubes/{cubeId}/deploy", h.DeployCube)

		// Query Analytics
		r.Get("/analytics/queries", h.GetQueryAnalytics)
		r.Get("/analytics/performance", h.GetPerformanceMetrics)
		r.Get("/analytics/usage", h.GetUsageMetrics)
		r.Get("/analytics/slow-queries", h.GetSlowQueries)

		// Pre-aggregation Management
		r.Get("/preaggs", h.ListPreAggregations)
		r.Get("/preaggs/suggestions", h.GetPreAggSuggestions)
		r.Post("/preaggs/suggestions/{suggestionId}/approve", h.ApprovePreAggSuggestion)
		r.Post("/preaggs/suggestions/{suggestionId}/reject", h.RejectPreAggSuggestion)
		r.Post("/preaggs/refresh", h.TriggerPreAggRefresh)

		// Scheduled Reports
		r.Get("/reports", h.ListScheduledReports)
		r.Post("/reports", h.CreateScheduledReport)
		r.Get("/reports/{reportId}", h.GetScheduledReport)
		r.Put("/reports/{reportId}", h.UpdateScheduledReport)
		r.Delete("/reports/{reportId}", h.DeleteScheduledReport)
		r.Post("/reports/{reportId}/run", h.RunReportNow)

		// Cache Management
		r.Get("/cache/stats", h.GetCacheStats)
		r.Post("/cache/clear", h.ClearCache)
		r.Post("/cache/warm", h.WarmCache)

		// Admin Users
		r.Get("/users", h.ListAdminUsers)
		r.Post("/users", h.CreateAdminUser)
		r.Put("/users/{userId}", h.UpdateAdminUser)
		r.Delete("/users/{userId}", h.DeleteAdminUser)
	})
}

// Context helpers
func getTenantID(r *http.Request) uuid.UUID {
	tidStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	if tidStr == "" {
		tidStr = r.URL.Query().Get("tenant_id")
	}
	tid, _ := uuid.Parse(tidStr)
	return tid
}

func getDatasourceID(r *http.Request) uuid.UUID {
	dsStr := r.Header.Get("X-Tenant-Datasource-ID")
	if dsStr == "" {
		dsStr = r.URL.Query().Get("datasource_id")
	}
	ds, _ := uuid.Parse(dsStr)
	return ds
}

func getOrgID(r *http.Request) *uuid.UUID {
	orgStr := r.Header.Get("X-Organization-ID")
	if orgStr == "" {
		orgStr = r.URL.Query().Get("organization_id")
	}
	if orgStr == "" {
		return nil
	}
	org, err := uuid.Parse(orgStr)
	if err != nil {
		return nil
	}
	return &org
}

// Dashboard handlers
func (h *CubeAdminHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgID(r)

	stats, err := h.service.GetDashboardStats(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Organization handlers
func (h *CubeAdminHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.service.GetOrganizationHierarchy(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orgs)
}

func (h *CubeAdminHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var org Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.service.CreateOrganization(r.Context(), org)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CubeAdminHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	org, err := h.service.GetOrganization(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h *CubeAdminHandler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	var org Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	org.ID = orgID

	updated, err := h.service.UpdateOrganization(r.Context(), org)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Tenant handlers
func (h *CubeAdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgID(r)
	if orgID == nil {
		writeError(w, http.StatusBadRequest, "organization_id required")
		return
	}

	tenants, err := h.service.GetTenantsForOrganization(r.Context(), *orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tenants)
}

func (h *CubeAdminHandler) CreateTenantConfig(w http.ResponseWriter, r *http.Request) {
	var config TenantCubeConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.service.CreateTenantConfig(r.Context(), config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CubeAdminHandler) GetTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantId")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	config, err := h.service.GetTenantConfig(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (h *CubeAdminHandler) UpdateTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantId")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	var config TenantCubeConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	config.TenantID = tenantID

	updated, err := h.service.UpdateTenantConfig(r.Context(), config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *CubeAdminHandler) DeleteTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantId")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	if err := h.service.DeleteTenantConfig(r.Context(), tenantID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// Cube Catalog handlers
func (h *CubeAdminHandler) ListCubes(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	datasourceID := getDatasourceID(r)

	cubes, err := h.service.GetCubeCatalog(r.Context(), tenantID, datasourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cubes)
}

func (h *CubeAdminHandler) CreateCube(w http.ResponseWriter, r *http.Request) {
	var cube CubeCatalogEntry
	if err := json.NewDecoder(r.Body).Decode(&cube); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Set tenant context from request
	cube.TenantID = getTenantID(r)
	cube.DatasourceID = getDatasourceID(r)

	created, err := h.service.CreateCube(r.Context(), cube)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CubeAdminHandler) GetCube(w http.ResponseWriter, r *http.Request) {
	cubeIDStr := chi.URLParam(r, "cubeId")
	cubeID, err := uuid.Parse(cubeIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cube ID")
		return
	}

	cube, err := h.service.GetCube(r.Context(), cubeID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cube)
}

func (h *CubeAdminHandler) UpdateCube(w http.ResponseWriter, r *http.Request) {
	cubeIDStr := chi.URLParam(r, "cubeId")
	cubeID, err := uuid.Parse(cubeIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cube ID")
		return
	}

	var cube CubeCatalogEntry
	if err := json.NewDecoder(r.Body).Decode(&cube); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cube.ID = cubeID

	updated, err := h.service.UpdateCube(r.Context(), cube)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *CubeAdminHandler) DeleteCube(w http.ResponseWriter, r *http.Request) {
	cubeIDStr := chi.URLParam(r, "cubeId")
	cubeID, err := uuid.Parse(cubeIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cube ID")
		return
	}

	if err := h.service.DeleteCube(r.Context(), cubeID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *CubeAdminHandler) ValidateCube(w http.ResponseWriter, r *http.Request) {
	cubeIDStr := chi.URLParam(r, "cubeId")
	cubeID, err := uuid.Parse(cubeIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cube ID")
		return
	}

	result, err := h.service.ValidateCube(r.Context(), cubeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *CubeAdminHandler) DeployCube(w http.ResponseWriter, r *http.Request) {
	cubeIDStr := chi.URLParam(r, "cubeId")
	cubeID, err := uuid.Parse(cubeIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cube ID")
		return
	}

	if err := h.service.DeployCube(r.Context(), cubeID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deployed"})
}

// Query Analytics handlers
func (h *CubeAdminHandler) GetQueryAnalytics(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	sinceStr := r.URL.Query().Get("since")
	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	analytics, err := h.service.GetQueryAnalytics(r.Context(), tenantID, since, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, analytics)
}

func (h *CubeAdminHandler) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	metrics, err := h.service.GetPerformanceMetrics(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *CubeAdminHandler) GetUsageMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	metrics, err := h.service.GetUsageMetrics(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *CubeAdminHandler) GetSlowQueries(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	thresholdMs := 1000 // default 1 second
	if t := r.URL.Query().Get("threshold_ms"); t != "" {
		if thresh, err := strconv.Atoi(t); err == nil {
			thresholdMs = thresh
		}
	}

	slowQueries, err := h.service.GetSlowQueries(r.Context(), tenantID, thresholdMs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, slowQueries)
}

// Pre-aggregation handlers
func (h *CubeAdminHandler) ListPreAggregations(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	preAggs, err := h.service.ListPreAggregations(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preAggs)
}

func (h *CubeAdminHandler) GetPreAggSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	suggestions, err := h.service.GetPreAggSuggestions(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, suggestions)
}

func (h *CubeAdminHandler) ApprovePreAggSuggestion(w http.ResponseWriter, r *http.Request) {
	suggestionIDStr := chi.URLParam(r, "suggestionId")
	suggestionID, err := uuid.Parse(suggestionIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid suggestion ID")
		return
	}

	// Get reviewer ID from context (assume set by auth middleware)
	reviewerID := uuid.New() // TODO: Get from auth context

	if err := h.service.ApprovePreAggSuggestion(r.Context(), suggestionID, reviewerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *CubeAdminHandler) RejectPreAggSuggestion(w http.ResponseWriter, r *http.Request) {
	suggestionIDStr := chi.URLParam(r, "suggestionId")
	suggestionID, err := uuid.Parse(suggestionIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid suggestion ID")
		return
	}

	// Get reviewer ID from context
	reviewerID := uuid.New() // TODO: Get from auth context

	if err := h.service.RejectPreAggSuggestion(r.Context(), suggestionID, reviewerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (h *CubeAdminHandler) TriggerPreAggRefresh(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	if err := h.service.TriggerPreAggRefresh(r.Context(), tenantID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "refresh_triggered"})
}

// Scheduled Reports handlers
func (h *CubeAdminHandler) ListScheduledReports(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	reports, err := h.service.GetScheduledReports(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reports)
}

func (h *CubeAdminHandler) CreateScheduledReport(w http.ResponseWriter, r *http.Request) {
	var report ScheduledReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	report.TenantID = getTenantID(r)
	report.DatasourceID = getDatasourceID(r)

	created, err := h.service.CreateScheduledReport(r.Context(), report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CubeAdminHandler) GetScheduledReport(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	report, err := h.service.GetScheduledReport(r.Context(), reportID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *CubeAdminHandler) UpdateScheduledReport(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	var report ScheduledReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	report.ID = reportID

	updated, err := h.service.UpdateScheduledReport(r.Context(), report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *CubeAdminHandler) DeleteScheduledReport(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	if err := h.service.DeleteScheduledReport(r.Context(), reportID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *CubeAdminHandler) RunReportNow(w http.ResponseWriter, r *http.Request) {
	reportIDStr := chi.URLParam(r, "reportId")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report ID")
		return
	}

	if err := h.service.RunReportNow(r.Context(), reportID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "running"})
}

// Cache Management handlers
func (h *CubeAdminHandler) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	stats, err := h.service.GetCacheStats(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *CubeAdminHandler) ClearCache(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	if err := h.service.ClearCache(r.Context(), tenantID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (h *CubeAdminHandler) WarmCache(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	if err := h.service.WarmCache(r.Context(), tenantID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "warming"})
}

// Admin User handlers
func (h *CubeAdminHandler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgID(r)
	if orgID == nil {
		writeError(w, http.StatusBadRequest, "organization_id required")
		return
	}

	users, err := h.service.ListAdminUsers(r.Context(), *orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *CubeAdminHandler) CreateAdminUser(w http.ResponseWriter, r *http.Request) {
	var user CubeAdminUser
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.service.CreateAdminUser(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CubeAdminHandler) UpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var user CubeAdminUser
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user.ID = userID

	updated, err := h.service.UpdateAdminUser(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *CubeAdminHandler) DeleteAdminUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DeleteAdminUser(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
