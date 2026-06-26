package clientportal

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Handler handles client portal HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new client portal handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetPreferences retrieves portal preferences
// GET /api/portal/preferences
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	clientIDStr := r.Header.Get("X-Client-ID")
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid client ID"})
		return
	}

	prefs, err := h.service.GetPreferences(r.Context(), clientID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

// UpdatePreferences updates portal preferences
// PUT /api/portal/preferences
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	clientIDStr := r.Header.Get("X-Client-ID")
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid client ID"})
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	err = h.service.UpdatePreferences(r.Context(), clientID, updates)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Preferences updated successfully"})
}

// TrackAnalytics tracks a portal event
// POST /api/portal/analytics
func (h *Handler) TrackAnalytics(w http.ResponseWriter, r *http.Request) {
	clientIDStr := r.Header.Get("X-Client-ID")
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid client ID"})
		return
	}

	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid tenant ID"})
		return
	}

	var event AnalyticsEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	event.ClientID = clientID
	event.TenantID = tenantID

	// Extract device info from headers
	deviceType := r.Header.Get("X-Device-Type")
	if deviceType != "" {
		event.DeviceType = &deviceType
	}

	ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	event.IPAddress = &ipAddress

	err = h.service.TrackEvent(r.Context(), &event)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Event tracked"})
}

// GetMetrics retrieves engagement metrics
// GET /api/portal/metrics?days=30
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	clientIDStr := r.Header.Get("X-Client-ID")
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid client ID"})
		return
	}

	days := r.URL.Query().Get("days")
	if days == "" {
		days = "30"
	}
	var daysInt int
	if _, err := fmt.Sscanf(days, "%d", &daysInt); err != nil {
		daysInt = 30
	}

	metrics, err := h.service.GetEngagementMetrics(r.Context(), clientID, daysInt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// RegisterRoutes registers client portal routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/portal", func(r chi.Router) {
		r.Get("/preferences", h.GetPreferences)
		r.Put("/preferences", h.UpdatePreferences)
		r.Post("/analytics", h.TrackAnalytics)
		r.Get("/metrics", h.GetMetrics)
	})
}
