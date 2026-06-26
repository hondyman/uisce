package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/middleware"
	"calendar-service/internal/oauth"
	"calendar-service/internal/services"
	"calendar-service/internal/sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type SyncRequest struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	AuthCode string `json:"auth_code"`
}

type SyncResponse struct {
	SyncID  string `json:"sync_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type SyncHandler struct {
	processor    *sync.GoogleSyncProcessor
	oauth2       *oauth.GoogleOAuth2Provider
	msProcessor  *sync.MicrosoftSyncProcessor
	msOauth2     *oauth.MicrosoftOAuth2Provider
	auditService services.AuditService
	logger       *logrus.Entry
}

func NewSyncHandler(
	processor *sync.GoogleSyncProcessor,
	oauth2 *oauth.GoogleOAuth2Provider,
	msProcessor *sync.MicrosoftSyncProcessor,
	msOauth2 *oauth.MicrosoftOAuth2Provider,
	auditService services.AuditService,
	logger *logrus.Entry,
) *SyncHandler {
	return &SyncHandler{
		processor:    processor,
		oauth2:       oauth2,
		msProcessor:  msProcessor,
		msOauth2:     msOauth2,
		auditService: auditService,
		logger:       logger,
	}
}

// SyncGoogle handles POST /api/v1/sync/google
func (h *SyncHandler) SyncGoogle(w http.ResponseWriter, r *http.Request) {
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Exchange code for token
	token, err := h.oauth2.ExchangeCodeForToken(r.Context(), req.AuthCode)
	if err != nil {
		h.logger.WithError(err).Error("Failed to exchange code for token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to exchange code for token"})
		return
	}

	// Save token for user
	if err := h.oauth2.SaveUserToken(r.Context(), req.UserID, token); err != nil {
		h.logger.WithError(err).Error("Failed to save user token")
		writeJSONError(w, http.StatusInternalServerError, "Failed to save token")
		return
	}
	h.recordAudit(r.Context(), req.TenantID, req.UserID, "oauth_token_saved", map[string]interface{}{
		"provider":    "google",
		"auth_method": "code_exchange",
	})

	// Start sync
	result, err := h.processor.SyncUserCalendars(r.Context(), req.UserID, req.TenantID)
	if err != nil {
		h.logger.WithError(err).Error("Sync initiation failed")
		writeJSONError(w, http.StatusInternalServerError, "Sync initiation failed")
		return
	}
	h.recordAudit(r.Context(), req.TenantID, req.UserID, "sync_started", map[string]interface{}{
		"sync_id": result.ID,
		"status":  result.Status,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SyncResponse{
		SyncID:  result.ID,
		Status:  result.Status,
		Message: "Sync started successfully",
	})
}

// SyncMicrosoft handles POST /api/v1/sync/microsoft
func (h *SyncHandler) SyncMicrosoft(w http.ResponseWriter, r *http.Request) {
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// This assumes the auth code can be exchanged without a code verifier for simplicity,
	// but the UI should provide the verifier if PKCE is strictly required. Providing empty for now
	// as a placeholder. In a full implementation req should have CodeVerifier.
	token, err := h.msOauth2.ExchangeCodeForTokenWithPKCE(r.Context(), req.AuthCode, "placeholder_verifier")
	if err != nil {
		h.logger.WithError(err).Error("Failed to exchange microsoft code for token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to exchange code for token"})
		return
	}

	if err := h.msOauth2.SaveUserToken(r.Context(), req.UserID, token); err != nil {
		h.logger.WithError(err).Error("Failed to save microsoft user token")
		writeJSONError(w, http.StatusInternalServerError, "Failed to save token")
		return
	}
	h.recordAudit(r.Context(), req.TenantID, req.UserID, "oauth_token_saved", map[string]interface{}{
		"provider":    "microsoft",
		"auth_method": "code_exchange",
	})

	result, err := h.msProcessor.SyncUserCalendars(r.Context(), req.UserID, req.TenantID)
	if err != nil {
		h.logger.WithError(err).Error("Microsoft Sync initiation failed")
		writeJSONError(w, http.StatusInternalServerError, "Sync initiation failed")
		return
	}
	h.recordAudit(r.Context(), req.TenantID, req.UserID, "sync_started", map[string]interface{}{
		"sync_id":  result.ID,
		"status":   result.Status,
		"provider": "microsoft",
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SyncResponse{
		SyncID:  result.ID,
		Status:  result.Status,
		Message: "Microsoft Sync started successfully",
	})
}

// PushEventToGoogle - API Endpoint for manual upward sync trigger (Fallback/Testing)
func (h *SyncHandler) PushEventToGoogle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	eventID := vars["id"]

	userID, err := middleware.ExtractUserIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = h.processor.PushEvent(ctx, userID, eventID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to push event")
		http.Error(w, "Failed to push event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
}

// SyncAllToGoogle syncs all user events to Google Calendar
// @Summary Sync all events to Google
// @Tags sync
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sync/google/sync-all [post]
func (h *SyncHandler) SyncAllToGoogle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := middleware.ExtractUserIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Start async sync
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := h.processor.SyncAllToGoogle(bgCtx, userID); err != nil {
			h.logger.WithError(err).Error("Failed to sync all events to Google")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "started",
		"message": "Sync started in background",
	})
}

// GetSyncDirection returns current sync direction settings
// @Summary Get sync direction
// @Tags sync
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sync/google/direction [get]
func (h *SyncHandler) GetSyncDirection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"google_to_internal": true,
		"internal_to_google": true,
		"bi_directional":     true,
	})
}

// PKCEAuthResponse is returned when generating an OAuth URL with PKCE.
type PKCEAuthResponse struct {
	AuthURL   string `json:"auth_url"`
	State     string `json:"state"`
	ExpiresIn int64  `json:"expires_in_seconds"`
}

// PKCECallbackResponse is returned after handling the PKCE OAuth callback.
type PKCECallbackResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetPKCEAuthURL returns a PKCE-enabled authorization URL.
func (h *SyncHandler) GetPKCEAuthURL(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	tenantID := r.URL.Query().Get("tenant_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	pkce, err := oauth.GeneratePKCEParams()
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate PKCE parameters")
		writeJSONError(w, http.StatusInternalServerError, "Failed to generate PKCE parameters")
		return
	}

	state := uuid.NewString()
	if err := h.oauth2.StorePKCEState(r.Context(), state, &oauth.PKCEState{
		UserID:   userID,
		TenantID: tenantID,
		Params:   *pkce,
	}); err != nil {
		h.logger.WithError(err).Error("Failed to store PKCE state")
		writeJSONError(w, http.StatusInternalServerError, "Failed to store PKCE state")
		return
	}
	h.recordAudit(r.Context(), tenantID, userID, "pkce_state_created", map[string]interface{}{
		"state":     state,
		"expires":   h.oauth2.PKCEStateTTL().Seconds(),
		"pkce_mode": pkce.Method,
	})

	authURL := h.oauth2.GetAuthURLWithPKCE(state, pkce)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PKCEAuthResponse{
		AuthURL:   authURL,
		State:     state,
		ExpiresIn: int64(h.oauth2.PKCEStateTTL().Seconds()),
	})
}

// PKCECallback handles the Google OAuth callback that uses PKCE verification.
func (h *SyncHandler) PKCECallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeJSONError(w, http.StatusBadRequest, "code and state query parameters are required")
		return
	}

	pkceState, err := h.oauth2.RetrievePKCEState(r.Context(), state)
	if err != nil {
		h.logger.WithError(err).Error("PKCE state retrieval failed")
		writeJSONError(w, http.StatusBadRequest, "Invalid or expired state")
		return
	}

	token, err := h.oauth2.ExchangeCodeForTokenWithPKCE(r.Context(), code, pkceState.Params.Verifier)
	if err != nil {
		h.logger.WithError(err).Error("Failed to exchange PKCE code for token")
		writeJSONError(w, http.StatusInternalServerError, "Failed to exchange authorization code")
		return
	}

	if err := h.oauth2.SaveUserToken(r.Context(), pkceState.UserID, token); err != nil {
		h.logger.WithError(err).Error("Failed to save PKCE token")
		writeJSONError(w, http.StatusInternalServerError, "Failed to persist OAuth token")
		return
	}
	h.recordAudit(r.Context(), pkceState.TenantID, pkceState.UserID, "oauth_pkce_completed", map[string]interface{}{
		"state": state,
	})

	if h.processor != nil {
		if _, err := h.processor.SyncUserCalendars(r.Context(), pkceState.UserID, pkceState.TenantID); err != nil {
			h.logger.WithError(err).Error("Sync initiation failed after PKCE callback")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PKCECallbackResponse{
		Success: true,
		Message: "OAuth flow completed successfully",
	})
}

// GetMicrosoftPKCEAuthURL returns a PKCE-enabled authorization URL for Microsoft.
func (h *SyncHandler) GetMicrosoftPKCEAuthURL(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	tenantID := r.URL.Query().Get("tenant_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	pkce, err := oauth.GeneratePKCEParams()
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate Microsoft PKCE parameters")
		writeJSONError(w, http.StatusInternalServerError, "Failed to generate PKCE parameters")
		return
	}

	state := uuid.NewString()
	if err := h.msOauth2.StorePKCEState(r.Context(), state, &oauth.PKCEState{
		UserID:   userID,
		TenantID: tenantID,
		Params:   *pkce,
	}); err != nil {
		h.logger.WithError(err).Error("Failed to store Microsoft PKCE state")
		writeJSONError(w, http.StatusInternalServerError, "Failed to store PKCE state")
		return
	}
	h.recordAudit(r.Context(), tenantID, userID, "pkce_state_created", map[string]interface{}{
		"provider":  "microsoft",
		"state":     state,
		"expires":   h.msOauth2.PKCEStateTTL().Seconds(),
		"pkce_mode": pkce.Method,
	})

	authURL := h.msOauth2.GetAuthURLWithPKCE(state, pkce)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PKCEAuthResponse{
		AuthURL:   authURL,
		State:     state,
		ExpiresIn: int64(h.msOauth2.PKCEStateTTL().Seconds()),
	})
}

// MicrosoftPKCECallback handles the Microsoft OAuth callback that uses PKCE verification.
func (h *SyncHandler) MicrosoftPKCECallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeJSONError(w, http.StatusBadRequest, "code and state query parameters are required")
		return
	}

	pkceState, err := h.msOauth2.RetrievePKCEState(r.Context(), state)
	if err != nil {
		h.logger.WithError(err).Error("Microsoft PKCE state retrieval failed")
		writeJSONError(w, http.StatusBadRequest, "Invalid or expired state")
		return
	}

	token, err := h.msOauth2.ExchangeCodeForTokenWithPKCE(r.Context(), code, pkceState.Params.Verifier)
	if err != nil {
		h.logger.WithError(err).Error("Failed to exchange Microsoft PKCE code for token")
		writeJSONError(w, http.StatusInternalServerError, "Failed to exchange authorization code")
		return
	}

	if err := h.msOauth2.SaveUserToken(r.Context(), pkceState.UserID, token); err != nil {
		h.logger.WithError(err).Error("Failed to save Microsoft PKCE token")
		writeJSONError(w, http.StatusInternalServerError, "Failed to persist OAuth token")
		return
	}
	h.recordAudit(r.Context(), pkceState.TenantID, pkceState.UserID, "oauth_pkce_completed", map[string]interface{}{
		"provider": "microsoft",
		"state":    state,
	})

	if h.msProcessor != nil {
		if _, err := h.msProcessor.SyncUserCalendars(r.Context(), pkceState.UserID, pkceState.TenantID); err != nil {
			h.logger.WithError(err).Error("Microsoft Sync initiation failed after PKCE callback")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PKCECallbackResponse{
		Success: true,
		Message: "Microsoft OAuth flow completed successfully",
	})
}

// GetStatus handles GET /api/v1/sync/status
func (h *SyncHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing user_id parameter")
		return
	}

	status := h.processor.GetSyncStatus(userID)
	if status == nil {
		writeJSONError(w, http.StatusNotFound, "Sync not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// CancelSync handles GET /api/v1/sync/cancel
func (h *SyncHandler) CancelSync(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing user_id parameter")
		return
	}

	if err := h.processor.CancelSync(userID); err != nil {
		h.logger.WithError(err).Error("Failed to cancel sync")
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Sync cancelled"})
}

// ListActiveSyncs handles GET /api/v1/sync/active
func (h *SyncHandler) ListActiveSyncs(w http.ResponseWriter, r *http.Request) {
	syncs := h.processor.ListActiveSyncs()
	if syncs == nil {
		syncs = []*sync.SyncStatus{} // Return empty array instead of null
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(syncs)
}

func (h *SyncHandler) recordAudit(ctx context.Context, tenantID, userID, action string, values map[string]interface{}) {
	if h.auditService == nil || tenantID == "" || userID == "" || action == "" {
		return
	}
	entry := services.AuditEntry{
		TenantID:   tenantID,
		EntityType: "google_sync",
		EntityID:   userID,
		Action:     action,
		NewValues:  values,
		ChangedBy:  userID,
	}
	if err := h.auditService.Record(ctx, entry); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"user_id":   userID,
			"action":    action,
		}).Warn("Failed to record sync audit entry")
	}
}
