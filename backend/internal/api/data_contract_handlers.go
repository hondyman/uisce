package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/governance/contracts"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type DataContractHandler struct {
	gatekeeper *contracts.Gatekeeper
}

func NewDataContractHandler(gatekeeper *contracts.Gatekeeper) *DataContractHandler {
	return &DataContractHandler{gatekeeper: gatekeeper}
}

func (h *DataContractHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/governance/contracts", func(r chi.Router) {
		r.Post("/validate", h.Validate)
		r.Get("/violations", h.ListViolations)
		r.Post("/violations/{violationID}/approve", h.ApproveViolation)
		r.Post("/violations/{violationID}/reject", h.RejectViolation)
	})
}

func (h *DataContractHandler) Validate(w http.ResponseWriter, r *http.Request) {
	tenantID := extractValidatedTenantID(w, r)
	if tenantID == "" {
		return
	}

	var req contracts.ContractValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), "decode_error", nil)
		return
	}

	if req.TenantID != "" && req.TenantID != tenantID {
		http.Error(w, `{"error":"tenant_id mismatch","code":"tenant_mismatch"}`, http.StatusForbidden)
		return
	}
	req.TenantID = tenantID

	resp, err := h.gatekeeper.Validate(r.Context(), &req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "validation failed: "+err.Error(), "validation_error", nil)
		return
	}

	status := http.StatusOK
	if resp.HasCritical {
		status = http.StatusUnprocessableEntity
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (h *DataContractHandler) ListViolations(w http.ResponseWriter, r *http.Request) {
	tenantID := extractValidatedTenantID(w, r)
	if tenantID == "" {
		return
	}

	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	violations, err := h.gatekeeper.ListViolations(r.Context(), tenantID, status, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "db_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"violations": violations,
		"count":      len(violations),
	})
}

func (h *DataContractHandler) ApproveViolation(w http.ResponseWriter, r *http.Request) {
	tenantID := extractValidatedTenantID(w, r)
	if tenantID == "" {
		return
	}

	violationID := chi.URLParam(r, "violationID")
	if violationID == "" {
		writeJSONError(w, http.StatusBadRequest, "violationID is required", "missing_param", nil)
		return
	}

	claims, _ := jwtmiddleware.ValidateTokenFromRequest(r)
	reviewerID := "unknown"
	if claims != nil {
		reviewerID = claims.UserID
	}

	if err := h.gatekeeper.ApproveViolation(r.Context(), violationID, reviewerID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "db_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "approved",
		"violation_id": violationID,
		"reviewed_by":  reviewerID,
	})
}

func (h *DataContractHandler) RejectViolation(w http.ResponseWriter, r *http.Request) {
	tenantID := extractValidatedTenantID(w, r)
	if tenantID == "" {
		return
	}

	violationID := chi.URLParam(r, "violationID")
	if violationID == "" {
		writeJSONError(w, http.StatusBadRequest, "violationID is required", "missing_param", nil)
		return
	}

	var reqBody struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		reqBody.Reason = ""
	}

	claims, _ := jwtmiddleware.ValidateTokenFromRequest(r)
	reviewerID := "unknown"
	if claims != nil {
		reviewerID = claims.UserID
	}

	if err := h.gatekeeper.RejectViolation(r.Context(), violationID, reviewerID, reqBody.Reason); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "db_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "blocked",
		"violation_id": violationID,
		"reviewed_by":  reviewerID,
	})
}

func extractValidatedTenantID(w http.ResponseWriter, r *http.Request) string {
	claims, err := jwtmiddleware.ValidateTokenFromRequest(r)
	if err != nil || claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return ""
	}

	claimsTenant := claims.TenantID
	headerTenant := r.Header.Get("X-Tenant-ID")

	if headerTenant != "" && headerTenant != claimsTenant {
		http.Error(w, `{"error":"tenant_id mismatch","code":"tenant_mismatch"}`, http.StatusForbidden)
		return ""
	}

	if claimsTenant == "" && headerTenant != "" {
		claimsTenant = headerTenant
	}

	if claimsTenant == "" {
		http.Error(w, `{"error":"tenant_id required","code":"missing_tenant"}`, http.StatusBadRequest)
		return ""
	}

	return claimsTenant
}


