package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// TokenEncryptor handles AES-GCM encryption for sensitive tokens
type TokenEncryptor struct {
	block cipher.Block
}

// NewTokenEncryptor creates encryptor from 32-byte key
func NewTokenEncryptor(key []byte) (*TokenEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &TokenEncryptor{block: block}, nil
}

// Encrypt encrypts plaintext with AES-GCM
// Returns base64 encoded string
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	gcm, err := cipher.NewGCM(e.block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext with AES-GCM
// Expects base64 encoded string
func (e *TokenEncryptor) Decrypt(ciphertextB64 string) (string, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(e.block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// DecodeKey accepts a 32-byte raw key, a 64-char hex-encoded key, or a
// base64 (standard/URL-safe, padded/unpadded) encoding of 32 bytes. Returns
// the raw 32 bytes required for AES-256 / HMAC-SHA256.
func DecodeKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) == 32 {
		return []byte(s), nil
	}
	if len(s) == 64 {
		if raw, err := hex.DecodeString(s); err == nil && len(raw) == 32 {
			return raw, nil
		}
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if raw, err := enc.DecodeString(s); err == nil && len(raw) == 32 {
			return raw, nil
		}
	}
	return nil, fmt.Errorf("expected 32 raw bytes, 64 hex chars, or base64 of 32 bytes; got %d chars", len(s))
}

// LoadKeyFromEnv resolves a 32-byte server secret from envVar (raw/hex/base64).
// If unset and devFallbackEnvVar evaluates to "true", a process-lifetime
// random key is returned (logged as insecure — restarting the process
// invalidates anything sealed with it). Otherwise it errors, so callers fail
// closed rather than silently running with no real secret.
func LoadKeyFromEnv(envVar, devFallbackEnvVar string) ([]byte, error) {
	if v := os.Getenv(envVar); v != "" {
		return DecodeKey(v)
	}
	if os.Getenv(devFallbackEnvVar) == "true" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "[WARN] %s is unset; using a process-lifetime random key. Restarting will invalidate anything sealed with it. Set %s before deploying.\n", envVar, envVar)
		return key, nil
	}
	return nil, fmt.Errorf("%s is required; generate one with `openssl rand -base64 32`, or set %s=true for local development", envVar, devFallbackEnvVar)
}
