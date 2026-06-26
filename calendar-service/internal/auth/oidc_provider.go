package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCProvider handles OpenID Connect authentication
type OIDCProvider struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   oauth2.Config
	issuer   string
}

// OIDCConfig holds OIDC configuration
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// NewOIDCProvider initializes a new OIDC provider
func NewOIDCProvider(ctx context.Context, cfg OIDCConfig) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	oidcConfig := &oidc.Config{
		ClientID: cfg.ClientID,
	}
	verifier := provider.Verifier(oidcConfig)

	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	oauthConfig := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
	}

	return &OIDCProvider{
		provider: provider,
		verifier: verifier,
		config:   oauthConfig,
		issuer:   cfg.Issuer,
	}, nil
}

// GetAuthURL returns the OIDC login URL
func (p *OIDCProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// Exchange handles the code exchange and verifies the ID token
func (p *OIDCProvider) Exchange(ctx context.Context, code string) (*oidc.IDToken, error) {
	oauth2Token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token field in oauth2 token")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID Token: %w", err)
	}

	return idToken, nil
}
