package api

import (
	"encoding/json"
	"net/http"

	"calendar-service/internal/auth"

	"github.com/sirupsen/logrus"
)

// SSOHandler handles enterprise authentication requests
type SSOHandler struct {
	samlProvider *auth.SAMLProvider
	oidcProvider *auth.OIDCProvider
	logger       *logrus.Entry
}

// NewSSOHandler creates a new SSO handler
func NewSSOHandler(
	samlProvider *auth.SAMLProvider,
	oidcProvider *auth.OIDCProvider,
	logger *logrus.Entry,
) *SSOHandler {
	return &SSOHandler{
		samlProvider: samlProvider,
		oidcProvider: oidcProvider,
		logger:       logger.WithField("component", "sso_handler"),
	}
}

// SAMLLogin handles GET /api/v1/auth/saml/login
func (h *SSOHandler) SAMLLogin(w http.ResponseWriter, r *http.Request) {
	if h.samlProvider == nil {
		writeJSONError(w, http.StatusNotImplemented, "SAML not configured")
		return
	}
	h.samlProvider.GetAuthURL(w, r)
}

// OIDCLogin handles GET /api/v1/auth/oidc/login
func (h *SSOHandler) OIDCLogin(w http.ResponseWriter, r *http.Request) {
	if h.oidcProvider == nil {
		writeJSONError(w, http.StatusNotImplemented, "OIDC not configured")
		return
	}
	state := r.URL.Query().Get("state")
	url := h.oidcProvider.GetAuthURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

// OIDCCallback handles GET /api/v1/auth/oidc/callback
func (h *SSOHandler) OIDCCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing code")
		return
	}

	idToken, err := h.oidcProvider.Exchange(r.Context(), code)
	if err != nil {
		h.logger.WithError(err).Error("OIDC exchange failed")
		writeJSONError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}

	// In a real app, we'd issue a local JWT here
	resp := map[string]interface{}{
		"subject": idToken.Subject,
		"issuer":  idToken.Issuer,
		"expiry":  idToken.Expiry,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
