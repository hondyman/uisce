package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/wealth"

	"github.com/go-chi/chi/v5"
)

// ClientPortalHandlers contains handlers for client portal features
type ClientPortalHandlers struct {
	portalService *wealth.ClientPortalService
}

// NewClientPortalHandlers creates client portal handlers
func NewClientPortalHandlers(portalService *wealth.ClientPortalService) *ClientPortalHandlers {
	return &ClientPortalHandlers{
		portalService: portalService,
	}
}

// RegisterRoutes registers all client portal routes
func (h *ClientPortalHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/portal", func(r chi.Router) {
		// Messaging endpoints
		r.Post("/messages/send", h.SendMessage)
		r.Get("/messages/thread/{threadID}", h.GetMessageThread)

		// E-signature endpoints
		r.Post("/signatures/request", h.CreateSignatureRequest)
		r.Post("/signatures/{requestID}/sign", h.SignDocument)

		// Meeting endpoints
		r.Post("/meetings/schedule", h.ScheduleMeeting)
		r.Delete("/meetings/{meetingID}", h.CancelMeeting)

		// Activity feed
		r.Get("/activity/{familyID}", h.GetActivityFeed)

		// Notification preferences
		r.Put("/notifications/preferences", h.UpdateNotificationPreferences)
	})
}

// ==============================================================================
// MESSAGING HANDLERS
// ==============================================================================

type SendMessageRequest struct {
	FamilyID    string                     `json:"family_id"`
	SenderID    string                     `json:"sender_id"`
	SenderType  string                     `json:"sender_type"`
	RecipientID string                     `json:"recipient_id"`
	Subject     string                     `json:"subject"`
	Body        string                     `json:"body"`
	Priority    string                     `json:"priority"`
	Attachments []wealth.MessageAttachment `json:"attachments"`
}

func (h *ClientPortalHandlers) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	message, err := h.portalService.SendMessage(
		r.Context(),
		req.FamilyID,
		req.SenderID,
		req.SenderType,
		req.RecipientID,
		req.Subject,
		req.Body,
		req.Priority,
		req.Attachments,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

func (h *ClientPortalHandlers) GetMessageThread(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "threadID")
	userID := r.URL.Query().Get("user_id")

	messages, err := h.portalService.GetMessageThread(r.Context(), threadID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// ==============================================================================
// E-SIGNATURE HANDLERS
// ==============================================================================

type CreateSignatureRequestRequest struct {
	FamilyID       string          `json:"family_id"`
	DocumentName   string          `json:"document_name"`
	DocumentType   string          `json:"document_type"`
	DocumentURL    string          `json:"document_url"`
	Signers        []wealth.Signer `json:"signers"`
	ExpirationDays int             `json:"expiration_days"`
}

func (h *ClientPortalHandlers) CreateSignatureRequest(w http.ResponseWriter, r *http.Request) {
	var req CreateSignatureRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	request, err := h.portalService.CreateSignatureRequest(
		r.Context(),
		req.FamilyID,
		req.DocumentName,
		req.DocumentType,
		req.DocumentURL,
		req.Signers,
		req.ExpirationDays,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(request)
}

type SignDocumentRequest struct {
	SignerID      string `json:"signer_id"`
	SignatureData string `json:"signature_data"`
	IPAddress     string `json:"ip_address"`
}

func (h *ClientPortalHandlers) SignDocument(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "requestID")

	var req SignDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.portalService.SignDocument(
		r.Context(),
		requestID,
		req.SignerID,
		req.SignatureData,
		req.IPAddress,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// ==============================================================================
// MEETING HANDLERS
// ==============================================================================

type ScheduleMeetingRequest struct {
	FamilyID        string               `json:"family_id"`
	AdvisorID       string               `json:"advisor_id"`
	MeetingType     string               `json:"meeting_type"`
	Title           string               `json:"title"`
	ScheduledStart  time.Time            `json:"scheduled_start"`
	DurationMinutes int                  `json:"duration_minutes"`
	Participants    []wealth.Participant `json:"participants"`
	Agenda          []wealth.AgendaItem  `json:"agenda"`
}

func (h *ClientPortalHandlers) ScheduleMeeting(w http.ResponseWriter, r *http.Request) {
	var req ScheduleMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	meeting, err := h.portalService.ScheduleMeeting(
		r.Context(),
		req.FamilyID,
		req.AdvisorID,
		req.MeetingType,
		req.Title,
		req.ScheduledStart,
		req.DurationMinutes,
		req.Participants,
		req.Agenda,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meeting)
}

type CancelMeetingRequest struct {
	Reason string `json:"reason"`
}

func (h *ClientPortalHandlers) CancelMeeting(w http.ResponseWriter, r *http.Request) {
	meetingID := chi.URLParam(r, "meetingID")

	var req CancelMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.portalService.CancelMeeting(r.Context(), meetingID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// ==============================================================================
// ACTIVITY FEED HANDLERS
// ==============================================================================

func (h *ClientPortalHandlers) GetActivityFeed(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")
	limit := 50 // Default limit

	activities, err := h.portalService.GetActivityFeed(r.Context(), familyID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}

// ==============================================================================
// NOTIFICATION PREFERENCES HANDLERS
// ==============================================================================

func (h *ClientPortalHandlers) UpdateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	var prefs wealth.NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.portalService.UpdateNotificationPreferences(
		r.Context(),
		prefs.FamilyID,
		prefs.MemberID,
		prefs,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
