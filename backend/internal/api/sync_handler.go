package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/google"
	"github.com/hondyman/semlayer/backend/internal/oauth"
	"github.com/hondyman/semlayer/backend/internal/repository"
	"github.com/hondyman/semlayer/backend/internal/sync"
	"github.com/sirupsen/logrus"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SyncHandler handles Google Calendar sync endpoints
type SyncHandler struct {
	oauthProvider *oauth.GoogleOAuth2Provider
	syncProcessor *sync.SyncProcessor
	syncRepo      *repository.GoogleSyncRepo
	logger        *logrus.Entry
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(
	oauthProvider *oauth.GoogleOAuth2Provider,
	syncProcessor *sync.SyncProcessor,
	syncRepo *repository.GoogleSyncRepo, // Added param
	logger *logrus.Entry,
) *SyncHandler {
	return &SyncHandler{
		oauthProvider: oauthProvider,
		syncProcessor: syncProcessor,
		syncRepo:      syncRepo,
		logger:        logger.WithField("component", "sync_handler"),
	}
}

// RegisterRoutes registers the sync routes
func (h *SyncHandler) RegisterRoutes(r chi.Router) {
	r.Route("/sync/google", func(r chi.Router) {
		r.Get("/calendars", h.ListCalendars)
		r.Post("/sync", h.StartSync)
		r.Get("/status/{syncID}", h.GetSyncStatus)
		r.Post("/cancel/{syncID}", h.CancelSync)
		r.Get("/active", h.ListActiveSyncs)
		r.Get("/events", h.ListEvents)
	})
}

// ListEvents lists synced events for the user
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

	// We need to access the repo directly or via processor.
	// SyncHandler has oauthProvider, syncProcessor, logger.
	// It doesn't have direct access to repo unless we add it or expose it via processor.
	// Best practice: Inject repo into handler.
	// I'll add syncRepo to SyncHandler struct.

	// Temporarily: use syncProcessor.syncRepo if accessible (it's unexported)
	// So I need to update SyncHandler struct to include syncRepo.
	// (Note: This comment block was from a previous step, implementation below handles this)

	events, err := h.syncRepo.ListSyncedEvents(ctx, tenantID, userID, start, end)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list synced events")
		http.Error(w, "Failed to list events", http.StatusInternalServerError)
		return
	}

	// Handle timezone conversion if requested
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

// ListCalendars lists the user's Google Calendars
func (h *SyncHandler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.Header.Get("X-User-ID") // Assuming auth middleware sets this or header
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

	client, err := google.NewCalendarClient(google.CalendarClientConfig{
		OAuthProvider: h.oauthProvider,
		UserID:        userID,
		TenantID:      tenantID,
		Logger:        h.logger,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to create calendar client")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	calendars, err := client.ListCalendars(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list calendars")
		http.Error(w, "Failed to list calendars", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calendars": calendars,
	})
}

// StartSyncRequest represents the request body for starting a sync
type StartSyncRequest struct {
	GoogleCalendarID   string    `json:"google_calendar_id"`
	InternalCalendarID string    `json:"internal_calendar_id"` // Optional target
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
}

// StartSync initiates a sync job
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

	// Default time range if not provided (last 30 days to next 90 days)
	if req.StartTime.IsZero() {
		req.StartTime = time.Now().AddDate(0, 0, -30)
	}
	if req.EndTime.IsZero() {
		req.EndTime = time.Now().AddDate(0, 0, 90)
	}

	status, err := h.syncProcessor.StartSync(
		ctx,
		userID,
		tenantID,
		req.GoogleCalendarID,
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

// GetSyncStatus returns the status of a specific sync job
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

// CancelSync cancels a running sync job
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

// ListActiveSyncs returns all active sync jobs for the user
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
