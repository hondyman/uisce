package security

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/hondyman/uisce/backend/internal/logging"
)

// jwk is a single JSON Web Key from a JWKS endpoint (RSA keys only — the
// only key type Keycloak issues by default).
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

func (k jwk) publicKey() (*rsa.PublicKey, error) {
	if k.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type %q (only RSA is supported)", k.Kty)
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("invalid JWK modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("invalid JWK exponent: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

var jwksHTTPClient = &http.Client{Timeout: 5 * time.Second}

// FetchJWKS fetches and parses the RSA signing keys published at jwksURI
// (a standard OIDC JWKS endpoint, e.g. Keycloak's
// {base}/realms/{realm}/protocol/openid-connect/certs), keyed by "kid".
func FetchJWKS(jwksURI string) (map[string]*rsa.PublicKey, error) {
	resp, err := jwksHTTPClient.Get(jwksURI)
	if err != nil {
		return nil, fmt.Errorf("fetching JWKS from %s: %w", jwksURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint %s returned %d", jwksURI, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB cap
	if err != nil {
		return nil, fmt.Errorf("reading JWKS response from %s: %w", jwksURI, err)
	}

	var set jwkSet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, fmt.Errorf("parsing JWKS from %s: %w", jwksURI, err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kid == "" {
			continue
		}
		pub, err := k.publicKey()
		if err != nil {
			continue // skip unsupported/malformed keys, don't fail the whole set
		}
		keys[k.Kid] = pub
	}
	return keys, nil
}

// FetchAllTrustedKeys fetches and merges the signing keys for every active
// issuer in registry, keyed by "kid". A "kid" collision across two
// different issuers is a real (if unlikely — Keycloak generates random
// kids) risk: it would let a key from issuer A verify a token claiming to
// be from issuer B. On collision, the first issuer's key wins and the
// collision is logged rather than silently overwritten, so it's visible
// instead of a silent trust-boundary blur.
func FetchAllTrustedKeys(ctx context.Context, registry IssuerRegistry) (map[string]*rsa.PublicKey, error) {
	issuers, err := registry.ListActiveIssuers(ctx)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]*rsa.PublicKey)
	for _, iss := range issuers {
		keys, err := FetchJWKS(iss.JWKSURI)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("[jwks] failed fetching keys for issuer %q: %v", iss.Issuer, err)
			continue
		}
		for kid, key := range keys {
			if _, exists := merged[kid]; exists {
				logging.GetLogger().Sugar().Warnf("[jwks] kid %q collides across issuers; keeping first-seen key, this needs investigation", kid)
				continue
			}
			merged[kid] = key
		}
	}
	return merged, nil
}
