package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/repository"
	"github.com/hondyman/uisce/backend/internal/sync"
	"github.com/sirupsen/logrus"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type SyncHandler struct {
	syncProcessor *sync.SyncProcessor
	syncRepo     *repository.CalendarSyncRepo
	logger       *logrus.Entry
}

func NewSyncHandler(
	syncProcessor *sync.SyncProcessor,
	syncRepo *repository.CalendarSyncRepo,
	logger *logrus.Entry,
) *SyncHandler {
	return &SyncHandler{
		syncProcessor: syncProcessor,
		syncRepo:      syncRepo,
		logger:        logger.WithField("component", "sync_handler"),
	}
}

func (h *SyncHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/sync/calendars", func(r chi.Router) {
		r.Get("/calendars", h.ListCalendars)
		r.Post("/sync", h.StartSync)
		r.Get("/status/{syncID}", h.GetSyncStatus)
		r.Post("/cancel/{syncID}", h.CancelSync)
		r.Get("/active", h.ListActiveSyncs)
		r.Get("/events", h.ListEvents)
	})
}

func (h *SyncHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	start := time.Now().AddDate(0, 0, -30)
	end := time.Now().AddDate(0, 0, 90)

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "google"
	}

	events, err := h.syncRepo.ListSyncedEvents(ctx, tenantID, userID, start, end)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list synced events")
		http.Error(w, "Failed to list events", http.StatusInternalServerError)
		return
	}

	tz := r.URL.Query().Get("timezone")
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err == nil {
			for i := range events {
				events[i].StartTime = events[i].StartTime.In(loc)
				events[i].EndTime = events[i].EndTime.In(loc)
			}
		} else {
			h.logger.Warnf("Invalid timezone requested: %s", tz)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
	})
}

func (h *SyncHandler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "google"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calendars": []interface{}{},
		"provider":  provider,
	})
}

type StartSyncRequest struct {
	ExternalCalendarID string    `json:"external_calendar_id"`
	InternalCalendarID string    `json:"internal_calendar_id"`
	Provider           string    `json:"provider"`
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
}

func (h *SyncHandler) StartSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}

	var req StartSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.StartTime.IsZero() {
		req.StartTime = time.Now().AddDate(0, 0, -30)
	}
	if req.EndTime.IsZero() {
		req.EndTime = time.Now().AddDate(0, 0, 90)
	}

	provider := repository.Provider(req.Provider)
	if provider == "" {
		provider = repository.ProviderGoogle
	}

	status, err := h.syncProcessor.StartSync(
		ctx,
		userID,
		tenantID,
		provider,
		req.ExternalCalendarID,
		req.InternalCalendarID,
		sync.TimeRange{Start: req.StartTime, End: req.EndTime},
	)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start sync")
		http.Error(w, fmt.Sprintf("Failed to start sync: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *SyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	syncID := chi.URLParam(r, "syncID")
	if syncID == "" {
		http.Error(w, "Sync ID required", http.StatusBadRequest)
		return
	}

	status, err := h.syncProcessor.GetSyncStatus(syncID)
	if err != nil {
		http.Error(w, "Sync job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *SyncHandler) CancelSync(w http.ResponseWriter, r *http.Request) {
	syncID := chi.URLParam(r, "syncID")
	if syncID == "" {
		http.Error(w, "Sync ID required", http.StatusBadRequest)
		return
	}

	if err := h.syncProcessor.CancelSync(syncID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to cancel sync: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func (h *SyncHandler) ListActiveSyncs(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}

	activeSyncs := h.syncProcessor.ListActiveSyncs(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_syncs": activeSyncs,
	})
}
