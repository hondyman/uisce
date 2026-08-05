package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"calendar-service/internal/database"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/crewjam/saml"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type SSOService struct {
	dbClient      *database.Client
	logger        *logrus.Entry
	samlProviders map[string]*saml.ServiceProvider
	oidcProviders map[string]*oidc.Provider
}

type SSOServiceConfig struct {
	DBClient *database.Client
	Logger   *logrus.Entry
}

func NewSSOService(cfg SSOServiceConfig) *SSOService {
	return &SSOService{
		dbClient:      cfg.DBClient,
		logger:        cfg.Logger.WithField("component", "sso_service"),
		samlProviders: make(map[string]*saml.ServiceProvider),
		oidcProviders: make(map[string]*oidc.Provider),
	}
}

// SSOProvider represents an SSO provider configuration
type SSOProvider struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	ProviderType  string `json:"provider_type"`
	ProviderName  string `json:"provider_name"`
	IsActive      bool   `json:"is_active"`
	IsPrimary     bool   `json:"is_primary"`
	AutoProvision bool   `json:"auto_provision_users"`
	DefaultRole   string `json:"default_user_role"`

	// SAML
	SAMLEntityID    string `json:"saml_entity_id"`
	SAMLSSOURL      string `json:"saml_sso_url"`
	SAMLCertificate string `json:"saml_certificate"`

	// OIDC
	OIDCIssuer       string   `json:"oidc_issuer"`
	OIDCClientID     string   `json:"oidc_client_id"`
	OIDCClientSecret string   `json:"oidc_client_secret"`
	OIDCRedirectURI  string   `json:"oidc_redirect_uri"`
	OIDCScopes       []string `json:"oidc_scopes"`
}

// User represents a system user
type User struct {
	ID        string
	TenantID  string
	Email     string
	IDPUserID string
}

// InitializeProvider initializes an SSO provider
func (s *SSOService) InitializeProvider(ctx context.Context, provider *SSOProvider) error {
	switch provider.ProviderType {
	case "saml":
		return s.initializeSAMLProvider(ctx, provider)
	case "oidc":
		return s.initializeOIDCProvider(ctx, provider)
	default:
		return fmt.Errorf("unsupported provider type: %s", provider.ProviderType)
	}
}

// initializeSAMLProvider initializes a SAML provider
func (s *SSOService) initializeSAMLProvider(ctx context.Context, provider *SSOProvider) error {
	// Create SAML SP
	sp := &saml.ServiceProvider{
		EntityID: provider.SAMLEntityID,
		AcsURL:   *mustParseURL(fmt.Sprintf("%s/api/v1/sso/saml/acs", getBaseURL())),
	}

	s.samlProviders[provider.ID] = sp
	s.logger.WithField("provider_id", provider.ID).Info("SAML provider initialized")

	return nil
}

// initializeOIDCProvider initializes an OIDC provider
func (s *SSOService) initializeOIDCProvider(ctx context.Context, provider *SSOProvider) error {
	oidcProvider, err := oidc.NewProvider(ctx, provider.OIDCIssuer)
	if err != nil {
		return fmt.Errorf("create OIDC provider: %w", err)
	}

	s.oidcProviders[provider.ID] = oidcProvider
	s.logger.WithField("provider_id", provider.ID).Info("OIDC provider initialized")

	return nil
}

func (s *SSOService) storeSAMLSessionState(ctx context.Context, sessionID, providerID string) {
	// Not implemented: Should store in Redis/DB for verification on ACS
}

func (s *SSOService) storeOIDCSessionState(ctx context.Context, state, providerID string) {
	// Not implemented: Should store in Redis/DB for verification on callback
}

func (s *SSOService) getProviderConfig(ctx context.Context, providerID string) (*SSOProvider, error) {
	query := `
		SELECT id, tenant_id, provider_type, provider_name, is_active, is_primary,
			auto_provision_users, default_user_role, saml_entity_id, saml_sso_url,
			saml_certificate, oidc_issuer, oidc_client_id, oidc_client_secret,
			oidc_redirect_uri, oidc_scopes
		FROM sso_providers WHERE id = $1`

	var p SSOProvider
	var oidcScopes []byte
	err := s.dbClient.Pool().QueryRow(ctx, query, providerID).Scan(
		&p.ID, &p.TenantID, &p.ProviderType, &p.ProviderName, &p.IsActive, &p.IsPrimary,
		&p.AutoProvision, &p.DefaultRole, &p.SAMLEntityID, &p.SAMLSSOURL,
		&p.SAMLCertificate, &p.OIDCIssuer, &p.OIDCClientID, &p.OIDCClientSecret,
		&p.OIDCRedirectURI, &oidcScopes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query sso provider: %w", err)
	}
	if p.ID == "" {
		return nil, fmt.Errorf("sso provider not found: %s", providerID)
	}
	json.Unmarshal(oidcScopes, &p.OIDCScopes)
	return &p, nil
}

