package calendar

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Handler handles HTTP requests for the Compliance Calendar.
type Handler struct {
	service *Service
}

// NewHandler creates a new compliance calendar Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers routes on the provided chi Router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/compliance/calendar", func(r chi.Router) {
		r.Get("/", h.HandleGetCalendarEvents)
	})
}

// HandleGetCalendarEvents returns compliance calendar events for the authenticated tenant.
func (h *Handler) HandleGetCalendarEvents(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		if c, err := jwtmiddleware.ValidateTokenFromRequest(r); err == nil && c != nil {
			claims = c
		}
	}
	if claims == nil {
		http.Error(w, "unauthorized: missing claims", http.StatusUnauthorized)
		return
	}

	callerTenantStr := claims.TenantID
	if callerTenantStr == "" {
		http.Error(w, "unauthorized: tenant claim required", http.StatusUnauthorized)
		return
	}

	callerTenantID, err := uuid.Parse(callerTenantStr)
	if err != nil {
		http.Error(w, "unauthorized: invalid caller tenant id", http.StatusUnauthorized)
		return
	}

	requestedTenantStr := r.URL.Query().Get("tenant_id")
	targetTenantID := callerTenantID
	if requestedTenantStr != "" {
		if err := jwtmiddleware.ValidateTenantAccess(claims, requestedTenantStr); err != nil {
			http.Error(w, "forbidden: cross-tenant access denied", http.StatusForbidden)
			return
		}
		parsedReqID, err := uuid.Parse(requestedTenantStr)
		if err != nil {
			http.Error(w, "invalid tenant_id parameter", http.StatusBadRequest)
			return
		}
		targetTenantID = parsedReqID
	}

	filter := CalendarFilter{
		TenantID:     targetTenantID,
		Jurisdiction: r.URL.Query().Get("jurisdiction"),
		Regulation:   r.URL.Query().Get("regulation"),
		DeadlineType: r.URL.Query().Get("deadline_type"),
		Status:       r.URL.Query().Get("status"),
		ReferenceDay: time.Now().UTC(),
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filter.FromDate = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			filter.ToDate = t
		}
	}
	if refStr := r.URL.Query().Get("reference_date"); refStr != "" {
		if t, err := time.Parse("2006-01-02", refStr); err == nil {
			filter.ReferenceDay = t
		}
	}

	events, err := h.service.GetCalendarEvents(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to get calendar events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        events,
		"total_count": len(events),
		"tenant_id":   targetTenantID,
	})
}
