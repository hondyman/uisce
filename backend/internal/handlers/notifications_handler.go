package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hondyman/uisce/backend/internal/personalization"
	"github.com/hondyman/uisce/backend/internal/scheduler_intelligence"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type NotificationsHandler struct {
	personalizationSvc *personalization.Service
	schedulerSvc      *scheduler_intelligence.Service
}

func NewNotificationsHandler(personalizationSvc *personalization.Service, schedulerSvc *scheduler_intelligence.Service) *NotificationsHandler {
	return &NotificationsHandler{
		personalizationSvc: personalizationSvc,
		schedulerSvc:      schedulerSvc,
	}
}

func (h *NotificationsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/personalization/notifications", h.GetNotifications)
}

func (h *NotificationsHandler) RegisterMuxRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/personalization/notifications", h.GetNotifications).Methods("GET")
}

func (h *NotificationsHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant", http.StatusBadRequest)
		return
	}

	activeDrifts := 0
	if h.schedulerSvc != nil {
		count, err := h.schedulerSvc.CountPendingDriftSuggestions(ctx, tenantID)
		if err == nil {
			activeDrifts = count
		}
	}

	profile, _ := h.personalizationSvc.GetProfile(ctx, claims.TenantID, claims.UserID)
	pinnedBOs := []string{}
	if profile != nil {
		pinnedBOs = profile.PinnedBOKeys
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_drifts":  activeDrifts,
		"pinned_bo_keys": pinnedBOs,
	})
}
