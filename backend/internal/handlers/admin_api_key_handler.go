package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
)

type AdminAPIKeyHandler struct {
	Store services.APIKeyStore
}

type adminAPIKeyRequest struct {
	UserID      string   `json:"user_id"`
	TenantID    string   `json:"tenant_id"`
	TenantIDs   []string `json:"tenant_ids"`
	Roles       []string `json:"roles"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ExpiresAt   string   `json:"expires_at"`
}

func NewAdminAPIKeyHandler(store services.APIKeyStore) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{Store: store}
}

func (h *AdminAPIKeyHandler) RegisterRoutes(r chi.Router) {
	r.Post("/admin/api-keys", h.CreateAPIKey)
}

func (h *AdminAPIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Store == nil {
		http.Error(w, "api key store not configured", http.StatusInternalServerError)
		return
	}

	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		http.Error(w, "missing auth context", http.StatusUnauthorized)
		return
	}
	if !hasAdminRole(actor.Roles) {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return
	}

	var req adminAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	expiresAt, err := parseExpiresAt(req.ExpiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	key, _, err := h.Store.CreateKey(r.Context(), services.APIKeyCreateRequest{
		UserID:      strings.TrimSpace(req.UserID),
		TenantID:    strings.TrimSpace(req.TenantID),
		TenantIDs:   req.TenantIDs,
		Roles:       req.Roles,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		CreatedBy:   strings.TrimSpace(actor.UserID),
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"api_key": key})
}

func hasAdminRole(roles []string) bool {
	for _, role := range roles {
		switch strings.ToUpper(strings.TrimSpace(role)) {
		case "GLOBAL_OPS", "TENANT_ADMIN", "ADMIN":
			return true
		}
	}
	return false
}

func parseExpiresAt(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
