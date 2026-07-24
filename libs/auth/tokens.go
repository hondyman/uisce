package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type HS256Verifier struct {
	Secret []byte
}

func NewHS256Verifier(secret []byte) *HS256Verifier {
	return &HS256Verifier{Secret: secret}
}

func (v *HS256Verifier) Verify(tokenString string) (*Claims, error) {
	return VerifyHS256(tokenString, v.Secret)
}

func SignHS256(claims *Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func VerifyHS256(tokenString string, secret []byte) (*Claims, error) {
	stripped := strings.TrimSpace(tokenString)
	if strings.HasPrefix(strings.ToLower(stripped), "bearer ") {
		stripped = strings.TrimSpace(stripped[7:])
	}
	if stripped == "" {
		return nil, ErrMissingToken
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(stripped, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("token parse failed: %w", err)
	}
	if !token.Valid {
		return nil, ErrInvalidSignature
	}
	return claims, nil
}

func ExtractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("invalid authorization header format")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("empty token")
	}
	return token, nil
}

func ValidateRequestToken(r *http.Request, secret []byte) (*Claims, error) {
	token, err := ExtractBearerToken(r)
	if err != nil {
		return nil, err
	}
	return VerifyHS256(token, secret)
}

type JWKSCache struct {
	mu        sync.RWMutex
	keySets   map[string]*jwt_jwk_Set
	expiries  map[string]time.Time
ttl       time.Duration
}

type jwt_jwk_Set struct {
	Keys []jwk_Key `json:"keys"`
}

type jwk_Key struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewJWKSCache(ttl time.Duration) *JWKSCache {
	return &JWKSCache{
		keySets:  make(map[string]*jwt_jwk_Set),
		expiries: make(map[string]time.Time),
		ttl:      ttl,
	}
}

func (c *JWKSCache) Get(jwksURL string) (*jwt_jwk_Set, error) {
	c.mu.RLock()
	if set, ok := c.keySets[jwksURL]; ok {
		if time.Now().Before(c.expiries[jwksURL]) {
			c.mu.RUnlock()
			return set, nil
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if set, ok := c.keySets[jwksURL]; ok && time.Now().Before(c.expiries[jwksURL]) {
		return set, nil
	}

	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var set jwt_jwk_Set
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	c.keySets[jwksURL] = &set
	c.expiries[jwksURL] = time.Now().Add(c.ttl)
	return &set, nil
}

func (c *JWKSCache) GetRSAPublicKey(jwksURL, kid string) (*rsa.PublicKey, error) {
	set, err := c.Get(jwksURL)
	if err != nil {
		return nil, err
	}
	for _, key := range set.Keys {
		if key.Kid == kid && key.Kty == "RSA" {
			return parseRSAPublicKey(key.N, key.E)
		}
	}
	return nil, fmt.Errorf("key %s not found in JWKS", kid)
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	return &rsa.PublicKey{N: n, E: e}, nil
}

type RS256Verifier struct {
	Cache    *JWKSCache
	JWKSURL  string
	GetKey   func(jwksURL, kid string) (*rsa.PublicKey, error)
}

func NewRS256Verifier(jwksURL string) *RS256Verifier {
	cache := NewJWKSCache(15 * time.Minute)
	return &RS256Verifier{
		Cache:   cache,
		JWKSURL: jwksURL,
		GetKey:  cache.GetRSAPublicKey,
	}
}

func (v *RS256Verifier) Verify(tokenString string) (*Claims, error) {
	stripped := strings.TrimSpace(tokenString)
	if strings.HasPrefix(strings.ToLower(stripped), "bearer ") {
		stripped = strings.TrimSpace(stripped[7:])
	}

	var kid string
	_, _ = jwt.Parse(stripped, func(token *jwt.Token) (interface{}, error) {
		if kidStr, ok := token.Header["kid"].(string); ok {
			kid = kidStr
		}
		return nil, nil
	})

	if kid == "" {
		return nil, errors.New("token missing kid header")
	}

	pubKey, err := v.GetKey(v.JWKSURL, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(stripped, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("token parse failed: %w", err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidSignature
	}
	return claims, nil
}

func SignRS256(claims *Claims, privateKey *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

type TokenVerifier interface {
	Verify(tokenString string) (*Claims, error)
}

func GetJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET not configured")
	}
	return []byte(secret), nil
}
