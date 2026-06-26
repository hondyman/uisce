package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/metadata"
)

// EvidenceBundleHandler handles evidence bundle API requests
type EvidenceBundleHandler struct {
	service *metadata.EvidenceBundleService
}

// NewEvidenceBundleHandler creates a new evidence bundle handler
func NewEvidenceBundleHandler(service *metadata.EvidenceBundleService) *EvidenceBundleHandler {
	return &EvidenceBundleHandler{service: service}
}

// RegisterRoutes registers the routes for EvidenceBundleHandler.
func (h *EvidenceBundleHandler) RegisterRoutes(r chi.Router, approvalHandler *ApprovalHandler) {
	r.Route("/api/metadata/evidence/bundles", func(r chi.Router) {
		r.Get("/{id}", h.GetBundle)
		r.Get("/{id}/compliance-report", h.GetComplianceReport)
		r.Get("/{id}/stages", h.GetStages)
		r.Get("/{id}/approvals", approvalHandler.GetApprovalChain)
	})
}

// GetBundle retrieves a complete evidence bundle with all stages
// GET /api/metadata/evidence/bundles/:id
func (h *EvidenceBundleHandler) GetBundle(w http.ResponseWriter, r *http.Request) {
	bundleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid bundle ID"})
		return
	}

	bundle, err := h.service.GetBundle(r.Context(), bundleID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundle)
}

// GetComplianceReport generates a regulator-facing compliance report
// GET /api/metadata/evidence/bundles/:id/compliance-report
func (h *EvidenceBundleHandler) GetComplianceReport(w http.ResponseWriter, r *http.Request) {
	bundleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid bundle ID"})
		return
	}

	report, err := h.service.ExportComplianceReport(r.Context(), bundleID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Support both JSON and downloadable formats
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	case "download":
		// Set headers for file download
		w.Header().Set("Content-Disposition", "attachment; filename=compliance-report-"+bundleID.String()+".json")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid format. Supported: json, download"})
	}
}

// GetStages retrieves all stage evidence for a bundle
// GET /api/metadata/evidence/bundles/:id/stages
func (h *EvidenceBundleHandler) GetStages(w http.ResponseWriter, r *http.Request) {
	bundleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid bundle ID"})
		return
	}

	stages, err := h.service.GetStages(r.Context(), bundleID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stages)
}

// ApprovalHandler handles approval workflow API requests
type ApprovalHandler struct {
	service *metadata.ApprovalService
}

// NewApprovalHandler creates a new approval handler
func NewApprovalHandler(service *metadata.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{service: service}
}

// RegisterRoutes registers the routes for ApprovalHandler.
func (h *ApprovalHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/metadata/approvals", func(r chi.Router) {
		r.Get("/pending", h.GetPendingApprovals)
		r.Post("/{id}/approve", h.ApproveRequest)
		r.Post("/{id}/reject", h.RejectRequest)
	})
}

// GetPendingApprovals retrieves all pending approval requests for a role
// GET /api/metadata/approvals/pending?role=data_steward
func (h *ApprovalHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	requiredRole := r.URL.Query().Get("role")
	if requiredRole == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "role query parameter is required"})
		return
	}

	approvals, err := h.service.GetPendingApprovals(r.Context(), requiredRole)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approvals)
}

// ApproveRequest approves an upgrade deployment
// POST /api/metadata/approvals/:id/approve
func (h *ApprovalHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request ID"})
		return
	}

	var req struct {
		ApproverID    string `json:"approver_id" binding:"required"`
		Justification string `json:"justification" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	err = h.service.RecordDecision(r.Context(), requestID, req.ApproverID, "approved", req.Justification)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "approved",
		"message": "Approval recorded successfully",
	})
}

// RejectRequest rejects an upgrade deployment
// POST /api/metadata/approvals/:id/reject
func (h *ApprovalHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request ID"})
		return
	}

	var req struct {
		ApproverID    string `json:"approver_id" binding:"required"`
		Justification string `json:"justification" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	err = h.service.RecordDecision(r.Context(), requestID, req.ApproverID, "rejected", req.Justification)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "rejected",
		"message": "Rejection recorded successfully",
	})
}

// GetApprovalChain retrieves the complete approval history for a bundle
// GET /api/metadata/evidence/bundles/:id/approvals
func (h *ApprovalHandler) GetApprovalChain(w http.ResponseWriter, r *http.Request) {
	bundleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid bundle ID"})
		return
	}

	chain, err := h.service.GetApprovalChain(r.Context(), bundleID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chain)
}

// RegisterEvidenceBundleRoutes registers all evidence bundle API routes
func RegisterEvidenceBundleRoutes(r chi.Router, bundleService *metadata.EvidenceBundleService, approvalService *metadata.ApprovalService) {
	bundleHandler := NewEvidenceBundleHandler(bundleService)
	approvalHandler := NewApprovalHandler(approvalService)

	bundleHandler.RegisterRoutes(r, approvalHandler)
	approvalHandler.RegisterRoutes(r)
}
