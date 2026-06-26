package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/sirupsen/logrus"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// InternalEventHandler handles API requests for internal events
type InternalEventHandler struct {
	service *services.InternalEventService
	logger  *logrus.Entry
}

// NewInternalEventHandler creates a new handler
func NewInternalEventHandler(service *services.InternalEventService, logger *logrus.Entry) *InternalEventHandler {
	return &InternalEventHandler{
		service: service,
		logger:  logger.WithField("component", "internal_event_handler"),
	}
}

// RegisterRoutes registers routes
func (h *InternalEventHandler) RegisterRoutes(r chi.Router) {
	r.Route("/internal/events", func(r chi.Router) {
		r.Post("/", h.CreateEvent)
		// r.Put("/{id}", h.UpdateEvent)
		// r.Delete("/{id}", h.DeleteEvent)
	})
}

// CreateEvent creates a new internal event
func (h *InternalEventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Location    string    `json:"location"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
		IsAllDay    bool      `json:"is_all_day"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Mock Tenant/User from context (or header)
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	userIDStr := r.Header.Get("X-User-ID")

	if tenantIDStr == "" || userIDStr == "" {
		http.Error(w, "Missing X-Tenant-ID or X-User-ID", http.StatusBadRequest)
		return
	}

	tenantID, _ := uuid.Parse(tenantIDStr)
	userID, _ := uuid.Parse(userIDStr)

	// Determine logic for times (simplified)

	event := &models.InternalEvent{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Title:     req.Title,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		IsAllDay:  req.IsAllDay,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if req.Description != "" {
		event.Description = &req.Description
	}
	if req.Location != "" {
		event.Location = &req.Location
	}

	if err := h.service.CreateEvent(r.Context(), event); err != nil {
		h.logger.WithError(err).Error("Failed to create internal event")
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}
