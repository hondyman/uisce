package regulatory

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Handler handles HTTP requests for regulatory change management and steward triage/review.
type Handler struct {
	service *Service
}

// NewHandler creates a new regulatory Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers routes on the provided chi Router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/compliance/regulatory/cases", func(r chi.Router) {
		r.Get("/", h.HandleListCases)
		r.Get("/unaddressed", h.HandleListUnaddressedCases)
		r.Get("/{id}", h.HandleGetCase)
		r.Get("/{id}/steward-review", h.HandleGetStewardReview)
		r.Post("/intake", h.HandleIntakeCase)
		r.Post("/{id}/triage", h.HandleTriageCase)
		r.Post("/{id}/approve", h.HandleApproveCase)
		r.Post("/{id}/reject", h.HandleRejectCase)
		r.Post("/{id}/publish", h.HandlePublishRelease)
	})
}

func (h *Handler) getActor(r *http.Request) string {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims != nil && claims.Email != "" {
		return claims.Email
	}
	if userHdr := r.Header.Get("X-User-Email"); userHdr != "" {
		return userHdr
	}
	return "compliance_officer"
}

// HandleListUnaddressedCases returns operational SLA queue cases from compliance.v_unaddressed_regulatory_cases
func (h *Handler) HandleListUnaddressedCases(w http.ResponseWriter, r *http.Request) {
	cases, err := h.service.GetUnaddressedCases(r.Context())
	if err != nil {
		http.Error(w, "failed to get unaddressed cases: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        cases,
		"total_count": len(cases),
	})
}

// HandleListCases returns all cases with filtering
func (h *Handler) HandleListCases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	source := r.URL.Query().Get("source")

	query := `
		SELECT id, case_code, source, source_reference, title, description,
		       affected_rule_ids, classification, triage_notes, triaged_by, triaged_at,
		       status, is_escalated, escalation_count, escalated_at, last_escalated_at,
		       published_rule_versions, due_at, created_by, created_at, updated_at
		FROM compliance.regulatory_change_case
		WHERE 1=1
	`
	var args []any
	argIdx := 1
	if status != "" {
		query += " AND status = $" + string(rune('0'+argIdx))
		args = append(args, status)
		argIdx++
	}
	if source != "" {
		query += " AND source = $" + string(rune('0'+argIdx))
		args = append(args, source)
		argIdx++
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := h.service.db.QueryContext(ctx, query, args...)
	if err != nil {
		http.Error(w, "failed to query cases: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cases []*RegulatoryChangeCase
	for rows.Next() {
		c := &RegulatoryChangeCase{}
		var affectedRules []string
		var publishedJSON []byte
		err := rows.Scan(
			&c.ID, &c.CaseCode, &c.Source, &c.SourceReference, &c.Title, &c.Description,
			&c.AffectedRuleIDs, &c.Classification, &c.TriageNotes, &c.TriagedBy, &c.TriagedAt,
			&c.Status, &c.IsEscalated, &c.EscalationCount, &c.EscalatedAt, &c.LastEscalatedAt,
			&publishedJSON, &c.DueAt, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		)
		if err == nil {
			cases = append(cases, c)
		}
		_ = affectedRules
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        cases,
		"total_count": len(cases),
	})
}

// HandleGetCase returns single case with drafts and event audit trail
func (h *Handler) HandleGetCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	caseID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}

	c, err := h.service.GetCase(r.Context(), caseID)
	if err != nil {
		http.Error(w, "case not found: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

// HandleGetStewardReview returns the interactive visual diff & threshold comparison model
func (h *Handler) HandleGetStewardReview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	caseID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}

	reviewView, err := h.service.GetStewardReviewView(r.Context(), caseID)
	if err != nil {
		http.Error(w, "failed to get steward review view: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reviewView)
}

// HandleIntakeCase creates a new regulatory change case
func (h *Handler) HandleIntakeCase(w http.ResponseWriter, r *http.Request) {
	var req IntakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = h.getActor(r)
	}

	c, err := h.service.CreateCase(r.Context(), req)
	if err != nil {
		http.Error(w, "failed to create case: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}

// HandleTriageCase submits triage classification
func (h *Handler) HandleTriageCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	caseID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}

	var req TriageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}
	req.CaseID = caseID
	if req.TriagedBy == "" {
		req.TriagedBy = h.getActor(r)
	}

	err = h.service.TriageCase(r.Context(), req)
	if err != nil {
		http.Error(w, "failed to triage case: "+err.Error(), http.StatusBadRequest)
		return
	}

	c, err := h.service.GetCase(r.Context(), caseID)
	if err != nil {
		http.Error(w, "failed to reload case: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

// HandleApproveCase approves a draft rule within the case
func (h *Handler) HandleApproveCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	caseID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}

	var payload struct {
		ApproverNotes string `json:"approver_notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)

	approver := h.getActor(r)
	err = h.service.ApproveCase(r.Context(), caseID, approver, payload.ApproverNotes)
	if err != nil {
		http.Error(w, "failed to approve case draft: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "APPROVED", "case_id": caseID})
}

// HandleRejectCase marks case as rejected with reason
func (h *Handler) HandleRejectCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	caseID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if payload.Reason == "" {
		payload.Reason = "Rejected by steward"
	}

	actor := h.getActor(r)
	err = h.service.RejectCase(r.Context(), caseID, actor, payload.Reason)
	if err != nil {
		http.Error(w, "failed to reject case: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "REJECTED", "case_id": caseID})
}

// HandlePublishRelease releases approved drafts to production version snapshots
func (h *Handler) HandlePublishRelease(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	caseID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}

	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}
	req.CaseID = caseID
	if req.StewardID == "" {
		req.StewardID = h.getActor(r)
	}

	c, err := h.service.PublishRelease(r.Context(), req)
	if err != nil {
		http.Error(w, "failed to publish release: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}
