package canon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// Canonicalize returns a deterministic JSON string for the given map.
// Keys are sorted alphabetically.
func Canonicalize(m map[string]any) (string, error) {
	// Deterministic key-order JSON
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]any, len(m))
	for _, k := range keys {
		out[k] = m[k]
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Hash generates a SHA-256 hash of the content, linked to a parent hash and schema version.
// Format: v={version}|p={parent}|c={content}
func Hash(content string, parent string, schemaVersion int) string {
	h := sha256.New()
	h.Write([]byte("v="))
	h.Write([]byte{byte(schemaVersion)})
	h.Write([]byte("|p="))
	h.Write([]byte(parent))
	h.Write([]byte("|c="))
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// Sign generates an HMAC-SHA256 signature for the content using a secret key.
func Sign(content string, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(content))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks if the signature is valid for the content.
func Verify(content string, signature string, secretKey string) bool {
	expectedMAC := Sign(content, secretKey)
	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
