package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// DataQualityHandler handles Data Quality Sentinel & Nullability Profiling endpoints.
type DataQualityHandler struct {
	sentinel *analytics.DataQualitySentinelService
}

// NewDataQualityHandler creates a new DataQualityHandler instance.
func NewDataQualityHandler(db *sqlx.DB) *DataQualityHandler {
	return &DataQualityHandler{
		sentinel: analytics.NewDataQualitySentinelService(db),
	}
}

// RegisterRoutes registers data quality audit and gatekeeper endpoints.
func (h *DataQualityHandler) RegisterRoutes(r chi.Router) {
	r.Post("/business-objects/{id}/quality-audit", h.RunQualityAudit)
	r.Get("/business-objects/{id}/quality-summary", h.GetQualitySummary)
	r.Patch("/business-objects/{id}/fields/{fieldId}/fallback", h.SetFieldFallback)
}

// extractTenantID resolves the tenant for data-quality audit/gatekeeper
// operations. It NEVER trusts the raw X-Tenant-ID header: the header is
// only honored when a valid JWT is present and ValidateTenantAccess
// confirms that JWT is entitled to the requested tenant (including
// core/global admins, who may select any tenant).
func (h *DataQualityHandler) extractTenantID(r *http.Request) uuid.UUID {
	claims, err := jwtmiddleware.ValidateTokenFromRequest(r)
	if err != nil || claims == nil {
		return uuid.Nil
	}

	headerTid := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if headerTid == "" {
		if tid, err := uuid.Parse(claims.TenantID); err == nil {
			return tid
		}
		return uuid.Nil
	}

	if verr := jwtmiddleware.ValidateTenantAccess(claims, headerTid); verr != nil {
		return uuid.Nil
	}
	if tid, err := uuid.Parse(headerTid); err == nil {
		return tid
	}
	return uuid.Nil
}

// RunQualityAudit triggers reservoir sampling across all physical columns of a Business Object.
func (h *DataQualityHandler) RunQualityAudit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := h.extractTenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"valid tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	boIDStr := chi.URLParam(r, "id")
	boID, err := uuid.Parse(boIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid business_object_id"}`, http.StatusBadRequest)
		return
	}

	summary, err := h.sentinel.ProfileBusinessObjectQuality(r.Context(), tenantID, boID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Quality audit failed for BO %s: %v", boID, err)
		http.Error(w, `{"error":"failed executing data quality audit"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

// GetQualitySummary returns current health and gatekeeper state for a Business Object.
func (h *DataQualityHandler) GetQualitySummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := h.extractTenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"valid tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	boIDStr := chi.URLParam(r, "id")
	boID, err := uuid.Parse(boIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid business_object_id"}`, http.StatusBadRequest)
		return
	}

	summary, err := h.sentinel.GetQualitySummary(r.Context(), tenantID, boID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed getting quality summary for BO %s: %v", boID, err)
		http.Error(w, `{"error":"failed fetching quality summary"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

// SetFieldFallback sets a defensive fallback value to auto-resolve warnings via COALESCE.
func (h *DataQualityHandler) SetFieldFallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := h.extractTenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"valid tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	boIDStr := chi.URLParam(r, "id")
	boID, err := uuid.Parse(boIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid business_object_id"}`, http.StatusBadRequest)
		return
	}

	fieldIDStr := chi.URLParam(r, "fieldId")
	fieldID, err := uuid.Parse(fieldIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid field_id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		DefaultFallbackValue string `json:"default_fallback_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	err = h.sentinel.SetFieldFallback(r.Context(), tenantID, boID, fieldID, req.DefaultFallbackValue)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed setting field fallback for field %s: %v", fieldID, err)
		http.Error(w, `{"error":"failed updating field fallback"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":                 "success",
		"field_id":               fieldID,
		"default_fallback_value": req.DefaultFallbackValue,
		"quality_status":         "HEALTHY",
	})
}
