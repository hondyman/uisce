package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// refreshKeycloakJWKS fetches the current Keycloak JWKS document and loads
// every RSA signing key into secMgr, keyed by "kid". Called at startup and on
// a periodic timer so a Keycloak key rotation is picked up without a
// backend restart — see security_manager.go's ValidateToken, which now
// selects the key by the token's own "kid" header rather than trusting
// whichever key happened to be fetched first.
func refreshKeycloakJWKS(secMgr *services.SecurityManager, jwksURL string) error {
	tr := &http.Transport{}
	if os.Getenv("SKIP_TLS_VERIFY") == "true" {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Transport: tr, Timeout: 10 * 1e9}
	resp, err := client.Get(jwksURL)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read JWKS response: %w", err)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("parse JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" || (key.Alg != "" && key.Alg != "RS256") {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			log.Printf("[WARN] Failed to decode RSA modulus for kid=%s: %v", key.Kid, err)
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			log.Printf("[WARN] Failed to decode RSA exponent for kid=%s: %v", key.Kid, err)
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		keys[key.Kid] = &rsa.PublicKey{N: n, E: e}
	}

	if len(keys) == 0 {
		return fmt.Errorf("JWKS response contained no usable RSA keys")
	}

	secMgr.SetRSAPublicKeys(keys)
	log.Printf("[INFO] Loaded %d Keycloak RS256 key(s) from JWKS", len(keys))
	return nil
}

// TenantContext represents extracted tenant context
type TenantContext struct {
	TenantID     string
	DatasourceID string
}

// allowClientTenantHeaderFallback reports whether the X-Tenant-ID header may
// be trusted when the JWT carries no tenant claim. This must never be enabled
// in production: the header is fully client-controlled, so trusting it lets
// any caller assert an arbitrary tenant identity.
func allowClientTenantHeaderFallback() bool {
	return getEnv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false") == "true"
}

// extractTenantContext extracts tenant context from request headers (JWT-validated)
// WARNING: This function intentionally does NOT fall back to URL query params for security.
// Tenant ID must come from validated JWT claims; the X-Tenant-ID header is only
// trusted when ALLOW_CLIENT_TENANT_HEADER_FALLBACK=true (local/dev use only).
func extractTenantContext(r *http.Request) (*TenantContext, error) {
	var tenantID string
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if tenantID == "" && allowClientTenantHeaderFallback() {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		return nil, fmt.Errorf("tenant context not found: X-Tenant-ID and X-Tenant-Datasource-ID headers required")
	}

	return &TenantContext{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
	}, nil
}

// writeJSONError writes a structured JSON error response with the given status code.
func writeJSONError(w http.ResponseWriter, status int, msg string, errorCode string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:     msg,
		Code:      status,
		ErrorCode: errorCode,
		Details:   details,
	})
}

// getEnv returns the environment variable value if set; otherwise returns defaultValue.
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// buildApiDispatcherEncryptor constructs the AES-GCM TokenEncryptor used by the
// API dispatcher to encrypt/decrypt per-tenant API credentials at rest.
//
// Behavior:
//   - If API_TOKEN_ENCRYPTION_KEY is set and decodes to exactly 32 bytes (raw
//     or base64), that key is used.
//   - If API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK=true is set (or the server is
//     running in a non-prod mode), a process-lifetime random key is generated
//     and a warning is logged. THIS IS INSECURE — restarting the server will
//     invalidate every saved credential. Never enable in production.
//   - Otherwise the function returns an error and the server refuses to start.
func buildApiDispatcherEncryptor() (*security.TokenEncryptor, error) {
	keyEnv := os.Getenv("API_TOKEN_ENCRYPTION_KEY")
	allowDevFallback := getEnv("API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK", "false") == "true"

	if keyEnv != "" {
		key, err := decodeEncryptionKey(keyEnv)
		if err != nil {
			return nil, fmt.Errorf("API_TOKEN_ENCRYPTION_KEY is set but invalid: %w", err)
		}
		return security.NewTokenEncryptor(key)
	}

	if allowDevFallback {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate dev fallback key: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[WARN] API_TOKEN_ENCRYPTION_KEY is unset; using process-lifetime random key. Saved credentials will be unreadable after restart. Set API_TOKEN_ENCRYPTION_KEY before deploying.\n")
		return security.NewTokenEncryptor(key)
	}

	return nil, fmt.Errorf("API_TOKEN_ENCRYPTION_KEY is required; generate one with `openssl rand -base64 32` or set API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK=true for local development")
}

