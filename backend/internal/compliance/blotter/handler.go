package blotter

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Handler handles HTTP and WebSocket requests for compliance decision blotter.
type Handler struct {
	service *Service
	hub     *WebSocketHub
}

// NewHandler creates a new blotter Handler.
func NewHandler(service *Service, hub *WebSocketHub) *Handler {
	return &Handler{
		service: service,
		hub:     hub,
	}
}

// RegisterRoutes registers routes on the provided chi Router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/compliance/evaluations", func(r chi.Router) {
		r.Get("/", h.HandleListEvaluations)
		r.Get("/ws", h.HandleWebSocket)
		r.Get("/{lineage_id}", h.HandleGetEvidenceBundleByLineage)
		r.Get("/id/{eval_id}", h.HandleGetEvidenceBundleByID)
	})
}

// HandleListEvaluations returns paginated compliance evaluations with query filters.
func (h *Handler) HandleListEvaluations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant id: "+err.Error(), http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	filter := ListFilter{
		TenantID:    tenantID,
		RuleCode:    q.Get("rule_code"),
		ActionTaken: q.Get("action_taken"),
	}

	if pStr := q.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			filter.Page = p
		}
	}
	if psStr := q.Get("page_size"); psStr != "" {
		if ps, err := strconv.Atoi(psStr); err == nil {
			filter.PageSize = ps
		}
	}

	if orderIDStr := q.Get("order_id"); orderIDStr != "" {
		if u, err := uuid.Parse(orderIDStr); err == nil {
			filter.OrderID = &u
		}
	}

	if lineageIDStr := q.Get("lineage_id"); lineageIDStr != "" {
		if u, err := uuid.Parse(lineageIDStr); err == nil {
			filter.LineageID = &u
		}
	}

	if passedStr := q.Get("passed"); passedStr != "" {
		p := (passedStr == "true" || passedStr == "1")
		filter.Passed = &p
	}

	if fromStr := q.Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &t
		}
	}

	if toStr := q.Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &t
		}
	}

	result, err := h.service.ListEvaluations(ctx, filter)
	if err != nil {
		http.Error(w, "failed to list evaluations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HandleGetEvidenceBundleByLineage returns full explainability and cryptographic evidence for a lineage ID.
func (h *Handler) HandleGetEvidenceBundleByLineage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant id: "+err.Error(), http.StatusUnauthorized)
		return
	}

	lineageIDStr := chi.URLParam(r, "lineage_id")
	lineageID, err := uuid.Parse(lineageIDStr)
	if err != nil {
		http.Error(w, "invalid lineage_id format", http.StatusBadRequest)
		return
	}

	bundle, err := h.service.GetEvidenceBundleByLineageID(ctx, tenantID, lineageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bundle)
}

// HandleGetEvidenceBundleByID returns evidence bundle for a specific evaluation event ID.
func (h *Handler) HandleGetEvidenceBundleByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant id: "+err.Error(), http.StatusUnauthorized)
		return
	}

	evalIDStr := chi.URLParam(r, "eval_id")
	evalID, err := uuid.Parse(evalIDStr)
	if err != nil {
		http.Error(w, "invalid eval_id format", http.StatusBadRequest)
		return
	}

	bundle, err := h.service.GetEvidenceBundleByID(ctx, tenantID, evalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bundle)
}

// HandleWebSocket upgrades connection and streams live evaluations to connected workstation displays.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant id: "+err.Error(), http.StatusUnauthorized)
		return
	}

	h.hub.ServeWebSocket(w, r, tenantID)
}

func (h *Handler) getTenantID(r *http.Request) (uuid.UUID, error) {
	// Try JWT claims first
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		if claims.TenantID != "" {
			if u, err := uuid.Parse(claims.TenantID); err == nil {
				return u, nil
			}
		}
	}

	// Fallback to X-Tenant-ID header (for local testing / internal services)
	tenantIDHeader := r.Header.Get("X-Tenant-ID")
	if tenantIDHeader != "" {
		return uuid.Parse(tenantIDHeader)
	}

	// Fallback to query parameter (e.g. for WebSocket connections from browser)
	if tenantIDQuery := r.URL.Query().Get("tenant_id"); tenantIDQuery != "" {
		return uuid.Parse(tenantIDQuery)
	}

	return uuid.Nil, http.ErrNoCookie
}
