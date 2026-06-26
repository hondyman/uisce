package api

import (
	"encoding/json"
	"net/http"

	"calendar-service/internal/microsoft"
	"calendar-service/internal/oauth"
	"calendar-service/internal/sync"

	"github.com/sirupsen/logrus"
)

// MicrosoftHandler handles Microsoft-specific API requests
type MicrosoftHandler struct {
	oauthProvider *oauth.MicrosoftOAuth2Provider
	syncProcessor *sync.MicrosoftSyncProcessor
	logger        *logrus.Entry
}

// NewMicrosoftHandler creates a new Microsoft handler
func NewMicrosoftHandler(
	oauthProvider *oauth.MicrosoftOAuth2Provider,
	syncProcessor *sync.MicrosoftSyncProcessor,
	logger *logrus.Entry,
) *MicrosoftHandler {
	return &MicrosoftHandler{
		oauthProvider: oauthProvider,
		syncProcessor: syncProcessor,
		logger:        logger.WithField("component", "microsoft_handler"),
	}
}

// ListCalendars handles GET /api/v1/microsoft/calendars
func (h *MicrosoftHandler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing X-User-ID header")
		return
	}

	client, err := microsoft.NewGraphClient(microsoft.GraphClientConfig{
		OAuthProvider: h.oauthProvider,
		UserID:        userID,
		Logger:        h.logger,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to create Graph client")
		writeJSONError(w, http.StatusInternalServerError, "Failed to initialize Microsoft connection")
		return
	}

	calendars, err := client.ListCalendars(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list calendars")
		writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve calendars from Microsoft")
		return
	}

	// Transform to simple response
	type CalendarResp struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		IsPrimary bool   `json:"is_primary"`
	}

	resp := make([]CalendarResp, 0, len(calendars))
	for _, cal := range calendars {
		name := ""
		if cal.GetName() != nil {
			name = *cal.GetName()
		}
		id := ""
		if cal.GetId() != nil {
			id = *cal.GetId()
		}
		isPrimary := false
		if cal.GetIsDefaultCalendar() != nil {
			isPrimary = *cal.GetIsDefaultCalendar()
		}

		resp = append(resp, CalendarResp{
			ID:        id,
			Name:      name,
			IsPrimary: isPrimary,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// StartSync handles POST /api/v1/microsoft/sync
func (h *MicrosoftHandler) StartSync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.syncProcessor.SyncUserCalendars(r.Context(), req.UserID, req.TenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start Microsoft sync")
		writeJSONError(w, http.StatusInternalServerError, "Failed to initiate sync")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
