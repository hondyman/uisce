package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
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
