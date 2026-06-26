package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// NotificationAPIHandlers handles notification-related API endpoints
type NotificationAPIHandlers struct {
	notificationSvc *services.EngagementNotificationService
	campaignSvc     *services.NotificationCampaignService
}

// NewNotificationAPIHandlers creates new notification API handlers
func NewNotificationAPIHandlers(notificationSvc *services.EngagementNotificationService, campaignSvc *services.NotificationCampaignService) *NotificationAPIHandlers {
	return &NotificationAPIHandlers{
		notificationSvc: notificationSvc,
		campaignSvc:     campaignSvc,
	}
}

// GetUserNotifications retrieves notifications for a user
func (h *NotificationAPIHandlers) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	notifications, err := h.notificationSvc.GetUserNotifications(r.Context(), userID, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve notifications")
		return
	}

	respond(w, r, notifications, nil)
}

// CreateNotification creates a new notification
func (h *NotificationAPIHandlers) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var notification models.EngagementNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.notificationSvc.CreateNotification(r.Context(), &notification); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create notification")
		return
	}

	respond(w, r, notification, nil)
}

// SendNotification sends a notification immediately
func (h *NotificationAPIHandlers) SendNotification(w http.ResponseWriter, r *http.Request) {
	notificationID := chi.URLParam(r, "id")
	if notificationID == "" {
		respondWithError(w, http.StatusBadRequest, "Notification ID is required")
		return
	}

	if err := h.notificationSvc.SendNotification(r.Context(), notificationID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to send notification")
		return
	}

	respond(w, r, map[string]string{"status": "sent"}, nil)
}

// MarkNotificationAsRead marks a notification as read
func (h *NotificationAPIHandlers) MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	notificationID := chi.URLParam(r, "id")
	if notificationID == "" {
		respondWithError(w, http.StatusBadRequest, "Notification ID is required")
		return
	}

	// For now, we'll just return success since the middleware handles the actual marking
	respond(w, r, map[string]string{"status": "marked_as_read"}, nil)
}

// TrackEngagementEvent tracks user engagement with notifications
func (h *NotificationAPIHandlers) TrackEngagementEvent(w http.ResponseWriter, r *http.Request) {
	var event models.NotificationAnalytics
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	event.EventTimestamp = time.Now()

	if err := h.notificationSvc.TrackEngagementEvent(r.Context(), &event); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to track engagement event")
		return
	}

	respond(w, r, map[string]string{"status": "tracked"}, nil)
}

// GetUserPreferences retrieves user notification preferences
func (h *NotificationAPIHandlers) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	preferences, err := h.notificationSvc.GetUserPreferences(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve user preferences")
		return
	}

	respond(w, r, preferences, nil)
}

// UpdateUserPreferences updates user notification preferences
func (h *NotificationAPIHandlers) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	var preferences models.UserNotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&preferences); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	preferences.UserID = userID

	if err := h.notificationSvc.UpdateUserPreferences(r.Context(), &preferences); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update user preferences")
		return
	}

	respond(w, r, preferences, nil)
}

// CreateNotificationTemplate creates a new notification template
func (h *NotificationAPIHandlers) CreateNotificationTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.NotificationTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.notificationSvc.CreateNotificationTemplate(r.Context(), &template); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create notification template")
		return
	}

	respond(w, r, template, nil)
}

// GetEngagementAnalytics retrieves engagement analytics
func (h *NotificationAPIHandlers) GetEngagementAnalytics(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start")
	endDateStr := r.URL.Query().Get("end")

	if startDateStr == "" || endDateStr == "" {
		respondWithError(w, http.StatusBadRequest, "Start and end dates are required")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid start date format (use YYYY-MM-DD)")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid end date format (use YYYY-MM-DD)")
		return
	}

	analytics, err := h.notificationSvc.GetEngagementAnalytics(r.Context(), startDate, endDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve engagement analytics")
		return
	}

	respond(w, r, analytics, nil)
}

// CreateCampaign creates a new notification campaign
func (h *NotificationAPIHandlers) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var campaign models.NotificationCampaign
	if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.campaignSvc.CreateCampaign(r.Context(), &campaign); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create campaign")
		return
	}

	respond(w, r, campaign, nil)
}

// LaunchCampaign launches a notification campaign
func (h *NotificationAPIHandlers) LaunchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		respondWithError(w, http.StatusBadRequest, "Campaign ID is required")
		return
	}

	if err := h.campaignSvc.LaunchCampaign(r.Context(), campaignID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to launch campaign")
		return
	}

	respond(w, r, map[string]string{"status": "launched"}, nil)
}

// GetCampaign retrieves a campaign by ID
func (h *NotificationAPIHandlers) GetCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		respondWithError(w, http.StatusBadRequest, "Campaign ID is required")
		return
	}

	campaign, err := h.campaignSvc.GetCampaign(r.Context(), campaignID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve campaign")
		return
	}

	respond(w, r, campaign, nil)
}

// GetCampaignAnalytics retrieves analytics for a campaign
func (h *NotificationAPIHandlers) GetCampaignAnalytics(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		respondWithError(w, http.StatusBadRequest, "Campaign ID is required")
		return
	}

	analytics, err := h.campaignSvc.GetCampaignAnalytics(r.Context(), campaignID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve campaign analytics")
		return
	}

	respond(w, r, analytics, nil)
}

// GetActiveCampaigns retrieves all active campaigns
func (h *NotificationAPIHandlers) GetActiveCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.campaignSvc.GetActiveCampaigns(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve active campaigns")
		return
	}

	respond(w, r, campaigns, nil)
}

// PauseCampaign pauses a campaign
func (h *NotificationAPIHandlers) PauseCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		respondWithError(w, http.StatusBadRequest, "Campaign ID is required")
		return
	}

	if err := h.campaignSvc.PauseCampaign(r.Context(), campaignID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to pause campaign")
		return
	}

	respond(w, r, map[string]string{"status": "paused"}, nil)
}

// ResumeCampaign resumes a campaign
func (h *NotificationAPIHandlers) ResumeCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		respondWithError(w, http.StatusBadRequest, "Campaign ID is required")
		return
	}

	if err := h.campaignSvc.ResumeCampaign(r.Context(), campaignID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to resume campaign")
		return
	}

	respond(w, r, map[string]string{"status": "resumed"}, nil)
}

// StopCampaign stops a campaign
func (h *NotificationAPIHandlers) StopCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		respondWithError(w, http.StatusBadRequest, "Campaign ID is required")
		return
	}

	if err := h.campaignSvc.StopCampaign(r.Context(), campaignID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to stop campaign")
		return
	}

	respond(w, r, map[string]string{"status": "stopped"}, nil)
}

// Helper function to respond with error
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