// ensureProviderLoaded ensures the provider is initialized in memory
func (s *SSOService) ensureProviderLoaded(ctx context.Context, providerID string) error {
	if _, exists := s.samlProviders[providerID]; exists {
		return nil
	}
	if _, exists := s.oidcProviders[providerID]; exists {
		return nil
	}

	provider, err := s.getProviderConfig(ctx, providerID)
	if err != nil {
		return err
	}

	return s.InitializeProvider(ctx, provider)
}

// HandleSAMLLogin initiates SAML login flow
func (s *SSOService) HandleSAMLLogin(w http.ResponseWriter, r *http.Request, providerID string) {
	if err := s.ensureProviderLoaded(r.Context(), providerID); err != nil {
		http.Error(w, "SSO provider unavailable", http.StatusNotFound)
		return
	}

	sp := s.samlProviders[providerID]

	// Generate session ID
	sessionID := generateSessionID()

	// Store session state
	s.storeSAMLSessionState(r.Context(), sessionID, providerID)

	// Redirect to IdP
	redirectURL, err := sp.MakeRedirectAuthenticationRequest(sessionID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create SAML redirect")
		http.Error(w, "Failed to initiate SSO", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// HandleSAMLACS handles SAML ACS callback
func (s *SSOService) HandleSAMLACS(w http.ResponseWriter, r *http.Request, providerID string) {
	if err := s.ensureProviderLoaded(r.Context(), providerID); err != nil {
		http.Error(w, "SSO provider unavailable", http.StatusNotFound)
		return
	}

	sp := s.samlProviders[providerID]

	r.ParseForm()

	// Parse SAML response
	assertion, err := sp.ParseResponse(r, []string{""})
	if err != nil {
		s.logger.WithError(err).Error("Failed to parse SAML response")
		http.Error(w, "SSO authentication failed", http.StatusForbidden)
		return
	}

	// Extract user info
	userID := getUserIDFromAssertion(assertion)
	email := getEmailFromAssertion(assertion)
	attributes := getAttributesFromAssertion(assertion)

	// Create or update user
	user, err := s.getOrCreateUser(r.Context(), providerID, userID, email, attributes)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create/update user")
		http.Error(w, "Failed to create user account", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionToken, err := s.createSSOSession(r.Context(), user, providerID, assertion)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Redirect to app with token
	http.Redirect(w, r, fmt.Sprintf("/dashboard?token=%s", sessionToken), http.StatusFound)
}

func getUserIDFromAssertion(assertion *saml.Assertion) string {
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		return assertion.Subject.NameID.Value
	}
	return ""
}

func getEmailFromAssertion(assertion *saml.Assertion) string {
	// Try to find email in attributes
	for _, attrStatement := range assertion.AttributeStatements {
		for _, attr := range attrStatement.Attributes {
			if attr.Name == "email" || attr.Name == "mail" || attr.Name == "Email" {
				if len(attr.Values) > 0 {
					return attr.Values[0].Value
				}
			}
		}
	}
	return "sso-user@example.com"
}

func getAttributesFromAssertion(assertion *saml.Assertion) map[string]interface{} {
	attrs := make(map[string]interface{})
	for _, attrStatement := range assertion.AttributeStatements {
		for _, attr := range attrStatement.Attributes {
			if len(attr.Values) > 0 {
				attrs[attr.Name] = attr.Values[0].Value
			}
		}
	}
	return attrs
}

// HandleOIDCLogin initiates OIDC login flow
func (s *SSOService) HandleOIDCLogin(w http.ResponseWriter, r *http.Request, providerID string) {
	if err := s.ensureProviderLoaded(r.Context(), providerID); err != nil {
		http.Error(w, "SSO provider unavailable", http.StatusNotFound)
		return
	}

	provider := s.oidcProviders[providerID]

	// Get provider config
	providerConfig, err := s.getProviderConfig(r.Context(), providerID)
	if err != nil {
		http.Error(w, "Failed to get provider config", http.StatusInternalServerError)
		return
	}

	oauth2Config := &oauth2.Config{
		ClientID:     providerConfig.OIDCClientID,
		ClientSecret: providerConfig.OIDCClientSecret,
		RedirectURL:  providerConfig.OIDCRedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       append(providerConfig.OIDCScopes, oidc.ScopeOpenID),
	}

	// Generate state
	state := generateSessionID()
	s.storeOIDCSessionState(r.Context(), state, providerID)

	// Redirect to IdP
	authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleOIDCCallback handles OIDC callback
func (s *SSOService) HandleOIDCCallback(w http.ResponseWriter, r *http.Request, providerID string) {
	if err := s.ensureProviderLoaded(r.Context(), providerID); err != nil {
		http.Error(w, "SSO provider unavailable", http.StatusNotFound)
		return
	}

	provider := s.oidcProviders[providerID]

	// Get provider config
	providerConfig, err := s.getProviderConfig(r.Context(), providerID)
	if err != nil {
		http.Error(w, "Failed to get provider config", http.StatusInternalServerError)
		return
	}

	oauth2Config := &oauth2.Config{
		ClientID:     providerConfig.OIDCClientID,
		ClientSecret: providerConfig.OIDCClientSecret,
		RedirectURL:  providerConfig.OIDCRedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       providerConfig.OIDCScopes,
	}

	// Exchange code for token
	oauth2Token, err := oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		s.logger.WithError(err).Error("Failed to exchange code")
		http.Error(w, "SSO authentication failed", http.StatusForbidden)
		return
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID token in response", http.StatusInternalServerError)
		return
	}

	// Verify ID token
	verifier := provider.Verifier(&oidc.Config{ClientID: providerConfig.OIDCClientID})
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		s.logger.WithError(err).Error("Failed to verify ID token")
		http.Error(w, "SSO authentication failed", http.StatusForbidden)
		return
	}

	// Extract claims
	var claims struct {
		Sub    string   `json:"sub"`
		Email  string   `json:"email"`
		Name   string   `json:"name"`
		Groups []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		s.logger.WithError(err).Error("Failed to extract claims")
		http.Error(w, "Failed to extract user info", http.StatusInternalServerError)
		return
	}

	// Create or update user
	user, err := s.getOrCreateUser(r.Context(), providerID, claims.Sub, claims.Email, map[string]interface{}{
		"name":   claims.Name,
		"groups": claims.Groups,
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to create/update user")
		http.Error(w, "Failed to create user account", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionToken, err := s.createSSOSession(r.Context(), user, providerID, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Redirect to app with token
	http.Redirect(w, r, fmt.Sprintf("/dashboard?token=%s", sessionToken), http.StatusFound)
}

func (s *SSOService) getUserByIDPUser(ctx context.Context, providerID, idpUserID string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.tenant_id
		FROM users u
		JOIN sso_sessions ss ON u.id = ss.user_id
		WHERE ss.sso_provider_id = $1 AND ss.idp_user_id = $2
		LIMIT 1`

	var u User
	err := s.dbClient.Pool().QueryRow(ctx, query, providerID, idpUserID).Scan(&u.ID, &u.Email, &u.TenantID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return &u, nil
}

func (s *SSOService) createUser(ctx context.Context, tenantID, email string, attributes map[string]interface{}, role string) (*User, error) {
	username := email
	if name, ok := attributes["name"].(string); ok && name != "" {
		username = name
	}

	query := `INSERT INTO users (id, tenant_id, email, username, is_active, tenant_scope, created_at) VALUES (gen_random_uuid(), $1, $2, $3, true, 'single', NOW()) RETURNING id, email, tenant_id`

	var u User
	err := s.dbClient.Pool().QueryRow(ctx, query, tenantID, email, username).Scan(&u.ID, &u.Email, &u.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to provision user: %w", err)
	}
	return &u, nil
}

// Helper functions
func (s *SSOService) getOrCreateUser(ctx context.Context, providerID, idpUserID, email string, attributes map[string]interface{}) (*User, error) {
	// Check if user exists
	user, err := s.getUserByIDPUser(ctx, providerID, idpUserID)
	if err == nil {
		return user, nil
	}

	// Get provider config
	provider, err := s.getProviderConfig(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// Auto-provision if enabled
	if !provider.AutoProvision {
		return nil, fmt.Errorf("user not found and auto-provisioning disabled")
	}

	// Create new user
	user, err = s.createUser(ctx, provider.TenantID, email, attributes, provider.DefaultRole)
	if err != nil {
		return nil, err
	}
	user.IDPUserID = idpUserID
	return user, nil
}

func (s *SSOService) createSSOSession(ctx context.Context, user *User, providerID string, samlAssertion *saml.Assertion) (string, error) {
	sessionID := generateSessionID()
	expiresAt := time.Now().Add(24 * time.Hour)

	var idpAttributes []byte
	if samlAssertion != nil {
		idpAttributes, _ = json.Marshal(samlAssertion)
	}

	query := `INSERT INTO sso_sessions (id, tenant_id, user_id, sso_provider_id, session_id, idp_user_id, idp_email, idp_attributes, expires_at, last_activity_at, created_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`
	_, err := s.dbClient.Pool().Exec(ctx, query, user.TenantID, user.ID, providerID, sessionID, user.IDPUserID, user.Email, string(idpAttributes), expiresAt, time.Now())
	return sessionID, err
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func getBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		return "http://localhost:8081"
	}
	return baseURL
}