// decodeEncryptionKey accepts a 32-byte raw key, a 32-byte hex-encoded key,
// or a base64 (standard or URL-safe) encoding of a 32-byte key. Returns the
// raw 32 bytes that AES-256-GCM requires.
func decodeEncryptionKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) == 32 {
		return []byte(s), nil
	}
	if len(s) == 64 {
		raw, err := hex.DecodeString(s)
		if err == nil && len(raw) == 32 {
			return raw, nil
		}
	}
	if raw, err := base64.StdEncoding.DecodeString(s); err == nil && len(raw) == 32 {
		return raw, nil
	}
	if raw, err := base64.RawStdEncoding.DecodeString(s); err == nil && len(raw) == 32 {
		return raw, nil
	}
	if raw, err := base64.URLEncoding.DecodeString(s); err == nil && len(raw) == 32 {
		return raw, nil
	}
	if raw, err := base64.RawURLEncoding.DecodeString(s); err == nil && len(raw) == 32 {
		return raw, nil
	}
	return nil, fmt.Errorf("expected 32 raw bytes, 64 hex chars, or base64 of 32 bytes; got %d chars", len(s))
}

// respond is a small helper used across handlers to write JSON responses.
// It accepts a value (data) and an error; if error is non-nil it writes a
// structured JSON error response, otherwise it serializes the data as JSON.
func respond(w http.ResponseWriter, _r *http.Request, data interface{}, err error) {
	if err != nil {
		// If the error is an httpError with status, we could extract it —
		// keep it simple here and return 500 for now.
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "internal_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if data == nil {
		// Write an empty JSON object for nil data
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	_ = json.NewEncoder(w).Encode(data)
}

// toTitleCase converts snake/camel/underscore names into a human-friendly title.
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	// Replace underscores/dashes with spaces
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	parts := strings.Fields(s)
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

// sanitizeViewPayload is a permissive passthrough for view payloads. The
// real implementation may trim or filter fields for client-safe responses;
// for now return the payload unchanged.
func sanitizeViewPayload(v interface{}) interface{} {
	return v
}

// fileETag returns a simple ETag string for file bytes and FileInfo. Use
// the file's modification time and size with a tiny fingerprint of the
// payload to detect changes.
func fileETag(b []byte, fi os.FileInfo) string {
	if fi == nil {
		return ""
	}
	h := ""
	if len(b) > 0 {
		// Use a short hex prefix of the content for a lightweight fingerprint
		prefix := 8
		if len(b) < prefix {
			prefix = len(b)
		}
		h = hex.EncodeToString(b[:prefix])
	}
	return fmt.Sprintf("%d-%d-%s", fi.ModTime().Unix(), fi.Size(), h)
}

// parseIntDefault parses a string into int, returning defaultVal on error.
func parseIntDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}

// errorsIs is a small wrapper around errors.Is for call sites that expect this helper
func errorsIs(err, target error) bool {
	return errors.Is(err, target)
}

// generateRandomToken returns a URL-safe random token
func generateRandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}

// generateJobID returns a new UUID-based job id
func generateJobID() string {
	return uuid.New().String()
}

// nilIfNullInt64 returns a pointer to int64 if valid, otherwise nil
func nilIfNullInt64(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

// nilIfNullFloat64 returns a pointer to float64 if valid, otherwise nil
func nilIfNullFloat64(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// getSecureTenantID extracts tenant ID from validated JWT claims.
// SECURITY: This function intentionally does NOT fall back to URL query parameters,
// and only trusts the client-supplied X-Tenant-ID header when
// ALLOW_CLIENT_TENANT_HEADER_FALLBACK=true (local/dev use only).
func getSecureTenantID(r *http.Request) string {
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		return claims.TenantID
	}
	if allowClientTenantHeaderFallback() {
		if tid := r.Header.Get("X-Tenant-ID"); tid != "" {
			return tid
		}
	}
	return ""
}

// respondJSON responds with JSON
func respondJSON(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// nullString returns a pointer to the string if it's not empty, otherwise nil
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
