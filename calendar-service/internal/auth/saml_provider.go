package auth

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
)

// SAMLProvider handles SAML 2.0 authentication
type SAMLProvider struct {
	sp *samlsp.Middleware
}

// SAMLConfig holds configuration for SAML
type SAMLConfig struct {
	MetadataURL string
	BaseURL     string
	EntityID    string
	Certificate string
	PrivateKey  string
}

// NewSAMLProvider initializes a new SAML SP
func NewSAMLProvider(cfg SAMLConfig) (*SAMLProvider, error) {
	idpMetadataURL, err := url.Parse(cfg.MetadataURL)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata url: %w", err)
	}

	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient, *idpMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}

	keyPair, err := tls.X509KeyPair([]byte(cfg.Certificate), []byte(cfg.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("load keypair: %w", err)
	}
	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse leaf cert: %w", err)
	}

	rootURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	sp, err := samlsp.New(samlsp.Options{
		URL:               *rootURL,
		Key:               keyPair.PrivateKey.(crypto.Signer),
		Certificate:       keyPair.Leaf,
		IDPMetadata:       idpMetadata,
		EntityID:          cfg.EntityID,
		AllowIDPInitiated: true,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize saml sp: %w", err)
	}

	return &SAMLProvider{sp: sp}, nil
}

// GetAuthURL returns the IDP login URL
func (p *SAMLProvider) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	p.sp.HandleStartAuthFlow(w, r)
}

// HandleCallback processes the SAML assertion
func (p *SAMLProvider) HandleCallback(w http.ResponseWriter, r *http.Request) (*saml.Assertion, error) {
	// This is a simplified wrapper. The samlsp middleware usually handles this via ServeHTTP.
	// For custom integration, we'd use p.sp.ServiceProvider.ParseResponse
	return nil, fmt.Errorf("not implemented")
}
