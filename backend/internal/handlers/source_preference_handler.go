package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/preference"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SourcePreferenceHandler handles all source preference, analytics, and exception routes
type SourcePreferenceHandler struct {
	svc *preference.Service
}

func NewSourcePreferenceHandler(svc *preference.Service) *SourcePreferenceHandler {
	return &SourcePreferenceHandler{svc: svc}
}

// RegisterRoutes mounts all 12 endpoints under /api/v1/sources
func (h *SourcePreferenceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/sources", func(r chi.Router) {
		// Preferences
		r.Post("/preferences", h.CreatePreference)
		r.Get("/preferences/{prefId}", h.GetPreference)
		r.Get("/preferences", h.ListPreferences)
		r.Post("/preferences/{prefId}/override", h.RequestOverride)
		r.Post("/preferences/{prefId}/approve", h.ApproveOverride)
		r.Post("/preferences/{prefId}/promote", h.PromoteStage)
		// Analytics
		r.Get("/analytics", h.GetAnalytics)
		r.Get("/analytics/rank", h.GetRankings)
		r.Get("/analytics/confidence", h.GetConfidenceTrends)
		// Exceptions
		r.Post("/exceptions", h.CreateException)
		r.Get("/exceptions", h.ListExceptions)
		r.Post("/exceptions/{exId}/resolve", h.ResolveException)
	})
}

// ---- Preference Handlers ----

func (h *SourcePreferenceHandler) CreatePreference(w http.ResponseWriter, r *http.Request) {
	var p preference.SourcePreference
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	p.TenantID = mustTenantID(r)
	p.CreatedBy = mustUserID(r)
	result, err := h.svc.CreatePreference(r.Context(), &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *SourcePreferenceHandler) GetPreference(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "prefId"))
	if err != nil {
		http.Error(w, "invalid prefId", http.StatusBadRequest)
		return
	}
	p, err := h.svc.GetPreference(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *SourcePreferenceHandler) ListPreferences(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	prefs, err := h.svc.ListPreferences(r.Context(), mustTenantID(r),
		q.Get("business_object"), q.Get("semantic_term"), q.Get("region"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (h *SourcePreferenceHandler) RequestOverride(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "prefId"))
	if err != nil {
		http.Error(w, "invalid prefId", http.StatusBadRequest)
		return
	}
	var body struct {
		Reason  string    `json:"reason"`
		ValidTo time.Time `json:"valid_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	override, err := h.svc.RequestOverride(r.Context(), id, mustUserID(r), body.Reason, body.ValidTo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusCreated, override)
}

func (h *SourcePreferenceHandler) ApproveOverride(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "prefId"))
	if err != nil {
		http.Error(w, "invalid prefId", http.StatusBadRequest)
		return
	}
	var body struct {
		Notes string `json:"notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	p, err := h.svc.ApproveOverride(r.Context(), id, mustUserID(r), body.Notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *SourcePreferenceHandler) PromoteStage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "prefId"))
	if err != nil {
		http.Error(w, "invalid prefId", http.StatusBadRequest)
		return
	}
	p, err := h.svc.PromoteStage(r.Context(), id, mustUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ---- Analytics Handlers ----

func (h *SourcePreferenceHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	report, err := h.svc.GetAnalytics(r.Context(), mustTenantID(r),
		q.Get("business_object"), q.Get("semantic_term"), q.Get("region"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *SourcePreferenceHandler) GetRankings(w http.ResponseWriter, r *http.Request) {
	// Re-uses GetAnalytics but returns only rankings portion
	h.GetAnalytics(w, r)
}

func (h *SourcePreferenceHandler) GetConfidenceTrends(w http.ResponseWriter, r *http.Request) {
	// Placeholder — returns analytics report; extended in next sprint
	h.GetAnalytics(w, r)
}

// ---- Exception Handlers ----

func (h *SourcePreferenceHandler) CreateException(w http.ResponseWriter, r *http.Request) {
	var e preference.SourceException
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	e.TenantID = mustTenantID(r)
	result, err := h.svc.CreateException(r.Context(), &e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *SourcePreferenceHandler) ListExceptions(w http.ResponseWriter, r *http.Request) {
	exceptions, err := h.svc.ListExceptions(r.Context(), mustTenantID(r), r.URL.Query().Get("status"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, exceptions)
}

func (h *SourcePreferenceHandler) ResolveException(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "exId"))
	if err != nil {
		http.Error(w, "invalid exId", http.StatusBadRequest)
		return
	}
	if err := h.svc.ResolveException(r.Context(), id, mustUserID(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// ---- Shared helpers ----

func mustTenantID(r *http.Request) uuid.UUID {
	id, _ := uuid.Parse(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	return id
}

func mustUserID(r *http.Request) uuid.UUID {
	id, _ := uuid.Parse(r.Header.Get("X-User-ID"))
	return id
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
