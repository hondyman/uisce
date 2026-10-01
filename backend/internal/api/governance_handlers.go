package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/handlers"
	scheduler "github.com/hondyman/uisce/backend/internal/scheduler_intelligence"
)

// GovernanceHandler handles scheduler governance requests
type GovernanceHandler struct {
	governanceSvc *scheduler.GovernanceService
	auditSvc      *scheduler.AuditTrailService
	securityDeps  handlers.SecurityContextDeps
}

// NewGovernanceHandler creates a new governance handler
func NewGovernanceHandler(govSvc *scheduler.GovernanceService, auditSvc *scheduler.AuditTrailService, securityDeps handlers.SecurityContextDeps) *GovernanceHandler {
	return &GovernanceHandler{
		governanceSvc: govSvc,
		auditSvc:      auditSvc,
		securityDeps:  securityDeps,
	}
}

// RegisterRoutes registers governance routes.
// Slice 5 follow-up: S2 governance HTTP is retired with the satellite tables.
// Maker–checker for schedules lives on the one scheduler / shared governance later.
func (h *GovernanceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/scheduler/governance", func(r chi.Router) {
		r.Get("/changesets", h.retiredGovernance)
		r.Post("/changesets", h.retiredGovernance)
		r.Get("/changesets/{id}", h.retiredGovernance)
		r.Post("/changesets/{id}/approve", h.retiredGovernance)
		r.Post("/changesets/{id}/reject", h.retiredGovernance)
		r.Post("/changesets/{id}/apply", h.retiredGovernance)
		r.Post("/changesets/{id}/rollback", h.retiredGovernance)

		r.Get("/policies", h.retiredGovernance)
		r.Post("/policies", h.retiredGovernance)
		r.Get("/policies/{id}", h.retiredGovernance)
		r.Patch("/policies/{id}", h.retiredGovernance)
		r.Delete("/policies/{id}", h.retiredGovernance)

		r.Get("/audit", h.retiredGovernance)
		r.Get("/audit/entity/{type}/{id}", h.retiredGovernance)
		r.Get("/audit/stats", h.retiredGovernance)
	})
}

func (h *GovernanceHandler) retiredGovernance(w http.ResponseWriter, r *http.Request) {
	if _, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps); err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Link", "</api/schedules>; rel=\"successor-version\"")
	w.WriteHeader(http.StatusGone)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "scheduler_governance_retired",
		"message": "Scheduler Intelligence governance is closed. Use /api/schedules for timetable schedules (including workflow and job_dag).",
	})
}

// ListChangeSets returns change sets for a tenant
func (h *GovernanceHandler) ListChangeSets(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}
	tenantID, _ := uuid.Parse(secCtx.TenantID)

	var status *scheduler.ChangeSetStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := scheduler.ChangeSetStatus(s)
		status = &st
	}

	changeSets, err := h.governanceSvc.ListChangeSets(r.Context(), tenantID, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(changeSets)
}

// CreateChangeSet creates a new change set
func (h *GovernanceHandler) CreateChangeSet(w http.ResponseWriter, r *http.Request) {
	var cs scheduler.SchedulerChangeSet
	if err := json.NewDecoder(r.Body).Decode(&cs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.governanceSvc.CreateChangeSet(r.Context(), &cs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cs)
}

// GetChangeSet returns a single change set
func (h *GovernanceHandler) GetChangeSet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	cs, err := h.governanceSvc.GetChangeSet(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cs == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cs)
}

// ApproveChangeSet approves a change set
func (h *GovernanceHandler) ApproveChangeSet(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	var req struct {
		ApproverID string `json:"approver_id"`
		Role       string `json:"role"`
		Comment    string `json:"comment"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.governanceSvc.ApproveChangeSet(r.Context(), id, req.ApproverID, req.Role, req.Comment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// RejectChangeSet rejects a change set
func (h *GovernanceHandler) RejectChangeSet(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	var req struct {
		ApproverID string `json:"approver_id"`
		Role       string `json:"role"`
		Reason     string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.governanceSvc.RejectChangeSet(r.Context(), id, req.ApproverID, req.Role, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

// ApplyChangeSet applies an approved change set
func (h *GovernanceHandler) ApplyChangeSet(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	if err := h.governanceSvc.ApplyChangeSet(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "applied"})
}

// RollbackChangeSet rolls back an applied change
func (h *GovernanceHandler) RollbackChangeSet(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.governanceSvc.RollbackChangeSet(r.Context(), id, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rolled_back"})
}

// ListPolicies returns governance policies
func (h *GovernanceHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	// Would query policies from database
	policies := []scheduler.GovernancePolicy{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

// CreatePolicy creates a new governance policy
func (h *GovernanceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var policy scheduler.GovernancePolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Would save to database
	policy.ID = uuid.New()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(policy)
}

// GetPolicy returns a single policy
func (h *GovernanceHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	// Would fetch from database
	http.Error(w, "Not found", http.StatusNotFound)
}

// UpdatePolicy updates a policy
func (h *GovernanceHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	// Would update in database
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// DeletePolicy deletes a policy
func (h *GovernanceHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	// Would delete from database
	w.WriteHeader(http.StatusNoContent)
}

// GetAuditHistory returns audit records
func (h *GovernanceHandler) GetAuditHistory(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}
	tenantID, _ := uuid.Parse(secCtx.TenantID)
	limit := 100

	records, err := h.auditSvc.GetRecentActivityForTenant(r.Context(), tenantID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// GetEntityTimeline returns audit history for an entity
func (h *GovernanceHandler) GetEntityTimeline(w http.ResponseWriter, r *http.Request) {
	targetType := chi.URLParam(r, "type")
	targetID, _ := uuid.Parse(chi.URLParam(r, "id"))

	records, err := h.auditSvc.GetEntityTimeline(r.Context(), targetType, targetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// GetAuditStats returns aggregate audit statistics
func (h *GovernanceHandler) GetAuditStats(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}
	tenantID, _ := uuid.Parse(secCtx.TenantID)

	// Would parse from/to from query params
	stats, err := h.auditSvc.GetAuditStats(r.Context(), tenantID, time.Time{}, time.Time{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
